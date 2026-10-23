package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cris-17-21/Orbit/internal/atomicmemory"
	"github.com/Cris-17-21/Orbit/internal/claudecodeconfig"
	"github.com/Cris-17-21/Orbit/internal/codexconfig"
	"github.com/Cris-17-21/Orbit/internal/initmemory"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmationFunc func(initmemory.Plan) (bool, error)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Orbit: no se pudo determinar el directorio actual:", err)
		os.Exit(1)
	}

	code := execute(os.Args[1:], root, os.Stdout, os.Stderr, isInteractive(os.Stdin), func(plan initmemory.Plan) (bool, error) {
		return confirmPlan(plan, os.Stdin, os.Stdout)
	})
	os.Exit(code)
}

func execute(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc) int {
	return executeWithCodexLister(args, workingDirectory, stdout, stderr, interactive, confirm, listCodexServers)
}

type codexServerLister func() ([]string, error)

func executeWithCodexLister(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc, listServers codexServerLister) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], workingDirectory, stdout, stderr, interactive, confirm)
	case "memory":
		return runMemory(args[1:], workingDirectory, stdout, stderr, interactive, confirm, listServers)
	case "help":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "Orbit: comando desconocido %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runMemory(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc, listServers codexServerLister) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printMemoryHelp(stdout)
		return 0
	}
	if args[0] != "configure" {
		fmt.Fprintf(stderr, "Orbit: subcomando de memoria desconocido %q\n", args[0])
		printMemoryHelp(stderr)
		return 2
	}
	if len(args) < 2 {
		fmt.Fprintln(stderr, "Orbit: indica un agente para configurar Atomic Memory")
		fmt.Fprintln(stderr)
		printMemoryHelp(stderr)
		return 2
	}
	switch args[1] {
	case "codex":
		return runMemoryConfigureCodex(args[2:], workingDirectory, stdout, stderr, interactive, confirm, listServers)
	case "claude":
		return runMemoryConfigureClaude(args[2:], workingDirectory, stdout, stderr, interactive, confirm)
	default:
		fmt.Fprintf(stderr, "Orbit: todavía no se puede configurar Atomic Memory para %q\n\n", args[1])
		printMemoryHelp(stderr)
		return 2
	}
}

func runMemoryConfigureClaude(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc) int {
	flags := flag.NewFlagSet("orbit memory configure claude", flag.ContinueOnError)
	flags.SetOutput(stderr)
	backend := flags.String("backend", "", "backend: hosted-cloud, connected-local o self-hosted")
	apiURL := flags.String("api-url", "", "URL de API para connected-local o self-hosted")
	yes := flags.Bool("yes", false, "aplica el plan mostrado sin pedir confirmación interactiva")
	flags.Usage = func() { printMemoryHelp(stderr) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", flags.Arg(0))
		printMemoryHelp(stderr)
		return 2
	}
	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Orbit: no se encontró %s; ejecuta primero `orbit init`\n", filepath.ToSlash(filepath.Join(".orbit", "atomic-memory.json")))
		return 1
	}
	config, err := atomicmemory.ParseConfig(configBytes)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo leer la configuración de Atomic Memory:", err)
		return 1
	}
	previousBackend := config.Backend
	config.Mode = atomicmemory.ModePersonal
	if *backend != "" {
		config.Backend = atomicmemory.Backend(*backend)
		if config.Backend != previousBackend {
			config.APIURL = ""
		}
	} else if config.Backend == "" {
		config.Backend = atomicmemory.BackendHostedCloud
	}
	if *apiURL != "" {
		config.APIURL = *apiURL
	}
	if err := config.Validate(); err != nil {
		fmt.Fprintln(stderr, "Orbit: configuración inválida:", err)
		return 2
	}
	newConfigBytes, err := config.Marshal()
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo preparar la configuración de Atomic Memory:", err)
		return 1
	}
	serverBytes, err := claudecodeconfig.BuildServerConfig(config)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo preparar la configuración MCP:", err)
		return 1
	}
	entryPlan, err := initmemory.PlanManagedJSONEntry(root, initmemory.ManagedJSONEntryUpdate{
		Path: ".mcp.json", Section: "mcpServers", Key: claudecodeconfig.ServerName,
		ResourceID: "project-claude-atomic-memory-config", Source: "orbit-memory-configure", Content: serverBytes,
	})
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo revisar .mcp.json:", err)
		return 1
	}
	if entryPlan.Entry.Status == initmemory.StatusConflict {
		fmt.Fprintf(stderr, "Orbit: conflicto en %s: %s; no se aplicaron cambios\n", entryPlan.Entry.Path, entryPlan.Entry.Reason)
		return 1
	}
	entryUpdates := []initmemory.ManagedJSONEntryUpdate{}
	if entryPlan.Entry.Status != initmemory.StatusPresent {
		entryUpdates = append(entryUpdates, entryPlan.Update)
	}
	updates := []initmemory.ManagedFileUpdate{{
		Path: ".orbit/atomic-memory.json", ResourceID: "project-atomic-memory-config", Source: "orbit-memory-configure", Content: newConfigBytes,
	}}
	plan := initmemory.Plan{Root: root, Entries: []initmemory.Entry{{Path: ".orbit/atomic-memory.json", Status: initmemory.StatusModified}}}
	if entryPlan.Entry.Status == initmemory.StatusPresent {
		fmt.Fprintln(stdout, "Claude Code ya tiene la entrada atomicmemory de Orbit; no se duplicará.")
	} else {
		plan.Entries = append(plan.Entries, entryPlan.Entry)
		fmt.Fprintln(stdout, "Claude Code agregará Atomic Memory al alcance del proyecto.")
	}
	fmt.Fprintf(stdout, "\nBackend propuesto: %s\n", config.Backend)
	for _, entry := range plan.Entries {
		verb := "actualizar"
		if entry.Status == initmemory.StatusCreate {
			verb = "agregar"
		}
		fmt.Fprintf(stdout, "  • %s %s\n", verb, entry.Path)
	}
	if entryPlan.Entry.Status == initmemory.StatusCreate {
		fmt.Fprintln(stdout, "Claude Code puede pedirte aprobar el servidor al abrir este proyecto.")
	}
	if !*yes {
		if !interactive {
			fmt.Fprintln(stderr, "Orbit: este comando requiere confirmación; ejecútalo en una terminal o pasa --yes para aplicar el plan mostrado")
			return 2
		}
		if confirm == nil {
			fmt.Fprintln(stderr, "Orbit: la confirmación interactiva no está disponible")
			return 2
		}
		accepted, confirmErr := confirm(plan)
		if confirmErr != nil {
			fmt.Fprintln(stderr, "Orbit: falló la confirmación:", confirmErr)
			return 1
		}
		if !accepted {
			fmt.Fprintln(stdout, "Cancelado. No se escribió ningún archivo.")
			return 0
		}
	}
	if err := initmemory.UpdateManagedFilesWithJSONEntries(root, updates, entryUpdates); err != nil {
		fmt.Fprintln(stderr, "Orbit: no se aplicó la configuración:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Configuración guardada. Las credenciales y la identidad personal no se copiaron al repositorio.")
	fmt.Fprintln(stdout, "Esta integración instala herramientas MCP; el protocolo de memoria de Orbit se agregará en una tarea posterior.")
	return 0
}

func runMemoryConfigureCodex(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc, listServers codexServerLister) int {
	flags := flag.NewFlagSet("orbit memory configure codex", flag.ContinueOnError)
	flags.SetOutput(stderr)
	backend := flags.String("backend", "", "backend: hosted-cloud, connected-local o self-hosted")
	apiURL := flags.String("api-url", "", "URL de API para connected-local o self-hosted")
	yes := flags.Bool("yes", false, "aplica el plan mostrado sin pedir confirmación interactiva")
	flags.Usage = func() { printMemoryHelp(stderr) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", flags.Arg(0))
		printMemoryHelp(stderr)
		return 2
	}
	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "Orbit: no se encontró %s; ejecuta primero `orbit init`\n", filepath.ToSlash(filepath.Join(".orbit", "atomic-memory.json")))
		return 1
	}
	config, err := atomicmemory.ParseConfig(configBytes)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo leer la configuración de Atomic Memory:", err)
		return 1
	}
	previousBackend := config.Backend
	config.Mode = atomicmemory.ModePersonal
	if *backend != "" {
		config.Backend = atomicmemory.Backend(*backend)
		if config.Backend != previousBackend {
			config.APIURL = ""
		}
	} else if config.Backend == "" {
		config.Backend = atomicmemory.BackendHostedCloud
	}
	if *apiURL != "" {
		config.APIURL = *apiURL
	}
	if err := config.Validate(); err != nil {
		fmt.Fprintln(stderr, "Orbit: configuración inválida:", err)
		return 2
	}
	newConfigBytes, err := config.Marshal()
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo preparar la configuración de Atomic Memory:", err)
		return 1
	}
	if listServers == nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo consultar Codex; no se aplicaron cambios")
		return 1
	}
	servers, err := listServers()
	if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo consultar la lista MCP de Codex; no se aplicaron cambios")
		return 1
	}

	updates := []initmemory.ManagedFileUpdate{{
		Path: ".orbit/atomic-memory.json", ResourceID: "project-atomic-memory-config", Source: "orbit-memory-configure", Content: newConfigBytes,
	}}
	plan := initmemory.Plan{Root: root, Entries: []initmemory.Entry{{Path: ".orbit/atomic-memory.json", Status: initmemory.StatusModified}}}
	if codexconfig.HasAtomicMemoryServer(servers) {
		fmt.Fprintln(stdout, "Codex ya tiene un servidor llamado atomicmemory; Orbit no agregará otro ni leerá sus valores de entorno.")
		fmt.Fprintln(stdout, "Orbit actualizará solo la modalidad de Atomic Memory del proyecto.")
	} else {
		codexBytes, buildErr := codexconfig.BuildConfig(config)
		if buildErr != nil {
			fmt.Fprintln(stderr, "Orbit: no se pudo preparar la configuración MCP:", buildErr)
			return 1
		}
		updates = append(updates, initmemory.ManagedFileUpdate{
			Path: ".codex/config.toml", ResourceID: "project-codex-atomic-memory-config", Source: "orbit-memory-configure", Content: codexBytes,
		})
		plan.Entries = append(plan.Entries, initmemory.Entry{Path: ".codex/config.toml", Status: initmemory.StatusCreate})
		fmt.Fprintln(stdout, "Codex no tiene registrado atomicmemory; Orbit propondrá una configuración MCP de proyecto.")
		fmt.Fprintln(stdout, "Codex solo cargará esa configuración cuando el proyecto sea de confianza.")
	}
	fmt.Fprintf(stdout, "\nBackend propuesto: %s\n", config.Backend)
	for _, entry := range plan.Entries {
		fmt.Fprintf(stdout, "  • actualizar %s\n", entry.Path)
	}
	if !*yes {
		if !interactive {
			fmt.Fprintln(stderr, "Orbit: este comando requiere confirmación; ejecútalo en una terminal o pasa --yes para aplicar el plan mostrado")
			return 2
		}
		if confirm == nil {
			fmt.Fprintln(stderr, "Orbit: la confirmación interactiva no está disponible")
			return 2
		}
		accepted, confirmErr := confirm(plan)
		if confirmErr != nil {
			fmt.Fprintln(stderr, "Orbit: falló la confirmación:", confirmErr)
			return 1
		}
		if !accepted {
			fmt.Fprintln(stdout, "Cancelado. No se escribió ningún archivo.")
			return 0
		}
	}
	if err := initmemory.UpdateManagedFiles(root, updates); err != nil {
		fmt.Fprintln(stderr, "Orbit: no se aplicó la configuración:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Configuración guardada. Las credenciales y la identidad personal no se copiaron al repositorio.")
	return 0
}

func listCodexServers() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "codex", "mcp", "list", "--json").Output()
	if err != nil {
		return nil, errors.New("Codex no pudo listar sus servidores MCP")
	}
	return codexconfig.ParseServerNames(output)
}

func runInit(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc) int {
	flags := flag.NewFlagSet("orbit init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	yes := flags.Bool("yes", false, "aplica el plan mostrado sin pedir confirmación interactiva")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Uso: orbit init [--yes]")
		fmt.Fprintln(stderr, "Inicializa los archivos de memoria Markdown del proyecto.")
		fmt.Fprintln(stderr, "  --yes  confirma explícitamente el plan en un entorno no interactivo")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.Usage()
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", flags.Arg(0))
		flags.Usage()
		return 2
	}

	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	plan, err := initmemory.BuildPlan(root)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	printPlan(stdout, plan)

	if plan.CreateCount() > 0 && !*yes {
		if !interactive {
			fmt.Fprintln(stderr, "Orbit: este comando requiere confirmación; ejecútalo en una terminal o pasa --yes para aplicar el plan mostrado")
			return 2
		}
		if confirm == nil {
			fmt.Fprintln(stderr, "Orbit: la confirmación interactiva no está disponible")
			return 2
		}
		accepted, err := confirm(plan)
		if err != nil {
			fmt.Fprintln(stderr, "Orbit: falló la confirmación:", err)
			return 1
		}
		if !accepted {
			fmt.Fprintln(stdout, "Cancelado. No se escribió ningún archivo.")
			return 0
		}
	}

	result, err := initmemory.Apply(plan)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	printSummary(stdout, result)
	if len(result.Conflicts) > 0 {
		return 1
	}
	return 0
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, "Orbit — flujo colaborativo guiado por especificaciones para agentes de programación con IA")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Uso: orbit <comando> [opciones]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Comandos:")
	fmt.Fprintln(out, "  init       Inicializa la memoria Markdown del proyecto")
	fmt.Fprintln(out, "  memory     Configura una integración personal de Atomic Memory")
	fmt.Fprintln(out, "  help       Muestra esta ayuda")
}

func printMemoryHelp(out io.Writer) {
	fmt.Fprintln(out, "Uso: orbit memory configure <codex|claude> [--backend hosted-cloud|connected-local|self-hosted] [--api-url URL] [--yes]")
	fmt.Fprintln(out, "Configura Atomic Memory para la persona actual y el agente indicado, con una vista previa revisable.")
	fmt.Fprintln(out, "Sin --backend, conserva el backend del proyecto; si está desactivado, usa hosted-cloud.")
	fmt.Fprintln(out, "Las claves API y la identidad personal deben estar en el entorno local, nunca en Git.")
}

func printPlan(out io.Writer, plan initmemory.Plan) {
	fmt.Fprintf(out, "Orbit preparará la memoria del proyecto en %s\n", plan.Root)
	for _, entry := range plan.Entries {
		switch entry.Status {
		case initmemory.StatusCreate:
			fmt.Fprintf(out, "  + crear          %s\n", entry.Path)
		case initmemory.StatusPresent:
			fmt.Fprintf(out, "  = ya existe      %s\n", entry.Path)
		case initmemory.StatusConflict:
			fmt.Fprintf(out, "  ! conflicto      %s — %s\n", entry.Path, entry.Reason)
		case initmemory.StatusModified:
			fmt.Fprintf(out, "  ! modificado     %s — %s\n", entry.Path, entry.Reason)
		}
	}
}

func printSummary(out io.Writer, result initmemory.Result) {
	fmt.Fprintf(out, "\nCreados: %d archivo(s); ya existentes: %d; conflictos: %d.\n", len(result.Created), len(result.AlreadyPresent), len(result.Conflicts))
	for _, entry := range result.Conflicts {
		fmt.Fprintf(out, "  Revisa %s: %s\n", entry.Path, entry.Reason)
	}
}

type confirmationModel struct {
	plan      initmemory.Plan
	selected  int
	confirmed bool
	done      bool
}

func (m confirmationModel) Init() tea.Cmd { return nil }

func (m confirmationModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		m.selected = 0
	case "down", "j":
		m.selected = 1
	case "y", "s":
		m.confirmed = true
		m.done = true
		return m, tea.Quit
	case "n", "q", "esc", "ctrl+c":
		m.done = true
		return m, tea.Quit
	case "enter":
		m.confirmed = m.selected == 1
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m confirmationModel) View() string {
	var view strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Orbit · Confirmar cambios")
	view.WriteString(header + "\n\n")
	view.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Se aplicarán los siguientes cambios:") + "\n")
	for _, entry := range m.plan.Entries {
		if entry.Status != initmemory.StatusPresent {
			fmt.Fprintf(&view, "  %s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("•"), entry.Path)
		}
	}
	view.WriteString("\n¿Continuar?\n\n")
	choices := []string{"No, cancelar", "Sí, aplicar los cambios indicados"}
	for index, choice := range choices {
		marker := " "
		styledChoice := choice
		if m.selected == index {
			marker = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Render(">")
			styledChoice = lipgloss.NewStyle().Bold(true).Render(choice)
		}
		fmt.Fprintf(&view, "%s %s\n", marker, styledChoice)
	}
	view.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Usa ↑/↓ y Enter, o pulsa s/n.") + "\n")
	return view.String()
}

func confirmPlan(plan initmemory.Plan, input io.Reader, output io.Writer) (bool, error) {
	program := tea.NewProgram(confirmationModel{plan: plan}, tea.WithInput(input), tea.WithOutput(output))
	finalModel, err := program.Run()
	if err != nil {
		return false, err
	}
	confirmation, ok := finalModel.(confirmationModel)
	if !ok || !confirmation.done {
		return false, errors.New("la confirmación terminó sin una decisión")
	}
	return confirmation.confirmed, nil
}

func isInteractive(input *os.File) bool {
	info, err := input.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

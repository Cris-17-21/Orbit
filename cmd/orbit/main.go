package main

import (
	"context"
	"encoding/json"
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
	"github.com/Cris-17-21/Orbit/internal/skillinstall"
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
	case "doctor":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", args[1])
			printHelp(stderr)
			return 2
		}
		return runDoctor(workingDirectory, stdout, stderr, defaultDoctorDependencies())
	case "skill":
		return runSkill(args[1:], workingDirectory, stdout, stderr, interactive)
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

type doctorDependencies struct {
	lookPath   func(string) (string, error)
	listCodex  codexServerLister
	listClaude func() (string, error)
	getenv     func(string) string
}

func defaultDoctorDependencies() doctorDependencies {
	return doctorDependencies{
		lookPath:   exec.LookPath,
		listCodex:  listCodexServers,
		listClaude: listClaudeServers,
		getenv:     os.Getenv,
	}
}

func runDoctor(workingDirectory string, stdout, stderr io.Writer, deps doctorDependencies) int {
	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Orbit · Diagnóstico de Atomic Memory")
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	configBytes, err := os.ReadFile(configPath)
	var config atomicmemory.Config
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "Atomic Memory: sin configurar en este proyecto")
	} else if err != nil {
		fmt.Fprintln(stdout, "Atomic Memory: no se pudo leer la configuración")
	} else if config, err = atomicmemory.ParseConfig(configBytes); err != nil {
		fmt.Fprintln(stdout, "Atomic Memory: configuración inválida")
	} else if config.Mode == atomicmemory.ModeDisabled {
		fmt.Fprintln(stdout, "Atomic Memory: modalidad desactivada")
	} else {
		fmt.Fprintf(stdout, "Atomic Memory: modalidad %s (%s)\n", config.Mode, config.Backend)
		for _, name := range []string{"ATOMICMEMORY_API_KEY", "ATOMICMEMORY_SCOPE_USER"} {
			state := "falta"
			if deps.getenv != nil && deps.getenv(name) != "" {
				state = "definida"
			}
			fmt.Fprintf(stdout, "  %s: %s\n", name, state)
		}
	}

	codexPath, codexErr := deps.lookPath("codex")
	if codexErr != nil {
		fmt.Fprintln(stdout, "Codex: CLI no detectado")
	} else if deps.listCodex == nil {
		fmt.Fprintln(stdout, "Codex: CLI detectado; no se pudo consultar MCP")
	} else if servers, listErr := deps.listCodex(); listErr != nil {
		fmt.Fprintf(stdout, "Codex: CLI detectado (%s); no se pudo consultar MCP\n", filepath.Base(codexPath))
	} else if codexconfig.HasAtomicMemoryServer(servers) {
		fmt.Fprintln(stdout, "Codex: servidor registrado; conexión no verificada por el CLI (usa `/mcp` en Codex)")
	} else {
		fmt.Fprintln(stdout, "Codex: Atomic Memory no aparece en `codex mcp list`")
	}

	claudePath, claudeErr := deps.lookPath("claude")
	if claudeErr != nil {
		fmt.Fprintln(stdout, "Claude Code: CLI no detectado")
	} else if deps.listClaude == nil {
		fmt.Fprintln(stdout, "Claude Code: CLI detectado; no se pudo consultar MCP")
	} else if result, listErr := deps.listClaude(); listErr != nil {
		fmt.Fprintf(stdout, "Claude Code: CLI detectado (%s); no se pudo consultar MCP\n", filepath.Base(claudePath))
	} else {
		fmt.Fprintf(stdout, "Claude Code: %s\n", classifyClaudeAtomicMemoryStatus(result))
	}
	fmt.Fprintln(stdout, "Memoria compartida: no verificada; requiere validar el alcance con dos colaboradores.")
	return 0
}

func classifyClaudeAtomicMemoryStatus(output string) string {
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, claudecodeconfig.ServerName) {
			continue
		}
		switch {
		case strings.Contains(lower, "pending approval"):
			return "pendiente de aprobación (según `claude mcp list`)"
		case strings.Contains(lower, "needs authentication"):
			return "requiere autenticación (según `claude mcp list`)"
		case strings.Contains(lower, "failed to connect"):
			return "fallo al conectar (según `claude mcp list`)"
		case strings.Contains(lower, "connected"):
			return "estado conectado reportado por `claude mcp list`"
		case strings.Contains(lower, "disabled"):
			return "deshabilitado (según `claude mcp list`)"
		default:
			return "servidor listado; estado no reconocido por Orbit"
		}
	}
	return "Atomic Memory no aparece en `claude mcp list`"
}

func runSkill(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printSkillHelp(stdout)
		return 0
	}
	if args[0] != "install" {
		fmt.Fprintf(stderr, "Orbit: subcomando de skills desconocido %q\n", args[0])
		printSkillHelp(stderr)
		return 2
	}
	return runSkillInstall(args[1:], workingDirectory, stdout, stderr, interactive)
}

func runSkillInstall(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool) int {
	return runSkillInstallWithConfirmation(args, workingDirectory, stdout, stderr, interactive, confirmManagedPlan)
}

type managedConfirmationFunc func(initmemory.ManagedFilesPlan, io.Reader, io.Writer) (bool, error)

func runSkillInstallWithConfirmation(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm managedConfirmationFunc) int {
	flags := flag.NewFlagSet("orbit skill install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	codex := flags.Bool("codex", false, "instala la skill para Codex")
	claude := flags.Bool("claude", false, "instala la skill para Claude Code")
	antigravity := flags.Bool("antigravity", false, "instala la skill para Antigravity")
	yes := flags.Bool("yes", false, "aplica el plan mostrado sin pedir confirmación interactiva")
	dryRun := flags.Bool("dry-run", false, "muestra el plan sin escribir archivos")
	flags.Usage = func() { printSkillHelp(stderr) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", flags.Arg(0))
		printSkillHelp(stderr)
		return 2
	}
	if *yes && *dryRun {
		fmt.Fprintln(stderr, "Orbit: --yes y --dry-run son opciones incompatibles")
		return 2
	}
	var agents []skillinstall.Agent
	if *codex {
		agents = append(agents, skillinstall.Codex)
	}
	if *claude {
		agents = append(agents, skillinstall.ClaudeCode)
	}
	if *antigravity {
		agents = append(agents, skillinstall.Antigravity)
	}
	if len(agents) == 0 {
		if !interactive {
			fmt.Fprintln(stderr, "Orbit: selecciona uno o más agentes con --codex, --claude o --antigravity")
			return 2
		}
		var err error
		agents, err = selectSkillAgents(os.Stdin, stdout)
		if err != nil {
			fmt.Fprintln(stderr, "Orbit: no se pudo seleccionar agentes:", err)
			return 1
		}
		if len(agents) == 0 {
			fmt.Fprintln(stdout, "Cancelado. No se escribió ningún archivo.")
			return 0
		}
	}
	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	updates, err := skillinstall.Resolve(agents...)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	plan, err := initmemory.PlanManagedFiles(root, updates)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	printManagedPlan(stdout, plan, agents)
	if *dryRun {
		fmt.Fprintln(stdout, "Vista previa únicamente; no se escribió ningún archivo.")
		return 0
	}
	if managedCreateCount(plan) > 0 && !*yes {
		if !interactive {
			fmt.Fprintln(stderr, "Orbit: este comando requiere confirmación; ejecútalo en una terminal o pasa --yes para aplicar el plan mostrado")
			return 2
		}
		if confirm == nil {
			fmt.Fprintln(stderr, "Orbit: la confirmación interactiva no está disponible")
			return 2
		}
		accepted, err := confirm(plan, os.Stdin, stdout)
		if err != nil {
			fmt.Fprintln(stderr, "Orbit: falló la confirmación:", err)
			return 1
		}
		if !accepted {
			fmt.Fprintln(stdout, "Cancelado. No se escribió ningún archivo.")
			return 0
		}
	}
	result, err := initmemory.ApplyManagedFiles(plan)
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

func managedCreateCount(plan initmemory.ManagedFilesPlan) int {
	count := 0
	for _, entry := range plan.Entries {
		if entry.Status == initmemory.StatusCreate {
			count++
		}
	}
	return count
}

func printManagedPlan(out io.Writer, plan initmemory.ManagedFilesPlan, agents []skillinstall.Agent) {
	labels := make([]string, 0, len(agents))
	seen := make(map[skillinstall.Agent]bool)
	for _, agent := range agents {
		if seen[agent] {
			continue
		}
		seen[agent] = true
		switch agent {
		case skillinstall.Codex:
			labels = append(labels, "Codex")
		case skillinstall.ClaudeCode:
			labels = append(labels, "Claude Code")
		case skillinstall.Antigravity:
			labels = append(labels, "Antigravity")
		}
	}
	fmt.Fprintf(out, "Orbit instalará su skill para %s en %s\n", strings.Join(labels, ", "), plan.Root)
	if seen[skillinstall.Codex] && seen[skillinstall.Antigravity] {
		fmt.Fprintln(out, "Codex y Antigravity descubrirán la misma copia en .agents/skills/orbit/.")
	}
	for _, entry := range plan.Entries {
		switch entry.Status {
		case initmemory.StatusCreate:
			fmt.Fprintf(out, "  + instalar       %s\n", entry.Path)
		case initmemory.StatusPresent:
			fmt.Fprintf(out, "  = ya instalada   %s\n", entry.Path)
		case initmemory.StatusConflict, initmemory.StatusModified:
			fmt.Fprintf(out, "  ! conflicto      %s — %s\n", entry.Path, entry.Reason)
		}
	}
}

type skillAgentSelector struct {
	cursor   int
	selected map[int]bool
	done     bool
}

func (m skillAgentSelector) Init() tea.Cmd { return nil }
func (m skillAgentSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 2 {
			m.cursor++
		}
	case " ":
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "enter":
		m.done = true
		return m, tea.Quit
	case "q", "esc", "ctrl+c":
		m.selected = map[int]bool{}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}
func (m skillAgentSelector) View() string {
	var view strings.Builder
	fmt.Fprintln(&view, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Orbit · Agentes para instalar"))
	fmt.Fprintln(&view, "Selecciona con Espacio y confirma con Enter:")
	names := []string{"Codex", "Claude Code", "Antigravity"}
	for i, name := range names {
		marker, checked := " ", " "
		if m.cursor == i {
			marker = ">"
		}
		if m.selected[i] {
			checked = "x"
		}
		fmt.Fprintf(&view, "%s [%s] %s\n", marker, checked, name)
	}
	fmt.Fprintln(&view, "q cancela")
	return view.String()
}
func selectSkillAgents(input io.Reader, output io.Writer) ([]skillinstall.Agent, error) {
	model := skillAgentSelector{selected: make(map[int]bool)}
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output))
	final, err := program.Run()
	if err != nil {
		return nil, err
	}
	selection, ok := final.(skillAgentSelector)
	if !ok || !selection.done {
		return nil, errors.New("el selector terminó sin decisión")
	}
	var agents []skillinstall.Agent
	if selection.selected[0] {
		agents = append(agents, skillinstall.Codex)
	}
	if selection.selected[1] {
		agents = append(agents, skillinstall.ClaudeCode)
	}
	if selection.selected[2] {
		agents = append(agents, skillinstall.Antigravity)
	}
	return agents, nil
}

func confirmManagedPlan(plan initmemory.ManagedFilesPlan, input io.Reader, output io.Writer) (bool, error) {
	program := tea.NewProgram(confirmationModel{root: plan.Root, entries: plan.Entries}, tea.WithInput(input), tea.WithOutput(output))
	final, err := program.Run()
	if err != nil {
		return false, err
	}
	confirmation, ok := final.(confirmationModel)
	if !ok || !confirmation.done {
		return false, errors.New("la confirmación terminó sin una decisión")
	}
	return confirmation.confirmed, nil
}

func runMemory(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc, listServers codexServerLister) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printMemoryHelp(stdout)
		return 0
	}
	if args[0] == "status" {
		if len(args) != 1 {
			fmt.Fprintf(stderr, "Orbit: argumento inesperado %q\n", args[1])
			printMemoryHelp(stderr)
			return 2
		}
		return runMemoryStatus(workingDirectory, stdout, stderr)
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

func runMemoryStatus(workingDirectory string, stdout, stderr io.Writer) int {
	root, err := initmemory.FindGitRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Orbit:", err)
		return 1
	}
	configPath := filepath.Join(root, ".orbit", "atomic-memory.json")
	configBytes, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "Atomic Memory: sin configurar en este proyecto (no se encontró .orbit/atomic-memory.json).")
	} else if err != nil {
		fmt.Fprintln(stderr, "Orbit: no se pudo leer la configuración de Atomic Memory:", err)
		return 1
	} else {
		config, parseErr := atomicmemory.ParseConfig(configBytes)
		if parseErr != nil {
			fmt.Fprintln(stderr, "Orbit: configuración de Atomic Memory inválida:", parseErr)
			return 1
		}
		modeLabel := string(config.Mode)
		if config.Mode == atomicmemory.ModeDisabled {
			modeLabel = "desactivada"
		}
		fmt.Fprintf(stdout, "Modalidad: %s\n", modeLabel)
		if config.Backend != "" {
			fmt.Fprintf(stdout, "Backend: %s\n", config.Backend)
		}
	}

	printProjectMCPStatus(stdout, filepath.Join(root, ".codex", "config.toml"), filepath.Join(root, ".mcp.json"))
	fmt.Fprintln(stdout, "Conectividad: no verificada por `orbit memory status`; revisa el estado MCP en tu agente.")
	fmt.Fprintln(stdout, "Memoria compartida: no verificada; requiere validar el alcance con dos colaboradores.")
	return 0
}

func printProjectMCPStatus(out io.Writer, codexPath, claudePath string) {
	codexStatus := "no configurado"
	if data, err := os.ReadFile(codexPath); err == nil {
		if found, parseErr := codexconfig.HasAtomicMemoryConfig(data); parseErr != nil {
			codexStatus = "no se pudo inspeccionar"
		} else if found {
			codexStatus = "configurado en el proyecto"
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		codexStatus = "no se pudo inspeccionar"
	}
	claudeStatus := "no configurado"
	if data, err := os.ReadFile(claudePath); err == nil {
		var config struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if json.Unmarshal(data, &config) != nil {
			claudeStatus = "no se pudo inspeccionar"
		} else if _, found := config.MCPServers[claudecodeconfig.ServerName]; found {
			claudeStatus = "configurado en el proyecto"
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		claudeStatus = "no se pudo inspeccionar"
	}
	fmt.Fprintf(out, "Codex: %s\nClaude Code: %s\n", codexStatus, claudeStatus)
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

func listClaudeServers() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "claude", "mcp", "list").Output()
	if err != nil {
		return "", errors.New("Claude Code no pudo listar sus servidores MCP")
	}
	return string(output), nil
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
	fmt.Fprintln(out, "  doctor     Diagnostica la configuración de Atomic Memory y los agentes disponibles")
	fmt.Fprintln(out, "  memory     Configura una integración personal de Atomic Memory")
	fmt.Fprintln(out, "  skill      Instala skills de Orbit en un proyecto")
	fmt.Fprintln(out, "  help       Muestra esta ayuda")
}

func printSkillHelp(out io.Writer) {
	fmt.Fprintln(out, "Uso: orbit skill install [--codex] [--claude] [--antigravity] [--dry-run | --yes]")
	fmt.Fprintln(out, "Instala la skill de Orbit en el proyecto Git actual. En una terminal puedes elegir agentes interactivamente.")
	fmt.Fprintln(out, "Codex y Antigravity comparten .agents/skills/orbit/; Claude Code usa .claude/skills/orbit/.")
	fmt.Fprintln(out, "--dry-run muestra el plan sin escribir; sin terminal, la aplicación requiere --yes.")
}

func printMemoryHelp(out io.Writer) {
	fmt.Fprintln(out, "Uso: orbit memory <status | configure <codex|claude> [--backend hosted-cloud|connected-local|self-hosted] [--api-url URL] [--yes]>")
	fmt.Fprintln(out, "  status     Muestra la configuración MCP de proyecto sin revelar valores sensibles")
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
	root      string
	entries   []initmemory.Entry
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
	for _, entry := range m.entries {
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
	program := tea.NewProgram(confirmationModel{root: plan.Root, entries: plan.Entries}, tea.WithInput(input), tea.WithOutput(output))
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

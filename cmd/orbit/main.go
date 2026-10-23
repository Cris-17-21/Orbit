package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Cris-17-21/Orbit/internal/initmemory"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmationFunc func(initmemory.Plan) (bool, error)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Orbit: could not determine the current directory:", err)
		os.Exit(1)
	}

	code := execute(os.Args[1:], root, os.Stdout, os.Stderr, isInteractive(os.Stdin), func(plan initmemory.Plan) (bool, error) {
		return confirmPlan(plan, os.Stdin, os.Stdout)
	})
	os.Exit(code)
}

func execute(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], workingDirectory, stdout, stderr, interactive, confirm)
	case "help":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "Orbit: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runInit(args []string, workingDirectory string, stdout, stderr io.Writer, interactive bool, confirm confirmationFunc) int {
	flags := flag.NewFlagSet("orbit init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	yes := flags.Bool("yes", false, "apply the displayed plan without an interactive prompt")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: orbit init [--yes]")
		fmt.Fprintln(stderr, "Initialize the project's Markdown memory files.")
		fmt.Fprintln(stderr, "  --yes  explicitly confirm the plan in a non-interactive environment")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.Usage()
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Orbit: unexpected argument %q\n", flags.Arg(0))
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
			fmt.Fprintln(stderr, "Orbit: this command needs confirmation; run it in a terminal or pass --yes to apply the displayed plan")
			return 2
		}
		if confirm == nil {
			fmt.Fprintln(stderr, "Orbit: interactive confirmation is unavailable")
			return 2
		}
		accepted, err := confirm(plan)
		if err != nil {
			fmt.Fprintln(stderr, "Orbit: confirmation failed:", err)
			return 1
		}
		if !accepted {
			fmt.Fprintln(stdout, "Cancelled. No files were written.")
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
	fmt.Fprintln(out, "Orbit — a collaborative, specification-driven workflow for AI coding agents")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage: orbit <command> [options]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  init       Initialize project Markdown memory")
	fmt.Fprintln(out, "  help       Show this help")
}

func printPlan(out io.Writer, plan initmemory.Plan) {
	fmt.Fprintf(out, "Orbit will prepare project memory in %s\n", plan.Root)
	for _, entry := range plan.Entries {
		switch entry.Status {
		case initmemory.StatusCreate:
			fmt.Fprintf(out, "  + create         %s\n", entry.Path)
		case initmemory.StatusPresent:
			fmt.Fprintf(out, "  = already present %s\n", entry.Path)
		case initmemory.StatusConflict:
			fmt.Fprintf(out, "  ! conflict       %s — %s\n", entry.Path, entry.Reason)
		case initmemory.StatusModified:
			fmt.Fprintf(out, "  ! modified       %s — %s\n", entry.Path, entry.Reason)
		}
	}
}

func printSummary(out io.Writer, result initmemory.Result) {
	fmt.Fprintf(out, "\nCreated %d file(s); %d already present; %d conflict(s).\n", len(result.Created), len(result.AlreadyPresent), len(result.Conflicts))
	for _, entry := range result.Conflicts {
		fmt.Fprintf(out, "  Review %s: %s\n", entry.Path, entry.Reason)
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
	case "y":
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
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("Orbit · Initialize project memory")
	view.WriteString(header + "\n\n")
	view.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("The following files will be created:") + "\n")
	for _, entry := range m.plan.Entries {
		if entry.Status == initmemory.StatusCreate {
			fmt.Fprintf(&view, "  %s %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("+"), entry.Path)
		}
	}
	view.WriteString("\nExisting files will be preserved. Continue?\n\n")
	choices := []string{"No, cancel", "Yes, create the listed files"}
	for index, choice := range choices {
		marker := " "
		styledChoice := choice
		if m.selected == index {
			marker = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Render(">")
			styledChoice = lipgloss.NewStyle().Bold(true).Render(choice)
		}
		fmt.Fprintf(&view, "%s %s\n", marker, styledChoice)
	}
	view.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Use ↑/↓ and Enter, or press y/n.") + "\n")
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
		return false, errors.New("confirmation ended without a decision")
	}
	return confirmation.confirmed, nil
}

func isInteractive(input *os.File) bool {
	info, err := input.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

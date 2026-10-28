package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/term"
	"github.com/f-tuma/SPEC/internal/spec"
	"github.com/f-tuma/SPEC/internal/tui"
)

const help = `SPEC — spec-driven coding workflows

  spec                               Open the TUI in a terminal
  spec tui --project DIR             Interactive setup and version manager
  spec install --project DIR         Plugin + pinned OpenSpec + native setup
  spec status --project DIR          Read managed state
  spec update --project DIR --check  List stable Git releases
  spec update --project DIR          Update plugin (not the SPEC binary)
  spec rollback --project DIR        Restore the previous plugin
  spec uninstall --project DIR       Remove SPEC's agent setup and .spec state
  spec run --project DIR --agent claude|codex -- ARGS
  spec openspec --project DIR -- ARGS
  spec design --project DIR [--starter ID [--write]]
  spec audit|adopt|check|fingerprint --project DIR
  spec --version

All noninteractive reports are JSON. Use COMMAND --help for options.
Exit codes: 0 completed, 1 mechanical findings, 2 error, 3 client setup pending.
`

type stringsFlag []string

func (s *stringsFlag) String() string         { return fmt.Sprint([]string(*s)) }
func (s *stringsFlag) Set(value string) error { *s = append(*s, value); return nil }

func emit(out io.Writer, value any, code int) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return 2
	}
	return code
}
func fail(out io.Writer, err error) int {
	return emit(out, map[string]any{"status": "error", "error": err.Error(), "mechanical_result": "error", "semantic_result": "unknown"}, 2)
}

func Run(ctx context.Context, args []string, m *spec.Manager, in io.Reader, out, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "--version" || args[0] == "version") {
		fmt.Fprintln(out, spec.Version)
		return 0
	}
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		fmt.Fprint(out, help)
		return 0
	}
	operation := "tui"
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		operation = args[0]
		args = args[1:]
	}
	valid := map[string]bool{"tui": true, "install": true, "status": true, "update": true, "rollback": true, "run": true, "openspec": true, "audit": true, "adopt": true, "check": true, "fingerprint": true, "design": true, "uninstall": true}
	if !valid[operation] {
		return fail(out, fmt.Errorf("unknown command %q", operation))
	}
	options := spec.Options{}
	flags := flag.NewFlagSet(operation, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.Project, "project", "", "Existing project directory")
	tools := "claude,codex"
	agent, root, evidence, phase, language, starter, review := "", "", "", "orient", "", "", ""
	write, ui := false, false
	var inputs stringsFlag
	switch operation {
	case "tui", "install":
		flags.StringVar(&tools, "tools", tools, "claude, codex or claude,codex")
		flags.StringVar(&options.Cache, "cache-dir", spec.DefaultCache(), "OpenSpec cache outside the project")
		flags.StringVar(&options.Method, "openspec-method", "auto", "auto (native binary), binary, pnpm or npm")
		flags.StringVar(&options.OpenSpecBin, "openspec-bin", "", "Use an existing pinned OpenSpec executable")
		flags.BoolVar(&options.NoRegister, "no-register", false, "Skip agent setup (Claude Code plugin, Codex project skills)")
		flags.StringVar(&language, "document-language", "", "Language for project documents, e.g. English or Czech; conversation follows the chat")
	}
	if operation == "tui" || operation == "install" || operation == "update" || operation == "rollback" {
		flags.StringVar(&review, "review", "", "Design review: spec (SPEC's fresh-context review) or egm (installs the official EGM plugin)")
	}
	if operation == "install" || operation == "update" || operation == "rollback" || operation == "uninstall" {
		flags.BoolVar(&options.DryRun, "dry-run", false, "Preview without writing or installing")
	}
	if operation == "update" {
		flags.StringVar(&options.Source, "source", "", "Git release repository")
		flags.StringVar(&options.To, "to", "", "Stable version tag")
		flags.BoolVar(&options.Check, "check", false, "List available versions without installation")
		flags.BoolVar(&options.NoRegister, "no-register", false, "Skip agent setup refresh")
	}
	if operation == "rollback" {
		flags.BoolVar(&options.NoRegister, "no-register", false, "Skip agent setup refresh")
	}
	if operation == "run" {
		flags.StringVar(&agent, "agent", "", "claude or codex")
	}
	if operation == "audit" || operation == "adopt" || operation == "check" || operation == "fingerprint" {
		flags.StringVar(&root, "root", "", "Alias for --project, compatible with earlier checker mappings")
	}
	if operation == "adopt" {
		flags.BoolVar(&write, "write", false, "Exclusively create a draft mapping")
	}
	if operation == "design" {
		flags.StringVar(&starter, "starter", "", "Design starter ID; without it, list the starters")
		flags.BoolVar(&write, "write", false, "Create DESIGN.md from the starter; never overwrites")
	}
	if operation == "check" {
		flags.StringVar(&phase, "phase", phase, "orient, design, implement, verify or release")
		flags.BoolVar(&ui, "ui", false, "Require a mapped visual design source")
		flags.StringVar(&evidence, "evidence", "", "Repository-relative evidence receipt")
	}
	if operation == "fingerprint" {
		flags.Var(&inputs, "input", "Repository-relative file; repeat for each input")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return fail(out, err)
	}
	if operation != "run" && operation != "openspec" && flags.NArg() != 0 {
		return fail(out, fmt.Errorf("unexpected positional arguments"))
	}
	if root != "" {
		if options.Project != "" && options.Project != root {
			return fail(out, fmt.Errorf("use one consistent project/root"))
		}
		options.Project = root
	}
	var err error
	options.Tools, err = spec.Tools(tools)
	if err != nil {
		return fail(out, err)
	}
	flags.Visit(func(f *flag.Flag) {
		options.SetDocumentLanguage = options.SetDocumentLanguage || f.Name == "document-language"
		options.SetReview = options.SetReview || f.Name == "review"
	})
	if options.DocumentLanguage, err = spec.DocumentLanguage(language); err != nil {
		return fail(out, err)
	}
	if options.Review, err = spec.ReviewMode(review); err != nil {
		return fail(out, err)
	}
	if options.Method != "" && options.Method != "auto" && options.Method != "binary" && options.Method != "npm" && options.Method != "pnpm" {
		return fail(out, fmt.Errorf("invalid openspec-method"))
	}
	if operation == "tui" {
		input, inputOK := in.(*os.File)
		output, outputOK := out.(*os.File)
		if !inputOK || !outputOK || !term.IsTerminal(input.Fd()) || !term.IsTerminal(output.Fd()) {
			fmt.Fprint(out, help)
			return 0
		}
		if options.Project == "" {
			options.Project, _ = os.Getwd()
		}
		if err = tui.Run(ctx, m, options, in, out); err != nil {
			return fail(out, err)
		}
		return 0
	}
	project, err := spec.Project(options.Project)
	if err != nil {
		return fail(out, err)
	}
	options.Project = project
	switch operation {
	case "uninstall":
		result, code, err := m.Uninstall(ctx, options)
		if err != nil {
			return fail(out, err)
		}
		return emit(out, result, code)
	case "audit":
		return emit(out, spec.Audit(project), 0)
	case "design":
		if starter == "" {
			starters, err := spec.DesignStarters(m.Plugin)
			if err != nil {
				return fail(out, err)
			}
			list := []map[string]any{}
			for _, s := range starters {
				list = append(list, map[string]any{"id": s.ID, "name": s.Name, "description": s.Description, "fonts": s.Fonts, "colors": s.Colors})
			}
			return emit(out, map[string]any{"status": "listed", "starters": list, "next": "spec design --project DIR --starter ID --write"}, 0)
		}
		result, err := m.CreateDesign(project, starter, !write)
		if err != nil {
			return fail(out, err)
		}
		return emit(out, result, 0)
	case "adopt":
		result, err := spec.Adopt(project, write)
		if err != nil {
			return fail(out, err)
		}
		return emit(out, result, 0)
	case "check":
		result, err := spec.Check(project, phase, ui, evidence)
		if err != nil {
			return fail(out, err)
		}
		code := 0
		if result["mechanical_result"] == "fail" {
			code = 1
		}
		return emit(out, result, code)
	case "fingerprint":
		if len(inputs) == 0 {
			return fail(out, fmt.Errorf("fingerprint requires at least one --input"))
		}
		hashes := map[string]string{}
		for _, input := range inputs {
			digest, err := spec.Digest(project, input)
			if err != nil {
				return fail(out, err)
			}
			hashes[input] = digest
		}
		return emit(out, map[string]any{"schema_version": 1, "subject": hashes, "artifacts": map[string]string{}, "meaning": "Input fingerprint only; no check has been run"}, 0)
	case "openspec":
		result, err := m.OpenSpec(ctx, project, flags.Args())
		if err != nil {
			return fail(out, err)
		}
		return emit(out, result, 0)
	case "run":
		cmd, err := m.AgentCommand(ctx, project, agent, flags.Args())
		if err != nil {
			return fail(out, err)
		}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, stderr
		if err = cmd.Run(); err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				return exit.ExitCode()
			}
			return fail(out, err)
		}
		return 0
	default:
		result, code, err := m.Execute(ctx, operation, options)
		if err != nil {
			return fail(out, err)
		}
		return emit(out, result, code)
	}
}

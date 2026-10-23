# Orbit

Orbit is a collaborative, specification-driven workflow toolkit for AI coding agents.

The project is being designed and built incrementally. Its planned direction includes a Go CLI, shared SDD workflows and skills, adapters for Codex, Claude Code, and Antigravity, focused subagent roles, Markdown project memory, and optional Atomic Memory integration.

See the [product specification](docs/product-spec.md) for Orbit's consolidated design, [task orchestration](docs/task-orchestration.md) for the proposed workflow, and [memory contract](docs/memory-contract.md) for the initial format for durable project context.

## Build the CLI from source

Building from source requires Go 1.24 or newer. Run `go test ./...` and `go build -o <output-path>/orbit ./cmd/orbit`. From a target Git project, run the built `orbit` executable with `init` to review the files it plans to create. Add `--yes` only when you want to apply that plan without an interactive confirmation.

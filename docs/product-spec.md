# Orbit product specification

- status: draft for review
- product: Orbit
- updated: 2026-10-23
- repository: https://github.com/Cris-17-21/Orbit

This document consolidates Orbit's product direction and approved design decisions. It is not an implementation specification for a single feature. Items marked **Open** still need a product decision before the affected capability is built.

## Product intent

Orbit is an installable workflow toolkit for people building software with AI coding agents. It helps a team turn requests into accepted specifications, small implementation tasks, focused pull requests, and evidence-backed validation. It also helps collaborators carry useful project context across people, agents, and sessions.

Orbit is distributed from GitHub and configured through a Go CLI with an interactive terminal interface. It works with the coding agents a person already uses; Orbit does not provide or install the underlying models or agent products.

## Users and needs

### Primary users

- Software developers and product builders working with AI coding agents.
- Small product teams and collaborators who need shared rules, decisions, and task context.
- Teams building substantial SaaS products, including BI and data capabilities.

### User needs

- Turn an unclear request into a bounded task and verifiable acceptance criteria.
- Make business rules explicit and reviewable.
- Let the person responsible for a task accept its specification without requiring every partner to approve every step.
- Split work into small PRs that can be reviewed and integrated independently where dependencies allow.
- Reuse project context across agents and sessions without treating stale notes as current rules.
- Use specialized subagents when they help, while controlling context and token cost.
- Install, update, inspect, and remove Orbit without disrupting existing agent configuration or user-installed skills.

## Approved product decisions

- The project name is **Orbit**.
- The CLI is written in **Go** and uses **Bubble Tea** for its guided terminal interface.
- Initial agent integrations include **Codex, Claude Code, and Antigravity**.
- The workflow and source skills are shared; each agent has an adapter for its supported configuration formats and capabilities.
- Subagent roles are selected intelligently for the task or when directly requested; they are not mandatory workflow stages.
- Delegate independent, bounded work with only the necessary context and concise evidence-based results. Measure token savings rather than assuming delegation is cheaper.
- Project Markdown is the default, version-controlled memory format. Atomic Memory is optional. A hybrid mode is deferred until synchronization and conflict handling are designed.
- Memory records separate business rules, technical decisions, and project context. They carry type, content, scope, status, date, source, and owner. Current records identify who approved them; superseded records link to replacements when available.
- The installation experience is interactive and visually clear, supports keyboard navigation, and also works non-interactively through flags.
- Orbit manages only its own registered files. It must preserve user-installed skills and unrelated configuration during install, update, and uninstall.
- Orbit supports selecting a version, checking for updates, updating, and rolling back.

## Product experience

### Install Orbit

The user obtains Orbit from its GitHub repository or a published release and runs its CLI. The guided setup:

1. Selects user-wide or project-local installation.
2. Detects or lets the user select Codex, Claude Code, and/or Antigravity.
3. Shows the files and configuration areas Orbit will manage.
4. Offers project Markdown memory by default and asks whether to configure Atomic Memory.
5. Installs the selected version, reports conflicts without silently overwriting files, and shows how to start the orchestrator.

The install flow must have a non-interactive equivalent for scripts and repeatable team setup.

### Start a task

The user invokes Orbit's orchestrator from their chosen agent. Orbit identifies the repository and request, selects a lightweight or full SDD path, retrieves relevant project context, explores the code when needed, and drafts a specification for substantial work. The task owner reviews and accepts the specification. Orbit then breaks accepted scope into small tasks, delegates useful independent work, supports implementation and validation, and prepares focused PRs.

The detailed task-start behavior is described in [task orchestration](task-orchestration.md).

### Share project memory

Markdown memory lives in the project repository and is reviewable through Git. Atomic Memory, when configured, provides scoped retrieval and storage across sessions. Memory informs the work but does not independently approve a business rule or replace an accepted SDD artifact.

The initial memory rules are described in [the memory contract](memory-contract.md).

### Maintain Orbit

The CLI reports its installed version and managed files, checks for available versions, updates to a selected release, and restores a prior version. It includes `orbit doctor` to inspect Orbit's installation and report actionable configuration problems.

## Shared workflow

1. Understand the request and identify the project.
2. Choose lightweight work or full SDD based on scope, risk, and user preference.
3. Retrieve relevant Markdown memory and optional Atomic Memory context.
4. Explore existing behavior and project conventions.
5. Select only useful specialist roles and bounded subagent tasks.
6. Draft the specification, business rules, assumptions, open questions, and verifiable acceptance criteria.
7. Ask the task owner to accept or revise the specification.
8. Break accepted scope into small tasks with dependencies and validation evidence.
9. Implement, validate, and prepare small PRs.
10. Propose durable memory updates with source and status.

This workflow is flexible for small changes; not every task requires every phase or a subagent.

## Initial capability areas

### CLI

Proposed commands:

| Command | Purpose |
| --- | --- |
| `orbit` | Open the guided setup or main interactive menu |
| `orbit install` | Install a chosen Orbit version and selected agent integrations |
| `orbit init` | Initialize Orbit's project workflow and Markdown memory structure |
| `orbit doctor` | Diagnose managed files and configured integrations |
| `orbit update` | Update to a selected or latest compatible release |
| `orbit update --check` | Show available updates without changing files |
| `orbit rollback` | Restore a previously installed Orbit version |
| `orbit uninstall` | Remove Orbit-managed files and configuration only |

Command names and whether `install` and `init` remain separate are **Open** until the CLI design phase.

### Orchestrator and skills

Orbit packages a small entry skill for starting the workflow and focused skills for recurring activities such as exploration, requirements, SDD specification, task planning, BI/data analysis, implementation, testing, UI/UX, security, and review. The exact skill catalog is **Open** and should be derived from the agreed workflow and real use cases.

Skills are stored once as source content where possible. Agent adapters install or package them in each host's expected location. Existing user skills remain available and untouched. Name conflicts are reported and resolved without silent replacement.

### Subagent role catalog

Potential roles include repository exploration, requirements/business rules, technical design, UX, UI, frontend, backend/API, data/BI, security, implementation, testing, documentation, performance, and change review. This is a capability catalog, not a fixed roster of always-running agents. Which roles can be installed as native subagents and how they map per host are **Open**.

### Agent adapters

Each adapter documents and manages only the host-specific files needed to expose Orbit's instructions, skills, subagent profiles, and optional Atomic Memory MCP connection. Adapter capability differences must be visible in `doctor` and installation summaries. Orbit must avoid claiming identical delegation behavior across hosts when their runtime capabilities differ.

Supported agent details and version compatibility are **Open** and must be verified against the host's current official documentation during implementation.

### Project memory

Markdown is the no-service-required baseline. The initial layout and record contract are being established in `memory-contract.md` and the SaaS/BI example. Project-level Markdown can be committed and reviewed with normal PRs.

Atomic Memory setup is opt-in. The user selects or supplies the relevant project/workspace connection. Orbit must keep personal and shared scopes distinct, preserve provenance, and avoid silently promoting proposed or stale records. Exact authentication, identity mapping, and hosted versus local setup are **Open**.

### SaaS and BI example

The example demonstrates memory categories, a proposed business rule, a proposed technical decision, a draft invitation conversion metric specification, task breakdown, and a focused PR template. The example remains explicitly fictional and unaccepted until a task owner reviews it.

## Installation isolation and ownership

- Prefer project-local installation when a team wants the workflow shared through Git.
- Offer user-wide installation for personal use across projects.
- Track each file and configuration fragment that Orbit owns, including the installed Orbit version.
- Place Orbit-managed resources in a dedicated namespace or directory where the host allows it.
- Merge only Orbit's named configuration entries; preserve unrelated entries and formatting where practical.
- If an unmanaged file would be overwritten or a skill name conflicts, stop that write and explain a safe resolution.
- On uninstall, remove only Orbit-managed assets. Preserve user edits and unrelated files; report anything that needs manual cleanup.
- Updates show the target version and relevant changes, retain a recovery point, and support rollback.

The exact ownership manifest format and behavior when users edit managed files are **Open**.

## Quality attributes

- **Portable:** usable on supported Windows, macOS, and Linux environments; exact platform matrix is **Open**.
- **Safe:** no silent replacement of user configuration, no secrets in repository files, and explicit handling of Atomic Memory credentials.
- **Reviewable:** generated specifications, task plans, and memory updates are Markdown diffs where appropriate.
- **Incremental:** capabilities can be installed and updated by version without forcing every team to adopt every new skill immediately.
- **Efficient:** load detailed skill instructions and memory only when relevant; bound subagent inputs and outputs.
- **Diagnosable:** `orbit doctor` reports detected agents, managed assets, version, and actionable integration status.
- **Accessible:** interactive setup works with keyboard navigation and provides clear text output and non-interactive flags.

## Success signals

Orbit's usefulness should be evaluated with real team tasks. Candidate signals include:

- A new collaborator can configure Orbit and start a project workflow without manual file surgery.
- Specifications identify business rules and acceptance evidence before implementation begins.
- Work is delivered in smaller, focused PRs with traceable criteria and validation evidence.
- A later session or collaborator can retrieve relevant project decisions without treating proposals or superseded records as current.
- Delegation reduces duplicated investigation or main-agent context while accounting for subagent token cost and result quality.
- Installation, update, rollback, and uninstall preserve user-owned skills and configuration.

Numeric targets and the evaluation method are **Open**.

## Out of scope for the initial product definition

- Replacing Codex, Claude Code, or Antigravity.
- Requiring Atomic Memory for normal Orbit use.
- Building a hosted team memory service or a new agent runtime.
- Forcing every change through the full SDD sequence.
- Requiring all partners to approve every task or PR.
- Shipping a mandatory fixed set of subagents for every project.

## Open decisions

1. Minimum supported Go version and operating system/architecture matrix.
2. GitHub distribution method: release binaries, `go install`, package managers, or a combination.
3. Exact command names and interactive flow for install versus project initialization.
4. Project-local and user-wide directory conventions for each host, and rules for resolving skill conflicts.
5. Which SDD artifact format to adopt or adapt, and whether design documents are optional by default.
6. How the CLI records managed files, recognizes user edits, and handles update conflicts.
7. Atomic Memory authentication, project/workspace identity, and supported hosted/local connection modes.
8. Host-specific subagent capabilities, permission mapping, and token/cost telemetry availability.
9. Whether Orbit includes its own PR template as an opt-in file or only provides an example.
10. Release policy, compatibility guarantees, and how old configuration formats migrate.

Resolve each open decision before implementing the part that depends on it; unrelated development can continue in parallel where safe.

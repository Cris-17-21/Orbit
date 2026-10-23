# Specification: Initialize project Markdown memory with `orbit init`

- status: accepted
- task owner: Cris (Orbit task owner)
- created: 2026-10-23
- accepted by: Cris (Orbit task owner)
- accepted at: 2026-10-23

## Need and outcome

Teams should be able to start Orbit's SDD workflow with shared project memory even when they do not use Atomic Memory. Running `orbit init` should create the project's Markdown memory structure with clear, reviewable starter files and no invented business rules.

## Scope

### Included

- Initialize the three category files in the project-root `project-context/` directory, following Orbit's memory contract.
- Make the operation safe when files or directories already exist.
- Explain which files were created, already present, or need manual conflict resolution.

### Excluded

- Connecting or configuring Atomic Memory, MCP, or agent-specific integrations.
- Generating specifications, task plans, or business rules for the user's product.
- Deciding the final CLI distribution, all flags, or the full interactive setup.

## Context and business rules

| Reference | Status | Scope | Relevance to this task |
| --- | --- | --- | --- |
| `docs/memory-contract.md` | proposed design; accepted as the current working direction for this task | Orbit project memory | Markdown is the baseline; the initial layout contains `business-rules.md`, `technical-decisions.md`, and `project-context.md`. A current rule needs an owner and approval evidence. |
| `docs/install-ownership.md` | proposed design; accepted as the current working direction for this task | Orbit-managed project files | Orbit must not silently overwrite user-owned files; conflicts need a clear and safe resolution. |
| `docs/product-spec.md` | proposed product design; task owner accepted the Markdown-first direction in conversation | CLI and project memory | `orbit init` initializes a project workflow and Markdown memory without requiring Atomic Memory. |

This feature creates structure only. It must not create records marked `current`, invent domain rules, or treat example-project content as a real team's policy.

## Assumptions and decisions

### Confirmed

- Markdown files committed in the project are the baseline shared memory; Atomic Memory remains optional.
- The memory record format distinguishes `proposed`, `current`, and `superseded`; an item is not current without responsible-owner approval evidence.
- Orbit must preserve files it does not own and report conflicts instead of silently replacing them.
- For its first version, `orbit init` requires a Git repository. When invoked from a nested directory, it operates at the repository root.

### Assumptions to verify

- `orbit init` creates `project-context/business-rules.md`, `project-context/technical-decisions.md`, and `project-context/project-context.md` at the repository root, matching the current memory-contract layout.
- New files contain only concise headings and instructions linking to Orbit's record format; no fictional or pre-approved rules are seeded.
- On an interactive terminal, the command presents a preview and asks before creating files. `--yes` explicitly confirms the same plan for non-interactive use.
- Orbit records files it creates in `.orbit/manifest.json` with manifest schema version 1 and a digest for each created file. A valid existing Orbit manifest may be updated; an unsupported or invalid manifest stops initialization before writes.

### Implementation choices for this increment

- Use the stable Bubble Tea v1 API at `v1.3.10` for the interactive confirmation. Use `--yes` only as an explicit non-interactive confirmation.
- Set the development module baseline to Go 1.24, which is required by the selected Bubble Tea release. The published product's minimum Go version remains open until the release matrix is chosen.
- Store the project manifest at `.orbit/manifest.json` with schema version 1. Record only the project files this command creates, using their project-relative paths and SHA-256 digests.

## User scenarios

### Scenario: Start Markdown memory in a project without Atomic Memory

- Given a Git project with no Orbit memory files
- When the task owner runs `orbit init` and accepts the displayed file plan
- Then Orbit creates the three category files and reports their paths

### Scenario: Preserve existing project content

- Given one or more target files already exist and are not recorded as Orbit-owned
- When the task owner runs `orbit init`
- Then Orbit preserves their content and reports each conflict for manual review

### Scenario: Repeat initialization safely

- Given the memory structure was initialized previously
- When the task owner runs `orbit init` again
- Then Orbit does not duplicate content or replace user edits and reports the existing files

## Acceptance criteria

- [ ] **AC-1:** In an eligible Git project with no target directory, `orbit init` previews the three target paths and creates them only after confirmation.
- [ ] **AC-2:** Each generated file identifies its category and links to the record format; it contains no example business rules or records marked `current`.
- [ ] **AC-3:** Existing files that Orbit does not own remain byte-for-byte unchanged, and the command reports the path and conflict clearly.
- [ ] **AC-4:** Re-running initialization does not duplicate starter content or overwrite edits made after the first run.
- [ ] **AC-5:** The generated Markdown structure is usable without Atomic Memory or any external service.
- [ ] **AC-6:** The completion summary lists created, already-present, and conflicting paths separately and gives a next step for conflicts.
- [ ] **AC-7:** The manifest records each file created by Orbit with a stable resource ID and SHA-256 digest; a later run recognizes unchanged files and reports edited files without overwriting them.

## Dependencies and risks

- The implementation depends on the CLI command and safe ownership behavior described in `docs/install-ownership.md`.
- Orbit must distinguish files it created from pre-existing files so later initialization and uninstall do not claim user-owned content.
- A project-root path may conflict with an existing `project-context/` directory. The command should inspect and report this before writing.

## Open questions

- The product's minimum supported Go version remains open. For this development increment, set `go.mod` to Go 1.24, the minimum declared by the selected Bubble Tea v1.3.10 release; revisit the product support policy before publishing binaries.

## Validation evidence

| Acceptance criterion | Check or evidence | Result |
| --- | --- | --- |
| AC-1, AC-2, AC-5, AC-6 | CLI run in a temporary Git project; inspect created paths, contents, and summary | pending implementation |
| AC-3, AC-4, AC-7 | CLI tests with pre-existing files, post-init edits, and a missing managed file; compare file contents and manifest | pending implementation |

## Proposed task and PR boundaries

| Task | Outcome | Criteria covered | Dependencies | PR boundary |
| --- | --- | --- | --- | --- |
| T-1 | Add safe Markdown-memory initialization to `orbit init` | AC-1–AC-6 | CLI foundation and ownership tracking | One PR for the command's preview, safe writes, repeat behavior, and focused CLI checks |

## Acceptance

The task owner accepts this specification and authorizes implementation within its scope:

- accepted by: Cris (Orbit task owner)
- accepted at: 2026-10-23
- notes or requested changes: task owner accepted the Git repository requirement; remaining Go support policy is not a blocker for this increment

Implementation begins only after the task owner records acceptance. A material change to scope, business rules, or acceptance criteria returns to the owner for review.

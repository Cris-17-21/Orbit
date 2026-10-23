# Orbit task orchestration

This document defines the proposed task-start behavior for Orbit's orchestrator. It applies whether the user invokes Orbit through Codex, Claude Code, or Antigravity; agent-specific adapters expose the same workflow through each host's supported mechanisms.

## Goals

- Help the user turn a request into a clear, bounded change.
- Reuse relevant project context without treating stale or proposed memory as current policy.
- Delegate only work that is independently useful and bounded.
- Keep the task owner's review and acceptance local to the task.
- Keep the primary agent's context focused and measure whether delegation saves tokens overall.

## Task-start flow

### 1. Capture the request and identify the project

The orchestrator records the user's stated outcome, affected product area if known, and any constraints or requested roles. It identifies the repository root and checks for Orbit configuration and SDD artifacts. If the project cannot be identified, it asks the user to select or describe it.

It does not silently turn an idea into an implementation plan. It first summarizes its understanding and asks only questions whose answers could change scope, business behavior, data meaning, or acceptance.

### 2. Choose the amount of process

The orchestrator proposes a lightweight path for a small, low-risk change and the full SDD path for substantial, cross-cutting, business-rule-sensitive, data-sensitive, or explicitly SDD-requested work. The user may request a different level of structure.

The lightweight path still identifies scope and a way to check the result. The full path creates or updates proposal, specification, design when needed, and task breakdown artifacts.

### 3. Retrieve relevant project context

The orchestrator reads the relevant Markdown records in the project. If Atomic Memory is configured and available, it also searches the configured project/workspace scope for context relevant to the request. Retrieval should be narrow: search by project and task concepts, then include only memories needed for decisions at hand.

For each recalled item, preserve its ID or reference, source, scope, and declared status. Apply these rules:

- `current` records may inform the draft when their scope and source fit.
- `proposed` records are hypotheses to confirm, not binding requirements.
- `superseded` records are historical context and must not guide new behavior unless the task concerns that history.
- Missing sources, incompatible scope, or conflicting current records are surfaced to the task owner.
- Memory informs the work; it does not replace the accepted specification or the source code's observed behavior.

Personal memory is never included in shared project context unless the user explicitly asks to share it.

### 4. Explore the repository

For work that changes an existing project, inspect the relevant code, current behavior, tests or validation capabilities, and project conventions. Keep exploration proportional to the task. When useful, a read-only exploration subagent can return a short report with file paths and line references.

The orchestrator compares repository evidence with relevant memory. It records discrepancies as open questions rather than assuming that either source is automatically correct.

### 5. Propose specialization and delegation

The orchestrator selects roles from the task's needs, such as requirements, UX/UI, frontend, backend, data/BI, security, implementation, testing, or review. These are capabilities, not mandatory phases.

Before starting delegated work, it briefly tells the user which roles it recommends and why. It asks for explicit direction only when the user requested control over delegation or when a host requires a confirmation. The user may request a role directly.

Delegate when a subtask is independent, bounded, and has a clear expected result. Give the subagent only:

- its question or deliverable;
- the relevant paths or excerpts;
- applicable accepted specification sections and current business rules;
- the specific skill or role instructions it needs;
- tool and write permissions appropriate to the task.

Do not delegate short, dependent, or shared-file work that costs more to coordinate than to do directly. Avoid overlapping assignments. Request concise findings with evidence paths and unresolved questions; the orchestrator synthesizes them and does not paste full subagent transcripts into the main context.

### 6. Draft the specification

For a substantial change, the orchestrator drafts a specification with:

- user or business need and desired outcome;
- scope and exclusions;
- business rules with references to memory records;
- observable acceptance criteria and relevant scenarios;
- data/BI definitions when metrics or pipelines are involved;
- dependencies, risks, assumptions, and unresolved decisions;
- expected validation evidence.

The specification distinguishes confirmed facts from assumptions and proposals. Criteria should describe observable outcomes and avoid prescribing implementation details unless the implementation choice is itself a requirement.

Use the reusable [specification template](spec-template.md) as a starting point. The task owner may omit sections that do not apply, but must keep scope, acceptance criteria, and validation clear enough to review.

### 7. Ask the task owner to accept the specification

The orchestrator presents the draft and asks the person responsible for the task to accept it or request changes. It does not start implementation while material business decisions or acceptance conditions remain unresolved.

This is task-level acceptance. It does not require every partner to approve every specification. A team may separately define which shared policies or high-impact decisions require broader agreement.

After acceptance, record who accepted it and when in the project's SDD artifact. Acceptance authorizes work within that agreed scope; a material scope or business-rule change returns to the task owner for review.

### 8. Break accepted scope into small tasks

Only after acceptance, create an ordered task breakdown. Each task should state one outcome, its scope, dependencies, criteria covered, and validation evidence. Keep tasks small enough for a focused PR. Record parallelism only when tasks do not contend for the same files, interfaces, or mutable data contract.

### 9. Implement one bounded task at a time

The implementation agent receives the accepted specification, one task from the breakdown, its dependencies, and the criteria it must satisfy. It keeps the change within that task's scope and reports files changed, decisions made, and any deviation from the accepted plan.

If implementation reveals a material ambiguity, a changed business rule, or a need to expand scope, stop the affected work and bring the evidence and proposed adjustment to the task owner. Do not quietly change the specification or add unrelated improvements. The owner updates and re-accepts the affected scope before implementation continues. Independent tasks may proceed in parallel only when their interfaces and files do not conflict.

### 10. Validate against the accepted criteria

For each task, connect every applicable acceptance criterion to observable evidence. Use the repository's existing checks and add focused checks when the change needs them. Record commands and outcomes, and identify any criteria that could not be verified and why. A passing test suite alone does not prove a business rule or BI definition is correct; include domain or data evidence where relevant.

When a check fails, report the failure and whether it blocks acceptance. Do not weaken a criterion or remove a check just to make the change pass; propose a specification adjustment to the task owner when evidence shows the original criterion is wrong or infeasible.

### 11. Prepare a focused pull request

Each PR should deliver one coherent outcome and be reviewable on its own. Prefer one task per PR; combine tasks only when they form one inseparable change. A PR description links the accepted specification and task, lists the acceptance criteria addressed, summarizes the implementation, and includes validation evidence, known limitations, and any proposed memory updates. It calls out schema, migration, permission, or data-definition changes explicitly.

Keep unrelated formatting, cleanup, and speculative features out of the PR. If an accepted task cannot fit in a reviewable PR, split it into smaller tasks and revise dependencies before continuing.

### 12. Review, integrate, and close the task

The task owner or repository's designated reviewer checks behavior against the accepted specification, PR scope, validation evidence, and relevant business rules. Apply the repository's branch protection and review requirements; Orbit does not assume it may merge or bypass them. Resolve review findings in the same bounded scope or create a follow-up task.

After merge or an explicitly recorded alternative disposition, update the task status and leave a compact handoff with the resulting commit/PR, completed criteria, outstanding follow-ups, and validation evidence. If a durable decision emerged, propose a memory change under the [memory contract](memory-contract.md), with source and status. A proposed memory update is not presented as current policy until its responsible owner accepts it.

### 13. Persist task handoff context

Before the session ends or hands work to another agent, provide a compact handoff with:

- accepted specification and task references;
- completed work and current status;
- decisions made and their sources;
- blockers and open questions;
- next action and validation evidence still needed.

Save durable, reusable decisions to project Markdown as reviewable changes. If Atomic Memory is available, propose appropriate shared memories with provenance and status. Do not store raw conversation transcripts by default.

## Expected output before implementation

For a substantial task, the user should be able to review:

1. A concise understanding of the request and relevant existing context.
2. Any conflicts or unresolved product/business/data decisions.
3. The proposed specialist roles and delegations, with brief reasons.
4. The draft specification and acceptance criteria.
5. A task breakdown with PR boundaries and validation evidence.

The task owner accepts the specification before implementation begins. The exact artifacts and approval state remain visible in the project rather than living only in agent conversation history.

## Token-efficiency signals

The workflow should compare delegated work's cost with the context it saved. Track, when host capabilities allow:

- delegated task count and role;
- input/output token counts or the best available estimate;
- duplicate or discarded findings;
- whether the delegation changed the spec, plan, or implementation;
- overall time and quality evidence.

Do not claim that subagents save tokens solely because they use separate contexts. Use delegation when it improves focus, parallelism, or review coverage, and tune the policy from observed results.

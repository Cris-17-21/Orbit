# Orbit memory contract

Orbit stores durable project context as Markdown files committed with the project. The files are the reviewable source of truth when Atomic Memory is not configured. Atomic Memory is optional and is not required to use this format.

## Memory boundaries and scopes

Orbit separates three kinds of memory:

- **Personal agent memory** belongs to one person's agent or account. It may contain that person's working preferences, but it is never shared or copied into a project automatically.
- **Project memory** belongs to the repository and is the default shared record. It is reviewable in Git and works without an external memory service.
- **Team/workspace memory** is optional shared context held in Atomic Memory when the team has configured an appropriate shared scope. Its audience and access depend on the configured Atomic Memory project/workspace permissions.

The adapter must identify which scope it is querying and report when it cannot verify the audience or access mode. A private/personal connection is not treated as team memory. Orbit must not copy records between scopes automatically. A person can propose an explicit promotion from personal context to project/team memory, with source and owner recorded, but the responsible person must confirm it.

Project Markdown remains the baseline source of truth. Atomic Memory is an optional retrieval and persistence backend, not an implicit authority. A hybrid sync mode is deferred until conflict resolution, identity mapping, and deletion behavior are designed. If both sources are present and disagree, surface the conflict to the task owner; do not choose by timestamp or silently overwrite either record.

## Memory categories

Keep durable knowledge in one of these categories:

- **Business rules** describe product behavior, constraints, and exceptions.
- **Technical decisions** record a chosen technical direction and its rationale.
- **Project context** gives stable background that helps collaborators understand the project.

Keep change-specific specifications and task plans in their own SDD artifacts. Do not use this memory format as a replacement for an accepted specification.

## Required record fields

Each record must identify:

| Field | Meaning |
| --- | --- |
| `type` | `business-rule`, `technical-decision`, or `project-context` |
| `id` | Stable identifier within its category, such as `BR-001` or `ADR-001` |
| `title` | Short human-readable name |
| `content` | The rule, decision, or context to preserve |
| `scope` | Project, product area, service, or data domain where it applies |
| `status` | `proposed`, `current`, or `superseded` |
| `created` | Date first recorded, in ISO `YYYY-MM-DD` format |
| `updated` | Date last substantively reviewed, in ISO `YYYY-MM-DD` format |
| `source` | Link or reference to the discussion, accepted specification, PR, or other evidence |
| `owner` | Person responsible for maintaining the record |
| `review_after` | Optional date after which the record should be rechecked before relying on it |

For a record with `status: current`, include `approved_by` and `approved_at` to identify when it became accepted. A proposed record is context to consider, not an active rule. A superseded record must point to its replacement using `superseded_by` when one exists. Withdraw a record by changing its status to `superseded` with an explanatory source; preserve its history rather than silently deleting it.

`review_after` is a reminder, not an automatic expiry. Once that date has passed, Orbit must label the item **review due** and ask the task owner to confirm it before using it as a controlling business rule. Orbit must not change its status or approve a replacement automatically. A missing `review_after` does not prove that a record is fresh: check its source and scope, and ask when repository behavior, data definitions, or other current records conflict with it.

## Approval and change lifecycle

1. **Capture:** record a candidate as `proposed`, with its source, scope, and accountable owner. Ideas from conversation, agent inference, or Atomic Memory retrieval begin as proposals unless an explicit approval is recorded.
2. **Review:** the responsible owner checks wording, scope, evidence, and affected work. For a business rule, this is the product/domain owner or a person they delegated; for a technical decision, the designated technical owner; for project context, the contributor responsible for its accuracy.
3. **Accept:** promote the proposal to `current` only after that responsible person confirms it. Record `approved_by` and `approved_at`. A task owner can accept a task-scoped memory change as part of their specification review. A team-wide policy follows the team's separately agreed decision process; Orbit does not require every collaborator to approve every task.
4. **Revise:** create or edit a proposal that cites the current record. When accepted, mark the prior record `superseded` and link both records. The change should be reviewable in the project PR or in an Atomic Memory audit mechanism when that backend provides one.
5. **Recheck:** if evidence conflicts, required review is due, ownership is unknown, or scope is unclear, mark the item as needing human review in the task context. Do not treat it as controlling until resolved.

Atomic Memory may represent approval, provenance, scope, and history differently from the Markdown template. Its adapter must map these fields where supported and report any guarantees it cannot provide. Orbit must not claim an item is approved, shared, audited, or current solely because it was retrieved from Atomic Memory.

## Agent behavior

1. Read records relevant to the current task and cite their IDs when using them in a specification or implementation plan.
2. Treat `current` as the record's declared state, but check its source and scope. Ask the task owner when the source is missing, the scope does not fit, or another current record conflicts.
3. Never promote a proposal or memory recall to `current` without confirmation from the responsible person.
4. Preserve history by marking replaced records `superseded` and linking the replacement; do not silently rewrite a decision that has already guided work.
5. Propose memory changes as reviewable Markdown diffs. The task owner can approve task-level records; changes to team-wide policy follow the team's agreed decision process.
6. Keep personal agent memory separate from these project files. Do not copy personal preferences or private information into shared project memory without explicit direction.

## File organization

Store records in category-specific files so collaborators can find and review them:

```text
project-context/
  business-rules.md
  technical-decisions.md
  project-context.md
```

Each file may contain multiple records. Keep each record self-contained, and use the required fields below its heading.

## Record template

```markdown
### <ID>: <Title>

- type: <business-rule | technical-decision | project-context>
- scope: <area this applies to>
- status: <proposed | current | superseded>
- created: <YYYY-MM-DD>
- updated: <YYYY-MM-DD>
- source: <link or evidence reference>
- owner: <responsible person>
- approved_by: <required when current>
- approved_at: <required when current; YYYY-MM-DD>
- review_after: <optional; YYYY-MM-DD>
- superseded_by: <replacement ID, when applicable>

<Content>
```

## Example: business rule

```markdown
### BR-001: Invitation conversion definition

- type: business-rule
- scope: workspace invitations and activation reporting
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: illustrative Orbit example; not approved for a real product
- owner: product owner

An invitation counts as converted only after the invitee accepts it and completes account registration. Sending or opening an invitation does not count as conversion.
```

This example remains `proposed` because it illustrates the format; it is not a confirmed rule for a real product. A real project must replace the source and responsible-person values with its own evidence and owners.

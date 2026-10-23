# Orbit memory contract

Orbit stores durable project context as Markdown files committed with the project. The files are the reviewable source of truth when Atomic Memory is not configured. Atomic Memory is optional and is not required to use this format.

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

For a record with `status: current`, include `approved_by` to identify the person who confirmed it. A proposed record is context to consider, not an active rule. A superseded record must point to its replacement using `superseded_by` when one exists.

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

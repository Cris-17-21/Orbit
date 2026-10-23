# Technical decisions

### ADR-001: Keep the conversion metric tied to explicit lifecycle events

- type: technical-decision
- scope: invitation conversion analytics
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: `../specs/invitation-conversion/spec.md` (draft example)
- owner: technical lead (example role)

The metric should be derived from explicit invitation lifecycle events with stable invitation and workspace identifiers, rather than inferred from email opens. The implementation mechanism, event names, retention, and warehouse model remain open until the project stack is inspected and the design is accepted.

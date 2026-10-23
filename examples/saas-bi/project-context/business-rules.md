# Business rules

### BR-001: Invitation conversion definition

- type: business-rule
- scope: workspace invitations and activation reporting
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: `../specs/invitation-conversion/spec.md` (draft example)
- owner: product owner (example role)

An invitation counts as converted only after the invitee accepts it and completes account registration. Sending or opening an invitation does not count as conversion.

### BR-002: Invitation eligibility

- type: business-rule
- scope: workspace invitations
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: `../specs/invitation-conversion/spec.md` (draft example)
- owner: product owner (example role)

Only a workspace administrator may send an invitation. The task owner must decide whether invitations expire and how duplicate or already-registered email addresses are handled before this rule can be accepted.

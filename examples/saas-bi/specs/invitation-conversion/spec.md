# Invitation conversion metric (draft example)

- status: draft
- owner: task owner (to be assigned)
- created: 2026-10-22
- related business rules: `../../project-context/business-rules.md` (proposed BR-001 and BR-002)
- related technical decision: `../../project-context/technical-decisions.md` (proposed ADR-001)

This is a worked example, not an accepted product specification. Resolve the open questions and obtain acceptance from the task owner before implementation.

## Need

Workspace owners need to understand how many invitations lead to activated accounts so they can assess workspace onboarding.

## Goal

Define and report the invitation conversion rate for a selected workspace and time period.

## Scope

- Define which invitations enter the denominator.
- Define which invitees count as converted.
- Report invitation and conversion counts with a rate for a selected period.
- Preserve workspace identity so one customer's events cannot be attributed to another workspace.

## Out of scope

- Choosing a BI vendor or warehouse.
- Changing the invitation email or registration experience.
- Defining billing, retention, or broader product activation metrics.

## Proposed business rules

These rules are proposals linked to the project memory files; neither is approved yet:

- BR-001: an invitation converts after acceptance and completed account registration.
- BR-002: only workspace administrators may send invitations.

## Metric definition (proposal)

- **Denominator:** distinct valid invitations sent during the selected period.
- **Numerator:** invitations in that denominator whose invitee accepted and completed account registration by the reporting cutoff.
- **Conversion rate:** numerator divided by denominator, expressed as a percentage.
- Exclude test, cancelled, and invalid invitations if the product can identify them. The owner must confirm the exclusion rules.
- An empty denominator is reported as “no data,” not as 0%.

The reporting window, treatment of late conversions, resend behavior, and exact exclusion rules are unresolved and must be confirmed before acceptance.

## Acceptance criteria

1. Given a workspace and reporting period, the report shows the denominator count, numerator count, and conversion rate using the accepted metric definition.
2. An invitation contributes no more than once to either count, including when it is resent.
3. An invitation contributes to the numerator only after both acceptance and completed registration.
4. An invitation from another workspace never appears in the selected workspace's counts.
5. When the denominator is empty, the report shows the agreed no-data state and does not display a misleading numeric rate.
6. Given identical source data, workspace, date range, and accepted cutoff, repeated report runs return the same numerator, denominator, and rate.

## Open questions for the task owner

- What exact reporting period and timezone should the metric use?
- Can an invitee accept after the selected period and still count as a conversion? What cutoff applies?
- Does resending create a new invitation or update the existing invitation?
- Which invitation statuses are excluded, and can the system identify test invitations?
- What does “completed account registration” mean in this product's existing flow?
- Which dashboard or report surface should display the metric?
- What data source, warehouse model, and refresh latency are required?

## Validation evidence to attach during implementation

- Automated checks covering numerator, denominator, duplicates, workspace isolation, and the no-data state.
- A query or data-model review showing that the implemented calculation matches the accepted definition.
- A screenshot or report preview for a representative workspace and period, if a UI is in scope.

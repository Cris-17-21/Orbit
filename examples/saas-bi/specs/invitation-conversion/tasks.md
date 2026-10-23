# Invitation conversion metric tasks (draft)

- status: draft
- owner: task owner (to be assigned)
- specification: `spec.md`

These tasks are a proposed implementation breakdown, not authorization to implement. The task owner must first resolve the open questions and accept `spec.md`. Keep each task on its own small PR where practical. If the inspected codebase changes the dependencies or scope, revise this plan before coding.

## Task 1: Define invitation lifecycle data

- **Goal:** Identify the existing invitation and registration records/events needed to calculate conversion.
- **Scope:** Inspect current schemas and flows; document the stable invitation, workspace, sent, accepted, and registration-completed identifiers/timestamps. Propose a minimal event or data-model change only if the current data is insufficient.
- **Depends on:** Accepted `spec.md` and resolution of the event/data-source questions.
- **PR boundary:** Data contract or schema change only; do not add dashboard UI.
- **Acceptance evidence:** Schema or event-contract review showing that each invitation can be attributed to one workspace and deduplicated; migration and compatibility notes if a schema change is needed.
- **Covers:** Criteria 2, 3, and 4.

## Task 2: Calculate the conversion metric

- **Goal:** Produce numerator, denominator, rate, and no-data behavior from the accepted definition.
- **Scope:** Implement the metric query or service using the accepted reporting window, timezone, cutoff, resend, and exclusion rules.
- **Depends on:** Task 1, when it changes or establishes the required data contract.
- **PR boundary:** Metric calculation and focused validation; no dashboard layout changes.
- **Acceptance evidence:** Automated checks or reproducible query examples for denominator, conversion timing, duplicates/resends, workspace isolation, exclusions, cutoff, and empty denominator.
- **Covers:** Criteria 1–5.

## Task 3: Display the metric

- **Goal:** Show the agreed counts and rate to an authorized workspace user.
- **Scope:** Add the report surface, date range, loading/error/no-data states, and access checks required by the accepted design.
- **Depends on:** Task 2 and an accepted decision about the dashboard/report location and authorized roles.
- **PR boundary:** Report UI and its integration; avoid unrelated navigation or design-system changes.
- **Acceptance evidence:** UI checks for counts, rate, date range, loading/error/no-data states, and workspace authorization; attach a screenshot or preview when the project supports it.
- **Covers:** Criteria 1, 4, and 5.

## Task 4: Verify end-to-end behavior

- **Goal:** Confirm the shipped report matches the accepted specification across the full invitation lifecycle.
- **Scope:** Validate representative invitation data through ingestion/modeling and the report, including the accepted reporting cutoff.
- **Depends on:** Tasks 1–3.
- **PR boundary:** Add or adjust end-to-end validation and operational notes; fix product code in a separate PR if findings exceed this task's scope.
- **Acceptance evidence:** Recorded scenario results for a converted invitation, an unconverted invitation, duplicate/resend behavior, a different workspace, and an empty period; confirm repeat runs are stable for identical inputs.
- **Covers:** Criteria 1–6.

## Dependency sketch

```text
Accepted specification
          |
          v
Task 1: lifecycle data
          |
          v
Task 2: metric calculation
          |
          v
Task 3: report UI
          |
          v
Task 4: end-to-end verification
```

If inspection shows the current data already supports the metric, Task 1 may become a short evidence-only review and Task 2 can proceed without a schema change. The task owner should keep the PR boundary small and update this plan when that happens.

# Atomic Memory evaluation

- status: research and pilot proposal
- checked: 2026-10-23

This note records what Orbit can safely assume about Atomic Memory and proposes a small team-memory pilot. It does not configure an Atomic Memory account or agent host.

## What the documentation confirms

- Atomic Memory exposes an MCP server with `memory_search`, `memory_ingest`, `memory_package`, and `memory_list`. MCP gives compatible agents tools; it does not by itself define Orbit's business-rule approval workflow.
- Atomic Memory distinguishes personal user scope from a workspace scope. Its Open Source Core documentation describes workspace visibility for shared, agent-only, and restricted memories. Workspace reads identify both the workspace and the calling agent.
- The SDK concept of a team workspace uses a user plus namespace. The Core HTTP workspace contract uses a workspace ID and agent ID. These are related concepts but not interchangeable configuration fields; Orbit must verify the selected backend and interface before mapping them.
- Core's workspace is described as a collaboration boundary inside a user's tenancy, while the SDK says applications supply user identity. The docs do not establish how two separately authenticated human collaborators are joined to one workspace or how that identity is enforced at the MCP boundary. This is a pilot question, not a capability Orbit should assume.
- The current Hosted Cloud integration guidance says to omit `ATOMICMEMORY_SCOPE_NAMESPACE` until a documented scope-filter/ingest defect is fixed. Do not build Orbit's team-sharing setup around a Hosted Cloud namespace assumption.
- The published coding-agent integration list documents Codex and Claude Code, among other hosts, but does not list Antigravity. A generic MCP connection may be possible where the agent host supports MCP, but that pairing is unverified.
- Atomic Memory's retrieval result is context, not approval. Orbit must continue to apply its own status, provenance, scope, and responsible-owner rules from the [memory contract](memory-contract.md).

## Recommended pilot

Evaluate the collaboration behavior on a disposable Open Source Core workspace before choosing a production backend or implementing an installer integration. Use synthetic data. First check whether two separately authenticated collaborators can safely access one workspace. Do not share one human's credentials or emulate shared access by assigning everyone the same user ID. If the chosen setup cannot authenticate and authorize each collaborator, record that as a blocker and keep the pilot local to one operator.

1. Connect one supported host for collaborator A and one for collaborator B to the same test Core workspace, only after the identity/access boundary is understood.
2. Have A store a clearly marked **proposed** business rule with a stable ID, project scope, source, and owner.
3. Have B search for that rule and confirm the result retains enough identity and source information for Orbit to treat it as a proposal.
4. Store an `agent_only` scratch memory for A and confirm B cannot retrieve it. Store a shared workspace memory and confirm both can retrieve it.
5. Replace a test rule with a new version. Confirm the old item is not presented as current merely because it remains searchable, and verify that Orbit can preserve the replacement link and approval evidence.
6. Repeat the host connection check for Codex and Claude Code. Treat Antigravity as a separate compatibility check, not an assumed integration.

Keep the test records synthetic. Ingest distilled facts or handoff summaries, never raw conversation transcripts by default. Remove the disposable test workspace and test data after recording the result.

## Pilot acceptance criteria

- Both collaborators can retrieve an explicitly shared test record through their own agent identities.
- Each collaborator's identity and workspace membership are enforced by the service or an explicitly documented authenticated wrapper; a shared API key plus caller-supplied IDs alone does not count as verified collaborator isolation.
- A private/agent-only test record is not visible to the other collaborator.
- Search results can be associated with a stable record reference, source, and declared proposal/current/superseded state, or the pilot records the missing mapping as a blocker.
- Updating a memory does not cause Orbit to silently promote it or erase the prior approval trail.
- The host reports whether the MCP server is configured and reachable; configured-but-unverified is not reported as connected.
- The team records setup effort, required service credentials, model/provider costs if any, data location, and cleanup steps.

## Decision after the pilot

Choose among:

- keep Markdown + Git as the only shared memory for now;
- offer Atomic Memory as an optional retrieval layer while Markdown remains the reviewable authority; or
- investigate a self-hosted workspace setup further if its access controls and operating cost fit the team.

Do not promise Hosted Cloud project-shared memory until the documented namespace defect is resolved and verified. Do not promise Antigravity support until the MCP connection and scope behavior pass the pilot.

## Official sources checked

- [Atomic Memory integrations](https://docs.atomicstrata.ai/integrations/overview)
- [Atomic Memory scope contract](https://docs.atomicstrata.ai/platform/scope)
- [SDK scopes and identity](https://docs.atomicstrata.ai/sdk/concepts/scopes-and-identity)
- [Atomic Memory CLI and current Hosted Cloud notes](https://docs.atomicstrata.ai/cli)
- [Codex integration guide](https://docs.atomicstrata.ai/integrations/coding-agents/codex/local)
- [Claude Code integration guide](https://docs.atomicstrata.ai/integrations/coding-agents/claude-code/local)

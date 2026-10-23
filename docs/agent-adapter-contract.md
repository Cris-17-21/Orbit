# Orbit agent adapter contract

- status: design proposal
- updated: 2026-10-23

Orbit has one workflow and one source library of skills. An agent adapter maps that shared contract into a host's supported files and runtime capabilities. An adapter must not silently claim that two hosts behave identically.

## Shared adapter responsibilities

Every adapter must provide these operations to the Go CLI:

1. **Detect:** identify whether the host is installed, its version, active configuration roots, and usable project/user scopes.
2. **Describe capabilities:** report support and evidence for instructions, skills, subagent definitions, runtime delegation, MCP, lifecycle hooks, and usage telemetry.
3. **Plan:** return the exact files and structured configuration entries it intends to create, update, skip, or remove.
4. **Apply:** write only resources approved by Orbit's installer and recorded in the ownership manifest.
5. **Verify:** report whether the host can discover the installed workflow, skills, subagent profiles, and optional Atomic Memory connection.
6. **Remove:** remove or restore only manifest-owned resources, following the safe ownership rules in [installation ownership](install-ownership.md).

Capability values are:

- `native`: host provides the capability and the adapter can configure it.
- `manual`: the host supports it, but the user must complete a host UI, authentication, or trust action.
- `fallback`: Orbit can continue with reduced behavior in the main agent, without pretending the native capability exists.
- `unavailable`: the host/runtime version does not provide it or Orbit cannot verify it.

The adapter reports the host version and evidence for any `native` claim. Unknown versions use conservative capabilities and trigger a compatibility notice.

## Shared skill contract

- Keep the canonical skill source in Orbit's own repository namespace.
- Use the Agent Skills folder format with one `SKILL.md` per skill, concise `name` and `description` metadata, and optional supporting references/assets/scripts.
- Keep skills focused. Load detailed instructions only when a skill is selected or invoked.
- Prefix Orbit skill names and generated slash commands with `orbit-` where host naming rules allow it.
- Preserve user skill names and directories. A collision is handled by Orbit's safe installation conflict flow.
- Do not put long always-loaded workflow text in every host's global prompt. Install a short entry instruction that points the agent to the orchestrator skill.
- Keep templates and project artifacts in the repository; do not copy arbitrary project context into a user's global configuration.

The precise destination path and any host-specific metadata are owned by each adapter and must follow that host's current documentation.

## Shared subagent contract

A role profile is host-neutral content containing:

- stable role ID and short name;
- concise delegation description and trigger conditions;
- focused instructions and expected output format;
- required context references;
- minimum tools/permissions needed, classified as read-only or write-capable;
- suggested concurrency policy, with no model/provider hard-coded by default.

The adapter translates a role profile into native subagent files or plugin metadata when available. The orchestrator delegates only independent, bounded work. It passes only the relevant specification, memory records, files, and skills. It asks for concise findings with evidence references and unresolved questions.

When native subagents are not available in the active host/runtime, `fallback` means the primary agent does the work itself or asks the user to run a separate session. Orbit must not create nested prompts and call that a subagent, and must not claim token savings. If the host exposes usage, record main and delegated usage separately; otherwise report that token savings cannot be measured for that run.

## Capability matrix

This matrix records the current design baseline. Exact file paths and version support remain adapter implementation details and must be checked against official host documentation when each adapter is built.

| Capability | Codex | Claude Code | Antigravity |
| --- | --- | --- | --- |
| Project instructions | `AGENTS.md` recognized; keep Orbit entry guidance short and namespaced | `CLAUDE.md` plus skills; keep always-loaded guidance short | Rules and workspace skills; use project scope where possible |
| Shared skills | `SKILL.md` skills supported; package or install project skills using the host's documented scope | `SKILL.md` under Claude skill conventions; skill bodies load when invoked | Agent Skills supported; workspace skills and plugin-provided skills supported |
| Native subagent definitions | Runtime-dependent; verify installed Codex surface. OpenAI's managed Codex/Agents API documents multi-agent delegation | Native custom subagents with separate contexts, tool sets, permissions, and optional preloaded skills | Antigravity CLI documents asynchronous subagents; plugin bundles can include agent profiles |
| Delegation fallback | Use primary agent if active Codex runtime does not expose delegation | Use primary agent for work that does not benefit from delegation | Use primary agent if the active Antigravity surface lacks the required delegation control |
| MCP | Supported; configure through Codex's supported MCP settings/CLI | Supported with local, project, and user scopes | Supported; CLI/IDE can use global or workspace MCP configuration |
| Atomic Memory | Atomic Memory documents a Codex MCP integration | Atomic Memory documents a Claude Code integration, including memory skill and lifecycle hooks | Antigravity supports MCP, but Atomic Memory's coding-agent integration table does not currently list an Antigravity-specific adapter; begin with generic MCP setup and verify in a pilot |
| Usage/token visibility | Host/version-dependent; report only measured values | Provider/runtime-dependent; subagent usage may count toward the same usage limits | Host/version-dependent; report only measured values |

## Host-specific behavior

### Codex

- Install the shared orchestrator skill and short project instruction using the selected project or user scope.
- Add the Atomic Memory MCP entry only when the user opted in and credentials are available through a supported secure mechanism.
- Treat native subagent support as a runtime capability, not an assumption based solely on the agent name. The OpenAI Agents API uses a managed Codex harness and documents subagent orchestration; that does not automatically establish parity for every Codex client or version.
- Verify skills, MCP, and any available delegation with the installed Codex surface and report partial support clearly.

### Claude Code

- Install namespaced skills and project guidance using Claude Code's project-scoped conventions.
- Install namespaced subagent profiles with least-privilege tool access and concise descriptions; detailed instructions load only when the subagent runs.
- Configure Atomic Memory MCP at project scope when sharing should be project-specific. Respect Claude Code's project trust and approval flow; Orbit should report when the user must approve or authenticate the server.
- Do not copy credentials into checked-in settings or skill files.

### Antigravity

- Prefer its documented plugin bundle when that gives Orbit a stable namespaced container for skills, agent profiles, rules, MCP, and hooks.
- For project-local skills and MCP, use the workspace locations documented by the active Antigravity product surface.
- Since Atomic Memory currently documents no Antigravity-specific coding-agent package in its integration overview, configure its MCP endpoint as a generic server only after validating the current server schema, scope identity, and credential flow. `doctor` must label this route as generic MCP rather than a vendor-maintained Atomic Memory adapter.
- The host may require a UI-based trust or server approval step; report that as `manual` and verify the connection afterward.

## Atomic Memory normalization

When enabled, the Orbit orchestrator should use the same memory behavior regardless of host:

- Search the configured project/workspace scope at task start.
- Include only relevant memories and preserve their source, scope, status, and provenance.
- Use a stable identity for the human/workspace and a distinct caller identity for each adapter/agent where required.
- Store approved durable project knowledge only after proposing a reviewable update; keep personal memory separate.
- Use Atomic Memory's MCP tools through the host adapter. The workflow skill determines when and how to call the tools; the memory server handles storage and scope enforcement.
- If Atomic Memory is configured but unreachable, continue from project Markdown, warn that shared recall is unavailable, and do not claim the memory was written.

The adapter must check the current Atomic Memory scope and integration requirements at implementation time; provider scope and host connection behavior can change independently.

## Installation verification

`orbit doctor` reports, per selected host:

- detected host name and version;
- selected installation scope and managed targets;
- orchestrator instruction/skill discovery status;
- subagent definition and runtime delegation status separately;
- MCP server configured, requiring approval/authentication, connected, or unavailable;
- Atomic Memory mode: vendor integration, generic MCP, disabled, or unknown;
- any unsupported, manual, or fallback behavior.

`doctor` must not mark an MCP server connected solely because its config entry exists. It should use host/server status when available and otherwise report `configured, connection unverified`.

## Current documentation references

- [Codex and OpenAI skills](https://developers.openai.com/plugins/concepts/skills)
- [Codex/Agents API multi-agent](https://developers.openai.com/api/docs/guides/agents-api/multi-agent)
- [Claude Code skills](https://code.claude.com/docs/en/skills)
- [Claude Code subagents](https://code.claude.com/docs/en/sub-agents)
- [Claude Code MCP](https://code.claude.com/docs/en/mcp)
- [Antigravity plugins and skills](https://www.antigravity.google/docs/cli/plugins/)
- [Antigravity CLI subagents](https://www.antigravity.google/docs/cli/features/)
- [Antigravity MCP](https://www.antigravity.google/docs/mcp)
- [Atomic Memory integrations](https://docs.atomicstrata.ai/integrations/overview)
- [Atomic Memory scopes](https://docs.atomicstrata.ai/platform/scope)

These references describe separate products and release tracks. Recheck support, configuration paths, and authentication when implementing or updating an adapter.

## Open implementation decisions

- Minimum supported version for each agent and how adapters discover that version.
- Whether Codex plugin packaging, project-local skills, or both are used for each install scope.
- Exact role file conversion and permissions mapping for every native subagent system.
- Which Atomic Memory deployment modes are supported on each host at first release.
- What metrics are exposed by each host and how Orbit records token usage without storing sensitive prompts.
- How to test each adapter in CI without requiring a user's authenticated agent installation.

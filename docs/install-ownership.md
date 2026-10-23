# Orbit installation ownership and conflict handling

- status: design proposal
- updated: 2026-10-23

This document defines how the Orbit CLI should track the files and configuration entries it manages. The goal is to make project-local and user-wide installs repeatable while preserving existing agent customizations and skills.

## Ownership rules

1. Orbit owns only files or configuration entries recorded in its install manifest.
2. Orbit never claims an entire agent configuration file when it only needs to add one server, skill reference, or managed section.
3. Orbit places its resources under an Orbit-specific namespace wherever the agent supports one.
4. Existing user skills, files, settings, and unrelated configuration are preserved.
5. A conflict is shown to the user; it is never resolved by silently overwriting user content.
6. Install, update, and uninstall operate on a planned set of changes. The CLI shows the plan before applying it in interactive mode and supports a no-write preview in automation.

## Install manifest

Each installation scope has one Orbit-owned manifest. A project-local installation stores it in the project; a user-wide installation stores it in Orbit's user configuration directory.

The manifest records, at minimum:

| Field | Purpose |
| --- | --- |
| `schema_version` | Version of the manifest format so the CLI can migrate it safely |
| `orbit_version` | Exact Orbit release installed |
| `scope` | `project` or `user` |
| `project_root` | Canonical project root for a project installation |
| `installed_at` | Installation timestamp |
| `resources[]` | Files and configuration entries managed by Orbit |

Each resource record includes:

| Field | Purpose |
| --- | --- |
| `resource_id` | Stable Orbit identity, independent of the target path |
| `agent` | Adapter that owns or references the resource |
| `kind` | `file`, `config-entry`, or `managed-section` |
| `target` | Canonical path or structured key/section selector |
| `source` | Orbit package path and release version used to create it |
| `installed_digest` | Digest of the exact managed bytes or serialized entry last written |
| `previous_value` | Prior value or absence marker when Orbit added a config entry |
| `backup_ref` | Optional recovery copy for a replacement approved by the user |
| `state` | `managed`, `modified`, `conflict`, or `orphaned` |

The manifest must not contain credentials, API keys, or other secrets. If a host requires a credential, use the host's supported credential store or environment reference and record only that the connection is configured.

## Install flow

1. Resolve and validate the requested scope and project root.
2. Load the selected Orbit release and verify its published digest before installing it.
3. Detect selected agents and their existing skill/configuration paths without changing them.
4. Build an install plan listing every new file, config entry, conflict, and skipped optional integration.
5. Check the plan for path escape, duplicate targets, existing unowned files, and incompatible manifest versions.
6. Show the plan in the guided interface. In non-interactive mode, print it with `--dry-run`; require explicit flags for conflict choices.
7. Apply writes through temporary files and atomic replacement where the platform allows it. Preserve unrelated configuration entries and retain a recovery copy before any approved replacement.
8. Write or update the manifest only after all planned resources have been applied and checked. If a step fails, restore the prior state from the transaction journal and report any incomplete recovery.

The default conflict choice is to skip the conflicting target and keep going with non-conflicting resources, while clearly reporting that the integration is partial. The user can choose a distinct Orbit-prefixed name, a different installation scope, or a manual resolution. Orbit does not overwrite the target as a default.

## Update flow

For each manifest resource, compare the current target with the last `installed_digest`:

- **Unchanged managed resource:** replace it with the selected Orbit release and update the manifest digest.
- **User-modified managed file or section:** preserve it and report a conflict. Offer a side-by-side candidate, a three-way merge preview when a trustworthy base is available, or skip. Never replace it silently.
- **Missing resource:** offer to restore it from the selected release.
- **New resource in the release:** plan it as a new target, applying the normal collision checks.
- **Resource no longer shipped by Orbit:** mark it `orphaned` and offer removal; do not remove it automatically during update.

Before applying an update, create a recovery point containing the previous Orbit version, manifest, and exact prior contents of resources that will change. Rollback restores that recovery point only for resources whose current content still matches the version installed by the update. If a resource changed after the update, leave it in place and report a manual conflict instead of erasing newer work.

## Uninstall flow

Uninstall reads only the selected scope's manifest and previews the resources it would remove or restore:

- Remove an unchanged Orbit-managed file.
- Remove an Orbit-added config entry only if that exact entry still matches Orbit's installed digest; preserve other keys and comments when the format permits.
- Restore a previous config value only when the current value still equals Orbit's installed value.
- Leave user-modified resources in place, mark them for manual cleanup, and show their paths.
- Never recursively remove a parent folder that contains unowned resources.

After successful cleanup, remove the manifest. If conflicts remain, keep a small manifest/report of unresolved Orbit-owned resources until the user resolves or acknowledges them.

## Editing shared agent configuration

Some hosts keep MCP servers or settings in a shared JSON/TOML file. Orbit should prefer the host's supported CLI for adding a uniquely named server entry when that command preserves unrelated settings. Otherwise, the adapter must edit only a namespaced key or managed section and record the exact before/after value in the manifest.

If an adapter cannot make a narrow, verifiable edit without rewriting unrelated content, it should not write that file automatically. It can generate a patch for the user to apply and mark that integration as manual in `doctor`.

## Recovery and audit output

Every install, update, rollback, and uninstall produces a concise summary of:

- Orbit version before and after;
- files and config entries added, changed, restored, skipped, or left conflicted;
- backup/recovery location;
- integrations that need a manual step;
- next command for diagnosis.

The CLI should support a read-only `orbit doctor` view of manifest integrity, installed version, missing or modified resources, and agent integration status.

## Open implementation decisions

- Exact manifest filename and user configuration directory.
- Digest algorithm and canonical serialization for structured config entries.
- How long recovery points are retained and how the user removes them.
- Whether a release may rename a resource while preserving its stable `resource_id`.
- Which agent CLIs can safely add/remove scoped entries without editing files directly.
- How to handle shared configuration files edited concurrently by an agent during an install transaction.

Resolve these details in the CLI architecture and adapter specifications before implementing writes to each host's configuration.

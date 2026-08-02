# SDK capability sync

This directory tracks how much of the [Eon Go SDK](https://github.com/eon-io/eon-sdk-go)
surface this provider exposes, and records the triage decision for every SDK
operation. It is generated and updated by the analyzer in `tools/capsync`.

## Files

| File | What it is | Who edits it |
|---|---|---|
| `manifest.yaml` | Every SDK operation with its triage classification, reason, and coverage status. The pipeline's decision store. | capsync seeds new entries; **humans own the decisions** |
| `gap-report.md` | Human-readable gap report for the SDK release named in the manifest. | Generated — do not edit |
| `gap-report.json` | The same report, machine-readable. | Generated — do not edit |

## Running the analyzer

```sh
make gap-report                                  # analyze the SDK version in go.mod
go run ./tools/capsync -sdk-version latest       # analyze the newest SDK release
go run ./tools/capsync -sdk-version latest -update-manifest
go run ./tools/capsync -check                    # CI: fail if new/unreviewed operations exist
```

## Triage rules

Each operation is classified as `resource`, `data_source`, or `skip`. The
guiding heuristic: **if a practitioner cannot express it as desired state that
Terraform can reconcile and drift-detect, it does not belong in the provider.**

- **resource** — a durable, declaratively-managed object with stable identity
  and a real lifecycle (policies, connected accounts, vaults, roles,
  configuration singletons, override/exclusion toggles).
- **data_source** — a read-only lookup whose output is useful as input to
  other resources (accounts, vaults, roles, snapshots, permissions).
- **skip** — imperative one-shot actions (trigger a backup, cancel a job),
  job polling, reporting/analytics/metrics/billing, inventory plumbing,
  auth/session/credential endpoints, and the entire agentic surface
  (agents/chat/assistants/MCP), regardless of shape.

## Overriding a decision

Edit `classification`, `reason`, `terraform_name`, `notes`, or `needs_review`
in `manifest.yaml` and commit. capsync never rewrites those fields once an
entry exists, so overrides stick: an operation you mark `skip` is never
re-proposed, and one you promote to `resource`/`data_source` shows up as a gap
until it is implemented. The factual fields (`method`, `path`, `status`,
`covered_by`, `first_seen`) are recomputed on every run — don't edit those.

## Automation

Two workflows close the loop (nothing ever merges automatically — every PR
opens as a draft for human review):

- **SDK Capability Sync** (`.github/workflows/sdk-capability-sync.yml`) —
  bumps `eon-sdk-go`, reruns capsync, and opens a draft `sdk-sync` PR with the
  refreshed gap report and any newly seeded triage proposals. Runs on:
  - `repository_dispatch` with `event_type: sdk-release` (the SDK release
    pipeline POSTs `{"event_type":"sdk-release","client_payload":{"version":"vX.Y.Z"}}`
    to this repo's `/dispatches` endpoint — see the workflow header for the
    exact curl);
  - a daily scheduled fallback that compares the latest SDK tag with `go.mod`;
  - manual `workflow_dispatch`.

  Optional secret `SDK_SYNC_TOKEN` (PAT with repo scope): when set, CI runs
  automatically on the PRs this workflow opens; with the default
  `GITHUB_TOKEN`, GitHub suppresses those runs.

- **Implement Capability** (`.github/workflows/capability-implement.yml`) —
  dispatched with a capability name from the gap report (e.g.
  `eon_backup_posture_control`). Claude Code implements it end to end
  following the repo's existing patterns and opens a draft, labeled PR — one
  capability per PR, based on `main` or chained off an open sdk-sync branch.
  Requires the `ANTHROPIC_API_KEY` secret.

  Two dispatch modes:
  - **Manual** (default): a human picks the capability and runs the workflow.
  - **Automatic**: set the repository variable `CAPSYNC_AUTO_IMPLEMENT=true`
    and the sync workflow auto-dispatches implementation jobs (max 3 per run)
    for every gap whose triage is already settled — i.e. classified in the
    manifest on a previous run and not flagged `needs_review`. Operations the
    SDK release just introduced are never auto-implemented; they wait for a
    human to confirm the seeded classification in the sync PR first.

The manifest keeps both honest: implemented operations flip to `covered` on
the next run, operations you mark `skip` are never proposed again, and brand
new SDK operations always arrive as reviewable triage proposals in the
sdk-sync PR.

## Coverage detection

Coverage is detected statically: capsync maps each resource/data source in
`internal/provider` to the `internal/client` wrapper methods it calls, and each
wrapper to the SDK operations it invokes (including the hand-rolled HTTP calls
for endpoints missing from the SDK). When a gap ships, the next run flips its
status to `covered` automatically — no manifest edit needed.

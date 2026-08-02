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

The automation runs in the **private `eon-io/eon-service` repository** — the
same repo that generates and releases `eon-sdk-go` — so no automation secrets
or workflows live in this public repo. This repo only carries the analyzer
(`tools/capsync`) and the committed manifest/reports; the eon-service
workflows check this repo out, run the analyzer here, and push draft PRs back.
Nothing ever merges automatically.

- **`terraform-provider-capability-sync.yml`** (in eon-service) — chained
  directly off `update-sdk.yml`, the workflow that publishes each `eon-sdk-go`
  release, so a release triggers the sync in the same run; also runs on a
  daily scheduled fallback and manual dispatch. It bumps `eon-sdk-go` here,
  reruns capsync, optionally has Claude finalize the triage of newly
  discovered operations (`CAPSYNC_LLM_TRIAGE=true`; each decision is recorded
  as `notes: llm-triage(<version>): ...` and listed in the PR body — merging
  accepts it, editing the manifest overrides it), and opens a draft `sdk-sync`
  PR on this repo.

- **`terraform-provider-capability-implement.yml`** (in eon-service) —
  dispatched with a capability name from the gap report, manually or
  automatically by the sync run (`CAPSYNC_AUTO_IMPLEMENT=true`, max 3 per run,
  settled gaps only). Claude Code implements the capability end to end
  following this repo's patterns and opens a draft `capability-sync` PR here —
  one capability per PR.

Setup lives entirely in eon-service: `SDK_REPO_TOKEN` (repo scope on this
repo; already used to publish `eon-sdk-go`), `ANTHROPIC_API_KEY`, and the two
repository variables above.

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

# AGENTS.md

This file provides guidance to coding agents (e.g. Claude Code, claude.ai/code) when working with code in this repository.

## Repository purpose

Go module `kubeops.dev/supervisor` — a Kubernetes operator that gates **`Recommendation`s** (cluster-side suggestions to perform an operation: cert renewal, volume expansion, version patch, license renewal, etc.) behind **`MaintenanceWindow`s** and approval policies. An ops-manager (e.g. KubeDB, KubeVault) creates `Recommendation` objects when it detects an action is needed; supervisor decides *when* and *whether* to apply them based on namespace/cluster `MaintenanceWindow` schedules, parallelism limits, and `ApprovalPolicy`. See `design.md` for the original design notes and YouTube link.

CRDs (`supervisor.appscode.com/v1alpha1`):
- `Recommendation` — a recommended operation, with current approval/execution state.
- `MaintenanceWindow` (namespaced) — schedule for that namespace's recommendations.
- `ClusterMaintenanceWindow` (cluster-scoped) — cluster-wide schedule.
- `ApprovalPolicy` — auto-approval rules per resource/GroupKind.

The README is a Kubebuilder scaffold stub; treat this file and `design.md` as the source of truth.

The produced binary is `supervisor`.

## Architecture

- `cmd/supervisor/` — entry point.
- `pkg/cmds/` — Cobra root + run + webhook subcommands.
- `apis/supervisor/v1alpha1/`:
  - `recommendation_types.go`, `maintenancewindow_types.go`, `clustermaintenancewindow_types.go`, `approvalpolicy_types.go`.
  - `recommendation_helper.go`, `clock_helper.go`, `constants.go`.
  - `groupversion_info.go`, `doc.go`, generated `zz_generated.deepcopy.go` / `openapi_generated.go`.
  - `install/`, `fuzzer/` — scheme registration + fuzz helpers.
- `crds/` — generated CRD YAMLs.
- `pkg/controllers/supervisor/` — reconcilers for the four CRDs.
- `pkg/policy/` — `ApprovalPolicy` evaluation.
- `pkg/evaluator/` — recommendation evaluator (decides Approved / Rejected / WaitingForApproval).
- `pkg/maintenance/` — `MaintenanceWindow` time-window resolution.
- `pkg/deadline-manager/` — deadline tracking (recommendations that must run before a date).
- `pkg/parallelism/` — concurrency limits (how many recommendations may execute simultaneously).
- `pkg/shared/` — shared helpers.
- `pkg/webhooks/supervisor/` — admission validators/mutators.
- `artifacts/`, `doc/` — sample manifests and design docs.
- `Dockerfile.in` (PROD, distroless), `Dockerfile.dbg` (debian), `Dockerfile.ubi` (Red Hat certified) — three image variants.
- `PROJECT` — Kubebuilder metadata. Domain `appscode.com`, multigroup.
- `hack/`, `Makefile` — AppsCode build harness.
- `test/` — e2e tests.
- `vendor/` — checked-in deps.

CRD API group is `supervisor.appscode.com`.

## Common commands

All Make targets run inside `ghcr.io/appscode/golang-dev` — Docker must be running.

- `make ci` — CI pipeline.
- `make build` / `make all-build` — build host or all-platform binaries.
- `make gen` — regenerate clientset + manifests + openapi. Run after any change to `apis/supervisor/v1alpha1/*_types.go`.
- `make manifests` — regenerate CRDs only.
- `make clientset` — regenerate client code.
- `make openapi` — regenerate OpenAPI definitions.
- `make fmt`, `make lint`, `make unit-tests` / `make test` — standard.
- `make verify` — `verify-gen verify-modules`; `go mod tidy && go mod vendor` must leave the tree clean.
- `make container` — build PROD, DBG, and UBI images.
- `make push` — push all three; `make docker-manifest` writes multi-arch manifests; `make release` is the full publish flow.
- `make push-to-kind` / `make deploy-to-kind` — load into Kind and Helm-install.
- `make install` / `make uninstall` / `make purge` — Helm install lifecycle.
- `make add-license` / `make check-license` — manage license headers.

Run a single Go test (requires a local Go toolchain):

```
go test ./pkg/evaluator/... -run TestName -v
```

## Conventions

- Module path is `kubeops.dev/supervisor` (vanity URL). Imports must use that.
- License: see `LICENSE`; new files need the standard "Copyright AppsCode Inc. and Contributors" header (`make add-license`).
- Sign off commits (`git commit -s`); contributions follow the DCO.
- Vendor directory is checked in — `go mod tidy && go mod vendor` must leave the tree clean (enforced by `verify-modules`).
- Approval flow is the heart of this operator: `Recommendation` → evaluated (`pkg/evaluator/`) against `ApprovalPolicy` (`pkg/policy/`) and current `MaintenanceWindow` (`pkg/maintenance/`), then gated by `pkg/parallelism/` and tracked by `pkg/deadline-manager/`. Keep these concerns separated — don't merge the evaluator and parallelism limits into one package.
- Do not hand-edit `zz_generated.*.go`, `openapi_generated.go`, or `crds/*.yaml` — change `apis/supervisor/v1alpha1/*_types.go` and re-run `make gen`.
- Time-window logic uses `clock_helper.go` for testability; reach for that interface rather than calling `time.Now()` directly.
- Three Dockerfiles, one binary — keep `Dockerfile.in`, `Dockerfile.dbg`, and `Dockerfile.ubi` in sync.
- This is a **Kubebuilder multigroup project** — use `kubebuilder` to scaffold new APIs.

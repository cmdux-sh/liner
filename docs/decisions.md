# Decisions

## 2026-10-20 - Move Linux CI to the bounded Depot free trial

Arturo approved moving Linux CI to Depot through 2026-10-27T16:48:00Z, the same cutoff as Akast, to avoid automatic GitHub-hosted Actions usage. Preserve all existing checks and the manual GitHub fallback, and guard each Depot job before checkout. Native macOS and Windows builds remain local/manual because native Depot CI does not support those platforms. This decision does not authorize paid upgrades, package publication, or deployment. See [the trial coverage and rollback record](ci/depot-ci-trial.md).

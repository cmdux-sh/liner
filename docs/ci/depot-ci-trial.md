# Depot CI free trial

Decision recorded on 2026-10-20: Arturo approved moving Linux CI to Depot's
existing organization for the same free trial as Akast, to avoid automatic
GitHub-hosted Actions usage. This does not authorize paid upgrades.

The cutoff is **2026-10-27T16:48:00Z** (Unix timestamp `1793119680`). Every
Depot job checks this before checkout or dependency installation and fails closed
at or after the cutoff. Scheduling and sandbox startup can still consume usage;
this guard is not a billing cap. Disable the Depot workflows or repository
integration before expiry to prevent further scheduling. Do not extend the cutoff
or enable paid usage without Arturo's approval.

## Coverage

- `.depot/workflows/public-surface.yml` retains every release-metadata, public
  command, public-surface, and marketing build check on pull requests and pushes
  to `master`. Its job has a 20-minute limit.
- `.depot/workflows/platform-bundles.yml` remains manual, with Linux x64,
  Linux arm64, and the main `linersh` package as its only targets. It retains
  registry validation, package validation/tests, packing, and artifact upload;
  it does not publish packages. Target resolution is limited to 5 minutes,
  public validation to 20 minutes, and package builds to the existing 45 minutes.
- Depot labels are `depot-ubuntu-24.04` and `depot-ubuntu-24.04-arm`, including
  labels emitted by target resolution. Native Depot CI supports neither macOS
  nor Windows; those builds remain local/manual.
- The original `.github/workflows/platform-bundles.yml` remains unchanged as a
  manual fallback for all platform targets. The GitHub public-surface workflow
  has only its trigger changed to `workflow_dispatch`; all its job steps remain.
  Manually invoking either GitHub fallback consumes GitHub-hosted minutes.

Actions, language versions, permissions, and original validation commands are
preserved. No real library, runtime profile, provider credentials, public
mirroring, package publication, or deployment is part of this change.

## Activation and rollback

Depot's GitHub App must have access to this repository. Depot documents automatic trigger
registration on the default branch; inspect actual PR check runs because an
installed integration may validate a proposed workflow before merge. If no run
appears, validate with an explicitly selected Depot CLI run of the public-surface
workflow only. Do not run the manual packaging workflow as part of migration
acceptance.
No workflow references custom secrets or variables, so none need importing.

To roll back, disable Depot triggers first, then restore the original GitHub
public-surface pull-request and push triggers, using `master` as the push
branch. Keep the manual packaging fallback available. Restoring automatic
GitHub-hosted execution is a separate cost decision; do not silently enable it
when the trial guard expires.

References: [Depot CI overview and runner labels](https://depot.dev/docs/ci/overview)
and [migration quickstart](https://depot.dev/docs/ci/quickstart).

# Liner agent guidance

This development checkout uses the private `cmdux-sh/liner-private` repository and its `master` integration branch. Keep repository work on that existing remote. Public mirroring, package publication, releases, and marketing deployment need explicit authorization.

Use `docs/project/PRODUCT.md` for product behavior, `docs/project/DESIGN.md` for interface work, and `docs/project/CONTEXT.md` for vocabulary and boundaries. The active surfaces are the Python Core, Go TUI, TypeScript runner/npm shim, and `marketing/site/`. The previous Ink UI is decommissioned.

## Product and data boundaries

Preserve Core ownership of Project/Source maintenance, immutable identities, exact versioned Change Sets, approval-required risk checks, stale-plan guards, and durable Change Receipts. Extend the existing Core contract rather than adding direct canonical-file writers to the TUI. Preserve the distinction between a Tape, Mixtape, Liner Project, and Operating Layer.

Test with isolated synthetic Projects using `docs/development.md`; keep Arturo's normal Project library, source captures, private run logs, and provider credentials out of checkouts and PRs. An authenticated runtime profile needs an explicit selection. Source skills are reference artifacts, and generated Project pointers/adapters do not authorize automatic skill invocation in this development task.

Use [verification guidance](docs/agents/verification.md) for the affected Python, Go, runner, or marketing surface. Preserve applicable product acceptance and public-surface contracts. Read relevant existing tracked decisions when continuing that work; bounded direct work needs no new workflow or tracker setup.

## Working agreement

Follow Arturo's global `AGENTS.md` for collaboration, manual skill selection, and Git authorization. Agreed implementation includes task-owned checkpoints, pushes to the existing remote, and draft PR updates; merging and required acceptance remain explicit. Project Flow, Wayfinder, and Open Engine apply only when explicitly requested. Installed skills and issue labels do not invoke them.

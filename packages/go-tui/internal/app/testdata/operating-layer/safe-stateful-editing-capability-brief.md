# JTBD and knowledge map

## Capability Brief

### Reusable AI capability

Safely coordinate dependent mutations of a persistent, versioned document through MCP tools.

### Exact runtime output contract

For each requested change, produce:

1. **A preflight record:** observed target, revision, schema, authority, and invariants.
2. **A mutation plan:** intended diff, dependencies, preconditions, abort conditions, and recovery actions.
3. **An execution trace:** ordered attempts, sanitized inputs, returned identities, and validation outcomes.
4. **A verified closeout:** authoritative reread, invariant results, final revision, and terminal status.
5. **An append-only commit receipt:** successes, failures, compensations, warnings, unknowns, and follow-up.

### Internal job-to-be-done

When an authorized user requests a dependent multi-step edit, execute it without losing history or inventing certainty about partial application.

### Inferred research lanes

- Conditional writes, idempotency, reconciliation, recovery, and audit evidence.

### Required source roles

- Primary protocol authority and worked recovery cases.

### Source exclusions

- Exclude generic tool demos that do not address persistent state or partial failure.

### Runtime autonomy, abstention, and escalation

The future agent may act autonomously only for clearly scoped, authorized, reversible edits whose current revision, invariants, and recovery path are established.

After a timeout or ambiguous write response, reconcile authoritative state before retrying; never infer that an unobserved response means the change did not happen.

If the document revision changes after preflight, abort or rebase onto fresh state rather than overwriting concurrent work.

Human approval is required before destructive or irreversible commits, and the agent must escalate when recovery could discard valid work.

## Knowledge map

- State and identity
- Ordered execution
- Recovery and closeout

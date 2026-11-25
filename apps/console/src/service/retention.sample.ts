export const sampleDigest = "d".repeat(64);
const who = (id: string, login: string | null = id + "@example.invalid") => ({ id, login });

export const sample = {
  revision: "r9",
  now: "2026-11-17T09:00:00Z",
  state: "configured",
  policies: [
    { class: "claim", target: "retention-class:claim", policy: null, origin: "artifact_anchor", scope: "not_connected", archive_rule: "not_connected", pending: null, can_propose: true, propose_blocked: "" },
    {
      class: "source", target: "retention-class:source", policy: { revision: 1, period_days: 365, grace_days: 30, proposer: who("a1"), approver: who("a2"), approved_at: "2026-11-17T08:00:00Z" },
      origin: "artifact_anchor", scope: "not_connected", archive_rule: "not_connected", pending: 7, can_propose: false, propose_blocked: "already_proposed",
    },
  ],
  expiry: [
    {
      class: "source", population: "registered_sample_artifacts", counts: { eligible: 0, held: 1, guarding: 0, referenced: 1, not_due: 3, no_policy: 0, in_run: 1, deleted: 1 }, total: 7, more: false,
      artifacts: [
        { id: "sample-source-1", anchor: "2025-10-13T10:00:00Z", due_at: "2026-11-12T10:00:00Z", stores: ["sample-database"], access: "deleted", standing: "deleted", held: false, damaged: true, run: { target: "retention-run-1", label: "run/1" }, other_scope: false, shared: false, decision: "not_connected", holdable: true, hold_blocked: "", hold_target: "retention-artifact:sample-source-1" },
        { id: "sample-claim-x", anchor: "2025-10-13T10:00:00Z", due_at: null, stores: [], access: "ok", standing: "no_policy", held: false, damaged: false, run: null, other_scope: true, shared: true, holdable: false, hold_blocked: "target_too_long", hold_target: "" },
      ],
      usage: "not_connected", growth: "not_connected", bytes: "not_connected",
    },
  ],
  holds: {
    total: 1, more: false, tenant_target: "retention-tenant:sample-workspace",
    rows: [
      {
        id: "sample-hold-temporary", target: "retention-hold:sample-hold-temporary", owner: who("m"), reason: "Sample export review hold", placer: who("m"), placed_at: "2026-11-17T08:00:00Z", rationale: "Sample data",
        released: false, scope: { artifacts: [], workspace: true, classes: ["temporary"] }, counts: { not_preserved: 1, outcome_pending: 0, partially_deleted: 0 }, counts_scope: "workspace",
        outcomes: [{ artifact: "sample-temporary-1", outcome: "not_preserved", stores: { "sample-object-store": "deleted" } }], outcomes_more: false, release: 6, can_propose_release: false, release_blocked: "already_proposed",
      },
    ],
    references: { population: "registered_sample_artifacts", total: 1, more: false, rows: [{ id: "sample-source-3", class: "source" }] },
  },
  deletion: {
    total: 1, more: false, worker: "not_connected", retry: "not_connected", backup_expiry: "not_connected", propose_plan: "not_connected",
    rows: [
      {
        run: { target: "retention-run-2", label: "run/2" }, class: "temporary", revision: 1, state: "running", decision: "not_connected", next_chunk: 1, chunks: 3, chunk_size: 1, lease_seconds: 3600, reason: "r", damaged: 0,
        proposer: who("a1"), approver: who("a2"), stopper: null, counts: { pending: 2, deleting: 0, deleted: 1, failed: 0, held: 0, guarding: 0, referenced: 0, not_due: 0, stopped: 0 },
        stores: [{ store: "sample-object-store", target: 3, deleted: 1, failed: 0, excluded: 0, pending: 2 }], artifacts: { total: 3, more: false, rows: [{ id: "sample-temporary-1", outcome: "deleted" }] },
        can_stop: true, stop_blocked: "",
      },
    ],
  },
  restore: { state: "not_connected" },
  decisions: {
    watermark: 2, ledger: "in_memory", epoch: "not_connected", durable_commit: "not_connected", restore_check: "not_connected",
    tombstones: { window: 1000, total: 1, more: false, rows: [{ sequence: 2, artifact: "sample-temporary-1", class: "temporary", run: null, other_scope: true, at: "2026-11-17T08:00:00Z", decision: "not_connected" }] },
  },
  proposals: {
    total: 2, more: false,
    rows: [
      {
        number: 7, target: "retention-proposal-7", kind: "policy", state: "pending", created_at: "2026-11-17T08:00:00Z", proposer: who("a2"), approver: null, rationale: "Sample data", closed_by: null, closing_rationale: "",
        summary: { policy: { class: "source", period_days: 180, grace_days: 30, impact_at_proposal: { as_of: "2026-11-17T08:00:00.123Z", count: 2, digest: sampleDigest } }, plan: null, release: null },
        impact_required: true, impact_now: { as_of: "2026-11-17T09:00:00.456789Z", count: 2, digest: sampleDigest, workspace_count: 2 }, can_approve: true, approve_blocked: "", can_withdraw: false, withdraw_blocked: "not_proposer",
      },
      {
        number: 6, target: "retention-proposal-6", kind: "release", state: "pending", created_at: "2026-11-17T08:00:00Z", proposer: who("a1"), approver: null, rationale: "Sample data", closed_by: null, closing_rationale: "",
        summary: { policy: null, plan: null, release: { hold: "sample-hold-temporary", reason: "Sample review finished" } }, impact_required: false, impact_now: null,
        can_approve: false, approve_blocked: "own_proposal", can_withdraw: true, withdraw_blocked: "",
      },
    ],
  },
};

import { policyPageOf, trustChangeOf, type PolicyPage, type TrustChange } from "../../../../service/admin";

export const changeFixture = (over: Record<string, unknown> = {}): TrustChange =>
  trustChangeOf({
    id: "trust-0a1b2c3d-1", number: 1, state: "draft", change_revision: 1, change_digest: "c".repeat(64), old_epoch: { value: "1", generation: 0, number: 1 }, old_policy: 1,
    old_digest: "o".repeat(64), new_digest: "n".repeat(64), candidate: { methods: ["password"], mfa: "none", max_age_seconds: 3600, step_up: [] }, proposer: "sample-admin",
    impact: { members: 3, sessions: 2, affected_ids: ["sample-admin", "sample-admin-2", "sample-admin-3"], weakens_mfa: false, signs_out_everyone: true },
    can: {}, why_not: { approve: "", activate: "" }, history: [{ state: "draft", by: "sample-admin", at: "2026-11-18T09:00:00Z", rationale: "Shorter sign-in" }], ...over,
  });

export const pageFixture = (over: Record<string, unknown> = {}): PolicyPage =>
  policyPageOf({
    revision: "r5", now: "2026-11-18T09:00:00Z", signed_in_at: "2026-11-18T08:50:00Z", authenticated_at: "2026-11-18T08:50:00Z", act_until: "2026-11-18T09:05:00Z", step_up_seconds: 900,
    you: "sample-admin-3", started_at: "2026-11-18T08:00:00Z",
    policy: { methods: ["password"], mfa: "none", max_age_seconds: 43200, step_up: [], number: 1, epoch: { value: "1", generation: 0, number: 1 }, proposer: "", validator: "", approver: "", activator: "", at: "2026-11-18T08:00:00Z" },
    trust: { epoch: "1", epoch_generation: 0, epoch_number: 1, policy_number: 1, digest: "d".repeat(64), origin: "seed", hold: null },
    ledger: { state: "ok", records: 1, generation: 0 }, rights: { propose: true, validate: true, approve: true, activate: true }, holders: { qualifying_people: 3, members: 3, needed: 3, enrolment: "not_decided", ready: true, missing: [] },
    ...over,
  });

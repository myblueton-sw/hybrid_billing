import test from "node:test";
import assert from "node:assert/strict";
import { setupContextOf, SetupError, setupDate } from "./context";

function fixture() {
  return { revision: "r7", at: "2026-11-23T00:00:00Z", fields: {
    schema: "hb_issuance_setup_context_v1", contract: "contract-1", contract_name: "Example contract",
    on: "2024-02-29", lifecycle_sequence: "9007199254740993", contract_revision: "2",
    billing_profile: "profile-1", billing_profile_name: "Example billing profile", billing_revision: "3",
    buyer: "buyer-1", payer: "payer-1", issuer: "issuer-1", currency: "USD",
    scope_digest: "a".repeat(64), profile_basis: "current", current_eligibility: "unchecked",
  } };
}

test("retains exact source versions and historical selection without implying readiness", () => {
  const value = setupContextOf(fixture(), "contract-1", "2024-02-29");
  assert.equal(value.lifecycleSequence, "9007199254740993");
  assert.equal(value.on, "2024-02-29");
  assert.equal(value.profileBasis, "current");
  assert.equal(value.currentEligibility, "unchecked");
  assert.ok(Object.isFrozen(value));
});

test("refuses another contract or date instead of silently adopting current scope", () => {
  assert.throws(() => setupContextOf(fixture(), "contract-2", "2024-02-29"), SetupError);
  assert.throws(() => setupContextOf(fixture(), "contract-1", "2026-11-23"), SetupError);
  assert.equal(setupDate("2025-02-29"), false);
  assert.equal(setupDate("0000-01-01"), false);
});

test("rejects malformed or falsely eligible context atomically", () => {
  for (const [key, value] of [
    ["current_eligibility", "eligible"], ["profile_basis", "historical"],
    ["contract_revision", "0"], ["billing_revision", "18446744073709551616"],
    ["lifecycle_sequence", "01"], ["scope_digest", "bad"], ["currency", "usd"],
    ["issuer", ""], ["contract_name", "x".repeat(4097)], ["extra", "unexpected"],
  ]) {
    const envelope = fixture();
    Object.assign(envelope.fields, { [key]: value });
    assert.throws(() => setupContextOf(envelope, "contract-1", "2024-02-29"), SetupError, key);
  }
  const envelope = fixture();
  Reflect.deleteProperty(envelope.fields, "payer");
  assert.throws(() => setupContextOf(envelope, "contract-1", "2024-02-29"), SetupError);
});

test("accepts uint64 maximum without rounding and rejects invalid envelope evidence", () => {
  const envelope = fixture();
  envelope.fields.contract_revision = "18446744073709551615";
  assert.equal(setupContextOf(envelope, "contract-1", "2024-02-29").contractRevision, "18446744073709551615");
  envelope.at = "invalid";
  assert.throws(() => setupContextOf(envelope, "contract-1", "2024-02-29"), SetupError);
});

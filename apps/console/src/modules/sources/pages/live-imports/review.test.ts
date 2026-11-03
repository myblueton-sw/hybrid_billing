import test from "node:test";
import assert from "node:assert/strict";
import { receptionScope, validIntent } from "./review";
const profile = {
  profile: "synthetic",
  workspace: "a",
  connection: "c",
  account: "payer",
  period_start: "2026-10-01",
  period_end: "2026-11-01",
  versions: ["1.2", "1.3"],
  max_bytes: 8,
};
test("reception review detects every changed authoritative scope field", () => {
  for (const key of [
    "profile",
    "workspace",
    "connection",
    "account",
    "period_start",
    "period_end",
  ] as const)
    assert.notEqual(
      receptionScope(profile),
      receptionScope({ ...profile, [key]: "changed" }),
    );
  assert.notEqual(
    receptionScope(profile),
    receptionScope({ ...profile, max_bytes: 7 }),
  );
  assert.notEqual(
    receptionScope(profile),
    receptionScope({ ...profile, versions: ["1.3"] }),
  );
  assert.equal(
    receptionScope(profile),
    receptionScope({ ...profile, versions: ["1.3", "1.2"] }),
  );
});
test("reception review validates exact revision range and byte/version envelope", () => {
  const file = new File(["abc"], "synthetic.csv");
  assert.ok(validIntent(file, "1.3", "9223372036854775807", profile));
  for (const revision of ["0", "-1", "1.1", "1e2", "9223372036854775808"])
    assert.equal(validIntent(file, "1.3", revision, profile), false);
  assert.equal(
    validIntent(new File([], "empty.csv"), "1.3", "1", profile),
    false,
  );
  assert.equal(
    validIntent(new File(["123456789"], "oversize.csv"), "1.3", "1", profile),
    false,
  );
  assert.equal(validIntent(file, "2.0", "1", profile), false);
});

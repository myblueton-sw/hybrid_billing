import { test } from "node:test";
import assert from "node:assert/strict";
import type { Membership } from "../../../../service/admin";
import { bounds, dateParts, direction, endFields, endProblem, initialEnd, localInput, offsetLabel, periodOf, remaining, setting, utcOf } from "./period-model";

const m = (extra: Partial<Membership> = {}): Membership => ({ role: "member", state: "active", version: 1, role_from: "", role_until: "", role_period: "open",
  suspended_for: "", ledger: "", ...extra });

test("offsets are written as UTC plus or minus hours and minutes", () => {
  assert.equal(offsetLabel(-540), "UTC+09:00");
  assert.equal(offsetLabel(-180), "UTC+03:00");
  assert.equal(offsetLabel(300), "UTC-05:00");
  assert.equal(offsetLabel(0), "UTC+00:00");
  assert.equal(offsetLabel(-330), "UTC+05:30");
});

test("a local input converts to a whole-minute UTC time and back", () => {
  const utc = utcOf("2026-11-30T18:00");
  assert.match(utc, /^2026-11-3\dT\d{2}:00:00Z$|^2026-12-01T\d{2}:00:00Z$/);
  assert.equal(localInput(utc), "2026-11-30T18:00");
  assert.equal(utcOf("2026-02-30T10:00"), "");
  assert.equal(utcOf("soon"), "");
  assert.equal(localInput("nope"), "");
  const summer = utcOf("2026-07-01T12:00");
  const winter = utcOf("2026-01-01T12:00");
  assert.equal(localInput(summer), "2026-07-01T12:00");
  assert.equal(localInput(winter), "2026-01-01T12:00");
});

test("date parts never split inside a date and carry the zone", () => {
  const p = dateParts("en", "2026-11-30T09:00:00Z");
  assert.ok(p.date.includes("2026"));
  assert.match(p.zone, /^UTC[+-]\d{2}:\d{2}$/);
  assert.deepEqual(dateParts("ko", "x"), { date: "", time: "", zone: "" });
  assert.ok(dateParts("ar", "2026-11-30T09:00:00Z").time.length > 0);
});

test("remaining days come from the server time, less than a day is today", () => {
  assert.deepEqual(remaining("2026-11-21T09:00:00Z", "2026-11-18T09:00:00Z"), { days: 3, today: false });
  assert.deepEqual(remaining("2026-11-18T20:00:00Z", "2026-11-18T09:00:00Z"), { days: 0, today: true });
  assert.deepEqual(remaining("x", "2026-11-18T09:00:00Z"), { days: 0, today: false });
});

test("direction follows the stored end", () => {
  assert.equal(direction(m(), ""), "same");
  assert.equal(direction(m(), "2026-12-01T00:00:00Z"), "shorter");
  assert.equal(direction(m({ role_until: "2026-12-01T00:00:00Z" }), ""), "longer");
  assert.equal(direction(m({ role_until: "2026-12-01T00:00:00Z" }), "2026-12-02T00:00:00Z"), "longer");
  assert.equal(direction(m({ role_until: "2026-12-01T00:00:00Z" }), "2026-11-30T00:00:00Z"), "shorter");
  assert.equal(direction(m({ role_until: "2026-12-01T00:00:00Z" }), "2026-12-01T00:00:00Z"), "same");
  assert.equal(direction(m({ role_period: "ended", role_until: "2026-11-01T00:00:00Z" }), "2026-11-02T00:00:00Z"), "longer");
  assert.equal(direction(m({ suspended_for: "role_period_ended" }), "2026-11-30T00:00:00Z"), "longer");
});

test("bounds and problems use the server time, the longest period and the setter's end", () => {
  const b = bounds("2026-11-18T09:00:30Z", 2, "");
  assert.deepEqual(b, { min: "2026-11-18T09:02:00.000Z", max: "2026-11-20T09:00:00.000Z", own: "" });
  assert.equal(endProblem("", b), "missing");
  assert.equal(endProblem("2026-11-18T09:01:00Z", b), "early");
  assert.equal(endProblem("2026-11-21T00:00:00Z", b), "late");
  assert.equal(endProblem("2026-11-19T00:00:00Z", b), "");
  const own = bounds("2026-11-18T09:00:00Z", 2, "2026-11-19T00:00:00Z");
  assert.equal(own.max, "2026-11-19T00:00:00Z");
  assert.equal(endProblem("2026-11-19T01:00:00Z", own), "own");
  assert.deepEqual(bounds("x", 2, ""), { min: "", max: "", own: "" });
  assert.equal(endProblem("2026-11-19T01:00:00Z", { min: "", max: "", own: "" }), "");
});

test("the period of a row and settings lookup", () => {
  const base = { user_id: "u", login: "", email: "", state: "active" as const, epoch: 1, password_set: true, invitation_expires: "", link_purpose: "" as const,
    link_expires: "", installation_administrator: false, revocation: null };
  assert.deepEqual(periodOf({ ...base, membership: null }), { kind: "none", until: "" });
  assert.deepEqual(periodOf({ ...base, membership: m({ role_period: "ends_soon", role_until: "x" }) }), { kind: "ends_soon", until: "x" });
  assert.equal(setting([{ name: "a", value: 3 }], "a", 1), 3);
  assert.equal(setting([], "a", 1), 1);
});

test("the period form starts from the stored end", () => {
  const stored = "2026-12-01T10:00:00Z";
  assert.deepEqual(initialEnd(stored, ""), { kind: "at", local: localInput(stored) });
  assert.deepEqual(initialEnd("", ""), { kind: "none", local: "" });
  assert.deepEqual(initialEnd("", "2026-11-30T10:00:00Z"), { kind: "at", local: localInput("2026-11-30T10:00:00Z") });
  assert.deepEqual(endFields({ kind: "none", local: "x" }), { values: {}, utc: "" });
  assert.deepEqual(endFields({ kind: "at", local: "" }), { values: {}, utc: "" });
});

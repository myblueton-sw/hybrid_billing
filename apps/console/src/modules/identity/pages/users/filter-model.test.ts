import { test } from "node:test";
import assert from "node:assert/strict";
import type { UserRow } from "../../../../service/admin";
import { usersQueryString } from "../../../../service/admin";
import { appliedKeys, cursorUser, linkOf, selectableIds, loginParts, lostPerson, membershipEnded, merge, noFilters, parseLink, queryOf, sameFilters, sameInstant, validSearch, without } from "./filter-model";

const link = (q: string) => parseLink(new URLSearchParams(q));

test("link parameters are parsed and invalid ones are dropped with a notice", () => {
  assert.deepEqual(link("view=users&lang=ko"), { filters: noFilters, bad: false });
  assert.deepEqual(link("q=+%EA%B9%80+&st=suspended&ms=ended&role=member&per=ends_soon&si=set_link&adm=1"),
    { filters: { q: "김", st: "suspended", ms: "ended", role: "member", per: "ends_soon", si: "set_link", adm: true }, bad: false });
  for (const bad of ["st=gone", "ms=removed", "role=service", "per=soon", "si=sso", "adm=true", "q=", "q=%20%20", "q=" + "a".repeat(65), "st=active&st=suspended", "q=a&q=b", "adm=1&adm=1"]) {
    assert.equal(link(bad).bad, true, bad);
  }
  assert.deepEqual(link("st=gone&role=member").filters, { ...noFilters, role: "member" });
  assert.equal(link("q=" + "가".repeat(21) + "a").filters.q.length, 22);
  assert.equal(validSearch("가".repeat(21) + "ab"), false);
});

test("filters round trip through the link and the query", () => {
  const f = { q: "ann", st: "active", ms: "none", role: "", per: "", si: "not_set", adm: true };
  assert.deepEqual(linkOf(f), { q: "ann", st: "active", ms: "none", role: null, per: null, si: "not_set", adm: "1" });
  assert.deepEqual(linkOf(noFilters), { q: null, st: null, ms: null, role: null, per: null, si: null, adm: null });
  assert.deepEqual(link(new URLSearchParams(Object.entries(linkOf(f)).filter(([, v]) => v !== null) as [string, string][]).toString()).filters, f);
  assert.equal(usersQueryString(queryOf("users", f)), "q=ann&status=active&membership=none&signin=not_set&holder=true&limit=50");
  assert.equal(usersQueryString(queryOf("invitations", noFilters, "a:u1")), "invitation=pending&after=a%3Au1&limit=50");
  assert.equal(usersQueryString(queryOf("membership", { ...noFilters, role: "administrator", per: "open" })), "role=administrator&period=open&limit=50");
});

test("chips follow the applied filters and removal drops one", () => {
  const f = { ...noFilters, q: "x", per: "ended", adm: true };
  assert.deepEqual(appliedKeys(f), ["q", "per", "adm"]);
  assert.deepEqual(appliedKeys(without(f, "adm")), ["q", "per"]);
  assert.deepEqual(appliedKeys(without(f, "q")), ["per", "adm"]);
  assert.deepEqual(appliedKeys(noFilters), []);
  assert.ok(sameFilters(f, { ...f }));
  assert.ok(!sameFilters(f, without(f, "per")));
});

const row = (user_id: string, login = user_id): UserRow => ({ user_id, login, email: "", state: "active", epoch: 1, membership: null, password_set: false, invitation_expires: "",
  link_purpose: "", link_expires: "", installation_administrator: false, revocation: null });

test("pages merge by user ID so a departed person is shown once", () => {
  const first = [row("u1"), row("u2"), row("u3")];
  const out = merge(first, [row("u4"), row("u2", ""), row("u5")]);
  assert.deepEqual(out.map((r) => r.user_id), ["u1", "u2", "u3", "u4", "u5"]);
  assert.equal(out[1].login, "");
  assert.deepEqual(merge([], [row("a"), row("a")]).map((r) => r.user_id), ["a"]);
});

test("cursors and the ended membership word", () => {
  assert.deepEqual(cursorUser("a:user-1"), { departed: false, user: "user-1" });
  assert.deepEqual(cursorUser("d:user-2"), { departed: true, user: "user-2" });
  const m = { role: "member" as const, state: "active" as const, version: 1, role_from: "", role_until: "", role_period: "ended" as const, suspended_for: "" as const, ledger: "" as const };
  assert.equal(membershipEnded({ ...row("u"), membership: m }), true);
  assert.equal(membershipEnded({ ...row("u"), membership: { ...m, state: "suspended" } }), false);
  assert.equal(membershipEnded({ ...row("u"), membership: { ...m, role_period: "open" } }), false);
  assert.equal(membershipEnded(row("u")), false);
});

test("select loaded is empty while a read is in flight, and the open person is lost only from a row panel", () => {
  const rows = [row("u1"), row("me"), row("u2")];
  assert.deepEqual(selectableIds("ready", rows, "me"), ["u1", "u2"]);
  assert.equal(lostPerson("detail", "u9", rows), true);
  assert.equal(lostPerson("confirm", "u9", rows), true);
  assert.equal(lostPerson("period", "u9", rows), true);
  assert.equal(lostPerson("detail", "u1", rows), false);
  assert.equal(lostPerson("link", "u9", rows), false);
  assert.equal(lostPerson("create", "u9", rows), false);
  assert.equal(lostPerson("detail", "", rows), false);
});

test("equal expiry instants and login break points", () => {
  assert.equal(sameInstant("2026-11-25T10:00:00Z", "2026-11-25T10:00:00.000Z"), true);
  assert.equal(sameInstant("2026-11-25T10:00:00Z", "2026-11-25T10:00:01Z"), false);
  assert.equal(sameInstant("", ""), false);
  assert.deepEqual(loginParts("aaa1794996594070@example.invalid"), ["aaa1794996594070", "@example.", "invalid"]);
  assert.deepEqual(loginParts("admin"), ["admin"]);
});

test("a same-query reload keeps the visible rows loaded; a fresh read loads none", () => {
  const rows = [row("u1"), row("me")];
  assert.deepEqual(selectableIds("ready", rows, "me"), ["u1"]);
  assert.deepEqual(selectableIds("reloading", rows, "me"), ["u1"]);
  assert.deepEqual(selectableIds("fresh", rows, "me"), []);
});

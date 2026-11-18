import type { Locale } from "../../../../shared/types";
import { AdminError, type UserRow } from "../../../../service/admin";
import { c, retryPhrase } from "../../../../service/copy";
import { w, type Action } from "./live-copy";
import { fc } from "./filter-copy";

export const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function when(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short" }).format(d);
}

export function roleName(l: Locale, role: string): string {
  return role === "administrator" ? w.administrator(l) : role === "member" ? w.member(l) : w.noMembership(l);
}

export function accountName(l: Locale, state: string): string {
  return state === "suspended" ? w.suspended(l) : state === "departed" ? w.departed(l) : w.active(l);
}

export function setup(l: Locale, u: UserRow): string {
  if (u.link_purpose === "set") return fc.setLinkAt(l).replace("{at}", when(l, u.link_expires));
  if (u.link_purpose === "reset") return fc.resetLinkAt(l).replace("{at}", when(l, u.link_expires));
  if (u.invitation_expires) return w.invitationPending(l, when(l, u.invitation_expires));
  return u.password_set ? w.passwordSet(l) : w.noPassword(l);
}

export type Offer = { account: Action[]; membership: Action[]; sign: Action[]; notes: string[]; blocked: string };

export function offers(l: Locale, u: UserRow, self: string): Offer {
  const out: Offer = { account: [], membership: [], sign: [], notes: [], blocked: "" };
  if (u.user_id === self) return { ...out, blocked: w.selfReason(l) };
  if (u.state === "departed") return { ...out, blocked: w.departedReason(l) };
  out.account = u.state === "active" ? ["suspend_user", "depart_user"] : ["reinstate_user", "depart_user"];
  const m = u.membership;
  const promotable = !(m?.role === "member" && u.link_purpose);
  const ended = m?.role_period === "ended" || m?.suspended_for === "role_period_ended";
  if (!m) out.membership = u.invitation_expires || u.state !== "active" ? [] : ["invite"];
  else if (ended) out.membership = ["set_role_period", "remove_membership"];
  else if (m.state === "active") out.membership = promotable ? ["change_role", "set_role_period", "suspend_membership", "remove_membership"] : ["set_role_period", "suspend_membership", "remove_membership"];
  else out.membership = ["reinstate_membership", "set_role_period", "remove_membership"];
  if (!u.password_set) out.sign = u.login ? ["issue_set_link"] : [];
  else if (m?.role === "administrator") out.notes.push(w.adminResetReason(l));
  else if (u.state !== "active") out.notes.push(w.suspendedResetReason(l));
  else out.sign = ["issue_reset_link"];
  if (u.invitation_expires) out.sign.push("revoke_invitation");
  if (m?.state === "active" && m.role === "member" && u.link_purpose) out.notes.push(w.promotionReason(l));
  return out;
}

export function failureText(l: Locale, e: unknown): { title: string; body: string } {
  if (!(e instanceof AdminError)) return { title: w.saveFailed(l), body: w.unavailable(l) };
  switch (e.failure) {
    case "conflict":
      return { title: w.saveFailed(l), body: w.conflict(l) };
    case "stale":
      return { title: w.saveFailed(l), body: w.stale(l) };
    case "refused":
      return { title: w.refusedTitle(l), body: w.refused(l) };
    case "stage":
      return { title: w.saveFailed(l), body: w.stage(l) };
    case "denied":
      return { title: w.deniedTitle(l), body: w.denied(l) };
    case "unauthenticated":
      return { title: w.saveFailed(l), body: w.ended(l) };
    case "limited":
      return { title: w.saveFailed(l), body: c.limited(l, retryPhrase(l, e.retryAfter || 60)) };
    case "partial":
      return { title: w.saveFailed(l), body: w.partial(l, e.operation) };
    case "busy":
      return { title: w.saveFailed(l), body: w.busy(l) };
    case "invalid":
      return { title: c.check(l), body: "" };
    case "reauth":
    case "reauth_cancelled":
      return { title: w.saveFailed(l), body: c.reauthText(l, e.failure) };
    default:
      return { title: w.saveFailed(l), body: w.unavailable(l) };
  }
}

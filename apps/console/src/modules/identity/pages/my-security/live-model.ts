import type { SelfEnd, SelfEndRow, SelfItem, SelfPage, SelfSession } from "../../../../service/admin";
import { AdminError } from "../../../../service/admin";

export type SessionDisplay = { sessions: SelfSession[]; session_ends: SelfEnd[] };

export type Check = "" | "missing" | "mismatch" | "same";

export function checkChange(current: string, next: string, confirm: string): Check {
  if (!current || !next || !confirm) return "missing";
  if (next !== confirm) return "mismatch";
  return next === current ? "same" : "";
}

export function canChange(page: SelfPage): boolean {
  return page.password_set;
}

export function others(page: Pick<SessionDisplay, "sessions">): SelfSession[] {
  return page.sessions.filter((s) => !s.current);
}

export type Outcome = "wrong" | "locked" | "limited" | "form" | "conflict" | "lockedOut" | "signedOut" | "busy" | "unknown";

export function changeOutcome(e: unknown, hadWrong: boolean): Outcome {
  if (!(e instanceof AdminError)) return "unknown";
  switch (e.failure) {
    case "busy":
      return "busy";
    case "wrong":
    case "locked":
    case "limited":
    case "form":
    case "conflict":
      return e.failure;
    case "unauthenticated":
      return hadWrong ? "lockedOut" : "signedOut";
  }
  return "unknown";
}

export function changeUnsettled(outcome: Outcome): boolean {
  return outcome === "unknown";
}

export type RowOutcome = "complete" | "running" | "failed" | "gone";

export function rowOutcome(row: SelfEndRow): RowOutcome {
  if (row.result !== "ended") return "gone";
  const c3 = row.items.find((i) => i.item === "C3")?.state;
  if (c3 === "done" || c3 === "already_ended") return "complete";
  if (c3 === "failed") return "failed";
  return "running";
}

export type ItemGroup = "done" | "pending" | "failed" | "not_applicable" | "outside_scope";

export const groupOrder: ItemGroup[] = ["done", "pending", "failed", "not_applicable", "outside_scope"];

export function groupOf(item: SelfItem): ItemGroup {
  return item.state === "already_ended" ? "done" : item.state;
}

export function goneCount(op: SelfEnd): number {
  return op.sessions.filter((s) => s.result === "already_gone").length;
}

export function heldShown(op: SelfEnd): boolean {
  return op.kind === "others" && op.not_included.length > 0;
}

export function presentHandles(page: Pick<SessionDisplay, "sessions">): Set<string> {
  return new Set(page.sessions.map((s) => s.handle));
}

export function heldTags(page: SessionDisplay): Set<string> {
  const newest = page.session_ends.find((o) => o.kind === "others");
  if (!newest) return new Set();
  const present = presentHandles(page);
  return new Set(newest.not_included.map((s) => s.handle).filter((h) => present.has(h)));
}

export function openByDefault(op: SelfEnd, newest: boolean, present: Set<string>): boolean {
  return newest || op.state !== "complete" || (op.kind === "others" && op.not_included.some((s) => present.has(s.handle)));
}

export function running(page: Pick<SessionDisplay, "session_ends"> | null): boolean {
  return !!page && page.session_ends.some((o) => o.state === "cleanup_running");
}

export type SubmitOutcome = "changed" | "gone" | "signedOut" | "limited" | "reauth" | "refused" | "unknown";

export function submitOutcome(err: unknown, kind: "others" | "one"): SubmitOutcome {
  if (err instanceof AdminError) {
    if (err.failure === "invalid") return kind === "others" ? "changed" : "gone";
    if (err.failure === "unauthenticated") return "signedOut";
    if (err.failure === "limited") return "limited";
    if (err.failure === "reauth" || err.failure === "reauth_cancelled") return "reauth";
    if (err.failure === "refused") return "refused";
  }
  return "unknown";
}

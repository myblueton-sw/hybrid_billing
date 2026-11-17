import type { SelfPage, SelfSession } from "../../../../service/admin";
import { AdminError } from "../../../../service/admin";

export type Check = "" | "missing" | "mismatch" | "same";

export function checkChange(current: string, next: string, confirm: string): Check {
  if (!current || !next || !confirm) return "missing";
  if (next !== confirm) return "mismatch";
  return next === current ? "same" : "";
}

export function canChange(page: SelfPage): boolean {
  return page.password_set;
}

export function others(page: SelfPage): SelfSession[] {
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

export type EndOutcome = "gone" | "signedOut" | "limited" | "reauth" | "busy";

export function endOutcome(e: unknown): EndOutcome {
  if (e instanceof AdminError) {
    if (e.failure === "invalid") return "gone";
    if (e.failure === "unauthenticated") return "signedOut";
    if (e.failure === "limited") return "limited";
    if (e.failure === "reauth" || e.failure === "reauth_cancelled") return "reauth";
  }
  return "busy";
}

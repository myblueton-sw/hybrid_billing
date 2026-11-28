import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { w } from "../users/live-copy";
import { commandFailure, failureText, outcomeUnknown } from "../users/live-model";
import { a } from "./live-copy";
import { q } from "./request-copy";

export function describeRequest(l: Locale, e: unknown): { title: string; body: string } {
  if (e instanceof AdminError) {
    if (e.failure === "refused") return { title: a.refusedTitle(l), body: q.refusedRequest(l) };
    if (e.failure === "stage") return { title: a.refusedTitle(l), body: q.stageRequest(l) };
    if (e.failure === "busy") return { title: w.saveFailed(l), body: q.busyIncrease(l) };
    if (e.failure === "partial") return { title: w.unknownTitle(l), body: q.outcomeUnknown(l) };
    if (e.failure === "invalid") return { title: a.refusedTitle(l), body: a.invalidBody(l) };
    if (e.failure === "denied") return { title: a.refusedTitle(l), body: a.goneBody(l) };
    if (e.failure === "conflict" || e.failure === "stale") return { title: w.saveFailed(l), body: a.conflictBody(l) };
  }
  return commandFailure(l, e);
}

export function describeFor(kind: "grant" | "revoke" | "load") {
  return (l: Locale, e: unknown) => {
    if (kind === "grant") return describeRequest(l, e);
    if (e instanceof AdminError && kind !== "load") {
      if (e.failure === "refused") return { title: a.refusedTitle(l), body: a.refusedRevoke(l) };
      if (e.failure === "stage") return { title: a.refusedTitle(l), body: a.stageBody(l) };
      if (e.failure === "invalid") return { title: a.refusedTitle(l), body: a.invalidBody(l) };
      if (e.failure === "denied" && kind === "revoke") return { title: a.refusedTitle(l), body: a.goneBody(l) };
      if (e.failure === "conflict" || e.failure === "stale") return { title: w.saveFailed(l), body: a.conflictBody(l) };
    }
    if (kind !== "load" && outcomeUnknown(e)) return commandFailure(l, e);
    if (e instanceof AdminError && (e.failure === "unavailable" || e.failure === "busy")) return { title: w.saveFailed(l), body: a.unavailableBody(l) };
    return failureText(l, e);
  };
}

import type { Locale } from "../../../../shared/types";
import type { AccessHistoryPage, HistoryNote, HistoryNoteKind } from "../../../../service/access-history";
import { hc, periodMs, type Period } from "./history-copy";

export type NoteLine = { key: string; text: string; strong: boolean };

const order: (HistoryNoteKind | "recording")[] = [
  "recording",
  "purge_deferred",
  "deletion_overdue",
  "held",
  "early_deletion",
  "dropped",
  "unclean_stop",
  "damaged",
  "recovered_tail",
  "future_file",
];

const strongKinds = new Set<string>(["recording", "purge_deferred", "deletion_overdue"]);

export function fromFor(period: Period, now: string): string | undefined {
  if (period === "all") return undefined;
  const base = new Date(now).getTime();
  if (!now || Number.isNaN(base)) return undefined;
  return new Date(base - periodMs[period]).toISOString().replace(/\.\d{3}Z$/, "Z");
}

export function coverageGap(from: string | undefined, available: string | null): boolean {
  if (!from || !available) return false;
  return new Date(from).getTime() < new Date(available).getTime();
}

export function showSizeLimit(page: AccessHistoryPage): boolean {
  return page.effective_days < page.retention_days;
}

function noteText(l: Locale, n: HistoryNote, days: number, time: (iso: string) => string): string {
  const from = n.from ? time(n.from) : "";
  const to = n.to ? time(n.to) : "";
  const missing = () => (from && to ? hc.possiblyMissing(l, from, to) : hc.possiblyMissingPlain(l));
  switch (n.kind) {
    case "purge_deferred":
      return hc.deferred(l);
    case "deletion_overdue":
      return hc.overdue(l, days);
    case "held":
      return n.count === null ? missing() : hc.held(l, n.count, days);
    case "early_deletion":
      return hc.earlyDeletion(l, to);
    case "dropped":
      return n.count === null ? missing() : hc.dropped(l, n.count, from, to);
    case "unclean_stop":
      return hc.unclean(l, from);
    case "damaged":
      return n.count === null ? missing() : hc.damaged(l, n.count);
    case "recovered_tail":
      return hc.recoveredTail(l, from);
    case "future_file":
      return hc.futureFile(l);
  }
}

export function noteLines(l: Locale, page: AccessHistoryPage, time: (iso: string) => string): NoteLine[] {
  const out: (NoteLine & { rank: number; at: string })[] = [];
  if (page.recording.state === "stopped") {
    out.push({ key: "recording", text: hc.recordingStopped(l, page.recording.since ? time(page.recording.since) : ""), strong: true, rank: 0, at: "" });
  }
  page.notes.forEach((n, i) => {
    if (n.count === 0) return;
    out.push({ key: `${n.kind}-${i}`, text: noteText(l, n, page.retention_days, time), strong: strongKinds.has(n.kind), rank: order.indexOf(n.kind), at: n.from ?? "" });
  });
  out.sort((a, b) => a.rank - b.rank || (a.at < b.at ? 1 : a.at > b.at ? -1 : 0));
  return out.map(({ key, text, strong }) => ({ key, text, strong }));
}

export type HistoryState = "loading" | "failed" | "none-kept" | "empty" | "rows";

export function historyState(page: AccessHistoryPage | null, failed: boolean): HistoryState {
  if (failed) return "failed";
  if (!page) return "loading";
  if (page.events.length > 0) return "rows";
  return page.available_from === null ? "none-kept" : "empty";
}

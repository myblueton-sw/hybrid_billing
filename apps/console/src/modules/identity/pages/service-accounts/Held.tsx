import { useEffect, useRef, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { v } from "./live-copy";

type Held = "idle" | "sending" | "unknown" | "lapsed";

export function HeldBlock({ l, id, phase, lead, lapsed, focusKey, error, request }: { l: Locale; id: string; phase: Held; lead: string; lapsed: ReactNode; focusKey: number; error?: unknown; request?: string }) {
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focusKey) box.current?.focus(); }, [focusKey]);
  if (phase === "idle") return null;
  return (
    <div className="live-held" tabIndex={-1} ref={box}>
      {phase === "unknown" && <h3>{v.heldTitle(l)}</h3>}
      {phase === "lapsed" && <h3>{v.lapsedTitle(l)}</h3>}
      {phase === "sending" && <p id={id + "-held-lead"}>{v.sendingLead(l)}</p>}
      {phase === "unknown" && <p id={id + "-held-lead"}>{lead}</p>}
      {phase === "unknown" && <p id={id + "-locked-note"}>{v.lockedNote(l)}</p>}
      {phase === "lapsed" && lapsed}
      {phase === "unknown" && (request || (error instanceof AdminError && error.code)) && (
        <p className="live-meta">
          {request && <>{v.heldRequest(l)} <bdi dir="ltr" className="live-mono">{request}</bdi></>}
          {error instanceof AdminError && error.code && <>{request ? " · " : ""}{v.heldCode(l)} <bdi dir="ltr" className="live-mono">{error.code}</bdi></>}
        </p>
      )}
    </div>
  );
}

import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type Attempt, type CommandId, type Outcome, type PolicyPage } from "../../../../service/admin";
import { c as sc } from "../../../../service/copy";
import { Summary } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem } from "../users/Live";
import { w } from "../users/live-copy";
import { ConfirmBody, primaryLabel } from "./TrustConfirmBody";
import { ty } from "./trust-copy";
import { asTrustFailure, candidateOf, configDigest, fatal, trustFailure, valuesFor, type Confirming } from "./trust-model";

export type { Confirming };

const commandOf: Record<Confirming["kind"], CommandId> = {
  propose: "trust.propose", validate: "trust.validate", approve: "trust.approve", activate: "trust.activate", reject: "trust.reject", cancel: "trust.cancel",
};

export function ConfirmDialog({ l, page, api, k, opener, workspace, onDone, onClose, onSignedOut }: {
  l: Locale; page: PolicyPage; api: AdminClient; k: Confirming; opener: HTMLElement | null; workspace: string;
  onDone: (k: Confirming, out: Outcome) => void; onClose: (reload: boolean) => void; onSignedOut: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const title = useRef<HTMLHeadingElement>(null);
  const tried = useRef<Attempt>(attempt());
  const busyRef = useRef(false);
  const [reason, setReason] = useState("");
  const [problem, setProblem] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [dead, setDead] = useState(false);
  const [busy, setBusy] = useState(false);
  const [focusKey, setFocusKey] = useState(0);
  const [digest, setDigest] = useState("");
  useEffect(() => {
    const d = dialog.current;
    if (d && !d.open && typeof d.showModal === "function") d.showModal();
    title.current?.focus();
    return () => { if (d?.open) d.close(); };
  }, []);
  useEffect(() => {
    if (k.kind !== "propose" || !k.proposed) return;
    let live = true;
    configDigest(candidateOf(k.proposed.maxAge, k.proposed.step)).then((x) => { if (live) setDigest(x); }, () => undefined);
    return () => { live = false; };
  }, [k]);
  const finish = (reload: boolean) => {
    dialog.current?.close();
    onClose(reload);
    const kind = k.kind;
    window.setTimeout(() => {
      const target = opener && opener.isConnected ? opener
        : (kind === "propose" ? document.querySelector<HTMLElement>('.m02-form button[type="submit"]') : null)
          ?? document.querySelector<HTMLElement>(`[data-action="${kind}"]`)
          ?? document.querySelector<HTMLElement>(".m02-detail-title")
          ?? (kind === "propose" ? document.querySelector<HTMLElement>(".m02-propose") : null);
      target?.focus();
    }, 0);
  };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busyRef.current || dead) return;
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { setError(null); setFocusKey((n) => n + 1); return; }
    busyRef.current = true;
    setBusy(true);
    setError(null);
    try {
      const target = k.kind === "propose" && !k.change ? "trust" : k.change?.id ?? "";
      const out = await api.command(commandOf[k.kind], target, page.revision, reason, valuesFor(page, k), tried.current);
      dialog.current?.close();
      onDone(k, out);
    } catch (err) {
      const shown = asTrustFailure(err, page);
      if (shown instanceof AdminError && shown.failure === "unauthenticated") {
        dialog.current?.close();
        onSignedOut();
        return;
      }
      if (shown instanceof AdminError && shown.failure === "invalid" && shown.fields.some((f) => f !== "rationale")) setDead(true);
      if (fatal(shown)) setDead(true);
      if (shown instanceof AdminError && (shown.failure === "stale" || shown.failure === "conflict")) tried.current = attempt();
      setError(shown);
      setFocusKey((n) => n + 1);
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };
  const label = primaryLabel(l, k);
  return (
    <dialog className="m02-confirm" ref={dialog} aria-labelledby="m02-confirm-title"
      onCancel={(e) => { e.preventDefault(); if (!busyRef.current) finish(dead); }}>
      <form onSubmit={submit} noValidate>
        <h2 id="m02-confirm-title" tabIndex={-1} ref={title}>{ty.confirmTitle(l, label, k.change?.number ?? 0)}</h2>
        {problem && (
          <Summary locale={l} title={sc.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />
        )}
        {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={(l2, e2) => trustFailure(l2, e2, k.kind)} />}
        <ConfirmBody l={l} page={page} k={k} expectedDigest={digest} workspace={workspace} dead={dead} />
        {!dead && <Reason l={l} value={reason} onChange={(v) => { tried.current = attempt(); setReason(v); }} error={problem} />}
        <div className="live-buttons m02-confirm-actions">
          {!dead && <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : label}</button>}
          <button type="button" className="live-secondary" disabled={busy} onClick={() => finish(dead)}>{dead ? ty.reloadClose(l) : ty.goBack(l)}</button>
        </div>
      </form>
    </dialog>
  );
}

import { useEffect, useLayoutEffect, useMemo, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../shared/types";
import { reauthenticate } from "./admin";
import { c } from "./copy";
import { cancellable, outcomeOf, settleOnce } from "./reauth-model";

type Problem = "" | "empty" | "wrong";
type Message = "" | "changed" | "failed";

export function Reauthenticate({ locale: l, login, csrf, opener, onDone, onSignedOut }: {
  locale: Locale; login: string; csrf: string; opener: HTMLElement | null;
  onDone: (ok: boolean) => void; onSignedOut: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const field = useRef<HTMLInputElement>(null);
  const busy = useRef(false);
  const scope = useMemo(() => ({ csrf, active:true }), [csrf]);
  const currentScope = useRef(scope);
  currentScope.current = scope;
  const valid = () => scope.active && currentScope.current === scope;
  const finish = useMemo(() => settleOnce(onDone), [scope]);
  const [pending, setPending] = useState(false);
  const [problem, setProblem] = useState<Problem>("");
  const [message, setMessage] = useState<Message>("");
  const [status, setStatus] = useState("");
  const clear = () => { if (field.current) field.current.value = ""; };
  useLayoutEffect(() => {
    scope.active = true;busy.current = false;setPending(false);setProblem("");setMessage("");setStatus("");clear();
    return () => { scope.active = false;clear(); };
  }, [scope]);
  const restore = () => {
    let tries = 0;
    const go = () => {
      if (!valid() || !opener || !opener.isConnected) return;
      if (opener.matches(":disabled") && tries++ < 60) {
        requestAnimationFrame(go);
        return;
      }
      opener.focus();
    };
    go();
  };
  const close = (ok: boolean) => {
    if (!valid()) return;
    clear();
    dialog.current?.close();
    restore();
    finish(ok);
  };
  useEffect(() => {
    const d = dialog.current;
    if (d && !d.open) d.showModal();
    field.current?.focus();
    return () => {
      clear();
      if (d?.open) d.close();
    };
  }, []);
  useEffect(() => clear, [l]);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (busy.current) return;
    const password = field.current?.value ?? "";
    setMessage("");
    setStatus("");
    if (!password) {
      setProblem("empty");
      if (document.activeElement === field.current) setStatus(c.reauthEmpty(l));
      else field.current?.focus();
      return;
    }
    busy.current = true;
    setPending(true);
    setProblem("");
    const typing = document.activeElement === field.current;
    try {
      await reauthenticate(csrf, password);
      if (!valid()) return;
      close(true);
    } catch (err) {
      if (!valid()) return;
      const outcome = outcomeOf(err);
      if (outcome.kind === "wrong") {
        setProblem("wrong");
        if (typing && document.activeElement === field.current) setStatus(c.reauthWrong(l));
      } else if (outcome.kind === "signed-out") {
        clear();
        dialog.current?.close();
        onSignedOut();
        finish(false);
      } else {
        setMessage(outcome.kind === "changed" ? "changed" : "failed");
      }
    } finally {
      if (!valid()) return;
      clear();
      busy.current = false;
      setPending(false);
      if (dialog.current?.open) field.current?.focus();
    }
  };
  const errorText = problem === "wrong" ? c.reauthWrong(l) : problem === "empty" ? c.reauthEmpty(l) : "";
  const messageText = message === "changed" ? c.reauthChanged(l) : message === "failed" ? c.reauthFailed(l) : "";
  const described = ["reauth-hint", errorText ? "reauth-error" : ""].filter(Boolean).join(" ");
  return (
    <dialog
      ref={dialog}
      className="service-reauth"
      lang={l}
      dir={l === "ar" ? "rtl" : "ltr"}
      aria-labelledby="reauth-title"
      aria-describedby="reauth-body"
      onCancel={(e) => {
        e.preventDefault();
        if (cancellable(busy.current)) close(false);
      }}
    >
      <h2 id="reauth-title">{c.reauthTitle(l)}</h2>
      <p id="reauth-body" className="service-reauth-body">{c.reauthBody(l)}</p>
      {messageText && <p className="service-reauth-message" role="alert">{messageText}</p>}
      <form method="post" noValidate onSubmit={submit} aria-busy={pending || undefined}>
        <div className="service-field">
          <label htmlFor="reauth-login">{c.signInEmail(l)}</label>
          <bdi dir="ltr">
            <input id="reauth-login" type="email" dir="ltr" autoComplete="username" value={login} readOnly />
          </bdi>
        </div>
        <div className="service-field">
          <label htmlFor="reauth-password">{c.password(l)}</label>
          <p className="service-hint" id="reauth-hint">{c.reauthHint(l)}</p>
          {errorText && (
            <p className="service-field-error" id="reauth-error">
              <span className="visually-hidden">{c.errorPrefix(l)} </span>
              {errorText}
            </p>
          )}
          <input
            id="reauth-password" ref={field} type="password" dir="ltr" autoComplete="current-password"
            spellCheck={false} autoCapitalize="none" aria-invalid={errorText ? true : undefined} aria-describedby={described}
          />
        </div>
        <div className="dialog-actions service-reauth-actions">
          <button type="button" className="service-retry" disabled={pending} onClick={() => { if (cancellable(busy.current)) close(false); }}>
            {c.reauthCancel(l)}
          </button>
          <button type="submit" className="primary service-submit" disabled={pending} aria-busy={pending || undefined}>
            {pending ? c.reauthConfirming(l) : c.reauthConfirm(l)}
          </button>
        </div>
      </form>
      <p className="visually-hidden" role="status" aria-live="polite">{status}</p>
    </dialog>
  );
}

import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import type { PolicyPage, RevisionRow, TrustChange } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { Summary, type Problems } from "../../../../service/Service";
import { w } from "../users/live-copy";
import { p } from "./live-copy";
import { policyStep, stepMinutes, wholeMinutes } from "./live-model";
import { ty } from "./trust-copy";
import type { Proposed } from "./trust-model";

export type { Proposed };

function sameAs(r: RevisionRow | null, v: Proposed): boolean {
  return r !== null && r.max_age_seconds === v.maxAge * 60 && policyStep(r) === v.step * 60 && r.mfa === "none";
}

export function ProposeForm({ l, page, revise, initial, onReview, onCancel }: {
  l: Locale; page: PolicyPage; revise: TrustChange | null; initial: Proposed | null; onReview: (v: Proposed) => void; onCancel: () => void;
}) {
  const base = revise ? revise.candidate : page.policy;
  const startStep = initial ? initial.step * 60 : base ? policyStep(base) : 0;
  const [age, setAge] = useState(initial ? String(initial.maxAge) : base ? String(Math.round(base.max_age_seconds / 60)) : "");
  const [stepOn, setStepOn] = useState(startStep > 0);
  const [step, setStep] = useState(startStep ? String(Math.round(startStep / 60)) : "");
  const [problems, setProblems] = useState<Problems>([]);
  const [focusKey, setFocusKey] = useState(0);
  const [touched, setTouched] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  const maxAge = wholeMinutes(age, 5, 720);
  const stepValue = stepOn ? stepMinutes(step, maxAge) : 0;
  const same = maxAge !== null && stepValue !== null && sameAs(page.policy, { maxAge, step: stepValue });
  const submit = (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (maxAge === null) found.push({ field: "m02-age", message: p.errMaxAge(l) });
    if (stepOn && stepValue === null) found.push({ field: "m02-step", message: p.errStep(l, maxAge ?? 720) });
    if (!found.length && same) found.push({ field: "m02-age", message: ty.fail.unchanged(l) });
    setProblems(found);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    onReview({ maxAge: maxAge ?? 0, step: stepValue ?? 0 });
  };
  const field = (id: string) => problems.find((x) => x.field === id)?.message;
  const minutesField = (id: string, label: string, value: string, set: (v: string) => void, hint: string) => (
    <div className="service-field live-minutes">
      <label htmlFor={id}>{label}</label>
      <p className="service-hint" id={id + "-hint"}>{hint}</p>
      {field(id) && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{c.errorPrefix(l)} </span>{field(id)}</p>}
      <span className="live-unit">
        <input id={id} inputMode="numeric" dir="ltr" value={value} aria-invalid={field(id) ? true : undefined}
          aria-describedby={[id + "-hint", field(id) ? id + "-error" : ""].filter(Boolean).join(" ")} onChange={(e) => { setTouched(true); set(e.target.value); }} />
        <span>{p.minutes(l)}</span>
      </span>
    </div>
  );
  return (
    <form className="live-step m02-form" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{revise ? ty.reviseTitle(l, revise.number) : ty.formTitle(l)}</h2>
      <p className="live-impact">{ty.formLead(l)}</p>
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {minutesField("m02-age", p.maxAge(l), age, setAge, p.maxAgeHint(l))}
      <fieldset className="live-roles">
        <legend>{p.stepUp(l)}</legend>
        <label><input type="radio" name="m02-step-on" checked={!stepOn} onChange={() => { setTouched(true); setStepOn(false); }} />{p.stepNone(l)}</label>
        <label><input type="radio" name="m02-step-on" checked={stepOn} onChange={() => { setTouched(true); setStepOn(true); }} />{p.stepWithin(l)}</label>
      </fieldset>
      {stepOn && minutesField("m02-step", p.stepMinutes(l), step, setStep, p.stepHint(l))}
      {same && touched && <p className="live-note">{ty.fail.unchanged(l)}</p>}
      <div className="live-buttons">
        <button type="submit" className="live-primary">{ty.review(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

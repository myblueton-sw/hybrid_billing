import type { Locale } from "../../../../shared/types";
import type { RequirementRow } from "../../../../service/release-evidence";
import { c as sc } from "../../../../service/copy";
import { f } from "./live-copy";
import { noReplacement, replacementOptions, replacementProblem, retireValues, textProblem } from "./live-model";
import type { FieldErrors, Fields } from "./LiveCommand";

export function retireTextError(l: Locale, v: string): string {
  const p = textProblem(v);
  if (p === "missing") return f.retireReasonMissing(l);
  if (p === "long") return f.retireReasonLong(l);
  if (p === "control") return f.retireReasonControl(l);
  return "";
}

export function replacementError(l: Locale, problem: string): string {
  return problem === "required" ? f.replacementRequired(l) : problem === "choose" ? f.replacementChoose(l) : "";
}

export function retireFields(l: Locale, r: RequirementRow, frozen = ""): Fields {
  const required = r.replacement_required === true;
  const listed = replacementOptions(r);
  const options = frozen && frozen !== noReplacement && !listed.some((o) => o.target === frozen) ? [...listed, { target: frozen, label: frozen }] : listed;
  return {
    keys: ["reason", "replacement"],
    saved: ["reason"],
    check: (d) => {
      const e: FieldErrors = {};
      const text = retireTextError(l, d.reason ?? "");
      if (text) e.reason = text;
      const choice = replacementError(l, replacementProblem(d.replacement ?? "", required, options));
      if (choice) e.replacement = choice;
      return e;
    },
    values: (d) => retireValues(d),
    render: (d, set, errors, id) => (
      <>
        <div className="service-field p05-text-field">
          <label htmlFor={id + "-reason"}>{f.retireReason(l)}</label>
          <p className="service-hint" id={id + "-reason-hint"}>{f.retireReasonHint(l)}</p>
          {errors.reason && <p className="service-field-error" id={id + "-reason-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors.reason}</p>}
          <input
            id={id + "-reason"} type="text" autoComplete="off" dir="auto" value={d.reason ?? ""} aria-invalid={!!errors.reason || undefined}
            aria-describedby={[id + "-reason-hint", errors.reason ? id + "-reason-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("reason", e.target.value)}
          />
        </div>
        <div className="service-field p05-select-field">
          <label htmlFor={id + "-replacement"}>{f.replacement(l)}</label>
          <p className="service-hint" id={id + "-replacement-hint"}>{f.replacementHint(l)}</p>
          {r.replacements.more === true && <p className="service-hint">{f.replacementMore(l)}</p>}
          {errors.replacement && <p className="service-field-error" id={id + "-replacement-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors.replacement}</p>}
          <select
            id={id + "-replacement"} value={d.replacement ?? ""} aria-invalid={!!errors.replacement || undefined}
            aria-describedby={[id + "-replacement-hint", errors.replacement ? id + "-replacement-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("replacement", e.target.value)}
          >
            <option value="">{f.choose(l)}</option>
            {!required && <option value={noReplacement}>{f.noReplacementOption(l)}</option>}
            {options.map((o) => <option key={o.target} value={o.target}>{"\u2066" + o.label + "\u2069"}</option>)}
          </select>
        </div>
      </>
    ),
  };
}

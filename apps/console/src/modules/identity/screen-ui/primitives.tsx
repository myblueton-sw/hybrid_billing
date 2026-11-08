import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../../shared/types";
import { say, type Props, type Words } from "./types";
import { labels, common } from "./copy";
import { fieldsFor, reviewState, validateDraft, type Rule } from "./model";
export function Field({
  name,
  l,
  type = "text",
  optional = false,
}: {
  name: string;
  l: Locale;
  type?: string;
  optional?: boolean;
}) {
  return (
    <label className="security-field" htmlFor={`security-${name}`}>
      <span>
        {say(l, labels[name])}
        {optional ? (
          <small> {say(l, ["Optional", "선택", "اختياري"])}</small>
        ) : null}
      </span>
      <input
        id={`security-${name}`}
        name={name}
        type={type}
        maxLength={256}
        autoComplete="off"
        autoCapitalize="none"
        spellCheck={false}
        dir="auto"
      />
    </label>
  );
}
export function Choice({
  name,
  l,
  options,
}: {
  name: string;
  l: Locale;
  options: readonly (readonly [string, Words])[];
}) {
  return (
    <label className="security-field" htmlFor={`security-${name}`}>
      <span>{say(l, labels[name])}</span>
      <select id={`security-${name}`} name={name} defaultValue="">
        <option value="">
          {say(l, [
            "Choose for this local draft",
            "로컬 초안에서 선택",
            "اختر لهذه المسودة المحلية",
          ])}
        </option>
        {options.map(([value, words]) => (
          <option value={value} key={value}>
            {say(l, words)}
          </option>
        ))}
      </select>
    </label>
  );
}
export function Note({ l, words }: { l: Locale; words: Words }) {
  return <p className="security-note">{say(l, words)}</p>;
}
export function Observations({
  l,
  title,
  fields,
}: {
  l: Locale;
  title: Words;
  fields: readonly Words[];
}) {
  return (
    <section className="security-observations">
      <h2>{say(l, title)}</h2>
      <dl>
        {fields.map((f, i) => (
          <div key={i}>
            <dt>{say(l, f)}</dt>
            <dd>{say(l, common.unknown)}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
export function LocalForm({
  props,
  title,
  children,
  rules,
  help,
  items = [],
  variant = "",
}: {
  props: Props;
  title: Words;
  children: ReactNode;
  rules: Record<string, Rule>;
  help?: Words;
  items?: string[];
  variant?: string;
}) {
  const { screenId, view, locale: l, onState, onDirty } = props;
  const [instance] = useState(() => crypto.randomUUID());
  const [revision, setRevision] = useState(0),
    [dirty, setDirty] = useState(false),
    [stage, setStage] = useState("editing"),
    [errors, setErrors] = useState<string[]>([]);
  const form = useRef<HTMLFormElement>(null),
    message = useRef<HTMLDivElement>(null);
  const fields = fieldsFor(screenId, view);
  useEffect(() => {
    onState(
      reviewState(view, revision, dirty, stage, errors, instance, variant),
      [
        ...fields.map((f) => `${screenId}:input:${view}:${f}`),
        `${screenId}:action:${view}:check-local`,
        `${screenId}:action:${view}:clear`,
        `${screenId}:state:unconnected`,
        `${screenId}:state:${view}:local-validation`,
        `${screenId}:data:${view}:source-observations`,
        ...items.map((i) => `${screenId}:${i}`),
      ],
    );
  }, [
    view,
    revision,
    dirty,
    stage,
    errors,
    onState,
    screenId,
    instance,
    variant,
  ]);
  useEffect(() => () => onDirty(false), [onDirty]);
  function clear() {
    form.current?.reset();
    form.current
      ?.querySelectorAll("[aria-invalid]")
      .forEach((e) => e.removeAttribute("aria-invalid"));
    setDirty(false);
    onDirty(false);
    setErrors([]);
    setStage("editing");
    setRevision((n) => n + 1);
  }
  return (
    <section className="security-draft">
      <div className="security-section-heading">
        <h2>{say(l, title)}</h2>
        <span>{say(l, common.local)}</span>
      </div>
      {help && <Note l={l} words={help} />}
      <form
        ref={form}
        noValidate
        onInput={() => {
          form.current
            ?.querySelectorAll("[aria-invalid]")
            .forEach((e) => e.removeAttribute("aria-invalid"));
          setDirty(true);
          onDirty(true);
          setStage("editing");
          setErrors([]);
          setRevision((n) => n + 1);
        }}
        onSubmit={(e) => {
          e.preventDefault();
          const next = validateDraft(new FormData(e.currentTarget), rules);
          setErrors(next);
          setStage(next.length ? "invalid" : "checked");
          setRevision((n) => n + 1);
          for (const input of e.currentTarget.elements) {
            if (
              input instanceof HTMLInputElement ||
              input instanceof HTMLSelectElement
            )
              input.setAttribute(
                "aria-invalid",
                String(next.includes(input.name)),
              );
          }
          if (next.length)
            (
              e.currentTarget.elements.namedItem(next[0]) as HTMLElement
            )?.focus();
          else message.current?.focus();
        }}
      >
        <div className="security-fields">{children}</div>
        <div className="security-actions">
          <button type="submit">{say(l, common.check)}</button>
          <button type="button" onClick={clear}>
            {say(l, common.clear)}
          </button>
        </div>
        <div
          className="security-feedback"
          ref={message}
          tabIndex={-1}
          role={stage === "invalid" ? "alert" : "status"}
        >
          {stage === "invalid" ? (
            <>
              <p>{say(l, common.invalid)}</p>
              <ul>
                {errors.map((f) => (
                  <li key={f}>
                    <a
                      href={`#security-${f}`}
                      onClick={(e) => {
                        e.preventDefault();
                        form.current
                          ?.querySelector<HTMLElement>(`[name="${f}"]`)
                          ?.focus();
                      }}
                    >
                      {say(l, labels[f])}
                    </a>
                  </li>
                ))}
              </ul>
            </>
          ) : stage === "checked" ? (
            <p>{say(l, common.checked)}</p>
          ) : null}
        </div>
      </form>
      <Note l={l} words={common.boundary} />
    </section>
  );
}
export function StaticState({
  props,
  items = [],
}: {
  props: Props;
  items?: string[];
}) {
  useEffect(() => {
    props.onDirty(false);
    props.onState(reviewState(props.view, 0, false, "unconnected", []), [
      `${props.screenId}:state:unconnected`,
      ...items.map((i) => `${props.screenId}:${i}`),
    ]);
  }, [props.screenId, props.view, props.onState, props.onDirty]);
  return null;
}
export function Unconnected({ l, words }: { l: Locale; words: Words }) {
  return (
    <div className="security-connection" role="status">
      <strong>{say(l, common.unconnected)}</strong>
      <p>{say(l, words)}</p>
    </div>
  );
}
export function Links({
  l,
  links,
}: {
  l: Locale;
  links: readonly (readonly [string, Words])[];
}) {
  return (
    <nav
      className="security-links"
      aria-label={say(l, [
        "Related security tasks",
        "관련 보안 업무",
        "مهام أمنية ذات صلة",
      ])}
    >
      {links.map(([path, label]) => (
        <a key={path} href={`#${path}?lang=${l}`}>
          {say(l, label)}
        </a>
      ))}
    </nav>
  );
}

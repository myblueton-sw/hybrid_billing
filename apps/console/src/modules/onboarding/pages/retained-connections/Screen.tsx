import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import {
  query,
  queryString,
  identifier,
  recordKeys,
  type Data,
  type Query,
  type Registration,
} from "./contract";
import { load, ReadError, type ReadResult } from "./api";
import { text, timestamp, words, type Words } from "./copy";
import "./screen.css";
type Props = {
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (state: string, items: string[]) => void;
};
type State = {
  key: string;
  result?: ReadResult;
  error?: string;
  requestId?: string;
};
function href(
  params: URLSearchParams,
  q: Query,
  changes: {
    q?: string;
    after?: number;
    snapshot?: string;
    permission_revision?: string;
  },
) {
  const next = new URLSearchParams(params);
  for (const key of [...next.keys()])
    if (key.startsWith("connection-")) next.delete(key);
  next.set("view", q.view);
  const prefix = changes.q ?? q.q;
  if (prefix) next.set("connection-q", prefix);
  if (changes.after) next.set("connection-after", String(changes.after));
  if (changes.snapshot) next.set("connection-snapshot", changes.snapshot);
  if (changes.permission_revision)
    next.set("connection-permission", changes.permission_revision);
  return "#/connections?" + next.toString();
}
function DateValue({ value }: { value: string }) {
  return (
    <time dateTime={value}>
      <bdi dir="ltr">{timestamp(value)}</bdi>
    </time>
  );
}
function RegistrationDetail({
  record,
  locale,
  onClose,
}: {
  record: Registration;
  locale: Locale;
  onClose: () => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    heading.current?.focus({ preventScroll: true });
    heading.current?.scrollIntoView({ block: "nearest" });
  }, [record.id]);
  const fields: [Words, ReactNode][] = [
    [
      words.evidenceId,
      <bdi dir="ltr" className="retained-connection-id">
        {record.id}
      </bdi>,
    ],
    [words.version, <bdi dir="ltr">{record.focus_version}</bdi>],
    [
      words.grain,
      text(locale, record.grain === "day" ? words.day : words.month),
    ],
    [words.from, <DateValue value={record.window_start} />],
    [words.to, <DateValue value={record.window_end} />],
    [
      words.domains,
      <>
        <bdi dir="ltr">{record.registered_domains}</bdi>
        <p className="retained-connection-note">
          {text(locale, words.domainLimit)}
        </p>
      </>,
    ],
    [
      words.profile,
      <>
        {text(
          locale,
          record.semantic_profile === "focus-consumed-v1"
            ? words.consumed
            : words.exploratory,
        )}
        <p>
          <bdi dir="ltr">{record.semantic_profile}</bdi>
        </p>
      </>,
    ],
  ];
  return (
    <section
      className="retained-connection-detail"
      aria-label={text(locale, words.detail)}
    >
      <div className="retained-connection-detail-heading">
        <h2 ref={heading} tabIndex={-1}>
          <bdi dir="ltr">{record.connection}</bdi>
        </h2>
        <button
          type="button"
          onClick={onClose}
          aria-label={text(locale, words.close)}
        >
          <span className="retained-connection-close-wide">
            {text(locale, words.close)}
          </span>
          <span className="retained-connection-close-mobile">
            {text(locale, words.back)}
          </span>
        </button>
      </div>
      <dl>
        {fields.map(([label, value]) => (
          <div key={label[1]}>
            <dt>{text(locale, label)}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      <section className="retained-connection-policy">
        <h3>{text(locale, words.policy)}</h3>
        <dl>
          <div>
            <dt>{text(locale, words.lateness)}</dt>
            <dd>
              <bdi dir="ltr">{record.allowed_lateness_seconds}</bdi>
            </dd>
          </div>
          <div>
            <dt>{text(locale, words.replacements)}</dt>
            <dd>
              <bdi dir="ltr">{record.replacement_limit}</bdi>
            </dd>
          </div>
        </dl>
      </section>
      <a
        href={`#/imports?${new URLSearchParams({ lang: locale, "receipt-q": record.connection }).toString()}`}
      >
        {text(locale, words.receipts)}
      </a>
      <p className="retained-connection-note">
        {text(locale, words.destination)}
      </p>
    </section>
  );
}
export default function RetainedConnections(props: Props) {
  const locale = props.locale;
  let requested: Query | undefined;
  try {
    requested = query(props.params, props.view);
  } catch {}
  const serialized = requested ? queryString(requested) : "invalid";
  const [attempt, setAttempt] = useState(0),
    [editRevision, setEditRevision] = useState(0);
  const [state, setState] = useState<State>({ key: "" });
  const [search, setSearch] = useState(requested?.q ?? ""),
    [invalid, setInvalid] = useState(false);
  const [selection, setSelection] = useState<{
    key: string;
    snapshot: string;
    id: string;
  } | null>(null);
  const key = JSON.stringify([
    locale,
    serialized,
    props.params.toString(),
    attempt,
  ]);
  const current = state.key === key ? state : undefined;
  const data: Data | undefined =
    current?.result?.kind === "loaded" ? current.result.data : undefined;
  const phase =
    !requested || current?.error
      ? "error"
      : data
        ? "loaded"
        : current?.result?.kind === "unconnected"
          ? "unconnected"
          : "loading";
  const selected =
    data && selection?.key === key && selection.snapshot === data.snapshot
      ? data.records.find((record) => record.id === selection.id)
      : undefined;
  const supported =
    props.view === "connections" || props.view === "capabilities";
  const seq = useRef(0),
    input = useRef<HTMLInputElement>(null),
    heading = useRef<HTMLHeadingElement>(null);
  const status = useRef<HTMLParagraphElement>(null),
    trigger = useRef<HTMLButtonElement | null>(null);
  const focusStatus = useRef(false);
  useEffect(() => {
    const controller = new AbortController(),
      sequence = ++seq.current;
    if (selection) heading.current?.focus({ preventScroll: true });
    setSelection(null);
    setState({ key });
    setSearch(requested?.q ?? "");
    setInvalid(false);
    if (requested)
      load(requested, controller.signal)
        .then((result) => {
          if (!controller.signal.aborted && sequence === seq.current)
            setState({ key, result });
        })
        .catch((error) => {
          if (!controller.signal.aborted && sequence === seq.current)
            setState({
              key,
              error:
                error instanceof ReadError ? error.code : "service_unavailable",
              requestId:
                error instanceof ReadError ? error.requestId : undefined,
            });
        });
    return () => controller.abort();
  }, [key]);
  useEffect(() => {
    props.onState(
      JSON.stringify({
        profile: "retained-focus-connections-v1",
        phase,
        view: props.view,
        requested: requested ?? null,
        request_authority: "unverified",
        editRevision,
        source: data
          ? {
              auth_scope: data.auth_scope,
              permission_revision: data.permission_revision,
              snapshot: data.snapshot,
              total: data.total,
              capabilities: data.capabilities,
            }
          : null,
        error: phase === "error" ? (current?.error ?? "invalid_query") : null,
        selected: selected?.id ?? null,
      }),
      [
        `S01:data:${props.view}`,
        "S01:scope:connection",
        "S01:state:async",
        "S01:action:filter",
        "S01:action:refresh",
        "S01:action:inspect",
        ...recordKeys.map((field) => `S01:field:${field}`),
      ],
    );
  }, [
    props.onState,
    props.view,
    phase,
    serialized,
    data,
    current?.error,
    selected?.id,
    editRevision,
  ]);
  useEffect(() => {
    if (phase !== "loading" && focusStatus.current) {
      focusStatus.current = false;
      (phase === "loaded" && data?.total
        ? heading.current
        : status.current
      )?.focus({ preventScroll: true });
    }
  }, [phase, data]);
  const close = () => {
    setSelection(null);
    setEditRevision((n) => n + 1);
    requestAnimationFrame(() => {
      if (trigger.current?.isConnected) trigger.current.focus();
      else heading.current?.focus();
    });
  };
  const reload = () => {
    focusStatus.current = true;
    setSelection(null);
    setState({ key: "" });
    setEditRevision((n) => n + 1);
    if (
      requested &&
      (requested.after !== 0 ||
        requested.snapshot ||
        requested.permission_revision)
    )
      location.hash = href(props.params, requested, { after: 0 });
    else if (requested) setAttempt((n) => n + 1);
    else {
      const next = new URLSearchParams(props.params);
      for (const name of [...next.keys()])
        if (name.startsWith("connection-") || name === "view")
          next.delete(name);
      next.set("view", "connections");
      location.hash = "#/connections?" + next.toString();
    }
  };
  const move = (after: number) => {
    if (!data || !requested) return;
    focusStatus.current = true;
    setSelection(null);
    setState({ key: "" });
    location.hash = href(props.params, requested, {
      after,
      snapshot: data.snapshot,
      permission_revision: data.permission_revision,
    });
  };
  const errorCode = requested ? current?.error : "invalid_query";
  const failure =
    errorCode === "scope_denied" || errorCode === "permission"
      ? words.denied
      : errorCode === "query_snapshot_changed" ||
          errorCode === "snapshot_required"
        ? words.changed
        : errorCode === "invalid_response"
          ? words.invalidResponse
          : words.error;
  const counter =
    data && data.total !== null ? (
      <span className="retained-connection-counter">
        {text(locale, words.range)}:{" "}
        <bdi dir="ltr">
          {data.total === 0
            ? "0"
            : `${data.selection.after + 1}–${data.selection.after + data.records.length}`}
        </bdi>
        {" / "}
        <bdi dir="ltr">{data.total}</bdi>
      </span>
    ) : null;
  const pager =
    data && data.total !== null && data.total > 0 && requested ? (
      <nav
        className="retained-connection-pagination"
        aria-label={text(locale, words.page)}
      >
        <span>
          {text(locale, words.page)}{" "}
          <bdi dir="ltr">
            {Math.floor(data.selection.after / 25) + 1} /{" "}
            {Math.ceil(data.total / 25)}
          </bdi>
        </span>
        <div>
          <button
            type="button"
            disabled={requested.after === 0}
            onClick={() => move(Math.max(0, requested.after - 25))}
          >
            {text(locale, words.previous)}
          </button>
          <button
            type="button"
            disabled={data.next_after === null}
            onClick={() => data.next_after !== null && move(data.next_after)}
          >
            {text(locale, words.next)}
          </button>
        </div>
      </nav>
    ) : null;
  return (
    <section
      className="retained-connections-screen"
      data-read-phase={phase}
      aria-busy={phase === "loading"}
      onKeyDown={(event) => {
        if (event.key === "Escape" && selected) {
          event.preventDefault();
          event.stopPropagation();
          close();
        }
      }}
    >
      <header className="retained-connection-context">
        <h2 ref={heading} tabIndex={-1}>
          {text(locale, words.context)}
        </h2>
        <p>{text(locale, words.boundary)}</p>
        {data && (
          <dl className="retained-connection-scope">
            <div>
              <dt>{text(locale, words.scope)}</dt>
              <dd>
                {text(
                  locale,
                  data.audience === "company" ? words.company : words.tenant,
                )}
              </dd>
            </div>
            <div>
              <dt>{text(locale, words.members)}</dt>
              <dd>
                <bdi dir="ltr">{data.member_count}</bdi>
              </dd>
            </div>
            <div>
              <dt>{text(locale, words.observed)}</dt>
              <dd>
                <DateValue value={data.observed_at} />
              </dd>
            </div>
          </dl>
        )}
      </header>
      <div
        className={`retained-connection-workspace ${selected ? "retained-connection-has-detail" : ""}`}
      >
        <div className="retained-connection-list-area">
          {supported && (
            <>
              <div className="retained-connection-list-heading">
                <h2>
                  {text(
                    locale,
                    props.view === "capabilities"
                      ? words.capabilities
                      : words.list,
                  )}
                </h2>
                {counter}
              </div>
              {props.view === "capabilities" && data && (
                <>
                  <dl className="retained-connection-capabilities">
                    <div>
                      <dt>{text(locale, words.list)}</dt>
                      <dd>{text(locale, words.fullyOwned)}</dd>
                    </div>
                    <div>
                      <dt>{text(locale, words.capabilities)}</dt>
                      <dd>{text(locale, words.registrationOnly)}</dd>
                    </div>
                    <div>
                      <dt>{text(locale, words.externalAuth)}</dt>
                      <dd>{text(locale, words.notConnected)}</dd>
                    </div>
                    <div>
                      <dt>{text(locale, words.diagnosticEvidence)}</dt>
                      <dd>{text(locale, words.notConnected)}</dd>
                    </div>
                    <div>
                      <dt>{text(locale, words.supplierSupport)}</dt>
                      <dd>{text(locale, words.unknown)}</dd>
                    </div>
                  </dl>
                  <p>{text(locale, words.capabilityLimit)}</p>
                </>
              )}
              <form
                role="search"
                className="retained-connection-filter"
                onSubmit={(event) => {
                  event.preventDefault();
                  setEditRevision((n) => n + 1);
                  if (!identifier(search)) {
                    setInvalid(true);
                    input.current?.focus();
                    return;
                  }
                  if (!requested) return;
                  setInvalid(false);
                  focusStatus.current = true;
                  setSelection(null);
                  setState({ key: "" });
                  if (search === requested.q && requested.after === 0)
                    setAttempt((n) => n + 1);
                  else
                    location.hash = href(props.params, requested, {
                      q: search,
                      after: 0,
                    });
                }}
              >
                <label htmlFor="retained-connection-search">
                  {text(locale, words.search)}
                  <input
                    id="retained-connection-search"
                    ref={input}
                    type="search"
                    value={search}
                    maxLength={128}
                    autoComplete="off"
                    spellCheck={false}
                    dir="ltr"
                    aria-invalid={invalid || undefined}
                    aria-describedby={
                      invalid ? "retained-connection-search-error" : undefined
                    }
                    onChange={(event) => {
                      setSearch(event.target.value);
                      setInvalid(false);
                      setEditRevision((n) => n + 1);
                    }}
                  />
                </label>
                <div className="retained-connection-actions">
                  <button type="submit">{text(locale, words.apply)}</button>
                  <button
                    type="button"
                    onClick={() => {
                      setSearch("");
                      setEditRevision((n) => n + 1);
                      if (!requested) return;
                      if (!requested.q && requested.after === 0) reload();
                      else {
                        focusStatus.current = true;
                        setSelection(null);
                        setState({ key: "" });
                        location.hash = href(props.params, requested, {
                          q: "",
                          after: 0,
                        });
                      }
                    }}
                  >
                    {text(locale, words.clear)}
                  </button>
                  {phase !== "error" && phase !== "unconnected" && (
                    <button
                      type="button"
                      onClick={reload}
                      disabled={phase === "loading"}
                    >
                      {text(locale, words.refresh)}
                    </button>
                  )}
                </div>
                {invalid && (
                  <p id="retained-connection-search-error" role="alert">
                    {text(locale, words.invalid)}
                  </p>
                )}
              </form>
            </>
          )}
          <p
            ref={status}
            className="retained-connection-status"
            role={phase === "error" ? "alert" : "status"}
            aria-live="polite"
            tabIndex={-1}
          >
            {phase === "loading"
              ? text(locale, words.loading)
              : phase === "unconnected"
                ? text(locale, words.unconnected)
                : phase === "error"
                  ? text(locale, failure)
                  : !supported
                    ? text(
                        locale,
                        props.view === "security"
                          ? words.security
                          : words.diagnostics,
                      )
                    : data?.total === 0
                      ? text(
                          locale,
                          requested?.q ? words.emptyFiltered : words.emptyScope,
                        )
                      : null}
          </p>
          {(phase === "error" || phase === "unconnected") && (
            <div className="retained-connection-error">
              {errorCode && (
                <p>
                  {text(locale, words.errorCode)}:{" "}
                  <bdi dir="ltr">{errorCode}</bdi>
                </p>
              )}
              {current?.requestId && (
                <p>
                  {text(locale, words.requestId)}:{" "}
                  <bdi dir="ltr">{current.requestId}</bdi>
                </p>
              )}
              <button type="button" onClick={reload}>
                {text(locale, words.retry)}
              </button>
            </div>
          )}
          {phase === "loaded" && !supported && (
            <p>
              {text(
                locale,
                props.view === "security"
                  ? words.securityBody
                  : words.diagnosticsBody,
              )}
            </p>
          )}
          {!supported && phase !== "error" && phase !== "unconnected" && (
            <div className="retained-connection-refresh">
              <button
                type="button"
                disabled={phase === "loading"}
                onClick={reload}
              >
                {text(locale, words.refresh)}
              </button>
            </div>
          )}
          {phase === "loaded" && data && supported && (
            <>
              {data.total === 0 ? (
                <p className="retained-connection-note">
                  {text(locale, words.legacy)}
                </p>
              ) : (
                <>
                  {pager}
                  <p className="retained-connection-note">
                    {text(locale, words.tableHint)}
                  </p>
                  <table className="retained-connection-table">
                    <thead>
                      <tr>
                        <th>{text(locale, words.connection)}</th>
                        <th>{text(locale, words.interval)}</th>
                        <th className="retained-connection-col-medium">
                          {text(locale, words.domains)}
                        </th>
                        <th className="retained-connection-col-medium">
                          {text(locale, words.version)}
                        </th>
                        <th className="retained-connection-col-wide">
                          {text(locale, words.grain)}
                        </th>
                        <th className="retained-connection-col-profile">
                          {text(locale, words.profile)}
                        </th>
                        <th className="retained-connection-col-id">
                          {text(locale, words.evidenceId)}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.records.map((record) => (
                        <tr
                          key={record.id}
                          aria-selected={selected?.id === record.id}
                        >
                          <td>
                            <button
                              type="button"
                              className="retained-connection-row-button"
                              aria-expanded={selected?.id === record.id}
                              onClick={(event) => {
                                trigger.current = event.currentTarget;
                                setSelection({
                                  key,
                                  snapshot: data.snapshot,
                                  id: record.id,
                                });
                                setEditRevision((n) => n + 1);
                              }}
                            >
                              <bdi dir="ltr">{record.connection}</bdi>
                            </button>
                          </td>
                          <td>
                            <span className="retained-connection-interval">
                              <span>
                                <span className="retained-connection-interval-label">
                                  {text(locale, words.from)}
                                </span>
                                <DateValue value={record.window_start} />
                              </span>
                              <span>
                                <span className="retained-connection-interval-label">
                                  {text(locale, words.to)}
                                </span>
                                <DateValue value={record.window_end} />
                              </span>
                            </span>
                          </td>
                          <td className="retained-connection-col-medium">
                            <bdi dir="ltr">{record.registered_domains}</bdi>
                          </td>
                          <td className="retained-connection-col-medium">
                            <bdi dir="ltr">{record.focus_version}</bdi>
                          </td>
                          <td className="retained-connection-col-wide">
                            {text(
                              locale,
                              record.grain === "day" ? words.day : words.month,
                            )}
                          </td>
                          <td className="retained-connection-col-profile">
                            {text(
                              locale,
                              record.semantic_profile === "focus-consumed-v1"
                                ? words.consumed
                                : words.exploratory,
                            )}
                          </td>
                          <td className="retained-connection-col-id">
                            <bdi dir="ltr">{record.id.slice(0, 12)}</bdi>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {pager}
                </>
              )}
            </>
          )}
        </div>
        {selected && (
          <RegistrationDetail
            record={selected}
            locale={locale}
            onClose={close}
          />
        )}
      </div>
    </section>
  );
}

import { useCallback, useId, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type {
  PageQuery,
  ServicePage,
} from "../../../../service/service-query-types";
import { FailureBox } from "../users/Live";
import { w } from "../users/live-copy";
import { v } from "./live-copy";
import { AdminError } from "../../../../service/admin";
import { ErrorDiagnostics } from "../../../../service/CommandDiagnostics";
import { qcopy as t } from "./query-copy";
import { FieldMessage, fieldAttributes } from "./form-validation";
import type { FormField } from "./form-validation";
import { cursorTrail, queryKey } from "./query-model";
import { readIdentity, useScopedRead, type ReadState } from "./scoped-read";
import type { Recovery } from "./recovery-context";

export function usePanelHeading(target: string) {
  return useCallback(
    (heading: HTMLHeadingElement | null) => {
      heading?.focus();
    },
    [target],
  );
}

export const bdi = (id: string) => <bdi dir="ltr">{id}</bdi>;
function describeRead(l: Locale, error: unknown) {
  if (error instanceof AdminError && error.failure === "denied")
    return { title: v.loadFailed(l), body: t(l, "readDenied") };
  const changed =
    error instanceof AdminError &&
    (error.code === "revision_conflict" || error.code === "invalid_request");
  return {
    title: v.loadFailed(l),
    body: t(l, changed ? "readQueryChanged" : "readUnavailable"),
  };
}
export function Facts({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="live-facts">
      {items.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}
export function ReadBoundary<T>({
  l,
  read,
  reload,
  restart,
  children,
}: {
  l: Locale;
  read: ReadState<T>;
  reload: () => void;
  restart?: () => void;
  children: (value: T) => ReactNode;
}) {
  if (read.state === "loading") return <p role="status">{v.loading(l)}</p>;
  if (read.state === "error")
    return (
      <>
        <FailureBox
          l={l}
          error={read.error}
          focusKey={0}
          describe={describeRead}
        />
        <ErrorDiagnostics l={l} error={read.error} />
        <button type="button" className="live-secondary" onClick={reload}>
          {w.reload(l)}
        </button>
        {restart && (
          <button type="button" className="live-secondary" onClick={restart}>
            {t(l, "first")}
          </button>
        )}
      </>
    );
  return <>{children(read.value)}</>;
}
export function PageInfo<T>({
  l,
  page,
  first,
  next,
  previous,
}: {
  l: Locale;
  page: ServicePage<T>;
  first: () => void;
  next: () => void;
  previous: () => void;
}) {
  return (
    <div className="live-sections">
      <Facts
        items={[
          [
            t(l, page.count.certainty === "exact" ? "exact" : "lower"),
            page.count.value,
          ],
          [t(l, "reference"), bdi(page.reference_at)],
          [t(l, "observed"), bdi(page.observed_at)],
        ]}
      />
      {page.rows.length === 0 && (
        <p className="live-empty">{t(l, page.next ? "incomplete" : "empty")}</p>
      )}
      <nav aria-label={t(l, "size")} className="live-buttons">
        <button type="button" className="live-secondary" onClick={first}>
          {t(l, "first")}
        </button>
        <button type="button" className="live-secondary" onClick={previous}>
          {t(l, "previous")}
        </button>
        <button
          type="button"
          className="live-secondary"
          disabled={!page.next}
          onClick={next}
        >
          {t(l, "next")}
        </button>
      </nav>
    </div>
  );
}
export function Picker<T extends { id: string }>({
  l,
  recovery,
  name,
  scope,
  value,
  locked,
  emptyCatalog,
  load,
  resolve,
  label,
  onChange,
  field,
  error,
}: {
  l: Locale;
  recovery: Recovery;
  name: string;
  scope: string;
  value: string;
  locked?: boolean;
  emptyCatalog?: boolean;
  load: (query: PageQuery) => Promise<ServicePage<T>>;
  resolve: (id: string) => Promise<{ row: T }>;
  label: (row: T) => string;
  onChange: (id: string) => void;
  field?: FormField;
  error?: string;
}) {
  const generatedId = useId(),
    searchId = useId();
  const attributes = field
    ? fieldAttributes(field, error)
    : { id: generatedId };
  const fieldId = attributes.id;
  const controlId = field ? fieldId + "-select" : fieldId;
  const [q, setQ] = useState(""),
    [applied, setApplied] = useState(""),
    [after, setAfter] = useState("");
  const [trail] = useState(cursorTrail);
  const query = {
    workspace: scope.split("\n")[0],
    q: applied,
    after,
    limit: 10,
  };
  const read = useScopedRead(
    recovery,
    readIdentity("picker", name, scope, JSON.stringify(query)),
    () => load(query),
  );
  const known = useScopedRead(
    recovery,
    readIdentity("picker-known", name, scope, value),
    () => resolve(value),
    !!value,
  );
  const selected =
    known.result.state === "ready"
      ? label(known.result.value.row)
      : value + " · " + t(l, "unavailable");
  const key = queryKey(query);
  return (
    <fieldset
      className="live-sections"
      disabled={locked}
      {...(field ? attributes : {})}
      tabIndex={field ? -1 : undefined}
    >
      <legend>{name}</legend>
      {field && <FieldMessage field={field} error={error} />}
      <div className="service-field">
        <label htmlFor={searchId}>{t(l, "search")}</label>
        <input
          id={searchId}
          value={q}
          onChange={(e) => {
            if (e.target.value.length <= 256) setQ(e.target.value);
          }}
        />
        <button
          type="button"
          className="live-secondary"
          onClick={() => {
            setApplied(q);
            setAfter("");
            trail.clear();
          }}
        >
          {t(l, "search")}
        </button>
      </div>
      {value && <p className="live-note">{bdi(selected)}</p>}
      <ReadBoundary
        l={l}
        read={read.result}
        reload={read.reload}
        restart={() => {
          setAfter("");
          trail.clear();
          read.reload();
        }}
      >
        {(page) => (
          <>
            {emptyCatalog &&
              !query.q &&
              !query.after &&
              page.count.certainty === "exact" &&
              page.count.value === "0" && <p>{t(l, "emptyCatalog")}</p>}
            <label htmlFor={controlId}>{name}</label>
            <select
              {...attributes}
              id={controlId}
              className="live-select"
              value={value}
              onChange={(e) => onChange(e.target.value)}
            >
              <option value="">{t(l, "choose")}</option>
              {value && !page.rows.some((r) => r.id === value) && (
                <option value={value}>{selected}</option>
              )}
              {page.rows.map((row) => (
                <option key={row.id} value={row.id}>
                  {label(row)}
                </option>
              ))}
            </select>
            <PageInfo
              l={l}
              page={page}
              first={() => {
                setAfter("");
                trail.clear();
              }}
              previous={() => setAfter(trail.previous(key))}
              next={() => {
                if (trail.next(key, after, page.next)) setAfter(page.next);
              }}
            />
          </>
        )}
      </ReadBoundary>
    </fieldset>
  );
}

import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { Basis, FocusRow } from "../../../sources/collected-focus/api";
import {
  instantParts,
  provenanceClass,
  valueState,
} from "../../../sources/collected-focus/fields";
import {
  fieldLabel,
  Instant,
  missingLabel,
  provenanceLabel,
  zeroLabel,
} from "../../../sources/collected-focus/labels";
import { money } from "../../../sources/collected-focus/values";
import { basisLabel } from "./copy";
export function SplitId({ id }: { id: string }) {
  const middle = Math.ceil(id.length / 2);
  return (
    <bdi dir="ltr" className="infra-code">
      <span className="infra-token">{id.slice(0, middle)}</span>
      <wbr />
      <span className="infra-token">{id.slice(middle)}</span>
    </bdi>
  );
}
const wide = (value: string | null) =>
  value !== null && value.length > 16 ? "infra-identity-wide" : undefined;
export function RowIdentity({
  locale: l,
  row,
  showDelivery,
}: {
  locale: Locale;
  row: FocusRow;
  showDelivery: boolean;
}) {
  return (
    <>
      <strong>
        {row.service ??
          t(l, "Unspecified service", "서비스 미지정", "خدمة غير محدّدة")}
      </strong>
      <small>
        <bdi dir="ltr">{row.provider}</bdi>
      </small>
      <dl className="infra-identity">
        <dt className={wide(row.billing_account)}>
          {fieldLabel(l, "billing_account")}
        </dt>
        <dd className={wide(row.billing_account)}>
          <bdi dir="ltr">{row.billing_account}</bdi>
        </dd>
        <dt className={wide(row.sub_account)}>
          {fieldLabel(l, "sub_account")}
        </dt>
        <dd className={wide(row.sub_account)} data-identity="sub_account">
          {row.sub_account === null ? (
            <span className="infra-state">{missingLabel(l)}</span>
          ) : (
            <bdi dir="ltr">{row.sub_account}</bdi>
          )}
        </dd>
        {showDelivery && (
          <>
            <dt className={wide(row.batch)}>{fieldLabel(l, "batch")}</dt>
            <dd className={wide(row.batch)} data-identity="batch">
              <bdi dir="ltr" className="infra-code">
                {row.batch}
              </bdi>
            </dd>
          </>
        )}
        <dt>{fieldLabel(l, "ordinal")}</dt>
        <dd data-identity="ordinal">
          <bdi dir="ltr">{row.ordinal}</bdi>
        </dd>
        <dt>{fieldLabel(l, "evidence_class")}</dt>
        <dd data-identity="provenance">
          {provenanceLabel(l, provenanceClass(row.evidence_class))}
        </dd>
        <dt className="infra-identity-wide">{fieldLabel(l, "id")}</dt>
        <dd className="infra-identity-wide" data-identity="id">
          <SplitId id={row.id} />
        </dd>
      </dl>
    </>
  );
}
export function ChargePeriod({
  locale: l,
  row,
}: {
  locale: Locale;
  row: FocusRow;
}) {
  const interpreted =
    instantParts(row.charge_start) && instantParts(row.charge_end);
  return (
    <>
      {[row.charge_start, row.charge_end].map((instant, index) => (
        <span
          key={index}
          className={instantParts(instant) ? "infra-token" : "infra-raw"}
        >
          <Instant value={instant} />
        </span>
      ))}
      {!interpreted && (
        <small>
          {t(
            l,
            "Time format not interpreted",
            "시각 형식을 해석하지 못했습니다",
            "تعذّر تفسير صيغة الوقت",
          )}
        </small>
      )}
    </>
  );
}
function Amount({
  locale: l,
  row,
  basis,
}: {
  locale: Locale;
  row: FocusRow;
  basis: Basis;
}) {
  const state = valueState(row[basis]);
  return (
    <div className="infra-cost-line" data-basis={basis} data-value={state}>
      <small>{basisLabel(l, basis)}</small>
      {state === "missing" ? (
        <span className="infra-state">{missingLabel(l)}</span>
      ) : (
        <bdi dir="ltr">{money(row[basis], row.currency, l)}</bdi>
      )}
      {state === "zero" && (
        <span className="infra-state infra-state-note">{zeroLabel(l)}</span>
      )}
    </div>
  );
}
export function CostLines({
  locale: l,
  row,
  basis,
  included,
}: {
  locale: Locale;
  row: FocusRow;
  basis: Basis;
  included: boolean;
}) {
  return (
    <>
      <Amount locale={l} row={row} basis={basis} />
      <Amount
        locale={l}
        row={row}
        basis={basis === "billed_cost" ? "effective_cost" : "billed_cost"}
      />
      {included && (
        <small className="infra-included">
          {t(l, "Declared included row", "포함 선언 행", "صف مُعلن مشمول")}
        </small>
      )}
    </>
  );
}

import type { Locale } from "../../../../shared/types";
import type { EvidenceRecord } from "./contract";
import { t, reported, recordLabel } from "./copy";
import { Facts, SyntheticNotice, SourceValue } from "./Shared";
import EvidencePanels from "./EvidencePanels";
export default function RecordCard({
  locale: l,
  record: r,
  close,
}: {
  locale: Locale;
  record: EvidenceRecord;
  close: () => void;
}) {
  const a = r.authority,
    historical =
      a.kind === "unknown" ||
      a.freshness !== "current" ||
      a.reference === null ||
      a.observed_at === null;
  return (
    <section
      className="b11-result"
      aria-label={recordLabel(l, r.kind)}
      onKeyDown={(e) => {
        if (
          e.key === "Escape" &&
          !(e.target as HTMLElement).closest("details[open]")
        ) {
          e.preventDefault();
          close();
        }
      }}
    >
      <SyntheticNotice locale={l} />
      <header>
        <div>
          <h2 tabIndex={-1} data-b11-detail-heading>
            <bdi dir="ltr">{r.id}</bdi>
          </h2>
          <p>
            <bdi dir="auto">{r.label}</bdi>
          </p>
          <p className="b11-meta">{recordLabel(l, r.kind)}</p>
          <p className="b11-meta">
            {t(l, "Source revision", "원천 개정", "مراجعة المصدر")}{" "}
            <SourceValue locale={l} value={r.revision} />
          </p>
        </div>
        <button onClick={close}>
          {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
        </button>
      </header>
      <section className="b11-authority">
        <h3>
          {t(
            l,
            "Source authority and freshness",
            "원장 기준·관측 시각·최신성",
            "مرجعية المصدر ووقت الرصد والحداثة",
          )}
        </h3>
        <p>{reported(l, a.kind)}</p>
        <p>{reported(l, a.freshness)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Authority reference", "원장 기준 참조", "مرجع المرجعية"),
              a.reference,
            ],
            [
              t(
                l,
                "Source observation time",
                "원천 관측 시각",
                "وقت رصد المصدر",
              ),
              a.observed_at,
            ],
          ]}
        />
        {historical && (
          <p className="b11-historical" role="note">
            {t(
              l,
              "Last source observation. Current available balance and payment confirmation are unconfirmed.",
              "마지막 원천 관측값입니다. 현재 사용 가능액·수량과 수납 확정은 미확인입니다.",
              "هذا آخر رصد من المصدر. الرصيد المتاح الحالي وتأكيد السداد غير مؤكدين.",
            )}
          </p>
        )}
        <p className="b11-meta">
          {t(
            l,
            "Source declarations are not current permission, usable-balance certification or reconciliation. Signs and precision are shown as supplied, with currencies and units kept separate.",
            "원천 기록은 현재 권한·사용 가능 잔액·대사의 인증이 아닙니다. 부호와 정밀도를 그대로 유지하고 통화·단위를 구분합니다.",
            "إقرارات المصدر ليست صلاحية حالية ولا تصديقاً على الرصيد المتاح أو المطابقة. تُحفظ الإشارات والدقة مع فصل العملات والوحدات.",
          )}
        </p>
      </section>
      <section className="b11-scope">
        <h3>{t(l, "Source scope", "원천 범위", "نطاق المصدر")}</h3>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Legal entity", "법인", "الكيان القانوني"),
              r.target.entity,
              "auto",
            ],
            [t(l, "Customer", "고객", "العميل"), r.target.customer, "auto"],
            [t(l, "Contract", "계약", "العقد"), r.target.contract, "auto"],
            [t(l, "Period", "기간", "الفترة"), r.target.period],
            [
              t(l, "Source updated time", "원천 갱신 시각", "وقت تحديث المصدر"),
              r.updated_at,
            ],
          ]}
        />
      </section>
      <EvidencePanels locale={l} record={r} />
      <details className="b11-evidence">
        <summary>
          {t(
            l,
            "Source evidence notes",
            "원천 근거 메모",
            "ملاحظات أدلة المصدر",
          )}
        </summary>
        {r.blockers.length ? (
          <ul>
            {r.blockers.map((b) => (
              <li key={b.code}>
                <bdi dir="auto">{b.text}</bdi>
              </li>
            ))}
          </ul>
        ) : (
          <p>
            {t(
              l,
              "No source blocker text supplied. This is not readiness or financial clearance.",
              "원천 차단 사유를 제공하지 않았습니다. 처리 가능·금융 승인 판정은 아닙니다.",
              "لم تُقدّم أسباب حجب من المصدر. هذا ليس إقراراً بالجاهزية أو الموافقة المالية.",
            )}
          </p>
        )}
      </details>
      <p className="b11-readonly">
        {t(
          l,
          "Read-only source evidence. No inquiry, reply, consent, resolution, credit, hold, due date, receipt or payment is created or changed here.",
          "원천 근거 조회 전용입니다. 문의·답변·동의·해결·크레딧·보류·지급 기한·수납·결제를 여기서 생성하거나 변경하지 않습니다.",
          "أدلة مصدر للقراءة فقط. لا تُنشأ أو تُغيّر هنا استفسارات أو ردود أو موافقات أو تسويات أو أرصدة أو إجراءات تعليق أو مواعيد أو إيصالات أو مدفوعات.",
        )}
      </p>
    </section>
  );
}

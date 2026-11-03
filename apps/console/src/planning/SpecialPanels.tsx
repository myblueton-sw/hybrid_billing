import { useState } from "react";
import type { Locale } from "../shared/types";
import { local } from "./copy";
import { field, valueText } from "./fields";
import CandidateTable, { type CandidateRow } from "./CandidateTable";
export function OrganizationTree({
  locale: l,
  onSelect,
}: {
  locale: Locale;
  onSelect: (id: string) => void;
}) {
  const [selected, setSelected] = useState(""),
    [expanded, setExpanded] = useState(true),
    [active, setActive] = useState("parent");
  const names = ["Research", "Operations", "Shared services"];
  return (
    <aside className="planning-tree">
      <h2>
        {local(l, [
          "조직·관계",
          "Organizations and relationships",
          "المؤسسات والعلاقات",
        ])}
      </h2>
      <div
        role="tree"
        aria-label={local(l, [
          "조직 소유 관계",
          "Organization ownership",
          "ملكية المؤسسة",
        ])}
        onKeyDown={(e) => {
          const elements = [
            ...e.currentTarget.querySelectorAll<HTMLElement>(
              '[role="treeitem"]',
            ),
          ];
          const at = elements.indexOf(document.activeElement as HTMLElement);
          let next = at;
          if (e.key === "ArrowDown")
            next = Math.min(elements.length - 1, at + 1);
          else if (e.key === "ArrowUp") next = Math.max(0, at - 1);
          else if (e.key === "ArrowRight") {
            setExpanded(true);
            next = Math.min(elements.length - 1, at + 1);
          } else if (e.key === "ArrowLeft") {
            if (at > 0) next = 0;
            else setExpanded(false);
          } else return;
          e.preventDefault();
          const el = elements[next];
          if (el) {
            setActive(el.dataset.node!);
            el.focus();
          }
        }}
      >
        <button
          role="treeitem"
          data-node="parent"
          tabIndex={active === "parent" ? 0 : -1}
          aria-expanded={expanded}
          onClick={() => {
            setActive("parent");
            setExpanded(!expanded);
          }}
        >
          DEMO Group
        </button>
        {expanded && (
          <div role="group">
            {names.map((name, i) => (
              <button
                role="treeitem"
                data-node={String(i + 1)}
                tabIndex={active === String(i + 1) ? 0 : -1}
                aria-selected={selected === String(i + 1)}
                key={name}
                onClick={() => {
                  setActive(String(i + 1));
                  setSelected(String(i + 1));
                  onSelect(String(i + 1));
                }}
              >
                DEMO {name}
              </button>
            ))}
          </div>
        )}
      </div>
      <p>
        {local(l, [
          "소유 관계 보기 · 권한 부여와 별개",
          "Ownership view · separate from access grants",
          "عرض الملكية · منفصل عن منح الوصول",
        ])}
      </p>
      <button
        className="planning-tree-reset"
        onClick={() => {
          setSelected("");
          onSelect("");
        }}
      >
        {local(l, ["전체 예제", "All examples", "كل الأمثلة"])}
      </button>
    </aside>
  );
}
export function PermissionMatrix({ locale: l }: { locale: Locale }) {
  return (
    <section className="planning-panel">
      <h2>
        {local(l, [
          "권한 효력 미리보기",
          "Effective-access preview",
          "معاينة الوصول الفعلي",
        ])}
      </h2>
      <p>
        {local(l, [
          "예제 역할은 권한을 부여하지 않습니다. 서버 정책과 부여 범위는 미설정입니다.",
          "Example roles grant nothing. Server policy and grant scope are unconfigured.",
          "الأدوار التجريبية لا تمنح صلاحيات. سياسة الخادم ونطاق المنح غير مهيأين.",
        ])}
      </p>
      <CandidateTable
        locale={l}
        columns={["permission", "scope", "grant", "sensitivity"]}
        rows={["view", "draft", "approve", "export"].map((k) => ({
          id: k,
          values: {
            permission: `preview.${k}`,
            scope: "DEMO Research",
            grant: "unconfigured",
            sensitivity: "unconfigured",
          },
        }))}
      />
    </section>
  );
}
export function DocumentPreview({
  locale: l,
  row,
}: {
  locale: Locale;
  row: CandidateRow;
}) {
  return (
    <article className="planning-document">
      <header>
        <div>
          <p>
            {local(l, [
              "예제 문서 · 다운로드 파일 아님",
              "Example document · not a downloadable file",
              "مستند تجريبي · ليس ملفاً للتنزيل",
            ])}
          </p>
          <h2>
            <bdi>{row.values.document ?? row.id}</bdi>
          </h2>
        </div>
        <span>{valueText(l, row.values.delivery ?? "pending")}</span>
      </header>
      <dl>
        {["kind", "issuer", "period", "correction"].map((k) => (
          <div key={k}>
            <dt>{field(l, k)}</dt>
            <dd>
              <bdi>{valueText(l, row.values[k] ?? "unconfigured")}</bdi>
            </dd>
          </div>
        ))}
      </dl>
      <CandidateTable
        locale={l}
        columns={["component", "quantity", "unit", "net", "tax", "gross"]}
        rows={[
          {
            id: row.id,
            values: {
              ...row.values,
              component: "DEMO service",
              quantity: "100",
              unit: "hour",
            },
          },
        ]}
      />
      <p>
        {local(l, [
          "발행·전달·납부 상태는 각각 별도입니다.",
          "Issuance, delivery and payment are separate states.",
          "الإصدار والتسليم والدفع حالات منفصلة.",
        ])}
      </p>
    </article>
  );
}
export function DisclosurePreview({
  locale: l,
  row,
}: {
  locale: Locale;
  row: CandidateRow;
}) {
  const [selected, setSelected] = useState(["document", "quantity", "amount"]);
  return (
    <section className="planning-split">
      <div className="planning-panel">
        <h2>
          {local(l, [
            "공개 필드 미리보기",
            "Public-field preview",
            "معاينة الحقول العامة",
          ])}
        </h2>
        {["document", "quantity", "amount"].map((k) => (
          <label className="planning-check" key={k}>
            <input
              type="checkbox"
              checked={selected.includes(k)}
              onChange={(e) =>
                setSelected(
                  e.target.checked
                    ? [...selected, k]
                    : selected.filter((x) => x !== k),
                )
              }
            />
            {field(l, k)}
          </label>
        ))}
        <p>
          {local(l, [
            "내부 원가·마진·자격 참조는 이 공개 예제에 포함되지 않습니다.",
            "Internal cost, margin and credentials are excluded from this public example.",
            "التكلفة الداخلية والهامش والاعتماد مستبعدة من المثال العام.",
          ])}
        </p>
      </div>
      <div className="planning-panel">
        <h2>
          {local(l, [
            "수신자에게 보이는 내용",
            "Recipient view",
            "عرض المستلم",
          ])}
        </h2>
        {selected.length ? (
          <CandidateTable
            locale={l}
            columns={selected}
            rows={[
              {
                id: "PUBLIC-01",
                values: {
                  document: row.id,
                  quantity: row.values.quantity ?? "unknown",
                  amount: row.values.amount ?? "unknown",
                },
              },
            ]}
          />
        ) : (
          <p>
            {local(l, [
              "선택한 공개 필드 없음",
              "No public fields selected",
              "لم تحدد حقول عامة",
            ])}
          </p>
        )}
        <p>
          {local(l, [
            "전체 검증·파일 생성·전달은 미연결입니다.",
            "Full validation, file generation and delivery are not connected.",
            "التحقق الكامل وإنشاء الملفات والتسليم غير متصلة.",
          ])}
        </p>
      </div>
    </section>
  );
}
export function ReportChart({
  locale: l,
  rows,
}: {
  locale: Locale;
  rows: CandidateRow[];
}) {
  const maximum = Math.max(
    ...rows.map((r) =>
      Number((r.values.amount ?? "0").replace(/^[A-Z]{3} /, "")),
    ),
    1,
  );
  return (
    <section className="planning-panel">
      <h2>
        {local(l, [
          "조회 결과 · 예제 금액",
          "Filtered results · example amounts",
          "نتائج التصفية · مبالغ تجريبية",
        ])}
      </h2>
      <div
        className="planning-chart"
        role="img"
        aria-label={rows.map((r) => `${r.id}: ${r.values.amount}`).join("; ")}
      >
        {rows.map((r) => (
          <div key={r.id}>
            <span>{r.values.amount}</span>
            <div
              className="planning-bar"
              style={{
                blockSize: Math.max(
                  1,
                  (Number(r.values.amount.replace(/^[A-Z]{3} /, "")) /
                    maximum) *
                    140,
                ),
              }}
            />
            <bdi>{r.id}</bdi>
          </div>
        ))}
      </div>
      <CandidateTable
        locale={l}
        columns={["id", "period", "meaning", "amount"]}
        rows={rows}
      />
    </section>
  );
}
export function LifecyclePanel({
  locale: l,
  kind,
}: {
  locale: Locale;
  kind: string;
}) {
  const steps =
    kind === "P04"
      ? [
          "Request",
          "Durable decision",
          "Access blocked",
          "Cleanup",
          "Active deletion verified",
          "Backup expiry",
          "Policy complete",
        ]
      : kind === "P03"
        ? [
            "Restore set",
            "Isolation",
            "External reconciliation",
            "Decision freshness",
            "Resume review",
          ]
        : [
            "Input revision",
            "Calculation candidate",
            "Independent approval",
            "Issuance",
            "Delivery",
          ];
  return (
    <section className="planning-panel">
      <h2>{local(l, ["단계별 근거", "Stage evidence", "أدلة المراحل"])}</h2>
      <ol className="planning-lifecycle">
        {steps.map((s, i) => (
          <li key={s}>
            <span>{i + 1}</span>
            <strong>{s}</strong>
            <small>{valueText(l, "unconnected")}</small>
          </li>
        ))}
      </ol>
      <p>
        {local(l, [
          "요청 접수와 완료는 다릅니다. 실제 실행 상태는 연결되지 않았습니다.",
          "Request receipt is not completion. Actual execution state is not connected.",
          "استلام الطلب ليس اكتماله. حالة التنفيذ الفعلية غير متصلة.",
        ])}
      </p>
    </section>
  );
}
export function Conversation({
  locale: l,
  onDirty,
}: {
  locale: Locale;
  onDirty: (dirty: boolean) => void;
}) {
  const [input, setInput] = useState(""),
    [messages, setMessages] = useState<string[]>([]);
  return (
    <section className="planning-conversation">
      <div className="planning-panel">
        <h2>
          {local(l, [
            "현재 범위에서 질문하기",
            "Ask within the current scope",
            "اسأل ضمن النطاق الحالي",
          ])}
        </h2>
        <p>
          {local(l, [
            "설치·법인·고객·계정·기간을 확인하세요. 모델과 실행 서비스는 미연결입니다.",
            "Check installation, legal entity, customer, account and period. Model and execution services are not connected.",
            "تحقق من التثبيت والكيان والعميل والحساب والفترة. النموذج وخدمات التنفيذ غير متصلة.",
          ])}
        </p>
        <div aria-live="polite">
          {messages.map((m, i) => (
            <article key={i}>
              <h3>{local(l, ["내 질문", "My request", "طلبي"])}</h3>
              <p>{m}</p>
              <p>
                {local(l, [
                  "요청은 이 화면에만 기록되었습니다. 답변·실행 계획·작업을 생성하지 않았습니다.",
                  "Recorded only in this view. No answer, execution plan or job was generated.",
                  "سجل الطلب في هذا العرض فقط. لم يتم إنشاء إجابة أو خطة أو مهمة.",
                ])}
              </p>
            </article>
          ))}
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (input.trim()) {
              setMessages([...messages, input]);
              setInput("");
              onDirty(false);
            }
          }}
        >
          <label>
            {local(l, ["요청 내용", "Request", "الطلب"])}
            <textarea
              required
              value={input}
              onChange={(e) => {
                setInput(e.target.value);
                onDirty(true);
              }}
            />
          </label>
          <button>
            {local(l, [
              "예제 대화에 추가",
              "Add to example conversation",
              "إضافة إلى المحادثة التجريبية",
            ])}
          </button>
        </form>
      </div>
      <aside className="planning-panel">
        <h2>
          {local(l, ["실행 계획·결과", "Plan and results", "الخطة والنتائج"])}
        </h2>
        <p>
          {local(l, [
            "연결된 실행 계획 없음",
            "No connected execution plan",
            "لا توجد خطة تنفيذ متصلة",
          ])}
        </p>
        <p>
          {local(l, [
            "승인과 업무 완료는 모델 응답과 별개입니다.",
            "Approval and business completion are independent of model responses.",
            "الموافقة وإكمال العمل مستقلان عن رد النموذج.",
          ])}
        </p>
      </aside>
    </section>
  );
}

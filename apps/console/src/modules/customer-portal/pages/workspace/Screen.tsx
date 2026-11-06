import { useEffect, useState } from "react";
import { Draft, Evidence, Ledger, Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, unknown, type Props, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const reason: Words = [
  "본인에게 공개된 자료를 조회하는 서비스가 연결되지 않았습니다. 다른 고객 자료를 검색하거나 금액을 추정하지 않습니다.",
  "The reader for information disclosed to you is not connected. No other customer's records are searched and no amounts are estimated.",
  "خدمة قراءة المعلومات المفصح عنها لك غير متصلة. لا تُبحث سجلات عملاء آخرين ولا تُقدّر أي مبالغ.",
];
function MonthScope({
  p,
  kind,
  setKind,
}: {
  p: Props;
  kind: string;
  setKind: (kind: string) => void;
}) {
  const l = p.locale;
  const [period, setPeriod] = useState("");
  useEffect(
    () =>
      p.onState(
        JSON.stringify({
          view: p.view,
          source: "unconnected",
          kind,
          periodSelected: !!period,
        }),
        [
          `${p.screenId}:field:period`,
          `${p.screenId}:field:settlement-kind`,
          `${p.screenId}:state:unconnected`,
        ],
      ),
    [kind, period, p.onState, p.screenId, p.view],
  );
  return (
    <section
      className="ops-scope"
      aria-label={t(l, [
        "내 정산 조회 범위",
        "My billing view scope",
        "نطاق عرض فوترتي",
      ])}
    >
      <label>
        {t(l, [
          "정산 기간 · 조회 요청 초안",
          "Period · view request draft",
          "الفترة · مسودة طلب عرض",
        ])}
        <input
          type="month"
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
        />
      </label>
      <fieldset className="ops-radio">
        <legend>{t(l, ["정산 종류", "Settlement type", "نوع التسوية"])}</legend>
        <label>
          <input
            type="radio"
            name="portal-kind"
            checked={kind === "billing"}
            onChange={() => setKind("billing")}
          />
          {t(l, ["판매 청구", "Sales billing", "فوترة المبيعات"])}
        </label>
        <label>
          <input
            type="radio"
            name="portal-kind"
            checked={kind === "allocation"}
            onChange={() => setKind("allocation")}
          />
          {t(l, ["내부 배분", "Internal allocation", "التوزيع الداخلي"])}
        </label>
      </fieldset>
      <p>
        {t(l, [
          "허용 조직·기간·통화 목록 미연결. 선택은 조회 초안이며 자료 접근을 허용하지 않습니다.",
          "Permitted organizations, periods and currencies are not connected. This is a view draft and grants no data access.",
          "المنظمات والفترات والعملات المسموحة غير متصلة. هذا مسودة عرض ولا يمنح وصولًا للبيانات.",
        ])}
      </p>
    </section>
  );
}
function Payment({ p, kind }: { p: Props; kind: string }) {
  if (kind === "allocation")
    return (
      <Panel
        title={t(p.locale, [
          "내부 배분 조회 초안",
          "Internal allocation view draft",
          "مسودة عرض التوزيع الداخلي",
        ])}
      >
        <p>
          {t(p.locale, [
            "이 조회 초안에서는 납부·연체 판정을 제공하지 않습니다. 실제 공개 문서와 적용 조건은 미연결입니다.",
            "This view draft provides no payment or overdue determination. Actual disclosed documents and applicable conditions are not connected.",
            "لا تقدم مسودة العرض هذه حكمًا بالسداد أو التأخر. المستندات المفصح عنها فعليًا والشروط المنطبقة غير متصلة.",
          ])}
        </p>
      </Panel>
    );
  return (
    <Evidence
      locale={p.locale}
      title={[
        "납부 기한·확인 상태",
        "Payment due date and confirmation",
        "موعد السداد وحالة التأكيد",
      ]}
      fields={[
        [
          "납부 기한·적용 문서",
          "Due date and applicable document",
          "موعد السداد والمستند المعني",
        ],
        [
          "외부 납부 확인·기준시각",
          "External payment confirmation and as of",
          "تأكيد السداد الخارجي والوقت المرجعي",
        ],
        [
          "미납 잔액·통화",
          "Outstanding balance and currency",
          "الرصيد المستحق والعملة",
        ],
        [
          "현재 권위·수납·배정 근거",
          "Current authority and collection/allocation evidence",
          "الجهة المرجعية الحالية وأدلة التحصيل والتخصيص",
        ],
      ]}
      reason={[
        "문서 다운로드는 납부 확인이 아닙니다. 현재 권위·날짜·수납·배정 근거 없이는 완납·연체로 판정하지 않습니다.",
        "Downloading a document is separate from payment confirmation. Paid or overdue requires current authority, date and collection/allocation evidence.",
        "تنزيل المستند منفصل عن تأكيد السداد. إثبات السداد أو التأخر يتطلب الجهة المرجعية الحالية والتاريخ وأدلة التحصيل والتخصيص.",
      ]}
    />
  );
}
function Overview(p: Props) {
  const l = p.locale;
  const [kind, setKind] = useState("billing");
  return (
    <>
      <MonthScope p={p} kind={kind} setKind={setKind} />
      {p.view === "payment-terms" ? (
        <Payment p={p} kind={kind} />
      ) : p.view === "usage" ? (
        <Ledger
          locale={l}
          title={[
            "내 허용 사용량",
            "My disclosed usage",
            "استخدامي المفصح عنه",
          ]}
          columns={[
            ["서비스", "Service", "الخدمة"],
            ["기간", "Period", "الفترة"],
            ["수량·단위", "Quantity / unit", "الكمية والوحدة"],
            ["측정 상태", "Measurement state", "حالة القياس"],
          ]}
          reason={reason}
        />
      ) : p.view === "notifications" ? (
        <Ledger
          locale={l}
          title={[
            "내 문서·문의 알림",
            "My document and inquiry notifications",
            "إشعارات مستنداتي واستفساراتي",
          ]}
          columns={[
            ["알림", "Notification", "الإشعار"],
            ["관련 문서·문의", "Document / inquiry", "المستند والاستفسار"],
            ["공개 시각", "Published at", "وقت النشر"],
          ]}
          reason={reason}
        />
      ) : (
        <>
          <Panel
            title={t(
              l,
              kind === "billing"
                ? [
                    "판매 청구 조회 초안",
                    "Sales billing view draft",
                    "مسودة عرض فوترة المبيعات",
                  ]
                : [
                    "내부 배분 조회 초안",
                    "Internal allocation view draft",
                    "مسودة عرض التوزيع الداخلي",
                  ],
            )}
          >
            <div className="ops-billing-summary">
              <div>
                <p>
                  {t(l, [
                    "공개된 정산 금액",
                    "Disclosed settlement amount",
                    "مبلغ التسوية المفصح عنه",
                  ])}
                </p>
                <strong>{t(l, unknown)}</strong>
                <p>
                  {t(l, [
                    "통화·확정 여부·기준시각 미연결",
                    "Currency, finality and as-of evidence not connected",
                    "العملة وحالة الاعتماد والوقت المرجعي غير متصلة",
                  ])}
                </p>
              </div>
              <dl className="ops-facts">
                <div>
                  <dt>{t(l, ["내 조직", "My organization", "منظمتي"])}</dt>
                  <dd>{t(l, unknown)}</dd>
                </div>
                <div>
                  <dt>
                    {t(l, [
                      "공개 문서 상태",
                      "Disclosed document state",
                      "حالة المستند المفصح عنه",
                    ])}
                  </dt>
                  <dd>{t(l, unknown)}</dd>
                </div>
              </dl>
            </div>
            <p>{t(l, reason)}</p>
          </Panel>
          <div className="ops-equal-grid">
            <Panel
              title={t(l, [
                "문서 확인",
                "Review documents",
                "مراجعة المستندات",
              ])}
            >
              <p>
                {t(l, [
                  "공개 문서와 정정 관계를 확인하세요.",
                  "Inspect disclosed documents and correction relationships.",
                  "راجع المستندات المفصح عنها وعلاقات التصحيح.",
                ])}
              </p>
              <a href={`#/portal/documents?lang=${l}`}>
                {t(l, [
                  "내 문서·사용량 열기",
                  "Open my documents and usage",
                  "فتح مستنداتي واستخدامي",
                ])}
              </a>
            </Panel>
            <Panel
              title={t(l, [
                "문의·답변",
                "Inquiries and replies",
                "الاستفسارات والردود",
              ])}
            >
              <p>
                {t(l, [
                  "문서의 쟁점을 준비하고 공개 답변을 확인하세요.",
                  "Prepare document questions and inspect public replies.",
                  "جهّز أسئلة المستندات وراجع الردود العامة.",
                ])}
              </p>
              <a href={`#/portal/cases?lang=${l}`}>
                {t(l, ["내 문의 열기", "Open my inquiries", "فتح استفساراتي"])}
              </a>
            </Panel>
          </div>
        </>
      )}
    </>
  );
}
function Documents(p: Props) {
  const l = p.locale;
  const [kind, setKind] = useState("billing");
  return (
    <>
      <MonthScope p={p} kind={kind} setKind={setKind} />
      {p.view === "payment-terms" ? (
        <Payment p={p} kind={kind} />
      ) : (
        <div className="ops-grid">
          <div>
            {p.view === "detail" ? (
              <Evidence
                locale={l}
                title={["문서 상세", "Document details", "تفاصيل المستند"]}
                fields={[
                  [
                    "공개 번호·문서 ID",
                    "Public number and document ID",
                    "الرقم العام ومعرّف المستند",
                  ],
                  [
                    "발행자·수신자",
                    "Issuer and recipient",
                    "جهة الإصدار والمستلم",
                  ],
                  [
                    "문서 금액·세금·통화",
                    "Document amount, tax and currency",
                    "مبلغ المستند والضريبة والعملة",
                  ],
                  [
                    "원문·정정 관계",
                    "Original and correction relationships",
                    "علاقات الأصل والتصحيح",
                  ],
                ]}
                reason={reason}
              />
            ) : p.view === "downloads" ? (
              <Evidence
                locale={l}
                title={[
                  "공개 파일 상태",
                  "Public file availability",
                  "توافر الملفات العامة",
                ]}
                fields={[
                  [
                    "공개 패키지·정책 revision",
                    "Public package and policy revision",
                    "الحزمة العامة ومراجعة السياسة",
                  ],
                  ["제공 형식", "Available formats", "الصيغ المتاحة"],
                  [
                    "준비·복원·만료·권한 상태",
                    "Ready, restore, expiry and access state",
                    "حالات الجاهزية والاستعادة والانتهاء والوصول",
                  ],
                ]}
                reason={[
                  "현재 권한과 공개 패키지 확인 전에는 다운로드를 만들지 않습니다.",
                  "Downloads require current authority and a disclosed package.",
                  "يتطلب التنزيل صلاحية حالية وحزمة مفصحًا عنها.",
                ]}
              />
            ) : (
              <Ledger
                locale={l}
                title={
                  p.view === "usage"
                    ? [
                        "허용 사용량 명세",
                        "Disclosed usage detail",
                        "تفاصيل الاستخدام المفصح عنه",
                      ]
                    : p.view === "statement"
                      ? [
                          "판매·배분 명세",
                          "Sales and allocation statement",
                          "كشف المبيعات والتوزيع",
                        ]
                      : [
                          "내 공개 문서",
                          "My disclosed documents",
                          "مستنداتي المفصح عنها",
                        ]
                }
                columns={
                  p.view === "documents"
                    ? [
                        ["문서", "Document", "المستند"],
                        ["종류", "Type", "النوع"],
                        ["기간", "Period", "الفترة"],
                        ["금액·통화", "Amount / currency", "المبلغ والعملة"],
                        ["공개 상태", "Disclosure state", "حالة الإفصاح"],
                      ]
                    : [
                        [
                          "서비스·행 참조",
                          "Service / row reference",
                          "الخدمة ومرجع الصف",
                        ],
                        ["기간", "Period", "الفترة"],
                        ["수량·단위", "Quantity / unit", "الكمية والوحدة"],
                        [
                          "공개 금액·통화",
                          "Disclosed amount / currency",
                          "المبلغ المفصح عنه والعملة",
                        ],
                      ]
                }
                reason={reason}
              />
            )}
            <Draft
              props={p}
              title={[
                "문서 조회 조건 검토",
                "Review document search criteria",
                "مراجعة معايير البحث عن المستندات",
              ]}
              fields={[
                {
                  id: "document",
                  label: [
                    "내 공개 문서 참조",
                    "My disclosed document reference",
                    "مرجع مستندي المفصح عنه",
                  ],
                },
                {
                  id: "period",
                  label: ["정산 기간", "Settlement period", "فترة التسوية"],
                },
                {
                  id: "type",
                  label: ["문서 종류", "Document type", "نوع المستند"],
                  options: [
                    ["판매 청구", "Sales invoice", "فاتورة مبيعات"],
                    ["내부 배분", "Internal allocation", "توزيع داخلي"],
                    ["크레딧·정정", "Credit or correction", "رصيد أو تصحيح"],
                  ],
                },
              ]}
            />
          </div>
          <aside>
            <Panel
              title={t(l, [
                "문서와 정정",
                "Documents and corrections",
                "المستندات والتصحيحات",
              ])}
            >
              <p>
                {t(l, [
                  "정정 문서는 원문을 삭제하지 않습니다. 공개된 원문·정정 참조와 각 관계의 증감을 함께 확인합니다.",
                  "A correction retains the original. Inspect disclosed original/correction references and the change attached to each relationship.",
                  "يحافظ التصحيح على الأصل. راجع مراجع الأصل والتصحيح المفصح عنها والتغيير المرتبط بكل علاقة.",
                ])}
              </p>
              <a href={`#/portal/cases?lang=${l}&view=compose`}>
                {t(l, [
                  "문서 문의 준비",
                  "Prepare a document inquiry",
                  "إعداد استفسار عن مستند",
                ])}
              </a>
            </Panel>
          </aside>
        </div>
      )}
    </>
  );
}
function Cases(p: Props) {
  const l = p.locale;
  return (
    <div className="ops-grid">
      <div>
        {p.view === "compose" ? (
          <>
            <Panel
              title={t(l, [
                "문의 작성 전 확인",
                "Before preparing an inquiry",
                "قبل إعداد الاستفسار",
              ])}
            >
              <p>
                {t(l, [
                  "접수는 청구 취소·환불·납부 유예가 아닙니다. 자격증명과 결제 비밀을 입력하지 마세요. 접수 경로는 미연결입니다.",
                  "An inquiry does not cancel billing, refund payment or defer a due date. Do not enter credentials or payment secrets. Submission is not connected.",
                  "لا يلغي الاستفسار الفوترة ولا يسترد المبلغ ولا يؤجّل موعد السداد. لا تدخل بيانات اعتماد أو أسرار دفع. الإرسال غير متصل.",
                ])}
              </p>
            </Panel>
            <Draft
              props={p}
              title={["문의 초안", "Inquiry draft", "مسودة استفسار"]}
              fields={[
                {
                  id: "document",
                  label: [
                    "본인 공개 문서 참조",
                    "Your disclosed document reference",
                    "مرجع مستندك المفصح عنه",
                  ],
                  required: true,
                },
                {
                  id: "row",
                  label: [
                    "쟁점 행 참조 · 선택",
                    "Disputed row reference · optional",
                    "مرجع الصف محل الاستفسار · اختياري",
                  ],
                },
                {
                  id: "type",
                  label: ["문의 유형", "Inquiry type", "نوع الاستفسار"],
                  required: true,
                  options: [
                    ["금액", "Amount", "المبلغ"],
                    ["사용량", "Usage", "الاستخدام"],
                    ["문서 정보", "Document information", "معلومات المستند"],
                    ["다운로드", "Download", "التنزيل"],
                  ],
                },
                {
                  id: "subject",
                  label: ["제목", "Subject", "الموضوع"],
                  required: true,
                },
                {
                  id: "body",
                  label: ["문의 내용", "Inquiry details", "تفاصيل الاستفسار"],
                  kind: "textarea",
                  required: true,
                },
                {
                  id: "requested",
                  label: [
                    "회신 요청 기한 · 약속 아님",
                    "Requested reply date · no commitment",
                    "تاريخ الرد المطلوب · ليس التزامًا",
                  ],
                  kind: "date",
                },
              ]}
            />
          </>
        ) : p.view === "conversation" ? (
          <Panel
            title={t(l, [
              "공개 답변·자료",
              "Public replies and materials",
              "الردود والمواد العامة",
            ])}
          >
            <div className="ops-conversation">
              <h3>
                {t(l, [
                  "공개 대화 미연결",
                  "Public conversation not connected",
                  "المحادثة العامة غير متصلة",
                ])}
              </h3>
              <p>
                {t(l, [
                  "문의 참조와 본인 열람 권한을 확인한 후 공개 답변을 표시합니다. 작성 역할·시각·첨부 검사 상태를 함께 제공합니다.",
                  "Public replies require an inquiry reference and your current access. Author role, time and attachment inspection state accompany each reply.",
                  "تتطلب الردود العامة مرجع استفسار وصلاحية وصولك الحالية. يرافق كل رد دور الكاتب والوقت وحالة فحص المرفقات.",
                ])}
              </p>
            </div>
          </Panel>
        ) : p.view === "resolution" ? (
          <Evidence
            locale={l}
            title={[
              "담당 해결·고객 확인",
              "Agent resolution and customer confirmation",
              "حل المسؤول وتأكيد العميل",
            ]}
            fields={[
              [
                "담당 해결 상태·근거",
                "Agent resolution and evidence",
                "حالة حل المسؤول والأدلة",
              ],
              ["고객 확인", "Customer confirmation", "تأكيد العميل"],
              [
                "관련 공개 정정 문서",
                "Related disclosed correction documents",
                "مستندات التصحيح المفصح عنها ذات الصلة",
              ],
            ]}
            reason={[
              "해결과 고객 확인은 환불·정산·납부 완료를 뜻하지 않습니다.",
              "Resolution and customer confirmation do not establish refund, settlement or payment completion.",
              "الحل وتأكيد العميل لا يثبتان اكتمال الاسترداد أو التسوية أو السداد.",
            ]}
          />
        ) : (
          <Ledger
            locale={l}
            title={["내 문의", "My inquiries", "استفساراتي"]}
            columns={[
              ["문의", "Inquiry", "الاستفسار"],
              ["관련 문서", "Related document", "المستند المعني"],
              ["제목·상태", "Subject / state", "الموضوع والحالة"],
              ["최근 공개 답변", "Latest public reply", "آخر رد عام"],
              [
                "다음 요청·기한",
                "Next request / date",
                "الطلب التالي والتاريخ",
              ],
            ]}
            reason={reason}
          />
        )}
      </div>
      <aside>
        <Panel title={t(l, ["문의 흐름", "Inquiry flow", "مسار الاستفسار"])}>
          <ol>
            <li>
              {t(l, [
                "공개 문서와 쟁점 행 확인",
                "Identify disclosed document and row",
                "تحديد المستند المفصح عنه والصف",
              ])}
            </li>
            <li>
              {t(l, [
                "내용 검토 후 접수",
                "Review content, then submit",
                "مراجعة المحتوى ثم الإرسال",
              ])}
            </li>
            <li>
              {t(l, [
                "공개 답변·추가 자료 확인",
                "Inspect public replies and materials",
                "مراجعة الردود والمواد العامة",
              ])}
            </li>
            <li>
              {t(l, [
                "해결 근거와 정정 문서 확인",
                "Inspect resolution and correction documents",
                "مراجعة الحل ومستندات التصحيح",
              ])}
            </li>
          </ol>
          <a href={`#/portal/cases?lang=${l}&view=compose`}>
            {t(l, [
              "문의 초안 작성",
              "Prepare an inquiry draft",
              "إعداد مسودة استفسار",
            ])}
          </a>
        </Panel>
      </aside>
    </div>
  );
}
export default function Portal(p: Props) {
  useEffect(
    () =>
      p.onState(
        JSON.stringify({
          screen: p.screenId,
          view: p.view,
          source: "unconnected",
          audience: "customer",
        }),
        [`${p.screenId}:state:unconnected`, `${p.screenId}:data:${p.view}`],
      ),
    [p.screenId, p.view, p.onState],
  );
  return p.screenId === "C01" ? (
    <Overview {...p} />
  ) : p.screenId === "C02" ? (
    <Documents {...p} />
  ) : (
    <Cases {...p} />
  );
}

import { useState } from "react";
import { text, type Props } from "../../../../shared/domain-screen-ui/work/contracts";
import {
  Boundary,
  EmptyList,
  Facts,
  Label,
  Panel,
  ReviewForm,
  refInput,
  useReview,
} from "../../../../shared/domain-screen-ui/work/ui";
export function Organizations(p: Props) {
  const l = p.locale;
  const [asOf, setAsOf] = useState("");
  const [relation, setRelation] = useState("");
  const [name, setName] = useState("");
  const [members, setMembers] = useState("");
  const [entity, setEntity] = useState("");
  const [provider, setProvider] = useState("");
  const dirty = !!(asOf || relation || name || members || entity || provider);
  useReview(p, dirty);
  return (
    <>
      <Boundary l={l} />
      <div className="work-columns organization-columns">
        <Panel
          title={text(l, [
            "기준일과 관계",
            "As-of date and relationships",
            "التاريخ المرجعي والعلاقات",
          ])}
          id="O01:scope:filters"
        >
          <ReviewForm
            l={l}
            id="O01:action:review-scope"
            onClear={() => {
              setAsOf("");
              setRelation("");
              setEntity("");
              setProvider("");
              setName("");
              setMembers("");
            }}
          >
            <Label label={text(l, ["기준일", "As-of date", "التاريخ المرجعي"])}>
              <input
                required
                type="date"
                value={asOf}
                onChange={(e) => setAsOf(e.target.value)}
              />
            </Label>
            <Label
              label={text(l, [
                "법인 참조",
                "Legal entity reference",
                "مرجع الكيان القانوني",
              ])}
            >
              <input
                {...refInput}
                value={entity}
                onChange={(e) => setEntity(e.target.value)}
              />
            </Label>
            <Label
              label={text(l, ["관계 종류", "Relationship kind", "نوع العلاقة"])}
            >
              <select
                value={relation}
                onChange={(e) => setRelation(e.target.value)}
              >
                <option value="">
                  {text(l, ["전체 관계", "All relationships", "جميع العلاقات"])}
                </option>
                {[
                  ["ownership", "소유", "Ownership", "الملكية"],
                  ["view", "조회", "View", "العرض"],
                  ["allocation", "배분", "Allocation", "التوزيع"],
                  ["permission", "권한", "Permission", "الصلاحية"],
                ].map(([v, ko, en, ar]) => (
                  <option key={v} value={v}>
                    {text(l, [ko, en, ar])}
                  </option>
                ))}
              </select>
            </Label>
            <Label
              label={text(l, [
                "공급자 참조",
                "Provider reference",
                "مرجع المزود",
              ])}
            >
              <input
                {...refInput}
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
              />
            </Label>
          </ReviewForm>
          <p className="work-empty">
            {text(l, [
              "허용 조직 트리를 아직 받지 못했습니다. 비공개 하위 조직 수를 추정하지 않습니다.",
              "The permitted organization tree has not been received. Private child counts are not inferred.",
              "لم يتم استلام شجرة المؤسسات المسموح بها. لا يتم استنتاج أعداد الفروع الخاصة.",
            ])}
          </p>
        </Panel>
        <Panel
          title={text(
            l,
            p.view === "accounts"
              ? [
                  "원천 계정 관계",
                  "Source account relationships",
                  "علاقات حسابات المصدر",
                ]
              : p.view === "groups"
                ? ["조회 그룹 초안", "View group draft", "مسودة مجموعة العرض"]
                : p.view === "history"
                  ? [
                      "유효기간별 관계 이력",
                      "Relationship history by validity period",
                      "سجل العلاقات حسب فترة الصلاحية",
                    ]
                  : [
                      "조직과 귀속 관계",
                      "Organizations and attribution",
                      "المؤسسات والإسناد",
                    ],
          )}
          id={`O01:data:${p.view}`}
        >
          {p.view === "groups" ? (
            <ReviewForm
              l={l}
              id="O01:action:review-group"
              onClear={() => {
                setName("");
                setMembers("");
              }}
            >
              <Label
                label={text(l, ["그룹 이름", "Group name", "اسم المجموعة"])}
              >
                <input
                  required
                  maxLength={128}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </Label>
              <Label
                label={text(l, [
                  "구성원 참조",
                  "Member references",
                  "مراجع الأعضاء",
                ])}
                hint={text(l, [
                  "한 줄에 canonical ID 하나. 조회 그룹은 소유권이나 접근 권한을 부여하지 않습니다.",
                  "One canonical ID per line. A view group grants neither ownership nor access.",
                  "معرف أساسي واحد لكل سطر. مجموعة العرض لا تمنح ملكية أو صلاحية وصول.",
                ])}
              >
                <textarea
                  required
                  rows={5}
                  maxLength={2048}
                  value={members}
                  onChange={(e) => {
                    const v = e.target.value;
                    setMembers(v);
                    const ids = v.split(/\n/).map((x) => x.trim());
                    e.target.setCustomValidity(
                      ids.some(
                        (x) => !/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(x),
                      ) || new Set(ids).size !== ids.length
                        ? text(l, [
                            "중복 없는 유효 ID를 입력하세요.",
                            "Enter valid, unique IDs.",
                            "أدخل معرفات صالحة وفريدة.",
                          ])
                        : "",
                    );
                  }}
                />
              </Label>
            </ReviewForm>
          ) : (
            <EmptyList
              l={l}
              columns={
                p.view === "accounts"
                  ? [
                      ["계정", "Account", "الحساب"],
                      ["원천 범위", "Source scope", "نطاق المصدر"],
                      ["계정 역할", "Account role", "دور الحساب"],
                      ["관계", "Relationship", "العلاقة"],
                    ]
                  : [
                      ["조직", "Organization", "المؤسسة"],
                      ["상위 조직", "Parent", "الأصل"],
                      ["관계", "Relationship", "العلاقة"],
                      ["유효기간", "Validity interval", "فترة الصلاحية"],
                    ]
              }
              reason={[
                "조직·관계 조회 서비스가 연결되면 허용된 페이지를 표시합니다.",
                "Permitted pages will appear when the organization reader is connected.",
                "ستظهر الصفحات المسموح بها عند اتصال خدمة قراءة المؤسسات.",
              ]}
            />
          )}
          <Facts
            l={l}
            labels={[
              ["현재 revision", "Current revision", "المراجعة الحالية"],
              ["법인 참조", "Legal entity reference", "مرجع الكيان القانوني"],
              ["미귀속 사유", "Unattributed reason", "سبب عدم الإسناد"],
            ]}
          />
        </Panel>
      </div>
    </>
  );
}
export function OrganizationChange(p: Props) {
  const l = p.locale;
  const [kind, setKind] = useState("");
  const [target, setTarget] = useState("");
  const [date, setDate] = useState("");
  const [revision, setRevision] = useState("");
  const [checked, setChecked] = useState(false);
  useReview(
    p,
    !!(kind || target || date || revision),
    checked ? "local-input-reviewed" : "unconnected",
  );
  const steps = [
    ["targets", "변경 대상", "Targets", "الأهداف"],
    ["impact", "영향 검토", "Impact review", "مراجعة الأثر"],
    ["approval", "승인·예약", "Approval and schedule", "الموافقة والجدولة"],
    ["result", "적용 결과", "Application result", "نتيجة التطبيق"],
  ];
  return (
    <>
      <Boundary l={l} />
      <div className="work-columns change-columns">
        <aside className="work-step-list">
          <h2>{text(l, ["변경 계획", "Change plan", "خطة التغيير"])}</h2>
          <ol>
            {steps.map(([v, ko, en, ar]) => (
              <li key={v} aria-current={p.view === v ? "step" : undefined}>
                <strong>{text(l, [ko, en, ar])}</strong>
                <small>
                  {text(
                    l,
                    v === "targets" && checked
                      ? [
                          "입력 형식 검토됨",
                          "Input format reviewed",
                          "تمت مراجعة التنسيق",
                        ]
                      : [
                          "서버 근거 미연결",
                          "Server evidence unconnected",
                          "أدلة الخادم غير متصلة",
                        ],
                  )}
                </small>
              </li>
            ))}
          </ol>
        </aside>
        <Panel
          title={text(
            l,
            p.view === "targets"
              ? ["변경 범위 초안", "Change scope draft", "مسودة نطاق التغيير"]
              : p.view === "impact"
                ? [
                    "변경 전후 영향",
                    "Before and after impact",
                    "الأثر قبل التغيير وبعده",
                  ]
                : p.view === "approval"
                  ? [
                      "승인과 적용 전제",
                      "Approval and application prerequisites",
                      "متطلبات الموافقة والتطبيق",
                    ]
                  : [
                      "의존 단위별 결과",
                      "Results by dependency unit",
                      "النتائج حسب وحدة الاعتماد",
                    ],
          )}
        >
          {p.view === "targets" ? (
            <ReviewForm
              l={l}
              id="O02:action:review-target"
              onReviewed={() => setChecked(true)}
              onClear={() => {
                setKind("");
                setTarget("");
                setDate("");
                setRevision("");
                setChecked(false);
              }}
            >
              <div className="work-form-grid">
                <Label
                  label={text(l, ["변경 종류", "Change kind", "نوع التغيير"])}
                >
                  <select
                    required
                    value={kind}
                    onChange={(e) => {
                      setKind(e.target.value);
                      setChecked(false);
                    }}
                  >
                    <option value="">
                      {text(l, ["선택하세요", "Choose", "اختر"])}
                    </option>
                    {[
                      ["link", "연결", "Link", "ربط"],
                      ["unlink", "해제", "Unlink", "إلغاء الربط"],
                      ["move", "이동", "Move", "نقل"],
                      ["group", "조회 그룹", "View group", "مجموعة عرض"],
                    ].map(([v, ko, en, ar]) => (
                      <option key={v} value={v}>
                        {text(l, [ko, en, ar])}
                      </option>
                    ))}
                  </select>
                </Label>
                <Label
                  label={text(l, [
                    "대상 manifest 참조",
                    "Target manifest reference",
                    "مرجع بيان الأهداف",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={target}
                    onChange={(e) => {
                      setTarget(e.target.value);
                      setChecked(false);
                    }}
                  />
                </Label>
                <Label
                  label={text(l, [
                    "기대 revision 참조",
                    "Expected revision reference",
                    "مرجع المراجعة المتوقعة",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={revision}
                    onChange={(e) => {
                      setRevision(e.target.value);
                      setChecked(false);
                    }}
                  />
                </Label>
                <Label
                  label={text(l, [
                    "적용일시 초안",
                    "Effective time draft",
                    "مسودة وقت السريان",
                  ])}
                >
                  <input
                    type="datetime-local"
                    required
                    value={date}
                    onChange={(e) => {
                      setDate(e.target.value);
                      setChecked(false);
                    }}
                  />
                </Label>
              </div>
              <p>
                {text(l, [
                  "입력한 참조의 존재·권한·서버 revision은 아직 확인되지 않았습니다. 적용 시간대는 계약 근거가 필요합니다.",
                  "Existence, access and server revision of entered references are unverified. The effective time needs the contract timezone.",
                  "وجود المراجع وصلاحياتها ومراجعتها على الخادم غير متحقق. وقت السريان يتطلب المنطقة الزمنية للعقد.",
                ])}
              </p>
            </ReviewForm>
          ) : p.view === "impact" ? (
            <>
              <div className="work-comparison">
                <section>
                  <h3>{text(l, ["변경 전", "Before", "قبل"])}</h3>
                  <Facts
                    l={l}
                    labels={[
                      [
                        "관계·권한",
                        "Relationships and access",
                        "العلاقات والوصول",
                      ],
                      [
                        "원가·판매·배분",
                        "Cost, sales and allocation",
                        "التكلفة والمبيعات والتوزيع",
                      ],
                      [
                        "ERP·공개 파일",
                        "ERP and public files",
                        "ERP والملفات العامة",
                      ],
                    ]}
                  />
                </section>
                <section>
                  <h3>{text(l, ["변경 후", "After", "بعد"])}</h3>
                  <Facts
                    l={l}
                    labels={[
                      [
                        "관계·권한",
                        "Relationships and access",
                        "العلاقات والوصول",
                      ],
                      [
                        "원가·판매·배분",
                        "Cost, sales and allocation",
                        "التكلفة والمبيعات والتوزيع",
                      ],
                      [
                        "ERP·공개 파일",
                        "ERP and public files",
                        "ERP والملفات العامة",
                      ],
                    ]}
                  />
                </section>
              </div>
              <p>
                {text(l, [
                  "영향 계산 서비스 미연결. 공유 의존 범위를 검사하기 전에는 승인·적용할 수 없습니다.",
                  "Impact calculation is unconnected. Approval and application require checks across shared dependencies.",
                  "حساب الأثر غير متصل. الموافقة والتطبيق يتطلبان فحص الاعتمادات المشتركة.",
                ])}
              </p>
            </>
          ) : p.view === "approval" ? (
            <Facts
              l={l}
              labels={[
                [
                  "승인 대상 hash",
                  "Approved target hash",
                  "بصمة الهدف المعتمد",
                ],
                [
                  "현재 실행 권한",
                  "Current execution authority",
                  "صلاحية التنفيذ الحالية",
                ],
                [
                  "예약 시각·시간대",
                  "Scheduled time and timezone",
                  "وقت الجدولة والمنطقة الزمنية",
                ],
                [
                  "의존 범위 검증",
                  "Dependency scope validation",
                  "التحقق من نطاق الاعتماد",
                ],
              ]}
            />
          ) : (
            <>
              <Facts
                l={l}
                labels={[
                  ["성공 단위", "Succeeded units", "الوحدات الناجحة"],
                  ["보류 단위", "Held units", "الوحدات المعلقة"],
                  ["실패 단위", "Failed units", "الوحدات الفاشلة"],
                  ["미처리 단위", "Unprocessed units", "الوحدات غير المعالجة"],
                ]}
              />
              <p>
                {text(l, [
                  "접수된 변경 작업이 확인되지 않았습니다. 완료나 부분 성공을 추정하지 않습니다.",
                  "No accepted change job is available. Completion or partial success is not inferred.",
                  "لا تتوفر مهمة تغيير مقبولة. لا يتم استنتاج اكتمال أو نجاح جزئي.",
                ])}
              </p>
            </>
          )}
        </Panel>
      </div>
    </>
  );
}

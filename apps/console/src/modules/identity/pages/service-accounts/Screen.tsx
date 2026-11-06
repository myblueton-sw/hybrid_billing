import type { Props } from "../../screen-ui/types";
import {
  Field,
  LocalForm,
  Note,
  Observations,
  Unconnected,
} from "../../screen-ui/primitives";
import { reference, required } from "../../screen-ui/model";
export default function ServiceAccounts(props: Props) {
  const { locale: l, view } = props;
  return (
    <div className="security-screen">
      <Unconnected
        l={l}
        words={[
          "Service identities and credential metadata are not loaded. No account, key, rotation, revocation or ownership transfer is created.",
          "서비스 신원·자격 메타데이터를 불러오지 않았습니다. 계정·키 발급·회전·폐기·책임자 이관을 수행하지 않습니다.",
          "لم تُحمّل هويات الخدمة أو بيانات الاعتماد الوصفية. لا يُنشأ حساب أو مفتاح ولا يُنفذ تدوير أو إلغاء أو نقل مسؤولية.",
        ]}
      />
      <div className="security-workbench">
        {view === "accounts" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare service-account purpose",
              "서비스 계정 목적 준비",
              "إعداد غرض حساب الخدمة",
            ]}
            rules={{
              purpose: required,
              organizationRef: reference,
              ownerRef: reference,
              audience: required,
              scope: { required: true, kind: "service-scope" },
            }}
            items={["data:human-service-boundary"]}
          >
            <Field name="purpose" l={l} />
            <Field name="organizationRef" l={l} />
            <Field name="ownerRef" l={l} />
            <Field name="audience" l={l} />
            <Field name="scope" l={l} />
          </LocalForm>
        ) : view === "credentials" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare metadata inspection",
              "자격 메타데이터 검토 준비",
              "إعداد فحص البيانات الوصفية",
            ]}
            rules={{
              accountRef: reference,
              credentialRef: reference,
              version: required,
            }}
            items={["data:credential-metadata"]}
          >
            <Field name="accountRef" l={l} />
            <Field name="credentialRef" l={l} />
            <Field name="version" l={l} />
          </LocalForm>
        ) : view === "usage" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare use-history scope",
              "최근 사용 조회 조건 준비",
              "إعداد نطاق سجل الاستخدام",
            ]}
            rules={{
              accountRef: reference,
              starts: { required: true, kind: "date" },
              expires: { required: true, kind: "date" },
            }}
            items={["data:use-history"]}
          >
            <Field name="accountRef" l={l} />
            <Field name="starts" l={l} type="date" />
            <Field name="expires" l={l} type="date" />
          </LocalForm>
        ) : (
          <LocalForm
            props={props}
            title={[
              "Prepare accountable-owner handoff",
              "관리 책임자 이관 준비",
              "إعداد تسليم المسؤولية",
            ]}
            rules={{
              accountRef: reference,
              ownerRef: reference,
              reason: required,
            }}
            items={["data:owner-handoff"]}
          >
            <Field name="accountRef" l={l} />
            <Field name="ownerRef" l={l} />
            <Field name="reason" l={l} />
          </LocalForm>
        )}
        <Observations
          l={l}
          title={
            view === "credentials"
              ? [
                  "Reference, version and lifecycle",
                  "참조·버전·수명 주기",
                  "المرجع والإصدار ودورة الحياة",
                ]
              : view === "transfer"
                ? [
                    "Current responsibility evidence",
                    "현재 책임 근거",
                    "أدلة المسؤولية الحالية",
                  ]
                : [
                    "Service execution evidence",
                    "서비스 실행 근거",
                    "أدلة تنفيذ الخدمة",
                  ]
          }
          fields={
            view === "credentials"
              ? [
                  [
                    "Credential reference and version",
                    "자격 참조·버전",
                    "مرجع بيانات الاعتماد وإصدارها",
                  ],
                  [
                    "Expiry and rotation state",
                    "만료·회전 상태",
                    "حالة الانتهاء والتدوير",
                  ],
                  [
                    "Revocation effect and dependent work",
                    "폐기 효력·진행 작업 영향",
                    "أثر الإلغاء والعمل المرتبط",
                  ],
                ]
              : view === "transfer"
                ? [
                    [
                      "Current accountable owner",
                      "현재 관리 책임자",
                      "المسؤول الحالي",
                    ],
                    [
                      "Eligible replacement and approval",
                      "적격 새 책임자·승인",
                      "البديل المؤهل والموافقة",
                    ],
                    ["Handoff result", "이관 결과", "نتيجة التسليم"],
                  ]
                : [
                    [
                      "Stable service account ID",
                      "안정 서비스 계정 ID",
                      "معرف حساب الخدمة الثابت",
                    ],
                    [
                      "Purpose, audience and permitted scope",
                      "목적·audience·허용 범위",
                      "الغرض والجمهور والنطاق المسموح",
                    ],
                    [
                      "Last use and actual call scope",
                      "최근 사용·실제 호출 범위",
                      "آخر استخدام ونطاق الاستدعاء الفعلي",
                    ],
                    [
                      "User delegation intersection",
                      "사용자 위임 교집합",
                      "تقاطع تفويض المستخدم",
                    ],
                  ]
          }
        />
      </div>
      <Note
        l={l}
        words={[
          "Service execution is separate from human approval. A local scope check rejects explicit approval labels; all proposed scopes still require server evaluation. Only references and versions belong here, never secret values.",
          "서비스 실행과 사람 승인은 별개입니다. 명시적인 승인 범위 표기는 로컬에서 거절하며 모든 제안 범위는 서버 평가가 필요합니다. 자격 원문이 아닌 참조·버전만 입력하세요.",
          "تنفيذ الخدمة منفصل عن الموافقة البشرية. يرفض الفحص المحلي تسميات الموافقة الصريحة؛ وكل نطاق مقترح يحتاج تقييم الخادم. أدخل المراجع والإصدارات فقط، لا القيم السرية.",
        ]}
      />
    </div>
  );
}

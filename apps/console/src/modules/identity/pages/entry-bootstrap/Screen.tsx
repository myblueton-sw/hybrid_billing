import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  useEntryState,
  Email,
  Reference,
  Result,
  PreviewNext,
} from "../../entry-ui";
export default function Screen(props: ScreenProps) {
  const { locale: l, mode } = props,
    state = useEntryState(props, "entry-bootstrap", [
      "field:proof",
      "field:email",
      "action:preview",
      "navigate:entry-scope",
    ]);
  return (
    <div className="entry-form-layout">
      <section className="entry-card">
        <h1>
          {t(
            l,
            "Register the first administrator",
            "최초 관리자 등록",
            "تسجيل المسؤول الأول",
          )}
        </h1>
        <p>
          {t(
            l,
            "Prepare the first administrator for this installation.",
            "이 설치 환경을 관리할 최초 관리자 정보를 준비합니다.",
            "جهّز بيانات المسؤول الأول لهذا التثبيت.",
          )}
        </p>
        <form onSubmit={state.submit} onChange={state.changed}>
          <Reference locale={l} />
          <Email locale={l} />
          <button className="primary" type="submit">
            {t(
              l,
              "Preview registration request",
              "등록 요청 미리보기",
              "معاينة طلب التسجيل",
            )}
          </button>
        </form>
        {state.result && <Result locale={l} />}
      </section>
      <aside className="entry-help">
        <h2>
          {t(
            l,
            "Installation verification",
            "설치 환경 확인",
            "التحقق من التثبيت",
          )}
        </h2>
        <dl>
          <dt>
            {t(
              l,
              "First administrator eligibility",
              "최초 관리자 등록 가능 여부",
              "إمكانية تسجيل المسؤول الأول",
            )}
          </dt>
          <dd>
            {t(
              l,
              "Not verified · service not connected",
              "미확인 · 서비스 미연결",
              "غير متحقق · الخدمة غير متصلة",
            )}
          </dd>
        </dl>
        <p>
          {t(
            l,
            "Registration requires the installation’s verified setup context. Existing installations do not grant a new administrator here.",
            "등록에는 설치 환경의 검증된 초기 설정 문맥이 필요합니다. 이미 운영 중인 환경에서 관리자 권한을 새로 부여하지 않습니다.",
            "يتطلب التسجيل سياق إعداد موثقاً للتثبيت. لا تُمنح صلاحيات مسؤول جديد للتثبيتات القائمة هنا.",
          )}
        </p>
        <PreviewNext
          locale={l}
          mode={mode}
          id="entry-scope"
          label={t(
            l,
            "Preview workspace selection",
            "업무 공간 선택 화면 보기",
            "معاينة اختيار مساحة العمل",
          )}
        />
      </aside>
    </div>
  );
}

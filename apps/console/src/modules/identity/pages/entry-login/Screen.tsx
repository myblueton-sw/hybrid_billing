import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  useEntryState,
  Email,
  Result,
  EntryLink,
  PreviewNext,
} from "../../entry-ui";
export default function Screen(props: ScreenProps) {
  const { locale: l, mode } = props,
    state = useEntryState(props, "entry-login", [
      "field:workspace",
      "field:email",
      "action:preview",
      "navigate:entry-challenge",
      "navigate:entry-scope",
    ]);
  return (
    <div className="entry-form-layout">
      <section className="entry-card">
        <h1>{t(l, "Sign in", "로그인", "تسجيل الدخول")}</h1>
        <p>
          {t(
            l,
            "Enter your workspace and email address.",
            "업무 공간과 이메일 주소를 입력하세요.",
            "أدخل مساحة عملك وبريدك الإلكتروني.",
          )}
        </p>
        <form onSubmit={state.submit} onChange={state.changed}>
          <label>
            {t(l, "Workspace", "업무 공간", "مساحة العمل")}
            <input
              id="entry-workspace"
              required
              maxLength={100}
              autoComplete="off"
              placeholder="workspace-name"
            />
          </label>
          <Email locale={l} />
          <p className="entry-note">
            {t(
              l,
              "Sign-in methods are unavailable until the identity service is connected.",
              "인증 서비스 연결 후 조직에서 허용한 로그인 방식이 표시됩니다.",
              "تتوفر طرق تسجيل الدخول بعد اتصال خدمة الهوية.",
            )}
          </p>
          <button className="primary" type="submit">
            {t(
              l,
              "Preview sign-in request",
              "로그인 요청 미리보기",
              "معاينة طلب الدخول",
            )}
          </button>
        </form>
        {state.result && <Result locale={l} />}
        <EntryLink id="entry-challenge" locale={l} mode={mode}>
          {t(l, "Recover access", "접근 복구", "استعادة الوصول")}
        </EntryLink>
      </section>
      <aside className="entry-help">
        <h2>
          {t(
            l,
            "Your organization’s workspace",
            "소속 조직의 업무 공간",
            "مساحة عمل مؤسستك",
          )}
        </h2>
        <p>
          {t(
            l,
            "Use the workspace provided by your administrator. An email address alone does not grant membership.",
            "관리자에게 안내받은 업무 공간을 입력하세요. 이메일 주소만으로 조직에 가입되지 않습니다.",
            "استخدم مساحة العمل التي وفرها المسؤول. البريد الإلكتروني وحده لا يمنح العضوية.",
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

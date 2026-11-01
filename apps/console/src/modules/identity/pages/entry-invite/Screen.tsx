import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  useEntryState,
  Email,
  Reference,
  Result,
  EntryLink,
} from "../../entry-ui";
export default function Screen(props: ScreenProps) {
  const { locale: l, mode } = props,
    state = useEntryState(props, "entry-invite", [
      "field:proof",
      "field:email",
      "action:preview",
      "navigate:entry-login",
      "data:scope",
      "data:role",
      "data:expiry",
    ]);
  return (
    <div className="entry-form-layout">
      <section className="entry-card">
        <h1>
          {t(l, "Join by invitation", "초대받아 가입하기", "الانضمام عبر دعوة")}
        </h1>
        <p>
          {t(
            l,
            "Check the invitation reference and the invited email address.",
            "초대 참조와 초대받은 이메일 주소를 확인하세요.",
            "تحقق من مرجع الدعوة والبريد الإلكتروني المدعو.",
          )}
        </p>
        <form onSubmit={state.submit} onChange={state.changed}>
          <Reference locale={l} />
          <Email locale={l} />
          <button className="primary" type="submit">
            {t(
              l,
              "Preview invitation request",
              "초대 확인 요청 미리보기",
              "معاينة طلب الدعوة",
            )}
          </button>
        </form>
        {state.result && <Result locale={l} />}
        <EntryLink id="entry-login" locale={l} mode={mode}>
          {t(
            l,
            "Already joined? Sign in",
            "이미 가입했다면 로그인",
            "هل انضممت بالفعل؟ سجّل الدخول",
          )}
        </EntryLink>
      </section>
      <aside className="entry-help">
        <h2>{t(l, "Invitation details", "초대 정보", "تفاصيل الدعوة")}</h2>
        <dl>
          {[
            t(l, "Workspace", "업무 공간", "مساحة العمل"),
            t(l, "Role and scope", "역할과 허용 범위", "الدور والنطاق"),
            t(l, "Expires at", "만료일", "تاريخ الانتهاء"),
          ].map((label) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                {t(
                  l,
                  "Awaiting invitation verification",
                  "초대 검증 전",
                  "بانتظار التحقق من الدعوة",
                )}
              </dd>
            </div>
          ))}
        </dl>
        <p>
          {t(
            l,
            "For expired or invalid invitations, ask your inviter for a new invitation. This preview sends no request.",
            "초대가 만료되었거나 유효하지 않다면 초대한 담당자에게 새 초대를 요청하세요. 이 화면에서 요청을 발송하지는 않습니다.",
            "إذا انتهت الدعوة أو كانت غير صالحة، اطلب دعوة جديدة من المرسل. هذه المعاينة لا ترسل طلباً.",
          )}
        </p>
      </aside>
    </div>
  );
}

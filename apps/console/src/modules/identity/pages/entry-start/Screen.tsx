import { useEffect } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { EntryLink, entryUrl } from "../../entry-ui";
export default function Screen({ locale: l, mode, onState }: ScreenProps) {
  useEffect(
    () =>
      onState(JSON.stringify({ mode }), [
        "entry-start:region:layout",
        ...[
          "entry-login",
          "entry-invite",
          "entry-bootstrap",
          "entry-challenge",
        ].map((id) => `entry-start:navigate:${id}`),
      ]),
    [mode, onState],
  );
  return (
    <div className="entry-welcome">
      <section className="entry-intro">
        <span className="entry-eyebrow">Hybrid Billing</span>
        <h1>
          {t(
            l,
            "Your workspace starts here",
            "서비스를 시작하세요",
            "ابدأ من مساحة عملك",
          )}
        </h1>
        <p>
          {t(
            l,
            "Sign in to your workspace, join your team or set up a new installation.",
            "업무 공간에 로그인하거나 초대를 수락하세요. 처음 설치했다면 관리자 등록부터 시작합니다.",
            "سجّل الدخول إلى مساحة عملك، أو انضم إلى فريقك، أو جهّز تثبيتاً جديداً.",
          )}
        </p>
        <dl className="entry-installation">
          <div>
            <dt>{t(l, "Installation", "설치 환경", "التثبيت")}</dt>
            <dd>Hybrid Billing</dd>
          </div>
          <div>
            <dt>{t(l, "Connection", "연결 상태", "الاتصال")}</dt>
            <dd>
              {t(
                l,
                "Identity service not connected",
                "인증 서비스 미연결",
                "خدمة الهوية غير متصلة",
              )}
            </dd>
          </div>
        </dl>
      </section>
      <section className="entry-card entry-signin-card">
        <h2>
          {t(
            l,
            "Sign in to your workspace",
            "업무 공간 로그인",
            "الدخول إلى مساحة العمل",
          )}
        </h2>
        <p>
          {t(
            l,
            "Continue through your organization’s sign-in process.",
            "소속 조직의 로그인 절차를 진행합니다.",
            "تابع إجراءات تسجيل الدخول الخاصة بمؤسستك.",
          )}
        </p>
        <a
          className="primary entry-main-action"
          href={entryUrl("entry-login", l, mode)}
        >
          {t(l, "Continue to sign in", "로그인하기", "متابعة تسجيل الدخول")}
        </a>
        <div className="entry-secondary">
          <h3>
            {t(
              l,
              "Received an invitation?",
              "초대를 받으셨나요?",
              "هل تلقيت دعوة؟",
            )}
          </h3>
          <EntryLink id="entry-invite" locale={l} mode={mode}>
            {t(l, "Join by invitation", "초대 수락하기", "الانضمام عبر دعوة")}
          </EntryLink>
        </div>
        <div className="entry-secondary">
          <h3>
            {t(l, "First installation", "처음 설치하셨나요?", "التثبيت الأول")}
          </h3>
          <EntryLink id="entry-bootstrap" locale={l} mode={mode}>
            {t(
              l,
              "Register the first administrator",
              "최초 관리자 등록",
              "تسجيل المسؤول الأول",
            )}
          </EntryLink>
        </div>
        <EntryLink id="entry-challenge" locale={l} mode={mode}>
          {t(
            l,
            "Need help accessing your workspace?",
            "접근에 도움이 필요하신가요?",
            "هل تحتاج مساعدة للوصول؟",
          )}
        </EntryLink>
      </section>
    </div>
  );
}

import { useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  useEntryState,
  Email,
  Reference,
  Result,
  EntryLink,
  PreviewNext,
} from "../../entry-ui";
export default function Screen(props: ScreenProps) {
  const { locale: l, mode } = props,
    [recovery, setRecovery] = useState(false),
    state = useEntryState(
      props,
      "entry-challenge",
      [
        "field:proof",
        "field:email",
        "action:preview",
        "action:recovery",
        "navigate:entry-login",
        "navigate:entry-scope",
      ],
      recovery ? "recovery" : "verification",
    );
  return (
    <div className="entry-form-layout">
      <section className="entry-card">
        <h1>
          {t(
            l,
            "Verify identity or recover access",
            "추가 인증과 접근 복구",
            "التحقق من الهوية واستعادة الوصول",
          )}
        </h1>
        <div className="entry-tabs">
          <button
            aria-pressed={!recovery}
            onClick={() => {
              setRecovery(false);
              state.changed();
            }}
          >
            {t(l, "Additional verification", "추가 인증", "تحقق إضافي")}
          </button>
          <button
            aria-pressed={recovery}
            onClick={() => {
              setRecovery(true);
              state.changed();
            }}
          >
            {t(l, "Recover access", "접근 복구", "استعادة الوصول")}
          </button>
        </div>
        <p>
          {recovery
            ? t(
                l,
                "Prepare a recovery request for your workspace administrator.",
                "업무 공간 관리자에게 전달할 복구 요청을 준비합니다.",
                "جهّز طلب استعادة الوصول لمسؤول مساحة العمل.",
              )
            : t(
                l,
                "Review the verification reference for the current sign-in step.",
                "현재 로그인 단계의 인증 참조를 확인합니다.",
                "راجع مرجع التحقق لخطوة الدخول الحالية.",
              )}
        </p>
        <form
          key={String(recovery)}
          onSubmit={state.submit}
          onChange={state.changed}
        >
          <Reference locale={l} />
          <Email locale={l} />
          <button className="primary" type="submit">
            {recovery
              ? t(
                  l,
                  "Preview recovery request",
                  "복구 요청 미리보기",
                  "معاينة طلب الاستعادة",
                )
              : t(
                  l,
                  "Preview verification request",
                  "인증 요청 미리보기",
                  "معاينة طلب التحقق",
                )}
          </button>
        </form>
        {state.result && <Result locale={l} />}
        <EntryLink id="entry-login" locale={l} mode={mode}>
          {t(
            l,
            "Back to sign in",
            "로그인으로 돌아가기",
            "العودة لتسجيل الدخول",
          )}
        </EntryLink>
      </section>
      <aside className="entry-help">
        <h2>{t(l, "Verification status", "인증 상태", "حالة التحقق")}</h2>
        <p>
          {t(
            l,
            "No active challenge is connected. Recovery does not bypass your organization’s sign-in policy.",
            "연결된 인증 요청이 없습니다. 복구 절차가 조직의 로그인 정책을 우회하지 않습니다.",
            "لا يوجد طلب تحقق متصل. الاستعادة لا تتجاوز سياسة تسجيل الدخول للمؤسسة.",
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

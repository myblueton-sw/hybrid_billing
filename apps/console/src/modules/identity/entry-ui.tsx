import { useEffect, useState, type ReactNode } from "react";
import type { Locale, ScreenProps } from "../../shared/types";
import { text as t } from "../../shared/copy";
export function entryUrl(id: string, locale: Locale, mode = "synthetic") {
  const p = new URLSearchParams(location.hash.split("?")[1]);
  return `#${id}?${new URLSearchParams({ lang: locale, mode, profile: p.get("profile") ?? "standard" })}`;
}
export function EntryLink({
  id,
  locale,
  mode,
  children,
}: {
  id: string;
  locale: Locale;
  mode?: string;
  children: ReactNode;
}) {
  return <a href={entryUrl(id, locale, mode)}>{children}</a>;
}
export function useEntryState(
  props: ScreenProps,
  id: string,
  items: string[],
  variant = "default",
) {
  const [instance] = useState(() => crypto.randomUUID());
  const [revision, setRevision] = useState(0),
    [result, setResult] = useState(false);
  useEffect(
    () =>
      props.onState(
        JSON.stringify({
          instance,
          variant,
          revision,
          result,
          mode: props.mode,
        }),
        items.map((i) => `${id}:${i}`),
      ),
    [instance, variant, revision, result, props.mode, props.onState],
  );
  return {
    result,
    changed: () => {
      setRevision((n) => n + 1);
      setResult(false);
    },
    submit: (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      e.currentTarget.reset();
      setRevision((n) => n + 1);
      setResult(true);
    },
  };
}
export function Result({ locale: l }: { locale: Locale }) {
  return (
    <div className="entry-feedback" role="status">
      {t(
        l,
        "Request fields checked locally. The identity service is not connected; no account, session or membership was created.",
        "입력 형식을 확인했습니다. 인증 서비스가 연결되지 않아 계정·세션·가입은 생성되지 않았습니다.",
        "تم فحص الحقول محلياً. خدمة الهوية غير متصلة؛ لم يتم إنشاء حساب أو جلسة أو عضوية.",
      )}
    </div>
  );
}
export function Email({ locale: l }: { locale: Locale }) {
  return (
    <label>
      {t(l, "Email address", "이메일 주소", "البريد الإلكتروني")}
      <input
        id="entry-email"
        type="email"
        required
        maxLength={200}
        autoComplete="off"
        placeholder="name@example.com"
      />
    </label>
  );
}
export function Reference({ locale: l }: { locale: Locale }) {
  return (
    <label>
      {t(l, "Verification reference", "검증 참조", "مرجع التحقق")}
      <input
        id="entry-proof"
        required
        maxLength={100}
        autoComplete="off"
        placeholder="DEMO-REFERENCE"
      />
      <small>
        {t(
          l,
          "Use an example reference, never a password or verification token.",
          "예제 참조만 입력하세요. 비밀번호나 인증 토큰은 입력하지 마세요.",
          "استخدم مرجعاً تجريبياً، وليس كلمة مرور أو رمز تحقق.",
        )}
      </small>
    </label>
  );
}
export function PreviewNext({
  locale: l,
  mode,
  id,
  label,
}: {
  locale: Locale;
  mode: string;
  id: string;
  label: string;
}) {
  return mode === "synthetic" ? (
    <div className="entry-preview-next">
      <span>{t(l, "Screen preview", "화면 미리보기", "معاينة الشاشة")}</span>
      <EntryLink id={id} locale={l} mode={mode}>
        {label}
      </EntryLink>
    </div>
  ) : null;
}

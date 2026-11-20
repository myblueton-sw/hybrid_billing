import { useLayoutEffect, useMemo, useRef, useState } from "react";
import type { Locale } from "../shared/types";
import { useService } from "./context";
import type { AdminClient } from "./admin";
import { diagnosticOf, observeCommands, type CommandDiagnostic, type RowDiagnostic } from "./command-diagnostics";

const text = {
  title: { en: "Action request references", ko: "작업 요청 참조 정보", ar: "مراجع طلب الإجراء" },
  code: { en: "Server error code", ko: "서버 오류 코드", ar: "رمز خطأ الخادم" },
  request: { en: "Server request ID", ko: "서버 요청 ID", ar: "معرّف طلب الخادم" },
  client: { en: "Client request ID", ko: "클라이언트 요청 ID", ar: "معرّف طلب العميل" },
  action: { en: "Action ID", ko: "작업 ID", ar: "معرّف الإجراء" },
  target: { en: "Target ID", ko: "대상 ID", ar: "معرّف الهدف" },
  missing: { en: "Not available", ko: "확인할 수 없음", ar: "غير متاح" },
  next: { en: "Review the message beside the action for the next step. These references do not confirm whether the action completed.", ko: "다음 조치는 작업 옆의 안내에서 확인하세요. 이 참조 정보만으로 작업 완료 여부를 확인할 수 없습니다.", ar: "راجع الرسالة بجانب الإجراء لمعرفة الخطوة التالية. هذه المراجع لا تؤكد ما إذا كان الإجراء قد اكتمل." },
  cancelled: { en: "Password confirmation was cancelled here. Server details below refer to the preceding confirmation request.", ko: "이 화면에서 비밀번호 확인을 취소했습니다. 아래 서버 정보는 앞서 받은 확인 요청에 해당합니다.", ar: "أُلغي تأكيد كلمة المرور هنا. تشير تفاصيل الخادم أدناه إلى طلب التأكيد السابق." },
};

export function DiagnosticDetails({ l, detail }: { l: Locale; detail: CommandDiagnostic }) {
  return (
    <div className="command-diagnostics-details">
      {detail.cancelled && <p>{text.cancelled[l]}</p>}
      <dl className="live-facts">
        {(["code", "request", "client", "target"] as const).filter((name) => name !== "client" && name !== "target" || detail[name]).map((name) => (
          <div key={name}><dt>{text[name][l]}</dt><dd>{detail[name] ? <bdi dir="ltr" className="live-mono">{detail[name]}</bdi> : text.missing[l]}</dd></div>
        ))}
      </dl>
    </div>
  );
}

export function ErrorDiagnostics({ l, error }: { l: Locale; error: unknown }) {
  return <DiagnosticDetails l={l} detail={diagnosticOf(error)} />;
}

export function CommandDiagnostics({ l, detail }: { l: Locale; detail: CommandDiagnostic | null }) {
  if (!detail) return null;
  return (
    <section className="live-note-rule command-diagnostics" aria-label={text.title[l]} role="status">
      <h3>{text.title[l]}</h3>
      <DiagnosticDetails l={l} detail={detail} />
      <p>{text.next[l]}</p>
    </section>
  );
}

export function useCommandDiagnostics(api: AdminClient, scope: string, onStart?: () => void) {
  const begin = useRef(onStart);
  begin.current = onStart;
  const current = useRef<object | null>(null);
  const [record, setRecord] = useState<{ token: object; detail: CommandDiagnostic | null } | null>(null);
  const token = useMemo(() => ({ active: true }), [api, scope]);
  current.current = token;
  useLayoutEffect(() => {
    token.active = true;
    return () => { token.active = false; };
  }, [token]);
  const client = useMemo(() => observeCommands(api, (detail) => { if (detail === null) begin.current?.(); setRecord({ token, detail }); }, () => token.active && current.current === token), [api, token]);
  return { api: client, detail: record?.token === token ? record.detail : null };
}

export function RowDiagnostics({ l, entries }: { l: Locale; entries: RowDiagnostic[] }) {
  return <>{entries.map(({ command, detail }) => <div key={command} className="command-diagnostics-row">
    <p>{text.action[l]}: <bdi dir="ltr" className="live-mono">{command}</bdi></p>
    <DiagnosticDetails l={l} detail={detail} />
  </div>)}</>;
}

export function useCurrentCommand(command: string, target: string) {
  const service = useService();
  const scope = JSON.stringify([service?.session.workspace, service?.session.user, service?.session.csrf, command, target]);
  const current = useRef<object | null>(null);
  const token = useMemo(() => ({ active: true }), [scope]);
  current.current = token;
  useLayoutEffect(() => {
    token.active = true;
    return () => { token.active = false; };
  }, [token]);
  return useMemo(() => () => token.active && current.current === token, [token]);
}

export function useFailureDiagnostic(current: () => boolean) {
  const [saved, save] = useState<{ current: () => boolean; detail: CommandDiagnostic } | null>(null);
  return {
    diagnostic: saved?.current === current ? saved.detail : null,
    setDiagnostic: (detail: CommandDiagnostic | null) => {
      if (current()) save(detail ? { current, detail } : null);
    },
  };
}

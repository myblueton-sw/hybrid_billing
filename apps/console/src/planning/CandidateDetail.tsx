import { useEffect, useRef } from "react";
import { local } from "./copy";
import { field, valueText } from "./fields";
import type { Locale } from "../shared/types";
import type { CandidateRow } from "./CandidateTable";
export default function CandidateDetail({
  row,
  keys,
  locale: l,
  onClose,
}: {
  row: CandidateRow;
  keys: string[];
  locale: Locale;
  onClose: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null),
    previous = useRef(document.activeElement as HTMLElement);
  useEffect(() => {
    dialog.current?.showModal();
    return () => {
      dialog.current?.close();
      if (previous.current?.isConnected) previous.current.focus();
    };
  }, []);
  return (
    <dialog
      className="planning-detail"
      ref={dialog}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      aria-labelledby="planning-detail-title"
    >
      <header>
        <h2 id="planning-detail-title">
          <bdi>{row.id}</bdi>
        </h2>
        <button autoFocus onClick={onClose}>
          {local(l, ["닫기", "Close", "إغلاق"])}
        </button>
      </header>
      <p>
        {local(l, [
          "합성 데이터 상세 · 실제 서비스 미연결",
          "Synthetic detail · service not connected",
          "تفاصيل تجريبية · الخدمة غير متصلة",
        ])}
      </p>
      <dl>
        {[...new Set(keys)].map((k) => (
          <div key={k}>
            <dt>{field(l, k)}</dt>
            <dd>
              <bdi>
                {valueText(
                  l,
                  k === "id" ? row.id : (row.values[k] ?? "unconfigured"),
                )}
              </bdi>
            </dd>
          </div>
        ))}
      </dl>
      <section>
        <h3>
          {local(l, ["근거·개정", "Evidence and revision", "الدليل والمراجعة"])}
        </h3>
        <p>
          <bdi>
            {row.values.evidence ?? `${row.id}:EVIDENCE`} ·{" "}
            {row.values.revision ?? "r1"}
          </bdi>
        </p>
        <p>
          {local(l, [
            "예제 참조입니다. 원문 조회·승인·전달 서비스는 연결되지 않았습니다.",
            "Example reference. Original retrieval, approval and delivery services are not connected.",
            "مرجع تجريبي. خدمات الأصل والموافقة والتسليم غير متصلة.",
          ])}
        </p>
      </section>
    </dialog>
  );
}

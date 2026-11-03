import { useState } from "react";
import { local } from "./copy";
import { field } from "./fields";
import type { Locale } from "../shared/types";
export default function DraftForm({
  fields,
  locale: l,
  onDirty,
}: {
  fields: string[];
  locale: Locale;
  onDirty: (dirty: boolean) => void;
}) {
  const [values, setValues] = useState<Record<string, string>>({}),
    [saved, setSaved] = useState(false);
  return (
    <form
      className="planning-form"
      onSubmit={(e) => {
        e.preventDefault();
        setSaved(true);
        onDirty(false);
      }}
    >
      <h3>{local(l, ["검토용 초안", "Review draft", "مسودة للمراجعة"])}</h3>
      <p>
        {local(l, [
          "이 화면에서만 유지되는 예제입니다. 실제 등록·적용·전송은 수행하지 않습니다.",
          "An in-memory example only. No registration, application or transmission is performed.",
          "مثال في الذاكرة فقط. لا يتم التسجيل أو التطبيق أو الإرسال.",
        ])}
      </p>
      <div className="planning-field-grid">
        {fields.map((k) => (
          <label key={k}>
            {field(l, k)}
            <input
              required
              value={values[k] ?? ""}
              onChange={(e) => {
                setValues({ ...values, [k]: e.target.value });
                setSaved(false);
                onDirty(true);
              }}
              autoComplete="off"
            />
          </label>
        ))}
      </div>
      <button type="submit">
        {local(l, ["초안 확인", "Review draft", "مراجعة المسودة"])}
      </button>
      {saved && (
        <p role="status">
          {local(l, [
            "입력한 초안을 확인했습니다. 페이지를 나가면 지워집니다.",
            "Draft reviewed locally. It clears when leaving this page.",
            "تمت مراجعة المسودة محلياً. تمسح عند مغادرة الصفحة.",
          ])}
        </p>
      )}
    </form>
  );
}

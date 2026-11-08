import { useState } from "react";
import { Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props } from "../../../../shared/domain-screen-ui/operations/types";
const notices = [
  "react",
  "react-dom",
  "scheduler",
  "@fontsource/ibm-plex-sans",
  "@fontsource/ibm-plex-sans-kr",
  "@fontsource/ibm-plex-sans-arabic",
  "@fontsource/ibm-plex-mono",
];
export default function Notices({ locale: l }: Pick<Props, "locale">) {
  const [search, setSearch] = useState("");
  const filtered = notices.filter((n) =>
    n.includes(search.trim().toLowerCase()),
  );
  return (
    <Panel
      title={t(l, [
        "이 프론트엔드 번들의 원문 고지",
        "Original notices for this frontend bundle",
        "الإشعارات الأصلية لحزمة الواجهة هذه",
      ])}
    >
      <p>
        {t(l, [
          "빌드에 포함된 원문 LICENSE 파일입니다. 설치 권리·지원 계약 판정과 별도입니다.",
          "Original LICENSE files included in this build. Separate from installation entitlements and support-contract decisions.",
          "ملفات LICENSE الأصلية المضمّنة في هذا البناء. منفصلة عن استحقاقات التثبيت وقرارات عقد الدعم.",
        ])}
      </p>
      <label className="ops-notice-search">
        {t(l, ["패키지 검색", "Search packages", "البحث عن الحزم"])}
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          maxLength={100}
        />
      </label>
      <ul className="ops-notices">
        {filtered.map((n) => (
          <li key={n}>
            <bdi dir="ltr">{n}</bdi>
            <a
              href={`./licenses/${n.replaceAll("/", "-")}.txt`}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={t(l, [
                `${n} 원문 고지 새 탭에서 열기`,
                `Open original ${n} notice in new tab`,
                `فتح إشعار ${n} الأصلي في تبويب جديد`,
              ])}
            >
              {t(l, [
                "원문 고지 열기",
                "Open original notice",
                "فتح الإشعار الأصلي",
              ])}
            </a>
          </li>
        ))}
      </ul>
      {filtered.length === 0 && (
        <p role="status">
          {t(l, [
            "검색에 맞는 번들 패키지가 없습니다.",
            "No bundled package matches this search.",
            "لا توجد حزمة مضمنة تطابق هذا البحث.",
          ])}
        </p>
      )}
    </Panel>
  );
}

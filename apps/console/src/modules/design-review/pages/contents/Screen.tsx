import { useEffect } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { routeUrl } from "../../../../shared/navigation";
export default function Screen({
  locale: l,
  profile,
  extensionIds,
  onState,
}: ScreenProps) {
  useEffect(
    () =>
      onState(JSON.stringify({ profile, extensionIds }), [
        "contents:table:implementation",
        "contents:action:open-intake",
      ]),
    [profile, extensionIds.join(","), onState],
  );
  return (
    <>
      <p className="lead">
        {t(
          l,
          "One application. Reusable modules. Explicit installation profiles.",
          "하나의 앱에서 업무 모듈과 설치 구성을 조립합니다.",
          "تطبيق واحد بوحدات قابلة لإعادة الاستخدام وملفات تثبيت واضحة.",
        )}
      </p>
      <div className="metrics">
        <section>
          <strong>2</strong>
          <span>
            {t(
              l,
              "React screens implemented",
              "React 구현 화면",
              "شاشات React المنفذة",
            )}
          </span>
        </section>
        <section>
          <strong>112</strong>
          <span>
            {t(
              l,
              "Screens in the design inventory",
              "전체 설계 화면",
              "شاشات التصميم",
            )}
          </span>
        </section>
        <section>
          <strong>
            <bdi>{profile}</bdi>
          </strong>
          <span>
            {t(
              l,
              "Synthetic installation profile",
              "합성 설치 구성",
              "ملف التثبيت التجريبي",
            )}
          </span>
        </section>
      </div>
      <section className="panel">
        <h2>
          {t(
            l,
            "First implementation slice",
            "첫 구현 범위",
            "نطاق التنفيذ الأول",
          )}
        </h2>
        <p>
          {t(
            l,
            "The source-intake screen supports bounded queries and local screen review. The remaining product screens and backend services are not implemented in this increment.",
            "원본 수집 화면의 제한된 조회와 로컬 화면 검수를 구현했습니다. 나머지 제품 화면과 백엔드 서비스는 이번 범위에 포함되지 않습니다.",
            "تدعم شاشة استلام المصادر الاستعلام المحدود ومراجعة الشاشة محليًا. بقية الشاشات والخدمات الخلفية خارج هذا الإصدار.",
          )}
        </p>
        <a className="button primary" href={routeUrl("index-intake")}>
          {t(
            l,
            "Open source intake",
            "원본 수집 화면 열기",
            "فتح استلام المصادر",
          )}
        </a>
      </section>
      <section className="panel">
        <h2>{t(l, "Composition", "모듈 구성", "تركيب الوحدات")}</h2>
        <dl>
          <dt>{t(l, "Modules", "업무 모듈", "الوحدات")}</dt>
          <dd>
            <bdi>design-review · sources</bdi>
          </dd>
          <dt>
            {t(l, "Optional extensions", "추가 기능", "الإضافات الاختيارية")}
          </dt>
          <dd>
            <bdi>{extensionIds.join(", ") || "—"}</bdi>
          </dd>
        </dl>
      </section>
    </>
  );
}

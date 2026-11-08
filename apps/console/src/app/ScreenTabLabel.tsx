import type { Locale } from "../shared/types";
import { local, tabLabel } from "../planning/copy";
import { tabLabel as creditsTab } from "../modules/history/pages/credits-cases/copy";
import { tabLabel as claimTab } from "../modules/calculation/pages/claim-history/copy";
import { tabLabel as inboxTab } from "../modules/issuance/pages/approval-inbox/copy";
import { tabLabel as billingTab } from "../modules/calculation/pages/billing-runs/copy";
import { tabLabel as marketplaceTab } from "../modules/pricing/pages/marketplace/copy";
export default function ScreenTabLabel({
  id,
  locale: l,
  tab: t,
}: {
  id: string;
  locale: Locale;
  tab: { key: string; labelKo: string };
}) {
  const liveCredits = id === "B11",
    liveClaims = id === "B10",
    liveInbox = id === "B09",
    liveBilling = id === "B08",
    liveMarketplace = id === "B07",
    liveCommitments = id === "B06",
    liveFx = id === "B05",
    livePricing = id === "B01" || id === "B03" || id === "B04";
  return liveCredits
    ? creditsTab(l, t.key)
    : liveClaims
      ? claimTab(l, t.key)
      : liveInbox
        ? inboxTab(l, t.key)
        : liveBilling
          ? billingTab(l, t.key)
          : liveMarketplace
            ? marketplaceTab(l, t.key)
            : liveCommitments
              ? local(
                  l,
                  (
                    {
                      commitments: [
                        "약정 명세",
                        "Commitment details",
                        "تفاصيل الالتزامات",
                      ],
                      utilization: [
                        "사용·미사용 근거",
                        "Used and unused evidence",
                        "أدلة الاستخدام وعدم الاستخدام",
                      ],
                      benefits: ["혜택 흐름", "Benefit flow", "مسار المنافع"],
                      reconciliation: [
                        "대사 검토",
                        "Reconciliation review",
                        "مراجعة المطابقة",
                      ],
                    } as Record<string, [string, string, string]>
                  )[t.key],
                )
              : liveFx
                ? local(
                    l,
                    (
                      {
                        policies: [
                          "환산 규칙",
                          "Conversion rules",
                          "قواعد التحويل",
                        ],
                        imports: [
                          "환율 근거",
                          "Exchange-rate evidence",
                          "أدلة سعر الصرف",
                        ],
                        comparison: [
                          "환산 비교",
                          "Conversion comparison",
                          "مقارنة التحويل",
                        ],
                        approvals: [
                          "승인 이력",
                          "Approval history",
                          "سجل الاعتماد",
                        ],
                      } as Record<string, [string, string, string]>
                    )[t.key],
                  )
                : livePricing
                  ? local(
                      l,
                      (
                        {
                          rates: [
                            "요금 규칙",
                            "Pricing rules",
                            "قواعد التسعير",
                          ],
                          formula: [
                            "계산 단계",
                            "Calculation steps",
                            "خطوات الحساب",
                          ],
                          inheritance: [
                            "정책 상속",
                            "Policy inheritance",
                            "توريث السياسات",
                          ],
                          adjustments: [
                            "조정 검토",
                            "Adjustment review",
                            "مراجعة التعديل",
                          ],
                          impact: [
                            "금액 비교",
                            "Amount comparison",
                            "مقارنة المبالغ",
                          ],
                          policies: [
                            "배분 정책",
                            "Allocation policy",
                            "سياسة التوزيع",
                          ],
                          pools: [
                            "배분 기준액",
                            "Allocation basis",
                            "مبلغ أساس التوزيع",
                          ],
                          simulation: [
                            "배분 미리보기",
                            "Allocation preview",
                            "معاينة التوزيع",
                          ],
                          comparison: [
                            "결과 비교",
                            "Result comparison",
                            "مقارنة النتائج",
                          ],
                          summary: [
                            "구성 요약",
                            "Composition summary",
                            "ملخص المكوّنات",
                          ],
                          costs: ["원가 항목", "Cost items", "بنود التكلفة"],
                          products: [
                            "상품별 포함 항목",
                            "Included items by product",
                            "البنود المشمولة لكل منتج",
                          ],
                          approvals: [
                            "승인 이력",
                            "Approval history",
                            "سجل الاعتماد",
                          ],
                        } as Record<string, [string, string, string]>
                      )[t.key],
                    )
                  : tabLabel(l, t.key, t.labelKo);
}

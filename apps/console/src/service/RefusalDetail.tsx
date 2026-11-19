import type { Locale } from "../shared/types";
import type { RefusalDetail as Detail } from "./refusal";

export function RefusalDetail({ locale, detail }: { locale: Locale; detail?: Detail }) {
  if (!detail || (!detail.code && !detail.request)) return null;
  const code = locale === "ko" ? "오류 코드" : locale === "ar" ? "رمز الخطأ" : "Error code";
  const request = locale === "ko" ? "요청 ID" : locale === "ar" ? "معرّف الطلب" : "Request ID";
  return <dl className="service-refusal-detail">
    {detail.code && <div><dt>{code}</dt><dd><bdi dir="ltr">{detail.code}</bdi></dd></div>}
    {detail.request && <div><dt>{request}</dt><dd><bdi dir="ltr">{detail.request}</bdi></dd></div>}
  </dl>;
}

export function canonicalScope(query: URLSearchParams, screen: string) {
  const result = new URLSearchParams(query);
  const periodKey =
    screen === "U01"
      ? "overview-period"
      : screen === "U02"
        ? "close-period"
        : null;
  if (periodKey) {
    if (!result.has("period") && result.has(periodKey))
      result.set("period", result.get(periodKey)!);
    result.delete(periodKey);
  }
  if (screen === "U01") {
    if (!result.has("ccy") && result.has("ov-currency"))
      result.set("ccy", result.get("ov-currency")!);
    result.delete("ov-currency");
  }
  return result;
}

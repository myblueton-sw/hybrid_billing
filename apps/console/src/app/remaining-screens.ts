export const remainingGroups = {
  security: ["L01", "L02", "L03", "M01", "M02", "M03", "M04", "M05"],
  finance: ["I01", "I02", "I03", "I04", "R01", "R02", "B02"],
  work: ["O01", "O02", "O03", "U03", "U04", "S03", "A01"],
  operations: [
    "P01",
    "P02",
    "P03",
    "P04",
    "P05",
    "P06",
    "P07",
    "C01",
    "C02",
    "C03",
  ],
} as const;
export const remainingIds: ReadonlySet<string> = new Set(
  Object.values(remainingGroups).flat(),
);
export function presentationAudience(hash: string) {
  const path = hash.replace(/^#/, "").split("?")[0];
  if (
    [
      "/login",
      "/activate",
      "/recover",
      "entry-start",
      "entry-login",
      "entry-scope",
      "entry-bootstrap",
      "entry-invite",
      "entry-challenge",
    ].includes(path)
  )
    return "unauthenticated" as const;
  if (path.startsWith("/portal/")) return "customer" as const;
  return "admin" as const;
}

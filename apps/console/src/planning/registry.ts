import m0 from "../modules/identity/screens.json";
import m1 from "../modules/operations/screens.json";
import m2 from "../modules/organizations/screens.json";
import m3 from "../modules/onboarding/screens.json";
import m4 from "../modules/sources/screens.json";
import m5 from "../modules/pricing/screens.json";
import m6 from "../modules/calculation/screens.json";
import m7 from "../modules/issuance/screens.json";
import m8 from "../modules/history/screens.json";
import m9 from "../modules/delivery/screens.json";
import m10 from "../modules/analytics/screens.json";
import m11 from "../modules/assistant/screens.json";
import m12 from "../modules/administration/screens.json";
import m13 from "../modules/customer-portal/screens.json";
import legacy from "./legacy.json";
const planningOrder = [
  "L01",
  "L02",
  "L03",
  "U01",
  "U02",
  "U03",
  "U04",
  "O01",
  "O02",
  "O03",
  "S01",
  "S02",
  "S03",
  "S04",
  "S05",
  "S06",
  "B01",
  "B02",
  "B03",
  "B04",
  "B05",
  "B06",
  "B07",
  "B08",
  "B09",
  "B10",
  "B11",
  "I01",
  "I02",
  "I03",
  "I04",
  "R01",
  "R02",
  "A01",
  "M01",
  "M02",
  "M03",
  "M04",
  "M05",
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
];
export const planningScreens = [
  ...m0,
  ...m1,
  ...m2,
  ...m3,
  ...m4,
  ...m5,
  ...m6,
  ...m7,
  ...m8,
  ...m9,
  ...m10,
  ...m11,
  ...m12,
  ...m13,
].sort((a, b) => planningOrder.indexOf(a.id) - planningOrder.indexOf(b.id));
export type PlanningScreen = (typeof planningScreens)[number];
export const legacyRedirects: Record<
  string,
  { target: string; screen: string | null; note: string }
> = legacy;
export const legacyReviewMap = Object.fromEntries(
  Object.entries(legacy).map(([key, value]) => [key, value.screen]),
);
export const byScreenId = (id: string) =>
  planningScreens.find((s) => s.id === id);

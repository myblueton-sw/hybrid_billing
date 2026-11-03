import U03 from "../modules/operations/candidates/U03.json";
import U04 from "../modules/operations/candidates/U04.json";
import O01 from "../modules/organizations/candidates/O01.json";
import O02 from "../modules/organizations/candidates/O02.json";
import O03 from "../modules/organizations/candidates/O03.json";
import S01 from "../modules/onboarding/candidates/S01.json";
import S02 from "../modules/onboarding/candidates/S02.json";
import S03 from "../modules/sources/candidates/S03.json";
import S04 from "../modules/sources/candidates/S04.json";
import S05 from "../modules/sources/candidates/S05.json";
import S06 from "../modules/sources/candidates/S06.json";
import B01 from "../modules/pricing/candidates/B01.json";
import B02 from "../modules/sources/candidates/B02.json";
import B03 from "../modules/pricing/candidates/B03.json";
import B04 from "../modules/pricing/candidates/B04.json";
import B05 from "../modules/pricing/candidates/B05.json";
import B06 from "../modules/pricing/candidates/B06.json";
import B07 from "../modules/pricing/candidates/B07.json";
import B08 from "../modules/calculation/candidates/B08.json";
import B09 from "../modules/issuance/candidates/B09.json";
import B10 from "../modules/calculation/candidates/B10.json";
import B11 from "../modules/history/candidates/B11.json";
import I01 from "../modules/issuance/candidates/I01.json";
import I02 from "../modules/delivery/candidates/I02.json";
import I03 from "../modules/delivery/candidates/I03.json";
import I04 from "../modules/delivery/candidates/I04.json";
import R01 from "../modules/analytics/candidates/R01.json";
import R02 from "../modules/history/candidates/R02.json";
import M01 from "../modules/identity/candidates/M01.json";
import M02 from "../modules/identity/candidates/M02.json";
import M03 from "../modules/identity/candidates/M03.json";
import M04 from "../modules/identity/candidates/M04.json";
import M05 from "../modules/identity/candidates/M05.json";
import P01 from "../modules/administration/candidates/P01.json";
import P02 from "../modules/administration/candidates/P02.json";
import P03 from "../modules/administration/candidates/P03.json";
import P04 from "../modules/administration/candidates/P04.json";
import P05 from "../modules/administration/candidates/P05.json";
import P06 from "../modules/administration/candidates/P06.json";
import P07 from "../modules/administration/candidates/P07.json";
import C01 from "../modules/customer-portal/candidates/C01.json";
import C02 from "../modules/customer-portal/candidates/C02.json";
import C03 from "../modules/customer-portal/candidates/C03.json";
export type CandidateModel = {
  columns: string[];
  details: string[];
  values: Record<string, string>;
  views: Record<string, string[]>;
  draftFields: string[];
};
const U01: CandidateModel = {
  columns: ["id", "document", "state"],
  details: ["evidence", "due"],
  values: {
    document: "DEMO-DOC-01",
    state: "pending",
    due: "2026-11-30",
    event: "DEMO source revision received",
    received: "2026-11-03T09:00Z",
    kind: "Reconciliation review",
    owner: "DEMO-USER-01",
  },
  views: {
    tasks: ["id", "kind", "owner", "due", "state"],
    changes: ["event", "received", "revision"],
    "payment-terms": ["document", "due", "state"],
  },
  draftFields: [],
};
export const candidateModels: Record<string, CandidateModel> = {
  U01,
  U03,
  U04,
  O01,
  O02,
  O03,
  S01,
  S02,
  S03,
  S04,
  S05,
  S06,
  B01,
  B02,
  B03,
  B04,
  B05,
  B06,
  B07,
  B08,
  B09,
  B10,
  B11,
  I01,
  I02,
  I03,
  I04,
  R01,
  R02,
  M01,
  M02,
  M03,
  M04,
  M05,
  P01,
  P02,
  P03,
  P04,
  P05,
  P06,
  P07,
  C01,
  C02,
  C03,
};

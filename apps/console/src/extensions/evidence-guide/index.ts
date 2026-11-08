import Guide from "./Guide";
import type { Extension } from "../../shared/types";
export const evidenceGuide: Extension = {
  id: "evidence-guide",
  version: 1,
  module: "sources",
  slot: "sources/evidence/v1",
  slotVersion: 1,
  render: Guide,
};

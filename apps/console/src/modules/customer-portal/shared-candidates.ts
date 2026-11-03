import type { CandidateModel } from "../../planning/models";
export const customerTasks: CandidateModel = {
  columns: ["id", "document", "kind", "state", "received"],
  details: ["publicReply", "delivery"],
  values: {
    document: "DEMO-PUBLIC-01",
    kind: "Public document notification",
    state: "pending",
    received: "2026-11-03T09:00:00Z",
    publicReply: "pending",
    delivery: "pending",
  },
  views: {
    tasks: ["id", "document", "kind", "state"],
    notifications: ["document", "received", "state"],
    detail: ["document", "publicReply", "state"],
    delivery: ["document", "delivery", "received"],
    "payment-terms": ["document", "due", "state"],
  },
  draftFields: [],
};

import { test } from "node:test";
import assert from "node:assert/strict";
import {
  stages,
  stageIndex,
  stageUrl,
  destination,
  validContext,
} from "./model";
import type { Connection } from "../connections/api";
test("stage history is navigation and preserves exact connection context", () => {
  assert.equal(stageIndex("unknown"), 0);
  for (const [i, stage] of stages.entries()) {
    assert.equal(stageIndex(stage), i);
    assert.equal(
      stageUrl(stage, "ar", "connection-id"),
      "#/onboarding?lang=ar&service=local&view=" +
        stage +
        "&connection=connection-id",
    );
  }
});
test("destinations use returned role routes and exact candidate", () => {
  const data = {
    actions: ["imports"],
    latest_candidate: { release_id: "candidate", manifest_digest: "digest" },
  } as Connection;
  assert.equal(destination(data, "reconciliation", "en"), null);
  assert.equal(
    destination(data, "imports", "ko"),
    "#/imports/candidate?lang=ko&service=local",
  );
  const approver = {
    ...data,
    actions: ["source-data", "reconciliation"],
  } as Connection;
  assert.equal(destination(approver, "imports", "en"), null);
  assert.equal(
    destination(approver, "source-data", "en"),
    "#/source-data/candidate?lang=en&service=local&snapshot=digest",
  );
});

test("explicit context must be a UUID while absent context uses server scope", () => {
  assert.equal(validContext(undefined), true);
  assert.equal(validContext(""), false);
  assert.equal(validContext("invalid"), false);
  assert.equal(validContext("00000000-0000-0000-0000-000000000001"), true);
});

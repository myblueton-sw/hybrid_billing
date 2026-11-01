import type { Module } from "../../shared/types";
import { screens } from "../../shared/catalog";
import Operations from "./pages/operations/Screen";
import Close from "./pages/core-index-close/Screen";
import Tasks from "./pages/my-tasks/Screen";
export const operations: Module = {
  id: "operations",
  version: 1,
  requires: [],
  slots: [],
  routes: [
    { id: "operations", component: Operations },
    { id: "core-index-close", component: Close },
    { id: "my-tasks", component: Tasks },
  ].map((r) => ({ ...r, title: screens.find((s) => s.id === r.id)!.title })),
};

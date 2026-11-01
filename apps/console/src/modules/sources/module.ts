import type { Module } from "../../shared/types";
import Screen from "./pages/index-intake/Screen";
import { titles } from "../../shared/copy";
export const sources: Module = {
  id: "sources",
  version: 1,
  requires: [],
  routes: [
    { id: "index-intake", title: titles["index-intake"], component: Screen },
  ],
  slots: ["sources/evidence/v1"],
};

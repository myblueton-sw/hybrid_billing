import type { Module } from "../../shared/types";
import { screens } from "../../shared/catalog";
import Screen from "./pages/review-issue/Screen";
export const issuance: Module = {
  id: "issuance",
  version: 1,
  requires: ["operations"],
  slots: [],
  routes: [
    {
      id: "review-issue",
      title: screens.find((s) => s.id === "review-issue")!.title,
      component: Screen,
    },
  ],
};

import type { Module } from "../../shared/types";
import { screens } from "../../shared/catalog";
import Screen from "./pages/customer-detail/Screen";
export const organizations: Module = {
  id: "organizations",
  version: 1,
  requires: ["operations"],
  slots: [],
  routes: [
    {
      id: "customer-detail",
      title: screens.find((s) => s.id === "customer-detail")!.title,
      component: Screen,
    },
  ],
};

import type { Module } from "../../shared/types";
import { screens } from "../../shared/catalog";
import { setupRoutes } from "./store";
import { makeSetupScreen } from "./SetupScreen";
export const onboarding: Module = {
  id: "onboarding",
  version: 1,
  requires: [],
  slots: [],
  routes: setupRoutes.map((id) => ({
    id,
    title: screens.find((s) => s.id === id)!.title,
    component: makeSetupScreen(id),
  })),
};

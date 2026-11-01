import type { Module } from "../../shared/types";
import Screen from "./pages/contents/Screen";
import { titles } from "../../shared/copy";
export const designReview: Module = {
  id: "design-review",
  version: 1,
  requires: [],
  routes: [{ id: "contents", title: titles.contents, component: Screen }],
  slots: [],
};

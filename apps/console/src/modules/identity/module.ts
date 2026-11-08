import type { Module } from "../../shared/types";
import { screens } from "../../shared/catalog";
import Start from "./pages/entry-start/Screen";
import Login from "./pages/entry-login/Screen";
import Bootstrap from "./pages/entry-bootstrap/Screen";
import Invite from "./pages/entry-invite/Screen";
import Challenge from "./pages/entry-challenge/Screen";
import Scope from "./pages/entry-scope/Screen";
export const identity: Module = {
  id: "identity",
  version: 1,
  requires: [],
  slots: [],
  routes: [
    { id: "entry-start", component: Start },
    { id: "entry-login", component: Login },
    { id: "entry-bootstrap", component: Bootstrap },
    { id: "entry-invite", component: Invite },
    { id: "entry-challenge", component: Challenge },
    { id: "entry-scope", component: Scope },
  ].map((r) => ({ ...r, title: screens.find((s) => s.id === r.id)!.title })),
};

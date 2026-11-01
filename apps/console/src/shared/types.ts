import type { ComponentType } from "react";
export type Locale = "ko" | "en" | "ar";
export type Copy = Record<Locale, string>;
export type Mode = "unconnected" | "synthetic";
export type ScreenProps = {
  locale: Locale;
  mode: Mode;
  profile: string;
  extensionIds: string[];
  contributions: Extension[];
  onState: (state: string, items: string[]) => void;
};
export type Route = {
  id: string;
  title: Copy;
  component: ComponentType<ScreenProps>;
};
export type Module = {
  id: string;
  version: number;
  requires: string[];
  routes: Route[];
  slots: string[];
};
export type Extension = {
  id: string;
  version: number;
  module: string;
  slot: string;
  slotVersion: number;
  render: ComponentType<{ locale: Locale }>;
};
export type Profile = { id: string; modules: string[]; extensions: string[] };

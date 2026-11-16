import type { Locale } from "../shared/types";
import { c } from "./copy";

export function serviceMode(loc: Pick<Location, "pathname" | "search">): boolean {
  return loc.pathname === "/activate" || new URLSearchParams(loc.search).get("service") === "api";
}

export function localeOf(search: string): Locale {
  const v = new URLSearchParams(search).get("lang");
  return v === "en" || v === "ar" ? v : "ko";
}

export function shellLocale(hash: string, fallback: Locale): Locale {
  const q = new URLSearchParams(hash.split("?")[1] ?? "");
  return q.has("lang") ? localeOf("?" + q.toString()) : fallback;
}

export function serviceUrl(view: "signin" | "help", l: Locale, status = ""): string {
  const q = new URLSearchParams({ service: "api", lang: l });
  if (view === "help") q.set("view", "help");
  if (status) q.set("status", status);
  return "/?" + q.toString();
}

export function utf8Length(s: string): number {
  return new TextEncoder().encode(s).length;
}

export function asciiLower(s: string): string {
  return s.replace(/[A-Z]/g, (ch) => ch.toLowerCase());
}

export function loginValid(login: string): boolean {
  const n = utf8Length(login);
  return n >= 3 && n <= 200 && !/[\u0000-\u001f\u007f-\u009f]/.test(login);
}

export function emailValid(email: string): boolean {
  return email.length > 0 && utf8Length(email) <= 254 && email.includes("@");
}

export function passwordProblem(l: Locale, password: string, login: string): string {
  if (!password) return c.enterNew(l);
  if (utf8Length(password) < 12) return c.tooShort(l);
  if (utf8Length(password) > 256) return c.tooLong(l);
  if (password === asciiLower(login)) return c.sameAsLogin(l);
  return "";
}

export function afterSignIn(hash: string, l: Locale): string {
  const [path, query = ""] = hash.replace(/^#/, "").split("?");
  const target = path.startsWith("/") ? path : "/home";
  const q = new URLSearchParams(query);
  q.set("lang", l);
  return "#" + target + "?" + q.toString();
}

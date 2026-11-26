import type { ReactNode } from "react";

export function breakable(login: string): ReactNode {
  const at = login.indexOf("@");
  return at < 0 ? login : <>{login.slice(0, at)}<wbr />{login.slice(at)}</>;
}

export function snakeBreakable(name: string): ReactNode {
  const parts = name.split("_");
  return parts.map((part, i) => (i === 0 ? part : <span key={i}>_<wbr />{part}</span>));
}

export function withWho(text: string, mark: string, who: string): ReactNode {
  const at = text.indexOf(mark);
  if (at < 0) return text;
  return <>{text.slice(0, at)}<bdi dir="ltr">{breakable(who)}</bdi>{text.slice(at + mark.length)}</>;
}

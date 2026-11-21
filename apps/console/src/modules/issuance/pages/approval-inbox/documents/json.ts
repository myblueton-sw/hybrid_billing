import { DocumentError } from "./error";
export function strictJSON(text: string, maximum = 100000): unknown {
  const bad = (): never => { throw new DocumentError("invalid_document_response"); };
  if (new TextEncoder().encode(text).length > maximum) return bad();
  let position = 0, nodes = 0;
  const space = () => { while (/[\x20\t\r\n]/.test(text[position] ?? "x")) position++; };
  function string(): string {
    const match = /^"(?:[^"\\\x00-\x1f]|\\(?:["\\/bfnrt]|u[0-9a-fA-F]{4}))*"/.exec(text.slice(position));
    if (!match) return bad();
    position += match[0].length;
    const result = JSON.parse(match[0]) as string;
    if ([...result].some(c => c.length === 1 && c.charCodeAt(0) >= 0xd800 && c.charCodeAt(0) <= 0xdfff)) return bad();
    return result;
  }
  function value(depth: number): unknown {
    space();
    if (depth > 16 || ++nodes > 10000) return bad();
    const token = text[position];
    if (token === '"') return string();
    if (token === "{" || token === "[") {
      position++; space();
      const array = token === "[", end = array ? "]" : "}";
      const out: unknown[] | Record<string, unknown> = array ? [] : Object.create(null);
      if (text[position] === end) { position++; return out; }
      for (;;) {
        space();
        if (array) (out as unknown[]).push(value(depth + 1));
        else {
          if (text[position] !== '"') return bad();
          const key = string();
          if (Object.hasOwn(out, key)) return bad();
          space(); if (text[position++] !== ":") return bad();
          (out as Record<string, unknown>)[key] = value(depth + 1);
        }
        space();
        if (text[position] === end) { position++; return out; }
        if (text[position++] !== ",") return bad();
      }
    }
    const literal = /^(?:true|false|null)/.exec(text.slice(position));
    if (literal) { position += literal[0].length; return JSON.parse(literal[0]); }
    const number = /^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?/.exec(text.slice(position));
    if (!number) return bad();
    position += number[0].length;
    const result = Number(number[0]);
    if (!Number.isFinite(result)) return bad();
    return result;
  }
  const result = value(0); space();
  if (position !== text.length) return bad();
  return result;
}

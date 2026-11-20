export class ReadParseError extends Error {}
export type Parser<T> = (value: unknown) => T;
export function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new ReadParseError();
  return value as Record<string, unknown>;
}
export const text: Parser<string> = (value) => {
  if (typeof value !== "string") throw new ReadParseError();
  return value;
};
export const boolean: Parser<boolean> = (value) => {
  if (typeof value !== "boolean") throw new ReadParseError();
  return value;
};
export const integer: Parser<number> = (value) => {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) throw new ReadParseError();
  return value;
};
export const positive: Parser<number> = (value) => {
  const parsed = integer(value);
  if (parsed === 0) throw new ReadParseError();
  return parsed;
};
export function choice<const T extends string>(...values: T[]): Parser<T> {
  return (value) => {
    const found = values.find((item) => item === value);
    if (found === undefined) throw new ReadParseError();
    return found;
  };
}
export function array<T>(parse: Parser<T>): Parser<T[]> {
  return (value) => {
    if (!Array.isArray(value)) throw new ReadParseError();
    return value.map(parse);
  };
}
export function optional<T>(parse: Parser<T>): Parser<T | undefined> {
  return (value) => value === undefined ? undefined : parse(value);
}
export function shape<T>(fields: { [K in keyof T]-?: Parser<T[K]> }): Parser<T> {
  return (value) => {
    const source = record(value);
    const result = {} as T;
    for (const key in fields) result[key] = fields[key](source[key]);
    return result;
  };
}
export const strings = array(text);

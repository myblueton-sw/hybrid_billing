import { issues, profile, type Prepared, type Draft } from "./model";
type Store = Pick<Storage, "getItem" | "setItem" | "removeItem">;
const keys = (v: unknown, names: string[]) =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === names.length &&
  names.every((n) => Object.hasOwn(v, n));
export const storageKey = (source: Prepared) =>
  `hb-b03-draft-v1:${source.context}`;
export function restore(store: Store, source: Prepared): Draft | null {
  const raw = store.getItem(storageKey(source));
  if (raw === null) return null;
  if (raw.length > 32768) throw Error("draft_invalid");
  const v = JSON.parse(raw);
  if (
    !keys(v, ["version", "profile", "context", "draft"]) ||
    v.version !== 1 ||
    v.profile !== profile ||
    v.context !== source.context ||
    !keys(v.draft, ["name", "weights"])
  )
    throw Error("draft_invalid");
  const d = v.draft;
  if (
    typeof d.name !== "string" ||
    d.name.length > 120 ||
    !Array.isArray(d.weights) ||
    d.weights.length > 100 ||
    d.weights.some((w: unknown) => !keys(w, ["target", "percentage"]))
  )
    throw Error("draft_invalid");
  for (const w of d.weights)
    if (
      typeof w.target !== "string" ||
      !/^[a-f0-9]{64}$/.test(w.target) ||
      typeof w.percentage !== "string" ||
      w.percentage.length > 8
    )
      throw Error("draft_invalid");
  if (issues(d, source, true).length) throw Error("draft_invalid");
  return d;
}
export function save(store: Store, source: Prepared, draft: Draft) {
  if (issues(draft, source, true).length) throw Error("draft_invalid");
  const text = JSON.stringify({
    version: 1,
    profile,
    context: source.context,
    draft: {
      name: draft.name,
      weights: draft.weights.map((w) => ({
        target: w.target,
        percentage: w.percentage,
      })),
    },
  });
  if (text.length > 32768) throw Error("draft_invalid");
  store.setItem(storageKey(source), text);
  if (store.getItem(storageKey(source)) !== text)
    throw Error("draft_save_failed");
}

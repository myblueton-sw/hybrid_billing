import type { OverviewData } from "../../../sources/collected-focus/api";
import { digest } from "../../../../shared/review-store";
import { categories, issues, type Draft } from "./model";
export type Context = {
  company: string | null;
  source: string;
  selection: string;
  query: string;
  after: number;
  accountAfter: number;
};
type Store = Pick<Storage, "getItem" | "setItem" | "removeItem">;
const fields = (value: unknown, keys: string[]) =>
  !!value &&
  typeof value === "object" &&
  !Array.isArray(value) &&
  Object.keys(value).length === keys.length &&
  keys.every((k) => Object.hasOwn(value, k));
export async function context(data: OverviewData): Promise<Context> {
  return {
    company: data.company,
    source: data.source_snapshot,
    selection: data.snapshot,
    query: await digest(
      JSON.stringify({
        company: data.company,
        selection: data.selection,
        after: data.after,
        accountAfter: data.account_after,
      }),
    ),
    after: data.after,
    accountAfter: data.account_after,
  };
}
function assertContext(ctx: Context, data: OverviewData) {
  if (
    ctx.company !== data.company ||
    ctx.source !== data.source_snapshot ||
    ctx.selection !== data.snapshot ||
    ctx.after !== data.after ||
    ctx.accountAfter !== data.account_after ||
    !/^[a-f0-9]{64}$/.test(ctx.query)
  )
    throw Error("draft_context_changed");
}
export const storageKey = (ctx: Context) => `hb-b01-review-v1:${ctx.query}`;
export function restore(
  store: Store,
  ctx: Context,
  data: OverviewData,
): Draft | null {
  assertContext(ctx, data);
  const raw = store.getItem(storageKey(ctx));
  if (raw === null) return null;
  if (raw.length > 32768) throw Error("draft_invalid");
  const body = JSON.parse(raw);
  if (
    !fields(body, ["version", "context", "draft"]) ||
    body.version !== 1 ||
    !fields(body.context, [
      "company",
      "source",
      "selection",
      "query",
      "after",
      "accountAfter",
    ]) ||
    JSON.stringify(body.context) !== JSON.stringify(ctx)
  )
    throw Error("draft_context_changed");
  const d = body.draft;
  if (
    !fields(d, ["name", "entries"]) ||
    typeof d.name !== "string" ||
    d.name.length > 120 ||
    !Array.isArray(d.entries) ||
    d.entries.length > 25 ||
    d.entries.some(
      (e: unknown) => !fields(e, ["row", "category", "parent", "reference"]),
    )
  )
    throw Error("draft_invalid");
  for (const e of d.entries)
    if (
      typeof e.row !== "string" ||
      !/^[a-f0-9]{64}$/.test(e.row) ||
      !categories.includes(e.category) ||
      !(
        e.parent === null ||
        (typeof e.parent === "string" && /^[a-f0-9]{64}$/.test(e.parent))
      ) ||
      typeof e.reference !== "string" ||
      e.reference.length > 300
    )
      throw Error("draft_invalid");
  if (issues(d, data, true).length) throw Error("draft_invalid");
  return d;
}
export function save(
  store: Store,
  ctx: Context,
  draft: Draft,
  data: OverviewData,
) {
  assertContext(ctx, data);
  if (issues(draft, data, true).length) throw Error("draft_invalid");
  const text = JSON.stringify({
    version: 1,
    context: {
      company: ctx.company,
      source: ctx.source,
      selection: ctx.selection,
      query: ctx.query,
      after: ctx.after,
      accountAfter: ctx.accountAfter,
    },
    draft: {
      name: draft.name,
      entries: draft.entries.map((e) => ({
        row: e.row,
        category: e.category,
        parent: e.parent,
        reference: e.reference,
      })),
    },
  });
  if (text.length > 32768) throw Error("draft_bound");
  store.setItem(storageKey(ctx), text);
  if (store.getItem(storageKey(ctx)) !== text) throw Error("draft_save_failed");
}
export function clear(store: Store, ctx: Context) {
  store.removeItem(storageKey(ctx));
  if (store.getItem(storageKey(ctx)) !== null)
    throw Error("draft_clear_failed");
}

import { AdminError } from "../../../../service/admin";
import type { RecoveryList } from "../../../../service/recovery";
import { useMoreRows, type MoreRows } from "../../../../shared/more-rows";
import type { Locale } from "../../../../shared/types";
import { k } from "./live-copy";
import type { Shared } from "./LiveCommand";

export function useRows<T>(shared: Shared, installation: string, list: RecoveryList, first: T[] | null, next: string | null, of: (r: Record<string, unknown>) => T, idOf: (row: T) => string): MoreRows<T> {
  return useMoreRows<T>(shared.page.revision + "\u0000" + installation + "\u0000" + list, first, installation ? next : null, (after) => shared.api.recoveryRows(installation, list, after, of), idOf, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  });
}

export function RowsFoot<T>({ l, kind, rows, total, moves, more }: { l: Locale; kind: string; rows: MoreRows<T>; total: number | null; moves: boolean; more: string }) {
  const failed = rows.status === "failed";
  const shown = rows.rows?.length ?? 0;
  const c = "p03-" + kind;
  const cut = total !== null && shown > 0 && shown < total;
  if (!cut && !rows.reset && !rows.more && !failed) return null;
  return (
    <div className={"p03-rows-foot " + c + "-foot"}>
      {cut && <p className={"p03-small live-quiet " + c + "-count"}>{k.rowsShown(l, shown, total)}</p>}
      {moves && (cut || rows.more) && <p className={"p03-small live-quiet " + c + "-moves"}>{k.rowsMove(l)}</p>}
      {rows.reset && <p className={"p03-small " + c + "-reset"} role="status">{k.rowsReset(l)}</p>}
      {(rows.more || failed) && (
        <div className={"p03-lock-more " + c + "-more"}>
          {failed && <p role="alert">{k.rowsFailed(l)}</p>}
          <button type="button" className={"live-secondary p03-lock-more-button " + c + "-more-button"} aria-disabled={rows.status === "loading" || undefined} onClick={() => { if (rows.status !== "loading") rows.load(); }}>
            {failed ? k.lockTryAgain(l) : rows.status === "loading" ? k.lockLoading(l) : more}
          </button>
        </div>
      )}
    </div>
  );
}

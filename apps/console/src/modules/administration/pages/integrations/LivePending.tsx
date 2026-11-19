import type { OwnPending } from "../../../../service/integrations";
import { k } from "./live-copy";
import { ltr } from "./live-format";
import { CommandForm, RowNote, useOpen, type Shared } from "./LiveCommand";
import { lostRight, ownPendingFor, pendingCommand } from "./live-rights";
import { pendingNow, refusedNow } from "./live-model";
import { r } from "./rights-copy";

function Item({ p, ...shared }: Shared & { p: OwnPending }) {
  const { l } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<"withdraw">();
  const command = pendingCommand[p.kind];
  const kind = r.pendingKind(l, p.kind);
  const usable = shared.canCommand;
  const lost = lostRight(command, p.target);
  const kept = lost ? r.noLongerHeld(l, lost) : refusedNow(command, p.target) ? k.kept(l, "proposal") : "";
  return (
    <li className="p02-proposal" id={"p02-own-" + p.id}>
      <div className="p02-meta">
        <p>{kind} · <bdi className="live-mono">{p.id}</bdi> <span className="p02-tag">{r.pendingTag(l)}</span></p>
      </div>
      <div className="p02-body">
        <p>{r.forTarget(l)} {ltr(p.object || p.target)}</p>
        <RowNote kept={kept} blockers={[]} focus={focusNote} />
        {usable && !open && (
          <div className="live-buttons">
            <button
              type="button" className="live-secondary" aria-expanded={false} aria-label={r.withdrawName(l, kind, p.id)}
              data-opener={mark("withdraw")} onClick={() => toggle("withdraw")}
            >
              {k.withdraw(l)}
            </button>
          </div>
        )}
        {open && (
          <CommandForm
            {...shared} command={command} target={p.target} values={{}}
            title={r.confirmWithdraw(l)}
            facts={
              <>
                <div><dt>{kind}</dt><dd><bdi className="live-mono">{p.id}</bdi></dd></div>
                <div><dt>{r.forTarget(l)}</dt><dd>{ltr(p.object || p.target)}</dd></div>
              </>
            }
            impact={p.kind === "plugin_proposal" ? k.withdrawImpact(l) : r.withdrawImpact(l)}
            verb={r.withdrawName(l, kind, p.id)}
            done={() => r.withdrawnDone(l, kind, p.id)}
            onClose={close}
          />
        )}
        {!open && pendingNow(command, p.target) && <p className="live-note-rule" role="status">{k.pendingAttempt(l)}</p>}
      </div>
    </li>
  );
}

export function OwnPendingList({ view, ...shared }: Shared & { view: string }) {
  const { l, page } = shared;
  const items = ownPendingFor(page, view);
  if (items.length === 0) return null;
  return (
    <section className="p02-own" aria-labelledby="p02-own-title">
      <h3 id="p02-own-title">{r.ownTitle(l)}</h3>
      <p className="p02-small">{r.ownLead(l)}</p>
      <ul className="p02-list">{items.map((p) => <Item key={p.target} p={p} {...shared} />)}</ul>
    </section>
  );
}

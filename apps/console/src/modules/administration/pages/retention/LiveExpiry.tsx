import type { Locale } from "../../../../shared/types";
import { standingKeys, type ArtifactRow, type ClassGroup, type Counts } from "../../../../service/retention";
import { counted, plurals } from "./live-codes";
import { k } from "./live-copy";
import { artifactTarget, blockerPolicy, decisionLine, keptKey, pendingNow, refusedNow, rowId, runNumber } from "./live-model";
import { Code, ListOr, MoreLine, Rich, Stamp, href, ltr } from "./live-format";
import { RowNote, useOpen, type Shared } from "./LiveCommand";
import { HoldCommand, itemContext } from "./LiveHoldForm";

export function Strip({ l, counts, keys, label, cls }: { l: Locale; counts: Counts | null; keys: string[]; label: (l: Locale, key: string) => string; cls: string }) {
  return (
    <dl className={"p04-strip " + cls}>
      {keys.map((key) => {
        const n = counts ? counts[key] : null;
        return (
          <div key={key}>
            <dt>{label(l, key)}</dt>
            <dd>{n === null || n === undefined ? <span className="p04-word">{k.unknown(l)}</span> : <bdi dir="ltr">{n}</bdi>}</dd>
          </div>
        );
      })}
    </dl>
  );
}

export function DecisionSub({ l, decision, needed }: { l: Locale; decision: string; needed: boolean }) {
  const d = decisionLine(decision, needed);
  if (d.kind === "none") return null;
  const text = d.kind === "not_connected" ? k.decision(l, "not_connected") : d.kind === "code" ? k.decisionCode(l, d.code) : k.decisionUnknown(l);
  return <span className="live-sub p04-small-text"><Rich text={text} /></span>;
}

function ItemBody({ a, group, ...shared }: Shared & { a: ArtifactRow; group: ClassGroup }) {
  const { l, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<"hold">();
  const target = artifactTarget(a);
  const pending = target ? pendingNow("retention.place_hold", target) : undefined;
  const usable = canCommand && !!target && (a.holdable || !!pending);
  const refused = target ? refusedNow("retention.place_hold", target) : "";
  const kept = refused ? k.kept(l, keptKey("retention.place_hold", refused)) : "";
  const current = selection.artifact !== "" && selection.artifact === a.id;
  const n = runNumber(a.run);
  const blocked = !target
    ? a.holdable || a.hold_blocked !== "target_too_long" ? k.narrow(l, "no_command") : k.blocker(l, a.hold_blocked)
    : !usable && !a.holdable && blockerPolicy(a.hold_blocked, "place_hold") === "explain" ? k.blocker(l, a.hold_blocked) : "";
  return (
    <tbody className={"p04-row" + (current ? " live-selected" : "")} id={a.id ? rowId("item", a.id) : undefined}>
      <tr aria-current={current || undefined}>
        <td className="p04-first">
          {a.id ? ltr(a.id) : k.unknown(l)}
          {a.shared === true && <span className="live-sub p04-small-text">{k.shared(l)}</span>}
          {a.shared === null && <span className="live-sub p04-small-text">{k.shared(l)}: {k.unknown(l)}</span>}
        </td>
        <td data-label={k.colDue(l)}>
          {a.due_at === null ? k.never(l) : <Stamp l={l} at={a.due_at} />}
          <span className="live-sub live-quiet p04-small-text">{k.start(l)} <Stamp l={l} at={a.anchor} /></span>
        </td>
        <td data-label={k.colStores(l)}>
          <ListOr l={l} items={a.stores} empty={k.none(l)}>{(list) => list.map((s) => <span key={s} className="live-sub">{ltr(s)}</span>)}</ListOr>
        </td>
        <td data-label={k.colAccess(l)}>
          <Code l={l} value={a.access} say={k.access} />
          <DecisionSub l={l} decision={a.decision} needed={a.access !== "ok"} />
        </td>
        <td data-label={k.colStanding(l)}>
          <Code l={l} value={a.standing} say={k.standing} />
          {(a.held === true && a.standing !== "held") || a.damaged === true ? (
            <span className="live-sub p04-tags">
              {a.held === true && a.standing !== "held" && <span className="p04-tag">{k.standing(l, "held")}</span>}
              {a.damaged === true && <span className="p04-tag">{k.tagDamaged(l)}</span>}
            </span>
          ) : null}
        </td>
        <td data-label={k.colRun(l)}>
          {a.run && n && a.other_scope !== true ? (
            <a className="p04-link" href={href(l, "deletion", { run: n })}>{ltr(a.run.label)}</a>
          ) : a.other_scope === true ? (
            <span className="p04-small-text">{k.runOutside(l)}</span>
          ) : a.run ? (
            k.unknown(l)
          ) : a.other_scope === null ? (
            k.unknown(l)
          ) : (
            k.none(l)
          )}
        </td>
        <td className="p04-action">
          {usable && !open ? (
            <button type="button" className="live-secondary p04-opener" aria-expanded={false} data-opener={mark("hold")} onClick={() => toggle("hold")}>{k.placeHold(l)}</button>
          ) : blocked && !open ? (
            <span className="p04-small-text"><Rich text={blocked} /></span>
          ) : null}
        </td>
      </tr>
      {(open || kept || focusNote > 0) && (
        <tr className="p04-follow">
          <td colSpan={7}>
            <RowNote kept={kept} blockers={[]} focus={focusNote} extra={!open && focusNote > 0 && blocked ? <p><Rich text={blocked} /></p> : undefined} />
            {open && usable && (
              <HoldCommand {...shared} target={target} tenant={false} base={itemContext(a, group.class)} heading={k.holdHeading(l, a.id)} onClose={close} />
            )}
          </td>
        </tr>
      )}
    </tbody>
  );
}

function Group({ g, ...shared }: Shared & { g: ClassGroup }) {
  const { l, selection } = shared;
  const head = "p04-class-h-" + (g.class || "x");
  const current = selection.cls === g.class && !selection.artifact;
  const cls = k.className(l, g.class);
  return (
    <section className={"p04-class" + (current ? " live-selected" : "")} id={g.class ? rowId("class", g.class) : undefined} aria-labelledby={head} aria-current={current || undefined}>
      <div className="p04-head">
        <h3 id={head}><Code l={l} value={g.class} say={k.className} /></h3>
        <span className="p04-muted">{g.total === null ? k.unknown(l) : counted(l, g.total, plurals.items)}</span>
      </div>
      <Strip l={l} counts={g.counts} keys={standingKeys} label={k.standing} cls="p04-strip-7" />
      <p className="p04-small live-quiet p04-usage">
        {k.usage(l)}: <Code l={l} value={g.usage} say={k.connection} /> · {k.growth(l)}: <Code l={l} value={g.growth} say={k.connection} /> · {k.bytes(l)}:{" "}
        <Code l={l} value={g.bytes} say={k.connection} />
      </p>
      {g.population !== "registered_sample_artifacts" && <p className="p04-small"><Rich text={k.populationNote(l, g.population)} /></p>}
      <MoreLine l={l} more={g.more} total={g.total} forms={plurals.showingOf} />
      {g.artifacts === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : g.artifacts.length === 0 ? (
        <p className="live-empty">{g.total === 0 ? k.noItems(l) : k.unknown(l)}</p>
      ) : (
        <div className="live-table">
          <table className="p04-table p04-wide p04-items">
            <caption className="visually-hidden">{k.itemsCaption(l, cls)}</caption>
            <colgroup>
              <col className="p04-c18" /><col className="p04-c16" /><col className="p04-c16" /><col className="p04-c14" /><col className="p04-c14" /><col className="p04-c8" /><col className="p04-c14" />
            </colgroup>
            <thead>
              <tr>
                <th scope="col">{k.colItem(l)}</th>
                <th scope="col">{k.colDue(l)}</th>
                <th scope="col">{k.colStores(l)}</th>
                <th scope="col">{k.colAccess(l)}</th>
                <th scope="col">{k.colStanding(l)}</th>
                <th scope="col">{k.colRun(l)}</th>
                <th scope="col">{k.colAction(l)}</th>
              </tr>
            </thead>
            {g.artifacts.map((a, n) => <ItemBody key={a.id || n} a={a} group={g} {...shared} />)}
          </table>
        </div>
      )}
    </section>
  );
}

export function ExpiryView(shared: Shared) {
  const { l, page } = shared;
  const groups = page.expiry;
  const population = groups?.find((g) => g.population)?.population ?? "";
  return (
    <>
      <div className="p04-notes">
        <p><Rich text={k.populationNote(l, population)} /></p>
        <p>{k.decisionNote(l)}</p>
      </div>
      {groups === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : (
        <>
          <nav className="p04-jumps" aria-label={k.jumpsLabel(l)}>
            {groups.map((g) => (
              <a key={g.class} className="p04-link" href={href(l, "expiry", { class: g.class })}>
                <Code l={l} value={g.class} say={k.className} /> ({g.total === null ? k.unknown(l) : <bdi dir="ltr">{g.total}</bdi>})
              </a>
            ))}
          </nav>
          {groups.map((g, n) => <Group key={g.class || n} g={g} {...shared} />)}
        </>
      )}
    </>
  );
}

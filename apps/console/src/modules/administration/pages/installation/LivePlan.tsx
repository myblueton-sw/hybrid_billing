import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { InstallationPage, InstallItem, InstallPreview, InstallRelease } from "../../../../service/installation";
import { k } from "./live-copy";
import { actor, latestAcceptance, listCount, notApplicableOf, observationBlocks, previewBasis, showsEffects } from "./live-model";
import { Code, Known, fsi, ltr, stamp } from "./live-format";
import { Proposals, Propose, type Shared } from "./LiveProposals";

function List<T>({ l, items, render }: { l: Locale; items: T[] | null; render: (x: T) => ReactNode }) {
  if (items === null) return <p className="p01-value">{k.unknown(l)}</p>;
  if (items.length === 0) return <p className="p01-value">{k.none(l)}</p>;
  return <ul className="p01-effect-list">{items.map((x, i) => <li key={i}>{render(x)}</li>)}</ul>;
}

function Item({ l, i }: { l: Locale; i: InstallItem }) {
  return (
    <>
      <Known l={l} value={i.target} /> — <Code l={l} value={i.action} say={k.action} />
      {i.conflict && <span className="live-sub">{k.conflicts(l)}: <Code l={l} value={i.conflict} say={k.conflict} /></span>}
    </>
  );
}

function Conflicts({ l, v }: { l: Locale; v: InstallPreview }) {
  return (
    <div className="p01-conflicts">
      <h4>{k.conflicts(l)}</h4>
      {v.conflicts === null ? (
        <p className="p01-value">{k.unknown(l)}</p>
      ) : v.conflicts.length === 0 ? (
        <p className="p01-value">{k.none(l)}</p>
      ) : (
        <>
          <p><strong>{k.conflictLead(l)}</strong></p>
          <ul className="p01-effect-list">
            {v.conflicts.map((c, i) => (
              <li key={i}><Known l={l} value={c.target} /> — <Code l={l} value={c.conflict} say={k.conflict} /></li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

function Effects({ l, v }: { l: Locale; v: InstallPreview }) {
  const section = (title: string, body: ReactNode) => (
    <section className="p01-effect">
      <h4>{title}</h4>
      {body}
    </section>
  );
  return (
    <div className="p01-effects">
      {section(k.services(l), <List l={l} items={v.services} render={(i) => <Item l={l} i={i} />} />)}
      {section(k.ports(l), <List l={l} items={v.ports} render={(i) => <Item l={l} i={i} />} />)}
      {section(k.volumes(l), <List l={l} items={v.volumes} render={(i) => <Item l={l} i={i} />} />)}
      {section(k.dataEffects(l), <List l={l} items={v.data_effects} render={(d) => <><Known l={l} value={d.volume} /> — <Code l={l} value={d.effect} say={k.effect} /></>} />)}
      {section(k.permissions(l), <List l={l} items={v.permissions} render={(p) => <Code l={l} value={p} say={k.permission} />} />)}
      {section(k.leftovers(l), <List l={l} items={v.leftovers} render={(i) => <Item l={l} i={i} />} />)}
    </div>
  );
}

function availability(l: Locale, v: InstallPreview, installed: string): ReactNode {
  if (v.state === "available") return k.available(l);
  if (v.state === "unavailable") return v.reason ? k.notAvailable(l, v.reason) : k.unknown(l);
  if (v.state === "not_applicable") return k.notApplicable(l, notApplicableOf(v.kind === "upgrade" || v.kind === "rollback" ? v.kind : "install", installed));
  return <Code l={l} value={v.state} say={() => k.unknownCode(l, v.state)} />;
}

function Head({ l, v, installed }: { l: Locale; v: InstallPreview; installed: string }) {
  if (v.state === "not_applicable") return <p className="p01-value">{availability(l, v, installed)}</p>;
  return (
    <dl className="p01-plan-head">
      <div><dt>{k.version(l)}</dt><dd><Known l={l} value={v.version} /></dd></div>
      {(v.hash || v.state === "available") && <div className="p01-hash-fact"><dt>{k.hash(l)}</dt><dd className="p01-hash"><Known l={l} value={v.hash} /></dd></div>}
      <div><dt>{k.availability(l)}</dt><dd>{availability(l, v, installed)}</dd></div>
    </dl>
  );
}

function Pair({ l, page }: { l: Locale; page: InstallationPage }) {
  const cell = (title: string, v: InstallPreview) => (
    <div>
      <h4>{title}</h4>
      <Head l={l} v={v} installed={page.installed.state} />
      {showsEffects(v.state) && (
        <details className="p01-fold">
          <summary>{k.showEffects(l)}</summary>
          <Conflicts l={l} v={v} />
          <Effects l={l} v={v} />
        </details>
      )}
    </div>
  );
  return (
    <div className="p01-pair">
      {cell(k.upgrade(l), page.upgrade)}
      {cell(k.rollback(l), page.rollback)}
    </div>
  );
}

function Observation({ l, page }: { l: Locale; page: InstallationPage }) {
  const o = page.observation;
  return (
    <section className="p01-observation" aria-labelledby="p01-observation-title">
      <h3 id="p01-observation-title">{k.observation(l)} <span className="p01-tag"><Code l={l} value={o.state} say={k.obsTag} /></span></h3>
      {o.state === "missing" ? (
        <p>{k.obsMissing(l)}</p>
      ) : o.state === "fresh" || o.state === "stale" ? (
        <p>{k.observed(l, stamp(l, o.observed_at), stamp(l, o.expires_at))}</p>
      ) : (
        <p>{k.obsUnknown(l)}</p>
      )}
      {observationBlocks(o.state) && <p className="p01-blocks">{k.obsBlocks(l)}</p>}
      {o.state === "stale" && <p className="p01-small">{k.obsRestart(l)}</p>}
    </section>
  );
}

function count(l: Locale, v: unknown[] | number | null): string {
  if (typeof v === "number") return v === 0 ? k.none(l) : String(v);
  const c = listCount(v);
  return c === "unknown" ? k.unknown(l) : c === "none" ? k.none(l) : String(c);
}

function Installed({ l, page }: { l: Locale; page: InstallationPage }) {
  const i = page.installed;
  return (
    <section className="p01-installed" aria-labelledby="p01-installed-title">
      <h3 id="p01-installed-title">{k.installed(l)}</h3>
      {i.state === "not_installed" ? (
        <p>{k.notInstalled(l)}</p>
      ) : (
        <>
          <p><Known l={l} value={i.version} /> <span className="p01-tag"><Code l={l} value={i.state} say={k.installedTag} /></span></p>
          <dl className="live-facts">
            <div>
              <dt>{k.ownedServices(l)}</dt>
              <dd>
                {i.services === null ? k.unknown(l) : i.services.length === 0 ? k.none(l) : i.services.map((s, n) => (
                  <span key={n} className="live-sub"><Known l={l} value={s.name} />{s.ports.length > 0 && <> ({ltr(s.ports.join(", "))})</>}</span>
                ))}
              </dd>
            </div>
            <div>
              <dt>{k.ownedVolumes(l)}</dt>
              <dd>
                {i.volumes === null ? k.unknown(l) : i.volumes.length === 0 ? k.none(l) : i.volumes.map((v, n) => (
                  <span key={n} className="live-sub">
                    <Known l={l} value={v.host} /> — {v.data === null ? k.dataUnknown(l) : v.data ? k.containsData(l) : k.noData(l)}{v.detached ? ", " + k.detached(l) : ""}
                  </span>
                ))}
              </dd>
            </div>
            <div><dt>{k.history(l)}</dt><dd>{count(l, i.history)}</dd></div>
            <div><dt>{k.backups(l)}</dt><dd>{count(l, i.backups)}</dd></div>
          </dl>
        </>
      )}
    </section>
  );
}

function Mark({ l, r }: { l: Locale; r: InstallRelease }) {
  const m = r.mark;
  if (m === "unknown") return <>{k.unknown(l)}</>;
  if (!m) return <>{k.none(l)}</>;
  const by = actor(m.by_login, m.by) || k.unknown(l);
  const confirmed = actor(m.confirmed_by_login, m.confirmed_by);
  return (
    <>
      {k.markedBy(l, by)} {m.reason ? <bdi>{m.reason}</bdi> : k.unknown(l)}
      {confirmed && <span className="live-sub">{k.confirmedBy(l, confirmed)}</span>}
    </>
  );
}

function Releases({ l, page }: { l: Locale; page: InstallationPage }) {
  const words = (v: string[] | null) => (v === null ? k.unknown(l) : v.length === 0 ? k.none(l) : null);
  return (
    <section className="p01-releases" aria-labelledby="p01-releases-title">
      <h3 id="p01-releases-title">{k.releases(l)}</h3>
      {page.releases.length === 0 ? (
        <p className="live-empty">{k.noReleases(l)}</p>
      ) : (
        <div className="live-table">
          <table className="p01-table">
            <colgroup>
              <col className="p01-c10" /><col className="p01-c12" /><col className="p01-c12" /><col className="p01-c12" />
              <col className="p01-c22" /><col className="p01-c22" /><col className="p01-c10" />
            </colgroup>
            <thead>
              <tr>
                <th scope="col">{k.version(l)}</th><th scope="col">{k.keyVersion(l)}</th><th scope="col">{k.upgradeFrom(l)}</th><th scope="col">{k.platforms(l)}</th>
                <th scope="col">{k.services(l)}</th><th scope="col">{k.volumes(l)}</th><th scope="col">{k.mark(l)}</th>
              </tr>
            </thead>
            <tbody>
              {page.releases.map((r, n) => (
                <tr key={r.version || n}>
                  <td className="p01-first"><Known l={l} value={r.version} /></td>
                  <td data-label={k.keyVersion(l)}><Known l={l} value={r.key_version} /></td>
                  <td data-label={k.upgradeFrom(l)}>{words(r.upgrade_from) ?? ltr((r.upgrade_from ?? []).join(", "))}</td>
                  <td data-label={k.platforms(l)}>{words(r.platforms) ?? (r.platforms ?? []).map((p) => <span key={p} className="live-sub">{ltr(p)}</span>)}</td>
                  <td data-label={k.services(l)}>
                    {r.services === null ? k.unknown(l) : r.services.length === 0 ? k.none(l) : r.services.map((s, i) => {
                      const kind = r.artifacts?.find((a) => a.name === s.artifact)?.kind ?? "";
                      return (
                        <span key={i} className="p01-service">
                          <Known l={l} value={s.name} />
                          {s.ports.length > 0 && <span className="live-sub">{ltr(s.ports.join(", "))}</span>}
                          {kind && <span className="live-sub live-quiet"><Code l={l} value={kind} say={k.artifact} /></span>}
                        </span>
                      );
                    })}
                  </td>
                  <td data-label={k.volumes(l)}>
                    {r.volumes === null ? k.unknown(l) : r.volumes.length === 0 ? k.none(l) : r.volumes.map((v, i) => (
                      <span key={i} className="live-sub"><Known l={l} value={v.name} /> — {v.data === null ? k.dataUnknown(l) : v.data ? k.containsData(l) : k.noData(l)}</span>
                    ))}
                  </td>
                  <td data-label={k.mark(l)}><Mark l={l} r={r} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

export function PlanView(shared: Shared) {
  const { l, page } = shared;
  const v = page.preview;
  const basis = previewBasis(page);
  const accepted = latestAcceptance(page.proposals);
  return (
    <>
      <div className="p01-status">
        <Observation l={l} page={page} />
        <Installed l={l} page={page} />
      </div>
      <section className="p01-plan" aria-labelledby="p01-plan-title">
        <h3 id="p01-plan-title">{k.installPlan(l)}</h3>
        <Head l={l} v={v} installed={page.installed.state} />
        {basis && <p className="p01-small">{basis === "pending" ? k.basisPending(l) : k.basisRelease(l)}</p>}
        {accepted && (
          <p className="p01-small">
            {k.accepted(l, actor(accepted.approver_login || accepted.closed_by_login, accepted.approver || accepted.closed_by) || k.unknown(l))[0]}{" "}
            <Known l={l} value={accepted.summary.plan_version} />{" "}
            {k.accepted(l, actor(accepted.approver_login || accepted.closed_by_login, accepted.approver || accepted.closed_by) || k.unknown(l))[1]}
          </p>
        )}
        {showsEffects(v.state) && (
          <>
            <Conflicts l={l} v={v} />
            <Effects l={l} v={v} />
          </>
        )}
        <Pair l={l} page={page} />
        <Propose {...shared} />
      </section>
      <Proposals {...shared} />
      <Releases l={l} page={page} />
    </>
  );
}

export const now = (l: Locale, page: InstallationPage | null) => (page ? fsi(stamp(l, page.now)) : "—");

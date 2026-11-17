import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { PluginInstall, PluginVersion } from "../../../../service/integrations";
import { k, storageText } from "./live-copy";
import {
  blockerPolicy, capabilityOf, changeKey, frozenValues, installOf, keepDraft, draftOf, pendingNow, pluginIds, pluginTarget, proposeOffer, proposeValues, refusedNow, duplicateIn,
  type Change, type ChangeKind, changeKinds,
} from "./live-model";
import { Code, Known, ListOr, MarkText, Num, Prose, Who, fsi, ltr, stamp } from "./live-format";
import { CommandForm, RowNote, blockerLines, plain, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { Proposals } from "./LiveProposals";

function Registry({ l, page }: Shared) {
  const w = page.modules.workspace;
  const pubs = page.modules.publishers;
  return (
    <>
      <div className="p02-status">
        <section className="p02-workspace" aria-labelledby="p02-workspace-title">
          <h3 id="p02-workspace-title">{k.workspaceTitle(l)}</h3>
          {w === null ? (
            <p>{k.noWorkspace(l)}</p>
          ) : (
            <dl className="live-facts">
              <div><dt>{k.workspaceId(l)}</dt><dd><Known l={l} value={w.id} /></dd></div>
              <div><dt>{k.runtimeIdentity(l)}</dt><dd><Known l={l} value={w.runtime} /></dd></div>
              <div><dt>{k.verificationWs(l)}</dt><dd>{w.verification === null ? k.unknown(l) : w.verification ? k.yes(l) : k.no(l)}</dd></div>
            </dl>
          )}
        </section>
        <section className="p02-publishers" aria-labelledby="p02-publishers-title">
          <h3 id="p02-publishers-title">{k.publishersTitle(l)}</h3>
          <ListOr l={l} items={pubs} empty={k.noPublishers(l)}>
            {(list) =>
              list.map((p, i) => (
                <div key={p.id || i} className="p02-publisher">
                  <p className="p02-publisher-name"><Prose l={l} value={p.name} /> <span className="p02-muted"><Known l={l} value={p.id} /></span></p>
                  <p className="p02-small">{p.revocation === null ? k.notRevoked(l) : <MarkText l={l} m={p.revocation} lead={k.revokedBy} />}</p>
                  <ListOr l={l} items={p.keys} empty={k.noKeysPublisher(l)}>
                    {(keys) => (
                      <ul className="p02-effect-list">
                        {keys.map((key, n) => (
                          <li key={key.version || n}>
                            {k.keyLabel(l)} <Known l={l} value={key.version} /> · {k.validRange(l, stamp(l, key.not_before), stamp(l, key.not_after))} ·{" "}
                            {key.retired === null ? k.unknown(l) : key.retired ? k.retired(l) : k.current(l)} · {k.compromise(l)}: <MarkText l={l} m={key.compromise} lead={k.compromisedBy} />
                          </li>
                        ))}
                      </ul>
                    )}
                  </ListOr>
                </div>
              ))
            }
          </ListOr>
        </section>
      </div>
      <p className="p02-small live-quiet p02-band-note">{k.registeringNotConnected(l)}</p>
    </>
  );
}

function Endpoints({ l, i }: { l: Locale; i: PluginInstall }) {
  return (
    <>
      <h4>{k.endpointsTitle(l)}</h4>
      <ListOr l={l} items={i.endpoints} empty={k.noEndpoints(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c12" /><col className="p02-c58" /><col className="p02-c30" /></colgroup>
              <thead><tr><th scope="col">{k.colEndpoint(l)}</th><th scope="col">{k.colFingerprint(l)}</th><th scope="col">{k.colApprovedBy(l)}</th></tr></thead>
              <tbody>
                {list.map((e, n) => (
                  <tr key={e.id || n}>
                    <td className="p02-first"><Known l={l} value={e.id} /></td>
                    <td data-label={k.colFingerprint(l)} className="p02-wrap p02-digest"><Known l={l} value={e.fingerprint} /></td>
                    <td data-label={k.colApprovedBy(l)}><Who l={l} login={e.approver_login} id={e.approver} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
    </>
  );
}

type Runtime = "integrations.plugin_stop" | "integrations.plugin_disable";

function RuntimeActions({ i, ...shared }: Shared & { i: PluginInstall }) {
  const { l } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<Runtime>();
  const target = pluginTarget(i.plugin);
  const stopUsable = shared.canCommand && !!target && (i.can_stop || !!pendingNow("integrations.plugin_stop", target));
  const disableUsable = shared.canCommand && !!target && (i.can_disable || !!pendingNow("integrations.plugin_disable", target));
  const kept = target && (refusedNow("integrations.plugin_stop", target) || refusedNow("integrations.plugin_disable", target)) ? k.kept(l, "plugin") : "";
  const blockers: Blocker[] = [];
  if (!target) blockers.push(plain(k.noCommand(l)));
  if (target && !stopUsable && !i.can_stop && blockerPolicy(i.stop_blocked) === "explain") blockers.push(...blockerLines(l, i.stop_blocked, "stop", k.stop(l)));
  if (target && !disableUsable && !i.can_disable && blockerPolicy(i.disable_blocked) === "explain") blockers.push(...blockerLines(l, i.disable_blocked, "disable", k.disable(l)));
  return (
    <>
      <RowNote kept={kept} blockers={blockers} focus={focusNote} />
      {(stopUsable || disableUsable) && !open && (
        <div className="live-buttons">
          {stopUsable && (!open || open === "integrations.plugin_stop") && <button type="button" className="live-secondary" aria-expanded={open === "integrations.plugin_stop"} data-opener={mark("integrations.plugin_stop")} onClick={() => toggle("integrations.plugin_stop")}>{k.stop(l)}</button>}
          {disableUsable && (!open || open === "integrations.plugin_disable") && <button type="button" className="live-secondary" aria-expanded={open === "integrations.plugin_disable"} data-opener={mark("integrations.plugin_disable")} onClick={() => toggle("integrations.plugin_disable")}>{k.disable(l)}</button>}
        </div>
      )}
      {open && target && (
        <CommandForm
          key={open} {...shared} command={open} target={target} values={{}}
          title={open === "integrations.plugin_stop" ? k.confirmStop(l) : k.confirmDisable(l)}
          facts={
            <>
              <div><dt>{k.factPlugin(l)}</dt><dd><Known l={l} value={i.plugin} /></dd></div>
              <div><dt>{k.factInstalled(l)}</dt><dd><Known l={l} value={i.version} /> · <Code l={l} value={i.state} say={k.installState} /></dd></div>
            </>
          }
          impact={open === "integrations.plugin_stop" ? k.stopImpact(l) : k.disableImpact(l)}
          verb={open === "integrations.plugin_stop" ? k.stopVerb(l, i.plugin) : k.disableVerb(l, i.plugin)}
          done={(_, state) => k.runtimeDone(l, i.plugin, k.result(l, state))}
          onClose={close}
        />
      )}
    </>
  );
}

function ChangeText({ l, c }: { l: Locale; c: Change }) {
  const [before, after = ""] = k.change(l, c.kind, "\u0000").split("\u0000");
  return <>{before}{c.version ? ltr(c.version) : null}{after}</>;
}

function initialChange(plugin: string): Change | null {
  const target = pluginTarget(plugin);
  const frozen = frozenValues(pendingNow("integrations.plugin_propose", target));
  const from = frozen ?? draftOf("integrations.plugin_propose", target);
  const kind = from.kind as ChangeKind;
  if (!changeKinds.includes(kind)) return null;
  return { kind, version: from.version ?? "" };
}

function Propose({ plugin, ...shared }: Shared & { plugin: string }) {
  const { l, page, me } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<"propose">();
  const target = pluginTarget(plugin);
  const offer = proposeOffer(page, me, plugin);
  const kept = target ? pendingNow("integrations.plugin_propose", target) : undefined;
  const [choice, setChoice] = useState<Change | null>(() => initialChange(plugin));
  const frozen = frozenValues(kept);
  const frozenChange: Change | null = frozen && changeKinds.includes(frozen.kind as ChangeKind) ? { kind: frozen.kind as ChangeKind, version: frozen.version ?? "" } : null;
  const options = [...offer.options];
  if (frozenChange && !options.some((o) => changeKey(o) === changeKey(frozenChange))) options.push(frozenChange);
  const usable = shared.canCommand && !!target && options.length > 0 && (offer.options.length > 0 || !!kept);
  const chosen = choice && options.some((o) => changeKey(o) === changeKey(choice)) ? choice : null;
  const install = installOf(page, plugin);
  const refused = target ? refusedNow("integrations.plugin_propose", target) : false;
  const id = "p02-propose-" + plugin.replace(/[^A-Za-z0-9-]/g, "");
  const blocker = offer.blocker;
  const blockerText = !blocker ? "" : blocker.code === "duplicate" ? k.proposeBlocker(l, blocker.own ? "duplicate_own" : "duplicate", blocker.number) : k.proposeBlocker(l, blocker.code, "");
  const current = install ? (page.modules.plugins ?? []).find((v) => v.plugin === plugin && v.version === install.version) : undefined;

  const pickChange = (c: Change) => {
    setChoice(c);
    keepDraft("integrations.plugin_propose", target, proposeValues(c));
  };

  return (
    <section className="p02-propose" aria-label={k.proposeChange(l, plugin)}>
      <RowNote kept={refused ? k.kept(l, "propose_plugin") : ""} blockers={!usable && blockerText ? [plain(blockerText)] : []} focus={focusNote} />
      {usable && !open && (
        <div className="live-buttons">
          <button type="button" className="live-secondary" aria-expanded={open === "propose"} data-opener={mark("propose")} onClick={() => toggle("propose")}>{k.proposeChange(l, plugin)}</button>
        </div>
      )}
      {open && usable && (
        <CommandForm
          {...shared} command="integrations.plugin_propose" target={target} values={chosen ? proposeValues(chosen) : {}}
          title={k.confirmChange(l)}
          choices={
            <fieldset className="p02-choices" id={id + "-change"}>
              <legend>{k.changeLegend(l)}</legend>
              {options.map((o, n) => (
                <label key={changeKey(o)} className="p02-choice">
                  <input
                    type="radio" name={id + "-change"} id={n === 0 ? id + "-first" : undefined} checked={!!chosen && changeKey(chosen) === changeKey(o)}
                    onChange={() => pickChange(o)}
                  />
                  <span><ChangeText l={l} c={o} /></span>
                </label>
              ))}
            </fieldset>
          }
          choiceProblems={() => (chosen ? [] : [{ field: id + "-first", message: k.chooseChange(l) }])}
          facts={
            <>
              <div><dt>{k.factPlugin(l)}</dt><dd><Known l={l} value={plugin} /></dd></div>
              <div><dt>{k.changeLegend(l)}</dt><dd>{chosen ? <ChangeText l={l} c={chosen} /> : k.unknown(l)}</dd></div>
              {chosen?.version && <div><dt>{k.factVersion(l)}</dt><dd>{ltr(chosen.version)}</dd></div>}
              <div>
                <dt>{k.factInstalled(l)}</dt>
                <dd>{install ? <><Known l={l} value={install.version} /> · <Code l={l} value={install.state} say={k.installState} /></> : k.notInstalled(l)}</dd>
              </div>
              {chosen?.kind === "remove" && current && <div><dt>{k.storageAfterRemoval(l)}</dt><dd>{storageText(l, current.storage)}</dd></div>}
            </>
          }
          impact={k.proposeImpact(l)}
          verb={k.proposeVerb(l, chosen ? k.change(l, chosen.kind, chosen.version ? fsi(chosen.version) : "").trim() : "", plugin)}
          done={(fields, _, snapshot) => (duplicateIn(snapshot, fields.proposal ?? "") ? k.duplicateProposal(l, fields.proposal ?? "") : k.proposedDone(l, fields.proposal || k.unknown(l)))}
          onClose={close}
        />
      )}
    </section>
  );
}

function Installations(shared: Shared) {
  const { l, page } = shared;
  const ids = pluginIds(page);
  return (
    <section className="p02-section p02-installs" aria-labelledby="p02-installs-title">
      <h3 id="p02-installs-title">{k.installTitle(l)}</h3>
      {ids.length === 0 && page.modules.installations !== null && <p className="live-empty">{k.noVersions(l)}</p>}
      {ids.map((plugin) => {
        const i = installOf(page, plugin);
        const removed = i?.state === "removed";
        return (
          <article key={plugin} className="p02-install">
            <div className="p02-install-head">
              <span className="p02-install-name">{ltr(plugin)}</span>
              {i ? <span className="p02-install-version"><Known l={l} value={i.version} /></span> : null}
              {i ? <span className="p02-tag"><Code l={l} value={i.state} say={k.installState} /></span> : page.modules.installations === null ? <span>{k.unknown(l)}</span> : <span>{k.notInstalledPlugin(l)}</span>}
            </div>
            {i && (
              <>
                <dl className="p02-install-facts">
                  <div><dt>{k.colReason(l)}</dt><dd>{i.reason ? <Code l={l} value={i.reason} say={k.installReason} /> : k.none(l)}</dd></div>
                  <div><dt>{k.registryRevision(l)}</dt><dd className="p02-num"><Num l={l} n={i.revision} /></dd></div>
                  <div><dt>{k.tokens(l)}</dt><dd><Code l={l} value={i.tokens} say={k.connection} /></dd></div>
                  <div><dt>{k.runtime(l)}</dt><dd><Code l={l} value={i.runtime} say={k.connection} /></dd></div>
                </dl>
                <Endpoints l={l} i={i} />
                {removed && (
                  <>
                    <h4>{k.storageAfterRemoval(l)}</h4>
                    <p className="p02-small">{i.disposal ? <>{storageText(l, i.disposal)} · <Code l={l} value={i.disposal.migration} say={k.migration} /></> : k.unknown(l)}</p>
                  </>
                )}
                <RuntimeActions {...shared} i={i} />
              </>
            )}
            <Propose {...shared} plugin={plugin} />
          </article>
        );
      })}
    </section>
  );
}

function Signed({ l, keyVersion, publisher }: { l: Locale; keyVersion: string; publisher: string }) {
  const parts = k.signedTemplate(l).split(/(\{key\}|\{publisher\})/);
  return (
    <>
      {parts.map((part, n) =>
        part === "{key}" ? <Known key={n} l={l} value={keyVersion} /> : part === "{publisher}" ? <Known key={n} l={l} value={publisher} /> : <span key={n}>{part}</span>,
      )}
    </>
  );
}

function Capabilities({ l, v }: { l: Locale; v: PluginVersion }) {
  return (
    <ListOr l={l} items={v.capabilities}>
      {(list) =>
        list.map((c, n) => {
          const cap = capabilityOf(c);
          return <span key={c + n} className="live-sub">{"raw" in cap ? <>{k.unknown(l)} ({ltr(cap.raw)})</> : <>{k.capability(l, cap.kind)}: {ltr(cap.target)}</>}</span>;
        })
      }
    </ListOr>
  );
}

function Details({ l, v }: { l: Locale; v: PluginVersion }) {
  const num = (n: number | null, f: (l: Locale, n: number) => string) => (n === null ? k.unknown(l) : f(l, n));
  return (
    <details className="p02-fold">
      <summary>{k.limitsFold(l, fsi(v.version || k.unknown(l)))}</summary>
      <div className="p02-detail-grid">
        <section>
          <h4>{k.limits(l)}</h4>
          <ul className="p02-effect-list">
            <li>{k.timeoutLabel(l)}: {num(v.timeout_seconds, k.seconds)}</li>
            <li>{k.memoryLabel(l)}: {num(v.memory_mib, k.mib)}</li>
            <li>{k.concurrencyLabel(l)}: {num(v.concurrency, k.concurrent)}</li>
          </ul>
        </section>
        <section>
          <h4>{k.storageContract(l)}</h4>
          <ul className="p02-effect-list">
            <li>{k.namespace(l)}: <Known l={l} value={v.storage.namespace} /></li>
            <li>{k.owner(l)}: <Known l={l} value={v.storage.owner} /></li>
            <li>{storageText(l, v.storage)}</li>
            <li><Code l={l} value={v.storage.migration} say={k.migration} /></li>
          </ul>
        </section>
        <section>
          <h4>{k.screensTitle(l)}</h4>
          <ListOr l={l} items={v.screens} empty={k.noScreens(l)}>
            {(list) => (
              <ul className="p02-effect-list">
                {list.map((s, n) => (
                  <li key={s.id || n}>
                    <Prose l={l} value={s.label} /> · <Code l={l} value={s.component} say={k.component} />
                    <ListOr l={l} items={s.fields}>
                      {(fields) => fields.map((f, m) => <span key={f.name || m} className="live-sub"><Prose l={l} value={f.label} /> ({ltr(f.source || k.unknown(l))})</span>)}
                    </ListOr>
                  </li>
                ))}
              </ul>
            )}
          </ListOr>
        </section>
      </div>
    </details>
  );
}

function Versions({ l, page }: Shared) {
  const list = page.modules.plugins;
  return (
    <section className="p02-section p02-versions" aria-labelledby="p02-versions-title">
      <h3 id="p02-versions-title">{k.versionsTitle(l)}</h3>
      <ListOr l={l} items={list} empty={k.noVersions(l)}>
        {(rows) => (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c18" /><col className="p02-c22" /><col className="p02-c16" /><col className="p02-c14" /><col className="p02-c20" /><col className="p02-c10" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colPlugin(l)}</th><th scope="col">{k.colVersion(l)}</th><th scope="col">{k.colCapabilities(l)}</th>
                  <th scope="col">{k.colEgress(l)}</th><th scope="col">{k.colVerification(l)}</th><th scope="col">{k.colVulnerable(l)}</th>
                </tr>
              </thead>
              {rows.map((v, n) => (
                <tbody key={v.plugin + v.version + n} className="p02-group">
                  <tr>
                    <td className="p02-first"><Known l={l} value={v.plugin} /></td>
                    <td data-label={k.colVersion(l)}>
                      <Known l={l} value={v.version} />
                      <span className="live-sub p02-wrap p02-digest">{k.digest(l)} <Known l={l} value={v.digest} /></span>
                    </td>
                    <td data-label={k.colCapabilities(l)}><Capabilities l={l} v={v} /></td>
                    <td data-label={k.colEgress(l)}>
                      <span className="live-sub">{k.egress(l)}: <ListOr l={l} items={v.egress}>{(x) => x.map((e) => <span key={e} className="live-sub">{ltr(e)}</span>)}</ListOr></span>
                      <span className="live-sub">{k.endpointsLabel(l)}: <ListOr l={l} items={v.endpoints}>{(x) => x.map((e) => <span key={e} className="live-sub">{ltr(e)}</span>)}</ListOr></span>
                    </td>
                    <td data-label={k.colVerification(l)}>
                      <Signed l={l} keyVersion={v.key_version} publisher={v.publisher} />
                      <span className="live-sub">{k.core(l)} {ltr((v.core_min || k.unknown(l)) + " – " + (v.core_max || k.unknown(l)))}</span>
                    </td>
                    <td data-label={k.colVulnerable(l)}><MarkText l={l} m={v.vulnerable} lead={k.markedVulnerable} /></td>
                  </tr>
                  <tr className="p02-follow"><td colSpan={6}><Details l={l} v={v} /></td></tr>
                </tbody>
              ))}
            </table>
          </div>
        )}
      </ListOr>
    </section>
  );
}

export function ModulesView(shared: Shared) {
  return (
    <>
      <Registry {...shared} />
      <Installations {...shared} />
      <Proposals {...shared} />
      <Versions {...shared} />
    </>
  );
}

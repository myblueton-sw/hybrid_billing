import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient, OrgParty } from "../../../../service/admin";
import type { ContractDetail, ContractEvent } from "../../../../service/contracts";
import type { Session } from "../../../../service/api";
import { k } from "./live-copy";
import { actorText, canWithdraw, endPassed, maxPeriod, pendingNow, revisionText, spanDays, validDay, type Period } from "./live-model";
import { ActionBar, CommandForm, type Shared } from "./LiveActions";
import { day, ltr, nameOf, stamp } from "./live-format";
import { Documents } from "./setup/Documents";
import { partOf } from "./setup/scope";
import { PaymentTerms } from "./setup/PaymentTerms";

type Props = {
  l: Locale;
  view: string;
  detail: ContractDetail;
  parties: OrgParty[];
  api: AdminClient;
  session: Session;
  params: URLSearchParams;
  period: Period;
  setupOn: string;
  onParams: (extra: Record<string, string | null>) => void;
  onReload: () => Promise<void>;
  onDone: (text: string) => void;
  onDenied: () => void;
  onSignedOut: () => void;
};

const unconnected = ["recognition"];

export function Detail(p: Props) {
  const { l, view, detail } = p;
  const c = detail.contract;
  const title = useRef<HTMLHeadingElement>(null);
  useEffect(() => { title.current?.focus(); }, [c.id]);
  const shared = { l, detail, api: p.api, session: p.session, onDone: p.onDone, onReload: p.onReload, onDenied: p.onDenied, onSignedOut: p.onSignedOut };
  return (
    <div className="o03-detail">
      <h2 id="o03-contract-title" ref={title} tabIndex={-1}><bdi>{c.name || c.id}</bdi></h2>
      <p className="live-sub live-mono">{ltr(c.id)}</p>
      <dl className="o03-pair" aria-label={k.contractState(l) + " / " + k.finalSettlement(l)}>
        <div>
          <dt>{k.stateToday(l)}</dt>
          <dd><span className="o03-tag">{k.state(l, c.state)}</span></dd>
          <dd className="live-sub">{c.end ? (endPassed(c.end, detail.today) ? k.endedFrom(l, day(l, c.end)) : k.endsFrom(l, day(l, c.end))) : c.start ? k.since(l, day(l, c.start)) : k.notRecorded(l)}</dd>
        </div>
        <div>
          <dt>{k.settlementCurrent(l)}</dt>
          <dd><Settlement l={l} value={c.settlement} /></dd>
        </div>
      </dl>
      {c.end && endPassed(c.end, detail.today) && c.settlement === "pending_final" && <p className="live-note o03-pair-note">{k.endedPending(l)}</p>}
      <div className="o03-sections">
        {view === "schedule" ? <Schedule {...p} shared={shared} />
          : view === "documents" ? <Documents l={l} session={p.session} contract={c.id} on={p.setupOn} parties={p.parties} part={partOf(p.params.get("part"))} onPart={(part) => p.onParams({ part: part === "settings" ? null : part })} onSignedOut={p.onSignedOut} />
          : view === "products" ? <Products l={l} detail={detail} parties={p.parties} />
            : view === "termination" ? <Termination l={l} detail={detail} shared={shared} />
              : view === "payment-terms" ? <PaymentTerms l={l} session={p.session} contract={c.id} on={p.setupOn} onSignedOut={p.onSignedOut} />
              : unconnected.includes(view) ? <Unconnected l={l} view={view} />
                : <Parties l={l} detail={detail} parties={p.parties} />}
      </div>
    </div>
  );
}

export function Settlement({ l, value }: { l: Locale; value: string }) {
  return value === "none" ? <span className="live-quiet">{k.settlement(l, value)}</span> : <span className="o03-tag">{k.settlement(l, value)}</span>;
}

function Party({ l, id, parties }: { l: Locale; id: string; parties: OrgParty[] }) {
  if (!id) return <>{k.notRecorded(l)}</>;
  const name = nameOf(id, parties);
  return <><bdi>{name}</bdi>{name !== id && <span className="live-sub">{ltr(id)}</span>}</>;
}

function Parties({ l, detail, parties }: { l: Locale; detail: ContractDetail; parties: OrgParty[] }) {
  const c = detail.contract;
  return (
    <>
      <section aria-labelledby="o03-h-parties">
        <h3 id="o03-h-parties">{k.parties(l)}</h3>
        <dl className="live-facts live-facts-grid o03-facts">
          <div><dt>{k.buyer(l)}</dt><dd><Party l={l} id={c.buyer} parties={parties} /></dd></div>
          <div><dt>{k.payer(l)}</dt><dd><Party l={l} id={c.payer} parties={parties} /></dd></div>
          <div><dt>{k.issuer(l)}</dt><dd><Party l={l} id={c.issuer} parties={parties} /></dd></div>
          <div><dt>{k.colPath(l)}</dt><dd>{k.path(l, c.path)}</dd></div>
        </dl>
      </section>
      <section aria-labelledby="o03-h-setup">
        <h3 id="o03-h-setup">{k.billingSetup(l)}</h3>
        <dl className="live-facts live-facts-grid o03-facts">
          <div><dt>{k.profile(l)}</dt><dd>{c.profile_name ? <bdi>{c.profile_name}</bdi> : k.notRecorded(l)}{c.profile && <span className="live-sub">{ltr(c.profile)}</span>}</dd></div>
          <div><dt>{k.currency(l)}</dt><dd>{c.currency ? ltr(c.currency) : k.notRecorded(l)}</dd></div>
          <div><dt>{k.timezone(l)}</dt><dd>{c.timezone ? ltr(c.timezone) : k.notRecorded(l)}</dd></div>
          <div><dt>{k.cycle(l)}</dt><dd>{k.cycleWord(l, c.cycle)}</dd></div>
        </dl>
      </section>
      <section aria-labelledby="o03-h-terms">
        <h3 id="o03-h-terms">{k.terms(l)}</h3>
        {detail.terms.length === 0 ? <p className="live-empty">{k.noTerms(l)}</p> : (
          <div className="live-table">
            <table className="o03-terms o03-fixed">
              <colgroup><col className="o03-c10" /><col className="o03-c30" /><col className="o03-c30" /><col className="o03-c30" /></colgroup>
              <thead><tr><th scope="col">{k.colRevision(l)}</th><th scope="col">{k.customer(l)}</th><th scope="col">{k.seller(l)}</th><th scope="col">{k.transactionPath(l)}</th></tr></thead>
              <tbody>
                {detail.terms.map((t, i) => (
                  <tr key={String(t.revision) + i}>
                    <td className="o03-num">{t.revision === null ? k.unknown(l) : t.revision}</td>
                    <td data-label={k.customer(l)}><Party l={l} id={t.customer} parties={parties} /></td>
                    <td data-label={k.seller(l)}>{t.seller ? <Party l={l} id={t.seller} parties={parties} /> : k.noSeller(l)}</td>
                    <td data-label={k.transactionPath(l)}>{k.path(l, t.path)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  );
}

function rateText(l: Locale, v: number | null): string {
  const r = revisionText(v);
  return r === "na" ? k.notApplicable(l) : r === "unknown" ? k.unknown(l) : r;
}

function Schedule(p: Props & { shared: Shared }) {
  const { l, detail, shared } = p;
  const [open, setOpen] = useState("");
  const opener = useRef<HTMLButtonElement | null>(null);
  return (
    <>
      <section aria-labelledby="o03-h-events">
        <h3 id="o03-h-events">{k.events(l)}</h3>
        {detail.events.length === 0 ? <p className="live-empty">{k.noEvents(l)}</p> : (
          <div className="live-table">
            <table className="o03-events">
              <colgroup><col className="o03-c22" /><col className="o03-c22" /><col className="o03-c16" /><col className="o03-c40" /></colgroup>
              <thead><tr><th scope="col">{k.colDay(l)}</th><th scope="col">{k.colEvent(l)}</th><th scope="col">{k.colRate(l)}</th><th scope="col">{k.colAction(l)}</th></tr></thead>
              <tbody>
                {detail.events.map((e) => <EventRow key={e.id} l={l} e={e} detail={detail} open={open === e.id} shared={shared}
                  onOpen={(btn) => { opener.current = btn; setOpen(open === e.id ? "" : e.id); }}
                  onClose={(done) => { setOpen(""); if (!done) requestAnimationFrame(() => opener.current?.focus()); }} />)}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <ActionBar keys={["contract.suspend", "contract.resume", "contract.renew", "contract.end"]} title={k.changeLifecycle(l)} {...shared} />
      <Segments {...p} />
    </>
  );
}

function EventRow({ l, e, detail, open, shared, onOpen, onClose }: {
  l: Locale; e: ContractEvent; detail: ContractDetail; open: boolean; shared: Shared; onOpen: (b: HTMLButtonElement) => void; onClose: (done?: boolean) => void;
}) {
  const can = !!detail.revision && !!detail.contract.id && (canWithdraw(detail, e) || !!pendingNow("contract.withdraw:" + e.id, detail.contract.id));
  return (
    <>
      <tr>
        <td>{validDay(e.from) ? day(l, e.from) : k.notRecorded(l)}<span className="live-sub">{k.dayStart(l)}</span></td>
        <td data-label={k.colEvent(l)}>{k.event(l, e.kind)}</td>
        <td data-label={k.colRate(l)} className="o03-num">{rateText(l, e.revision)}</td>
        <td data-label={k.colAction(l)}>
          {can ? (
            <button type="button" className="live-secondary o03-row-action" aria-expanded={open} onClick={(ev) => onOpen(ev.currentTarget)}>
              {k.withdraw(l)}<span className="visually-hidden">{k.withdrawWhat(l, k.event(l, e.kind), day(l, e.from))}</span>
            </button>
          ) : (
            <span className="live-sub">{k.eventBlocked(l, e.withdraw_blocked)}</span>
          )}
        </td>
      </tr>
      {open && (
        <tr className="o03-form-row">
          <td colSpan={4}><CommandForm commandKey="contract.withdraw" event={e} {...shared} onClose={onClose} /></td>
        </tr>
      )}
    </>
  );
}

function Segments({ l, detail, period, onParams }: Props) {
  const [from, setFrom] = useState(period.from || detail.from);
  const [until, setUntil] = useState(period.until || detail.until);
  const [problem, setProblem] = useState(period.problem);
  useEffect(() => { setFrom(period.from || detail.from); setUntil(period.until || detail.until); }, [detail.from, detail.until, period.from, period.until]);
  const show = (e: FormEvent) => {
    e.preventDefault();
    const p = !validDay(from) ? "from" : !validDay(until) ? "until" : spanDays(from, until) <= 0 ? "order" : spanDays(from, until) > maxPeriod ? "span" : "";
    setProblem(p as Period["problem"]);
    if (!p) onParams({ from, until });
  };
  return (
    <section aria-labelledby="o03-h-segments">
      <h3 id="o03-h-segments">{k.segments(l)}</h3>
      <form className="o03-period" noValidate onSubmit={show}>
        <div className="service-field">
          <label htmlFor="o03-p-from">{k.periodFrom(l)}</label>
          {(problem === "from" || problem === "order" || problem === "span") && <p className="service-field-error" id="o03-p-error">{k.periodProblem(l, problem)}</p>}
          <input id="o03-p-from" type="date" className="o03-date" value={from} aria-invalid={problem === "from" || problem === "order" || problem === "span" || undefined}
            aria-describedby={problem === "from" || problem === "order" || problem === "span" ? "o03-p-error" : undefined} onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div className="service-field">
          <label htmlFor="o03-p-until">{k.periodUntil(l)}</label>
          {problem === "until" && <p className="service-field-error" id="o03-p-until-error">{k.periodProblem(l, problem)}</p>}
          <input id="o03-p-until" type="date" className="o03-date" value={until} aria-describedby={["o03-p-until-hint", problem === "until" ? "o03-p-until-error" : ""].filter(Boolean).join(" ")}
            aria-invalid={problem === "until" || undefined} onChange={(e) => setUntil(e.target.value)} />
          <p className="service-hint" id="o03-p-until-hint">{k.periodUntilHint(l)}</p>
        </div>
        <button type="submit" className="live-secondary">{k.showPeriod(l)}</button>
      </form>
      <p className="live-note-rule">{k.prorationNote(l)}</p>
      {detail.segments.length === 0 ? <p className="live-empty">{k.noSegments(l)}</p> : (
        <div className="live-table">
          <table className="o03-segments">
            <thead><tr><th scope="col">{k.colFrom(l)}</th><th scope="col">{k.colUntil(l)}</th><th scope="col">{k.colDays(l)}</th><th scope="col">{k.colState(l)}</th></tr></thead>
            <tbody>
              {detail.segments.map((g) => (
                <tr key={g.from}>
                  <td>{day(l, g.from)}</td>
                  <td data-label={k.colUntil(l)}>{day(l, g.until)}</td>
                  <td data-label={k.colDays(l)} className="o03-num">{g.days === null ? k.unknown(l) : g.days}</td>
                  <td data-label={k.colState(l)}><span className="o03-tag">{k.state(l, g.state)}</span>{g.revision ? <span className="live-sub">{k.colRate(l)} {g.revision}</span> : null}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function Products({ l, detail, parties }: { l: Locale; detail: ContractDetail; parties: OrgParty[] }) {
  return (
    <>
      {detail.terms.length === 0 && <p className="live-empty">{k.noTerms(l)}</p>}
      {detail.terms.map((t, i) => (
        <section key={String(t.revision) + i} aria-labelledby={"o03-h-rev-" + i}>
          <h3 id={"o03-h-rev-" + i}>{k.termsRevision(l, t.revision === null ? k.unknown(l) : String(t.revision))}</h3>
          <div className="live-table">
            <table className="o03-components o03-fixed">
              <colgroup><col className="o03-c22" /><col className="o03-c20" /><col className="o03-c40" /><col className="o03-c18" /></colgroup>
              <thead><tr><th scope="col">{k.components(l)}</th><th scope="col">{k.colKind(l)}</th><th scope="col">{k.colProviderPayment(l)}</th><th scope="col">{k.colSellerBillable(l)}</th></tr></thead>
              <tbody>
                {t.components.map((c) => (
                  <tr key={c.id}>
                    <td>{ltr(c.id)}</td>
                    <td data-label={k.colKind(l)}>{k.component(l, c.kind)}</td>
                    <td data-label={k.colProviderPayment(l)}>{c.provider_payment_responsible ? <Party l={l} id={c.provider_payment_responsible} parties={parties} /> : c.kind === "seller_service" ? k.notApplicableService(l) : k.notRecorded(l)}</td>
                    <td data-label={k.colSellerBillable(l)}>{c.seller_billable === null ? k.unknown(l) : c.seller_billable ? k.yes(l) : k.no(l)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ))}
      <Unconnected l={l} view="products" />
    </>
  );
}

function Unconnected({ l, view }: { l: Locale; view: string }) {
  return (
    <section className="o03-unconnected" aria-labelledby="o03-h-unconnected">
      <h3 id="o03-h-unconnected">{view === "products" ? k.ratesTitle(l) + " — " + k.unconnected(l) : k.unconnected(l)}</h3>
      <p>{k.unconnectedLead(l)}</p>
      <ul>{k.unconnectedItems(l, view).map((x) => <li key={x}>{x}</li>)}</ul>
    </section>
  );
}

function count(l: Locale, v: number | null): string {
  return v === null ? k.unknown(l) : v === 0 ? k.none(l) : String(v);
}

function Termination({ l, detail, shared }: { l: Locale; detail: ContractDetail; shared: Shared }) {
  const c = detail.contract;
  const last = detail.events[detail.events.length - 1];
  return (
    <>
      <section aria-labelledby="o03-h-final">
        <h3 id="o03-h-final">{k.finalFacts(l)}</h3>
        <dl className="live-facts live-facts-grid o03-final">
          <div><dt>{k.endDay(l)}</dt><dd>{c.end ? k.endedFrom(l, day(l, c.end)) : k.noEnd(l)}</dd></div>
          <div><dt>{k.lastEvent(l)}</dt><dd>{last ? k.event(l, last.kind) + " · " + day(l, last.from) : k.notRecorded(l)}</dd></div>
          <div><dt>{k.colGaps(l)}</dt><dd>{count(l, c.open_gaps)}</dd></div>
          <div><dt>{k.reopenings(l)}</dt><dd>{count(l, detail.reopenings)}</dd></div>
          <div><dt>{k.sequence(l)}</dt><dd>{c.sequence === null ? k.unknown(l) : c.sequence}</dd></div>
          <div><dt>{k.finalAmount(l)}</dt><dd>{k.notComputed(l)}<span className="live-sub">{k.notZero(l)}</span></dd></div>
        </dl>
      </section>
      <section aria-labelledby="o03-h-evidence">
        <h3 id="o03-h-evidence">{k.evidence(l)}</h3>
        {detail.evidence.length === 0 ? <p className="live-empty">{k.noEvidence(l)}</p> : (
          <div className="live-table">
            <table className="o03-evidence o03-fixed">
              <colgroup><col className="o03-c22" /><col className="o03-c16" /><col className="o03-c20" /><col className="o03-c22" /><col className="o03-c20" /></colgroup>
              <thead><tr><th scope="col">{k.colEvidence(l)}</th><th scope="col">{k.colCovered(l)}</th><th scope="col">{k.colOwner(l)}</th><th scope="col">{k.colFollow(l)}</th><th scope="col">{k.colWrite(l)}</th></tr></thead>
              <tbody>
                {detail.evidence.map((e) => (
                  <tr key={e.write}>
                    <td>
                      <span className="o03-tag">{e.kind === "secured" ? k.secured(l) : e.kind === "missing" ? k.missing(l) : k.unknown(l)}</span>{" "}
                      {e.kind === "secured" ? ltr(e.reference) : k.gap(l, e.reason)}
                      {e.resolves.length > 0 && <span className="live-sub">{k.resolves(l, e.resolves.length)}</span>}
                    </td>
                    <td data-label={k.colCovered(l)}>{e.covered_until ? day(l, e.covered_until) : k.notApplicable(l)}</td>
                    <td data-label={k.colOwner(l)}>{e.kind === "missing" ? <bdi>{actorText(e.owner_login, e.owner) || k.notRecorded(l)}</bdi> : k.notApplicable(l)}</td>
                    <td data-label={k.colFollow(l)}>{e.kind === "missing" ? <>{ltr(e.follow_up)}<span className="live-sub">{e.resolved ? k.resolved(l) : k.open(l)}</span></> : k.notApplicable(l)}</td>
                    <td data-label={k.colWrite(l)}>{ltr(e.write)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <ActionBar keys={["contract.evidence.secured", "contract.evidence.missing", "contract.settle", "contract.reopen"]} title={k.settlementActions(l)} {...shared} />
      <section aria-labelledby="o03-h-log">
        <h3 id="o03-h-log">{k.log(l)}</h3>
        {detail.records.length === 0 ? <p className="live-empty">{k.noRecords(l)}</p> : (
          <div className="live-table">
            <table className="o03-log o03-fixed">
              <colgroup><col className="o03-c20" /><col className="o03-c20" /><col className="o03-c40" /><col className="o03-c20" /></colgroup>
              <thead><tr><th scope="col">{k.colWhen(l)}</th><th scope="col">{k.colActor(l)}</th><th scope="col">{k.colWhat(l)}</th><th scope="col">{k.colReason(l)}</th></tr></thead>
              <tbody>
                {detail.records.map((r) => (
                  <tr key={r.write + String(r.sequence)}>
                    <td>{stamp(l, r.at) || k.notRecorded(l)}</td>
                    <td data-label={k.colActor(l)}><bdi>{actorText(r.actor_login, r.actor) || k.notRecorded(l)}</bdi></td>
                    <td data-label={k.colWhat(l)}>
                      {k.record(l, r.kind)}
                      {r.added.length > 0 && <span className="live-sub">{k.added(l, r.added.length)}</span>}
                      {r.withdrawn.length > 0 && <span className="live-sub">{k.withdrawn(l, r.withdrawn.length)}{r.cancel ? " · " + k.cancel(l, r.cancel) : ""}</span>}
                      {r.as_of && <span className="live-sub">{k.asOf(l, day(l, r.as_of))}</span>}
                      {r.reopen && <span className="live-sub">{k.reopenCause(l, r.reopen)}</span>}
                      <span className="live-sub o03-keep">{k.finalSettlement(l)}: {k.settlement(l, r.state)}</span>
                      <span className="live-sub">{ltr(r.write)} · #{r.sequence === null ? "?" : r.sequence}</span>
                    </td>
                    <td data-label={k.colReason(l)}><bdi>{r.reason || k.notRecorded(l)}</bdi></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {detail.earlier_records !== null && detail.earlier_records > 0 && <p className="live-meta">{k.earlier(l, detail.earlier_records)}</p>}
      </section>
    </>
  );
}

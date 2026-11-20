import type { Locale } from "../../../../shared/types";
import type { ClientGrant, GrantScope } from "../../../../service/integrations";
import { actionLabel, kindLabel } from "../../../identity/pages/access/live-copy";
import { k } from "./live-copy";
import { Code, Known, ListOr, Num, Who, fsi, ltr, stamp } from "./live-format";
import type { Shared } from "./LiveCommand";

function Selector({ l, value, period = false }: { l: Locale; value: string; period?: boolean }) {
  const word = k.scopeWord(l, value);
  if (word && (period ? value === "any" : value === "all")) return <>{word}</>;
  return <Known l={l} value={value} />;
}

function Scope({ l, s }: { l: Locale; s: GrantScope }) {
  return (
    <>
      <span className="live-sub">{k.scopeCustomer(l)}: <Selector l={l} value={s.customer} /></span>
      <span className="live-sub">{k.scopeLegal(l)}: <Selector l={l} value={s.legal_entity} /></span>
      <span className="live-sub">{k.scopeContract(l)}: <Selector l={l} value={s.contract} /></span>
      <span className="live-sub">{k.scopePeriod(l)}: <Selector l={l} value={s.period} period /></span>
    </>
  );
}

function Grants({ l, grants }: { l: Locale; grants: ClientGrant[] }) {
  return (
    <div className="live-table">
      <table className="p02-table">
        <colgroup>
          <col className="p02-c16" /><col className="p02-c12" /><col className="p02-c14" /><col className="p02-c16" /><col className="p02-c15" /><col className="p02-c15" /><col className="p02-c12" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">{k.colGrant(l)}</th><th scope="col">{k.colAction(l)}</th><th scope="col">{k.colDataKind(l)}</th><th scope="col">{k.colScope(l)}</th>
            <th scope="col">{k.colFrom(l)}</th><th scope="col">{k.colUntil(l)}</th><th scope="col">{k.colState(l)}</th>
          </tr>
        </thead>
        <tbody>
          {grants.map((g, i) => (
            <tr key={g.id || i}>
              <td className="p02-first p02-wrap"><Known l={l} value={g.id} /></td>
              <td data-label={k.colAction(l)}><Code l={l} value={g.action} say={(x, v) => actionLabel(x, v) ?? k.unknownCode(x, v)} /></td>
              <td data-label={k.colDataKind(l)}><Code l={l} value={g.kind} say={(x, v) => kindLabel(x, v) ?? k.unknownCode(x, v)} /></td>
              <td data-label={k.colScope(l)}><Scope l={l} s={g.scope} /></td>
              <td data-label={k.colFrom(l)}>{fsi(stamp(l, g.not_before))}</td>
              <td data-label={k.colUntil(l)}>{fsi(stamp(l, g.not_after))}</td>
              <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={g.state} say={k.grantState} /></span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function ClientsView({ l, page }: Pick<Shared, "l" | "page">) {
  const c = page.clients;
  const clients = c.clients;
  if (c.state !== "configured") return <p className="live-empty">{c.state === "not_permitted" ? k.clientsDenied(l) : c.state === "not_configured" ? k.clientsUnconfigured(l) : k.unknown(l)}</p>;
  return (
    <>
      <section className="p02-clients" aria-labelledby="p02-clients-title">
        <h3 id="p02-clients-title">{k.clientsH3(l)}</h3>
        {clients === null ? (
          <p className="live-empty">{k.unknown(l)}</p>
        ) : clients.length === 0 ? (
          <p className="live-empty">{k.noClients(l)}</p>
        ) : (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c30" /><col className="p02-c30" /><col className="p02-c20" /><col className="p02-c20" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colClient(l)}</th><th scope="col">{k.colKind(l)}</th><th scope="col">{k.colState(l)}</th><th scope="col" className="p02-num">{k.colGrants(l)}</th>
                </tr>
              </thead>
              <tbody>
                {clients.map((x, i) => (
                  <tr key={x.id || i}>
                    <td className="p02-first"><Known l={l} value={x.id} /></td>
                    <td data-label={k.colKind(l)}><Code l={l} value={x.kind} say={k.clientKind} /></td>
                    <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={x.state} say={k.clientState} /></span></td>
                    <td data-label={k.colGrants(l)} className="p02-num"><Num l={l} n={x.grants === null ? null : x.grants.length} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      {(clients ?? []).map((x, i) => (
        <section key={x.id || i} className="p02-section p02-grants" aria-labelledby={"p02-grants-title-" + i}>
          <h3 id={"p02-grants-title-" + i}>{k.grantsTitle(l)}</h3>
          <p className="live-meta">{k.colClient(l)} <Known l={l} value={x.id} /></p>
          <ListOr l={l} items={x.grants} empty={k.noGrants(l)}>
            {(grants) => <Grants l={l} grants={grants} />}
          </ListOr>
        </section>
      ))}
      <section className="p02-section p02-service-users" aria-labelledby="p02-service-users-title">
        <h3 id="p02-service-users-title">{k.serviceUsersTitle(l)}</h3>
        {c.service_users === null ? (
          <p className="live-empty">{k.unknown(l)}</p>
        ) : c.service_users.length === 0 ? (
          <p className="live-empty">{k.noServiceUsers(l)}</p>
        ) : (
          <div className="live-table">
            <table className="p02-table">
              <colgroup><col className="p02-c30" /><col className="p02-c16" /><col className="p02-c30" /><col className="p02-c24" /></colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colServiceUser(l)}</th><th scope="col">{k.colState(l)}</th><th scope="col">{k.colOwner(l)}</th><th scope="col" className="p02-num">{k.colUsable(l)}</th>
                </tr>
              </thead>
              <tbody>
                {c.service_users.map((u, i) => (
                  <tr key={u.id || i}>
                    <td className="p02-first"><Known l={l} value={u.id} /></td>
                    <td data-label={k.colState(l)}><span className="p02-tag"><Code l={l} value={u.state} say={k.serviceUserState} /></span></td>
                    <td data-label={k.colOwner(l)}><Who l={l} login={u.owner_login} id={u.owner} /></td>
                    <td data-label={k.colUsable(l)} className="p02-num">{u.usable_credentials === 0 ? k.noneUsable(l) : <Num l={l} n={u.usable_credentials} />}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {c.manage_on === "M04" ? (
          <p className="p02-manage"><a href={"#/service-accounts?lang=" + l}>{k.manageOn(l)}</a></p>
        ) : (
          <p className="p02-manage">{k.manageOnUnknown(l)} {k.unknown(l)}{c.manage_on && <> ({ltr(c.manage_on)})</>}</p>
        )}
      </section>
      <section className="p02-section p02-unconnected" aria-labelledby="p02-oauth-title">
        <h3 id="p02-oauth-title">{k.oauthTitle(l)}</h3>
        <dl className="live-facts">
          <div><dt>{k.oauthTitle(l)}</dt><dd><Code l={l} value={c.oauth_clients} say={k.connection} /></dd></div>
        </dl>
      </section>
    </>
  );
}

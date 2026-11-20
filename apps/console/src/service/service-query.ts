import { type Parser } from "./strict-read";
import { accountDetailOf, accountPageOf, audienceDetailOf, audienceOf, credentialDetailOf, credentialOf, declarationPageOf, detailOf, identifier, organizationDetailOf, organizationOf, ownerOf, pageOf, requireRead, scopeDetailOf, scopeOf } from "./service-query-read";
import type { AccountQuery, CredentialQuery, DeclarationQuery, PageQuery, ServiceDetail, ServicePage } from "./service-query-types";

type Send = (path: string, init: RequestInit, request: string) => Promise<Record<string, unknown>>;
type Validate = <T>(parse: () => T) => T;
type Area = "accounts" | "credentials" | "owners" | "organizations" | "audiences" | "scopes" | "history";
type Query = PageQuery & Record<string, string | number | undefined>;
const filters: Record<Area, string[]> = {
  accounts: ["organization", "owner", "audience", "scope", "expiry", "state", "declaration"],
  credentials: ["service", "state", "collection"], owners: [], organizations: [], audiences: [], scopes: [], history: ["version"],
};
function prepare(area: Area, query: PageQuery): Query {
  identifier(query.workspace);
  const allowed = ["workspace", "limit", "after", ...(area === "history" ? [] : ["q"]), ...filters[area]];
  requireRead(Object.keys(query).every((key) => allowed.includes(key)));
  if (query.limit !== undefined) requireRead(Number.isSafeInteger(query.limit) && query.limit >= 1 && query.limit <= 100);
  if (query.after !== undefined) requireRead(typeof query.after === "string" && query.after.length <= 4096);
  const copy = { ...query } as Query;
  for (const [key, value] of Object.entries(copy)) {
    requireRead(value === undefined || key === "limit" || typeof value === "string");
    if (value === "" || value === undefined) delete copy[key];
  }
  return copy;
}
function tail(query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) if (value !== undefined) params.set(key, String(value));
  return "?" + params.toString();
}
function expectedEcho<T>(area: Area, page: ServicePage<T>, query: Query) {
  const expected: Record<string, string | number> = { workspace: query.workspace, organization: "", owner: "", audience: "", scope: "", expiry: "", state: "", declaration: "", service: "", collection: "", version: "", limit: query.limit ?? page.query.limit };
  if (area === "accounts") Object.assign(expected, { expiry: "all", state: "all", declaration: "all" });
  if (area === "credentials") Object.assign(expected, { state: "all", collection: "all" });
  for (const [key, value] of Object.entries(query)) if (key !== "after" && key !== "q" && value !== undefined) expected[key] = value;
  requireRead(page.search_input === (query.q ?? "") && (query.q ? page.query.q !== "" : page.query.q === ""));
  requireRead(Object.entries(expected).every(([key, value]) => page.query[key as keyof typeof page.query] === value));
  if (!query.after) requireRead(BigInt(page.count.value) === BigInt(page.rows.length));
}
function ordered<T>(rows: T[], key: (row: T) => string | bigint) {
  for (let i = 1; i < rows.length; i++) requireRead(key(rows[i - 1]) < key(rows[i]));
}

export function serviceQueryClient(send: Send, request: () => string, validate: Validate) {
  async function read<T>(path: string, parse: Parser<T>, verify: (value: T) => void): Promise<T> {
    const body = await send(path, { method: "GET" }, request());
    return validate(() => { const value = parse(body); verify(value); return value; });
  }
  async function list<P extends ServicePage<unknown>>(area: Area, path: string, query: PageQuery, parse: Parser<P>, verify: (page: P) => void = () => {}) {
    const captured = validate(() => prepare(area, query));
    return read(path + tail(captured), parse, (page) => { expectedEcho(area, page, captured); verify(page); });
  }
  async function detail<T, D extends ServiceDetail<T>>(path: string, id: string, workspace: string, parse: Parser<D>, verify: (value: D) => void = () => {}) {
    const target = validate(() => identifier(id));
    const scope = validate(() => identifier(workspace));
    return read(path + encodeURIComponent(target) + tail({ workspace: scope }), parse, (value) => {
      requireRead(value.workspace === scope && (value.row as { id: string }).id === target);
      verify(value);
    });
  }
  return {
    serviceAccounts: (query: AccountQuery) => list("accounts", "/identity/services", query, accountPageOf, (p) => ordered(p.rows, (r) => r.id)),
    serviceAccount: (id: string, workspace: string) => detail("/identity/services/", id, workspace, accountDetailOf),
    serviceCredentials: (query: CredentialQuery) => list("credentials", "/identity/service-credentials", query, pageOf(credentialOf, "service_id_asc_credential_id_asc"), (p) => {
      ordered(p.rows, (r) => r.service + "\n" + r.id);
      if (p.query.service) requireRead(p.rows.every((r) => r.service === p.query.service));
    }),
    serviceCredential: (id: string, workspace: string) => detail("/identity/service-credentials/", id, workspace, credentialDetailOf),
    serviceOwners: (query: PageQuery) => list("owners", "/identity/service-owners", query, pageOf(ownerOf, "owner_id_asc"), (p) => { ordered(p.rows, (r) => r.id); requireRead(p.rows.every((r) => r.eligible_at_reference)); }),
    serviceOwner: (id: string, workspace: string) => detail("/identity/service-owners/", id, workspace, detailOf(ownerOf)),
    serviceOrganizations: (query: PageQuery) => list("organizations", "/identity/service-organizations", query, pageOf(organizationOf, "organization_id_asc"), (p) => ordered(p.rows, (r) => r.id)),
    serviceOrganization: (id: string, workspace: string) => detail("/identity/service-organizations/", id, workspace, organizationDetailOf),
    serviceAudiences: (query: PageQuery) => list("audiences", "/identity/service-audiences", query, pageOf(audienceOf, "audience_id_asc"), (p) => ordered(p.rows, (r) => r.id)),
    serviceAudience: (id: string, workspace: string) => detail("/identity/service-audiences/", id, workspace, audienceDetailOf),
    serviceScopes(audience: string, query: PageQuery) {
      const target = validate(() => identifier(audience));
      const workspace = query.workspace;
      return list("scopes", "/identity/service-audiences/" + encodeURIComponent(target) + "/scopes", query, pageOf(scopeOf, "scope_id_asc"), (p) => {
        ordered(p.rows, (r) => r.id);
        requireRead(p.rows.every((r) => r.audience === target && r.workspace === workspace));
      });
    },
    serviceScope(audience: string, id: string, workspace: string) {
      const target = validate(() => identifier(audience));
      const scope = workspace;
      return detail("/identity/service-audiences/" + encodeURIComponent(target) + "/scopes/", id, workspace, scopeDetailOf, (d) => requireRead(d.row.audience === target && d.row.workspace === scope));
    },
    serviceDeclarations(id: string, query: DeclarationQuery) {
      const target = validate(() => identifier(id));
      return list("history", "/identity/services/" + encodeURIComponent(target) + "/declarations", query, declarationPageOf, (p) => {
        ordered(p.rows, (r) => -BigInt(r.version));
        if (p.query.version) requireRead(p.rows.length <= 1 && p.rows.every((r) => r.version === p.query.version));
      });
    },
  };
}

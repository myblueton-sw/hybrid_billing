---
type: OperationsGuide
title: Configure the local service audience and scope catalog
description: Loads a nonsecret installation catalog at API startup while distinguishing configured declarations from effective grants and connected service execution.
status: draft
sources:
  - resource: ../../features/HYB-304/m04-declarations.md
  - resource: ../../features/HYB-304/declaration-evidence.json
---

# Availability and prerequisites

This HYB-304 candidate adds startup configuration for the bounded local verification API. It is not production installation certification or completed M04 integration. Run commands from the repository root with the project's supported Go toolchain. Use a disposable local data directory for verification; do not replace a protected running installation or its configuration as part of this procedure.

The catalog records explicitly configured recipient identifiers and named service restrictions. It creates no permission grant or OAuth client, does not authenticate service requests, and does not prove a recipient connection works. The candidate API now requires a complete current declaration for new service creation and for first execution of each credential issue or reactivation. The candidate now exposes dedicated bounded M04 reads and canonical selection details. Console picker/form binding remains pending; the existing Console create form does not yet submit the required metadata and is not a completed integration.

# File and startup

Create a local UTF8 JSON file containing an explicit `audiences` array. Supply its path once with `--service-catalog`. Paths are local filesystem paths, not URLs. The file must be readable and regular, at most1MiB; symlinks to regular files are resolved by the standard filesystem interface. Do not place credentials, tokens, secrets or customer data in it.

```sh
go run ./apps/api/cmd/hb-api --listen 127.0.0.1:8820 --seed none --data ./local-data --service-catalog ./service-catalog.json
```

Omitting the flag constructs a known empty catalog with no selectable default recipients/scopes. `{"audiences":[]}` is the explicit empty file equivalent. A supplied empty path or empty file is an error; it does not select defaults. A missing/unreadable/nonregular/oversize file, duplicate flag or invalid configuration prevents startup. Do not remove the flag as an automatic fallback for a failed configured catalog.

The file is read and validated before opening the listener or initializing installation data. The server retains immutable parsed metadata and its fingerprint, not raw file bytes. Changing the file does not change a running server; validate and restart through the installation's normal controlled lifecycle to load a new version. The JSON parser never fetches a resource identifier or expands an environment variable.

# Explicit wire contract

This illustrative file uses placeholder IDs and a placeholder recipient. Replace them with actual installation-owned declarations. With `--seed none`, the workspace identifier is `workspace`; sample verification uses `sample-workspace`. The CLI seed default remains `sample`. Language selection does not select either scope. The currently recognized CLI vocabulary includes the actual `focus_data` kind and its `rows` field. Other configured kinds/fields require a server configuration that actually registers them; arbitrary names are rejected.

```json
{
  "audiences": [
    {
      "workspace": "workspace",
      "id": "usage-api",
      "resource": "https://resource.example.invalid/api",
      "scopes": [
        {
          "id": "read-usage",
          "action": "view",
          "kind": "focus_data",
          "fields": ["rows"],
          "customer": {"id": "customer-id"},
          "legal_entity": {"id": "legal-entity-id"},
          "contract": {"id": "contract-id"},
          "period": {"from": "2026-01", "until": "2027-01"}
        }
      ]
    }
  ]
}
```

Selectors are exactly `{ "id": "canonical-id" }` or the explicitly unrestricted `{ "any": true }`. Do not omit a selector; omission is invalid. Literal ID `all` is an ID and differs from `any:true`. Period is exactly `{ "any": true }` or canonical YYYY-MM `from`/`until`, with the upper bound exclusive and strictly later. Mixed members are rejected even when their values are empty/false. Human-only approval and installation-only actions are excluded.

Audience/scope identifiers are at most128 ASCII bytes using letters, digits, `.`, `_`, `:` or `-`. The recipient is at most2048 bytes: an absolute http/https identifier with a host, or `urn:<namespace>:<nonempty-value>`. Userinfo, query parameters (including an empty `?`) and fragments are rejected. This format acceptance is not a transport-security or external compatibility certificate. Audience IDs are unique per workspace, scope IDs per audience. An audience with an explicit empty `scopes` array records zero configured scopes, not unrestricted access.

Unknown fields, duplicate JSON keys, case aliases, nulls, malformed UTF8 and unpaired Unicode surrogate escapes are rejected. Explicit valid Unicode pairs and replacement characters retain their supplied identity. Construction also enforces10000 audience/100000 scope technical count bounds and the1MiB normalized representation limit; encoded expansion can exceed that limit even when a programmatic input's strings are smaller.

The complete normalized workspace/recipient/restriction content determines the catalog fingerprint. Reordering entries/field sets does not change meaning; modifying a scope, selector or exclusive period does. Catalog counts describe configured metadata, not effective permissions. Raw internal ID continuation positions are not public authenticated cursors.

# Declaration commands and limits

`service.create` requires string-valued `name`, `owner`, `purpose`, `organization`, `catalog_version` and `scopes`. `service.declare` targets an existing service and replaces the complete declaration using `purpose`, `organization`, `catalog_version`, `scopes` and canonical unsigned decimal `declaration_version`; use `0` only when no declaration exists. Both use the existing administrator session, CSRF, step-up, rationale, expected global revision and idempotency key. The catalog version is the exact current lowercase SHA256 fingerprint, not a user-selected label. The response exposes `declaration_version`, without echoing purpose or restrictions.

`scopes` is a JSON array encoded as one string field, for example `[{"audience":"usage-api","scope":"read-usage"}]`. Entries have exactly these two canonical IDs; unknown/duplicate members, duplicate pairs, null, empty arrays and trailing input are refused. Each selected entry must exist in the current workspace's configured catalog. Scope selection records restrictions; it does not authorize service requests or create an effective grant.

Purpose is nonempty trimmed decoded text within the configured UTF8 byte limit. Controls and directional overrides/isolates are refused; ordinary Korean and Arabic text is supported. The existing outer command JSON decoder can replace malformed raw Unicode with U+FFFD before this validation. This decoded-text contract differs from the catalog file's strict raw Unicode rejection.

The responsible owner must be an active human with an effective membership in the same workspace. The owning organization must be currently eligible, including an internal unit's legal-entity parent. Named customer/legal-entity selectors must resolve to eligible legal entities. Named contracts must be active at the command acceptance date with a recorded current revision and eligible customer/seller; an explicit customer selector must match. These checks run on create, declaration update, reactivation and every new issue execution. They neither contact a recipient nor infer an undocumented issuer mapping.

| Startup flag | Default | Allowed range | Meaning |
| --- | --- | --- | --- |
| `--service-declaration-purpose-bytes` | 512 | 1–2048 | Decoded purpose UTF8 bytes |
| `--service-declaration-scopes` | 16 | 1–64 | Selected audience/scope pairs |
| `--service-declaration-versions` | 5 | 1–20 | Retained declaration versions per service |

The existing command field byte cap also applies to the serialized scope array. Retention keeps a bounded suffix; missing/pruned versions are unavailable. The service-account admission/key cap is200, including retired records. Existing services without declarations cannot issue new credentials or reactivate until explicitly completed through `service.declare`. Suspended services may repair declarations; retired services cannot. Suspension, revocation and retirement remain available as reductions. Current dependency failure refuses new effects without partially creating an account or declaration.

An already accepted command can replay with its original request fingerprint after a business reference becomes ineligible, retaining the existing current-caller, step-up and trust checks. Replay does not repeat identity/declaration/credential effects, and accepted secret pickup remains available through its existing rules. A declaration change never rewrites a held unknown-outcome issue request. Trust hold refuses a new declaration change because it can expand the recorded scope.

Declarations currently share the prelaunch identity/command process lifecycle and reset on restart. This is not durable service-account persistence. The configured catalog remains immutable while running; no runtime catalog-edit endpoint is added.

# Verification and failure handling

A successful `/api/v1/health` response confirms that the local API started; it does not confirm a recipient connection, grant, business-object eligibility or service-request authentication. The constructor tests separately verify an exact configured scope against the actual gate vocabulary, no grant creation, immutable input handling and raw-byte removal. Synthetic CLI tests verify configured/empty/omitted startup, health, file replacement and shutdown.

On a generic catalog/startup error, inspect the local file against this schema and actual registered vocabulary. Correct the input before starting again. Do not paste raw configuration or credentials into logs/tickets. Prior server/data remains untouched by a failed constructor; after a later listener or initialization error, use the normal startup diagnostics rather than inferring the catalog was the cause. Current verification is macOS arm64 plus static Linux/Windows arm64 checks; other-OS runtime behavior remains unverified.

# Bounded administration reads

The candidate replaces the embedded account/credential/owner population at `GET /api/v1/identity/services` with account summary pages. `GET /identity/service-credentials`, `/identity/service-owners`, `/identity/service-organizations` and `/identity/service-audiences` under the same `/api/v1` prefix are independent bounded lists. Audience scopes and retained declaration versions have separate paged subpaths. Canonical account/credential/owner/organization/scope details do not depend on membership in the displayed page. See the [complete query contract](../../features/HYB-304/m04-queries.md) and running `/api/v1/openapi.json` read catalog for exact paths, fields and admission.

Every data read requires the current installation administrator, session and CSRF header in the authenticated canonical boot workspace. No role-name assumption, selected ID or continuation grants access. Counts use decimal strings and state exact/lower_bound certainty; a scan-stopped empty page is not proof of no matching records. `admission` separately reports exact recorded account usage, including retired records, so filtered count never controls creation capacity.

| Startup flag | Default | Range | Meaning |
| --- | --- | --- | --- |
| `--service-query-page-size` | 50 | 1–100 | Output rows per page |
| `--service-query-scan` | 1000 | 100–10000 | Conservative charged work units per request |
| `--service-query-lifetime-seconds` | 300 | 30–3600 | Fixed continuation lifetime in seconds |
| `--credential-warning-days` | 14 | 1–90 | Query threshold for expiring credentials; no credential-lifetime change |

A large account can require multiple requests before its credential aggregate is complete. Continue with the exact returned query and opaque `next`; keep the original reference time and count certainty. A changed/expired/mismatched continuation returns409 and requires an explicit fresh query. Current command and known-detail eligibility are recomputed; reference-time list membership is not current permission or execution certification. A source change while projecting a known detail also returns409 without a mixed-source row.

Recorded uncollected credentials have an independent workspace query, even when the account table is filtered differently. Collection membership means active at the signed reference and not recorded as returned. Open/never pickup availability is a live observation for the original issuing session, without a pickup reference or secret in metadata. A not-returned/currently-not-open annotation alone does not certify irreversible deletion; temporary original-issuer authority loss can later change availability. Another administrator seeing open is not given collection rights. Returned once never proves safe storage. Preserve held same-request recovery; changing filters or metadata does not rewrite an unknown-outcome issue attempt.

The connected Console now consumes these contracts, including complete declaration forms, independent pending collection and exact metadata resolution. Common read limits expose command_value_bytes as a decimal string, rationale_bytes as min(256, configured command rationale), and the existing1048576-byte body_bytes bound. These report current execution bounds and do not change policy. An exact audience detail is configured metadata, not effective eligibility or a grant. Full M04 acceptance, complete failure/retry verification and production-installation certification remain pending.

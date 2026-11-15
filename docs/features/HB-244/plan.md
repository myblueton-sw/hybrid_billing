---
type: ImplementationPlan
title: Allowlisted support bundles from typed fields, with pseudonymised identifiers, mandatory preview and disabled-by-default external upload
description: Second increment of work item W-36 (part 2a). A register builds support bundles only from typed fields that a dual-controlled schema allowlists. Secret-class and undeclared fields never leave the site, and free text is never accepted. Identifying values are always replaced with pseudonyms that are consistent within a bundle. A human must preview the exact encoded bytes before export or upload. Export is local by default. External upload is allowed only to an approved destination after dual-controlled enablement per site, bound to the previewed digest.
status: draft
---

# Scope and ownership

[HB-244](https://linear.app/hybrid-billing/issue/HB-244) continues work item W-36 (requirement HB-36) after HB-243. It covers the support-bundle part of HB-36-I and of HB-36-V: "support bundles are built from non-identifying information and a preview, and automatic external upload is disabled by default", with verification "support bundle secret removal".

Base: main `6936e6b`. Root (Claude Code) owns planning, implementation and tests. A separate read-only reviewer reviews the design and the change, because the register decides what customer-environment data may leave the site. Machine: mac_mini.

Allowed files: `modules/support/*` and `docs/features/HB-244/*`.

Out of scope, recorded as residuals:
- upgrade and restore (increment 2b);
- collecting items from real hosts;
- the transport that sends an approved upload;
- persistence;
- screens.

# Observed baseline

- **HB-36-I and HB-36-V**, as quoted above.
- **No reusable code.** No existing module performs redaction or pseudonymization.
- **Data rules.**
  - `tenant-security.md` forbids storing credentials or customer data in logs and tickets.
  - The AGENTS rule forbids storing keys, credentials or customer data in code, documents, memory, logs or tickets.
  - A support bundle is therefore treated as data that may leave the product only after removal and review.
- **Sibling conventions.** `modules/installer` and `modules/plugins` provide the conventions for proposals, dual control and actors.

# Decision

Removing secrets from free text by pattern cannot be made complete. Round 1 of the design review showed that header ordering, unlisted key names, encodings, multi-line values and vendor formats all leak. The register therefore never accepts free text. A support bundle carries only typed fields that a reviewed schema allowlists. Anything else is dropped before it is stored, and a field that might carry a secret has no class that could carry it out.

A new package `modules/support` follows the sibling conventions:
- one mutex and an injected clock that never goes backward;
- `ErrInvalid`, `ErrNotFound`, `ErrConflict`, `ErrRejected` (with `Refusal{Reason}`) and `ErrLimit`;
- check order: form, lookup, repeat, rule, room;
- assistants refused as `ai_not_allowed` and system actors as `not_human` wherever a human is required.

It imports only the standard library.

## Proposals

Configuration uses numbered proposals with the same rules as `modules/installer`:
- `Propose`, then `Approve` by a different human after a full recheck, and `Withdraw` by the proposer;
- caps per scope and per proposer;
- bounded history.

| Kind | Payload | Effect |
| --- | --- | --- |
| `site` | `Site{ID, Tenant}` | Registered site. Upload starts disabled |
| `schema` | `Schema{Source, Fields}` | Replaces the allowlist for one source, register-wide |
| `destination` | `Destination{ID, Host, Port}` | Approved support destination, register-wide. `Host` is a canonical DNS name or IP literal, using the `modules/installer` rules |
| `upload` | `Upload{Site, Destination}` | Enables external upload for the site to that destination |

Tightening actions by one human take effect at once:
- `DisableUpload(site)`;
- `DisableDestination(destination)`, which stops uploads to that destination from every site.

Destination and enablement rules:
- **One destination per site.** A site has at most one enabled destination. A new `upload` change replaces it.
- **No re-enabling.** A disabled destination can never be enabled again. Its ID stays recorded, so a new `destination` proposal with the same ID is `ErrConflict`. A new ID must be approved instead.
- **Reset previews.** Changing a site's upload enablement, or disabling a destination, returns every previewed bundle of the affected sites to `draft` and clears their previewer sets, so a new preview is needed.

## Schemas

- **Sources.** `Source` comes from the closed set `log`, `config`, `diagnostic`, `metric` and `version`.
- **Fields.** A schema lists 1–128 `FieldSpec{Key, Class, Values}`, strictly ascending by key. Enum `Values` are strictly ascending as well. Keys are lowercase `[a-z][a-z0-9_]{0,63}`.
- **Refused keys.** A key is refused (`sensitive_key`) when it contains any of these fragments:
  - `pass`, `pwd`, `secret`, `token`, `key`, `credential`, `private`, `auth`, `cookie`, `session`;
  - `signature`, `sig`, `pin`, `otp`, `cert`, `dsn`, `connection`;
  - `card`, `pan`, `iban`, `account`, `customer`, `phone`, `ssn`, `tax`, `invoice`.

  Such fields cannot be allowlisted at all. Over-refusal is accepted: for example, `ping_ms` and `company` are refused. Collectors name their fields to avoid the fragments.
- **Classes.**

  | Class | Accepted value | Rendered as |
  | --- | --- | --- |
  | `enum` | one of the spec's 1–64 declared `Values`. Each value matches `[a-z][a-z0-9_.-]{0,63}` and is not shaped like a pseudonym | the value |
  | `integer` | a signed 64-bit integer | its decimal form |
  | `duration` | a non-negative duration of at most 366 days | integer milliseconds |
  | `timestamp` | a time | RFC 3339 in UTC, to the second |
  | `version` | `MAJOR.MINOR.PATCH` | the version |
  | `host`, `address`, `email`, `user` | any UTF-8 string of 1–256 bytes | `<class>-<n>` |
  | `identifier` | a UTF-8 string of 1–256 bytes, or an integer | `identifier-<n>` |

  A pseudonym `<class>-<n>` is numbered by order of first appearance, per class, across the whole bundle. Two values are equal when they match after trimming `unicode.IsSpace` characters from both ends and lowercasing ASCII letters. An integer identifier compares by its decimal form. The original values and the mapping are discarded once the bundle is built and are never stored or returned.

  Further rules:
  - An `enum` field renders the declared value, which must equal the input exactly. Nothing else is ever echoed.
  - A field must fill only the member its class uses. Any other member that is non-zero drops the field (`wrong_shape`).
  - Timestamps must fall in the years 1970–9999.
  - An item with a duplicate key is dropped whole (`duplicate_key`).
- **Schema changes.** Applying a schema change for a source purges every live bundle that holds an item of that source. Purged bundles read as `ErrNotFound`, so content allowed by a schema that has since been tightened cannot be exported or uploaded.
- **Pending schemas.** At most 16 `schema` proposals may be pending at once, which bounds their payload memory.

## Items and bundles

- **Item.** An input item is `Item{Source, Fields}`, where each field is `{Key, Int, Text, Time, Duration}`. The member used depends on the class.
- **Item bounds.** At most 256 items per bundle and 128 fields per item. Every text member is at most 256 bytes, and the input text totals at most 4 MiB. These limits are checked during form validation, before anything is built.
- **Building.** `Create(site, items, actor)` by a human builds the bundle:
  - an unknown source, an undeclared key, or a value that does not fit its class drops that field, and the drop is counted in the summary by source and reason only, never by key;
  - items left without fields are dropped;
  - a bundle needs at least one item after dropping (`empty`).
- **Kept items.** Kept items are numbered `1..n` in input order. Each holds its source and the rendered fields, sorted by key. There is no free-text name or message anywhere.
- **Encoding.** The bundle bytes are the length-prefixed encoding of:
  - the tag `hybrid-billing/support-bundle/v1`;
  - the site's opaque support handle (`site-<n>`, assigned at registration), never the site ID;
  - the bundle ID, numbered per site;
  - the item count, and for each item its number, source and field count, then each field's key, class and rendered value;
  - the summary counts.

  The digest is the SHA-256 of exactly these bytes.
- **Size.** Encoded size is at most 4 MiB per bundle (`ErrLimit`).
- **Lifetime.** A bundle starts in state `draft` and expires `BundleLife` after creation.

## Preview, export and upload

- **Preview.** `Preview(site, bundle, actor)` by a human returns the bytes, the decoded view and the digest. It records the human as a previewer, and the state becomes `previewed`.
- **Export.** `Export(site, bundle, digest, actor)` by a human who previewed the current state (`not_previewed`) requires the digest to match (`digest_mismatch`). It returns exactly the previewed bytes, whose SHA-256 equals the digest. It never contacts anything outside the product.
- **Upload.** `Upload(site, bundle, digest, destination, actor)` by a human who previewed the current state requires:
  - site upload is enabled for exactly that destination (`upload_disabled`), so upload is disabled by default;
  - the destination is approved and not disabled;
  - the digest matches.

  It records and returns `UploadRecord{Destination, Digest, By, At}` for the transport adapter. Repeating the call for the same destination returns the existing record. At most 16 records exist per bundle (`ErrLimit`). No system or assistant actor can upload, so there is no automatic upload.
- **Expiry.** Expired bundles are refused (`expired`) and purged on every access.
- **Discard.** `Discard(site, bundle, actor)` by a human deletes a bundle at once.

## Bounds

| Limit | Scope | Maximum |
| --- | --- | --- |
| `Sites` | register | 100,000 |
| `Destinations` | register | 64 |
| `Bundles` | per site, live | 100 |
| `SiteBytes` | per site, live encoded bytes | 64 MiB |
| `Bytes` | register, live encoded bytes | 4 GiB |
| `BundleLife` | — | 1 h to 30 d |
| `Proposals` | pending per scope | 1,000 |

Input totals are checked before any processing.

## Residuals

- Collecting typed fields from real hosts and logs: the collector maps product events to schema fields.
- Shipping default schemas with the product.
- The transport, which must connect only to the approved destination host and port.
- Authorizing actors against the site's tenant (`tenant-security.md`), and AR-09 support delegation.
- Linking support sites to installer sites.
- Persistence and screens.
- **Numeric fields and enum literals.** A dual-controlled schema could still declare:
  - an `integer` field that holds a secret or identifying number;
  - an `enum` value that is an identifying literal, such as a hostname or tenant slug.

  The refused key list reduces this risk, and schema approval is the control.

# Verification plan

Literal tests:
- **Schema form.**
  - Every form rule.
  - Every sensitive key fragment.
  - Pseudonym-shaped enum values.
- **Classes.**
  - Every class with accepted and refused values at its boundaries.
  - Rendering of each class.
- **Dropping.**
  - Undeclared keys, unknown sources and mistyped values dropped and counted without their keys.
  - Items left without fields dropped.
  - `empty`.
- **Adversarial values.** Every adversarial secret form from the round-1 review supplied as an undeclared field, as a free value of an `enum` field, or as a pseudonym-class value never appears in the bytes.
- **Pseudonyms.**
  - Consistent across items within a bundle.
  - Independent across bundles.
  - Case and whitespace equivalence.
  - Never reversible from the register.
- **Encoding.**
  - A known-answer test of the encoding and digest.
  - Every field changes the digest.
  - Exported bytes hash to the digest.
- **Flow.**
  - Preview before export and upload.
  - Digest mismatch.
  - Re-preview after enablement changes and after a destination is disabled.
  - Upload binding to the destination, and repeat idempotence.
  - Expiry boundaries and purge.
  - Discard.
- **Bounds and isolation.**
  - Every bound, including the byte budgets.
  - Isolation between sites.
  - Refusal of system and assistant actors.

Additional literal tests:
- a tightened schema purges live bundles;
- the numeric-identifier key fragments, and integer identifiers;
- a disabled destination cannot be re-approved;
- a duplicate destination ID;
- a second `upload` replaces the first;
- the previewer set is cleared on reset;
- per-field and total input limits;
- duplicate keys, extra members and the timestamp range;
- trimming equivalence;
- the pending `schema` cap.

Property test: random items mixing declared and undeclared keys, with values drawn from a corpus of unique marked values of at least 12 bytes (secrets, e-mail addresses, addresses, hostnames and identifiers). The corpus also includes declared enum values supplied to undeclared keys. No corpus value appears anywhere in the bundle bytes. Pseudonyms are consistent. The digest equals the SHA-256 of the exported bytes.

Checks:
- `-race -count=10`, a 386 build, gofmt and vet;
- the whole repository with `-race`;
- 100% coverage and mutation testing.

# Review history
- Round 1 (independent reviewer, read-only): FAIL.
  - H1–H4 showed that pattern-based removal from free text leaks: header ordering, unlisted keys, syntaxes, encodings, multi-line values, vendor formats and invisible characters. Free text was replaced by allowlisted typed fields, with secret-class keys unrepresentable.
  - M1. The identifier list was removed, because identifying values are pseudonymised by class.
  - M2. Upload is now bound to the destination, the destination has an endpoint, `DisableDestination` was added, and previews are reset on enablement changes.
  - M3. Hosts and users are pseudonym classes, and items have no free names.
  - M4. No tokenization is needed.
  - M5. Byte budgets were added at site and register level, with output bounds and the empty rule.
  - M6. The digest equals the SHA-256 of the exact exported bytes, including the summary.
  - M7. No identifier matching is needed.
  - L1. The identifier list is no longer stored.
  - L2. Pseudonym-shaped enum values are refused.
  - L3. Upload repeats are idempotent.
  - L4 and L5. Recorded as residuals.
- Round 2 (same reviewer): PASS. Low and medium items were folded in:
  - N1. A schema change purges the affected live bundles.
  - N2. Numeric-identifier fragments are refused, and integer identifiers are allowed.
  - N3. Destination and enablement rules are specified, and previewers are cleared on reset.
  - N4. Input limits added.
  - N5. Recorded as a residual.
  - N6. Field semantics specified.
  - N7. Over-refusal noted.
  - N8. The property corpus uses unique markers.
  - N9. A cap on pending schema proposals.

# Change review resolution

The change review (independent reviewer, read-only) returned PASS with low items only. Resolved:
- L1. The bytes carry an opaque per-site handle (`site-<n>`) instead of the site ID.
- L2. Bundle IDs are numbered per site, so counts from other tenants are not revealed.
- L3. The schema is validated before it is copied.
- L4. The plan now states that enum values are ordered.
- L5. A concurrent-use test now runs Create, Preview, Export, Upload, Discard and DisableUpload under `-race`.
- L6. A test now exercises the register-wide expiry sweep.
- L7. Tests added for isolation between sites and for replacing the upload destination. The property test checks the kept count.
- L8. The property model lowercases ASCII only, and the corpus includes a non-ASCII case pair.
- L9. `itoa` uses `strconv`.

The re-review (same reviewer, read-only) returned PASS. Its informational note is recorded as a residual: site handles are numbered register-wide, so a handle reveals roughly how many sites were registered before it. Random handles are a possible follow-up.

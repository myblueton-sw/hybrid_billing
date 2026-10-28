---
type: Requirements
title: Noncommercial, commercial evaluation and purchased license modes
description: Defines future license mode selection, signed commercial entitlement and nonblocking expiry behavior without changing public license grants.
status: draft
sources:
  - resource: https://linear.app/hybrid-billing/issue/HB-37/define-noncommercial-commercial-trial-and-purchased-license
  - resource: https://polyformproject.org/licenses/noncommercial/1.0.0
  - resource: https://keygen.sh/docs/choosing-a-licensing-model/offline-licenses/
---

# Decision and scope

Implementation status: **NOT_IMPLEMENTED**. This document records product requirements accepted by the owner in conversation on 2026-10-28. It is not a license grant, executed commercial agreement, implemented activation service or completed runtime test.

[HB-37](https://linear.app/hybrid-billing/issue/HB-37/define-noncommercial-commercial-trial-and-purchased-license) depends on [HB-36](https://linear.app/hybrid-billing/issue/HB-36/adopt-source-available-noncommercial-terms-with-separate-commercial), the separately reviewed PolyForm transition. This branch started from a historical MIT base; HB-36 subsequently merged into main at `4b4d22fe1f31bf9a013c26f4fb842e7e32a900da`. Integrate these requirements with that adopted transition; this document does not retroactively restrict prior MIT copies. The applicable [license scope](../../LICENSING.md), [public license](../../LICENSE) and third-party grants govern actual rights.

The intended public policy is source-available under unmodified PolyForm Noncommercial 1.0.0, not OSI open source. Its permitted personal and organizational uses remain intact. The selected product mode records declared use and evidence; it cannot prove a user's real purpose. Corporate email, IP address or employer name alone must not classify use as commercial or authorize it.

# Explicit mode selection

| Mode | Permission basis | Required product behavior |
| --- | --- | --- |
| Noncommercial | Applicable public-license permitted purpose | Show the selected mode and license information, with no 14-day countdown or trial-expiry banner. Optional free key registration is not a condition of rights already granted by the public license. |
| Commercial evaluation | Separate express permission from the authorized rightsholder for a 14-day evaluation | Show accepted terms reference, authoritative start and end, and remaining time. Selecting the mode or downloading the software alone must not grant commercial evaluation permission. |
| Purchased commercial | Actual commercial agreement and authentic signed entitlement from an authorized issuer | Show the contracted company, agreement reference, permitted scope and term derived from verified entitlement. A key is evidence of that grant, not a substitute for the agreement. |

Noncommercial selection is a declaration, not an exemption for commercial activity outside the public license's permissions. Qualified organizational uses remain permitted according to PolyForm itself. Switching modes must preserve existing records and must not create a fresh trial or grant rights that the selected mode lacks.

# Evaluation lifecycle and time

- Start an evaluation only after explicit evaluation permission and acceptance are recorded. Record grant identifier, authorized company/subject, terms version/reference, acceptance, activation identifier, authoritative UTC start and end, and permitted scope.
- Define the 14-day interval as 336 hours from the recorded activation instant, with an inclusive start and exclusive end, subject to the actual evaluation agreement. Terms presented for acceptance must match this interval; a different agreement requires reconciliation rather than silently changing the product timer.
- Use a separately established authoritative time source and durable activation record. Reinstalling, restoring a backup, changing machines or rolling the client clock backward must not automatically reset or extend the grant.
- Before implementation, review the authority, identity binding, replay handling, persistence, restore behavior and clock anomaly policy. A purely local self-hosted check cannot be asserted to resist an operator controlling both code and storage.
- Decide online and offline activation/time verification with evidence. An offline signed record can establish issuer authenticity; it alone does not prove current wall-clock time, prevent a restored state or establish current revocation status.
- If time or grant verification is unavailable, show an explicit unverified state. Do not invent a new start time, silently extend the term or label network failure as expiry.

# Purchased license registration

- Provide a registration path for a signed key or license file. Verify its signature against a trusted public verification key before trusting its contents. Private signing keys must never be shipped in the client, repository, logs or customer installation.
- Bind signed claims to the actual authorized issuer/rightsholder, contracted legal entity, agreement/grant identifier, product, scope, validity start and end or explicitly agreed perpetual term, and verification key identifier. Do not fabricate a company, agreement, price or entitlement from the selected mode.
- Signature validity alone does not establish ownership of licensed material. The issuer must possess the commercial licensing authority for all material covered by the grant, including any external contributions.
- Reject altered payloads as invalid and distinguish unsupported or unknown signing keys as unverifiable. Present a verified company/scope mismatch separately from expiration. Do not replace a currently valid registration with an invalid submitted key.
- Define key rotation, issuance custody, authorized operator access, revocation handling and offline verification limits in an independently reviewed implementation design. No vendor, algorithm, activation endpoint, device limit or recurring online-check schedule is selected here.
- License registration is distinct from application authentication, tenant authorization and billing calculations. It must not bypass those controls or change customer invoice amounts.

# Status and expiry experience

| State | User-facing outcome |
| --- | --- |
| Noncommercial selected | License/mode information; no commercial trial-expiry message. |
| Evaluation active | Show the authorized evaluation end and remaining time. |
| Commercial entitlement active | Show verified company, scope and term; no evaluation countdown. |
| Evaluation or commercial term expired | Persistent nonblocking warning stating which term expired; offer key registration and contact. |
| Invalid or unverifiable key | Explain integrity or verification failure without presenting the key as active or expired. |
| Verification service/network unavailable | Explain inability to verify and retain the last verified evidence with its timestamp; do not represent stale evidence as fresh confirmation. |

The expiry warning must remain discoverable during ordinary use without trapping focus or blocking workflows. It must offer **Register license key** and **Contact myblueton@gmail.com**; contact is a user-initiated action. No automatic email, remote lock, data deletion, or denial of historical access/export is authorized. Technical access continuing after expiry does not extend commercial permission; the warning must say the applicable commercial grant has ended and direct the user to obtain permission before further commercial use.

# Acceptance scenarios for implementation

These are future verification criteria, all **NOT_IMPLEMENTED / NOT_RUN** in HB-37.

| ID | Given / when | Expected result |
| --- | --- | --- |
| LM-01 | A permitted noncommercial user selects the mode and passes 14 days without registering a key | No trial-expiry warning or registration barrier; public-license information remains available. |
| LM-02 | A user selects commercial evaluation without an express accepted grant | No active trial or implied commercial permission; show the permission/acceptance step. |
| LM-03 | A valid 14-day evaluation is activated, then evaluated immediately before and at its recorded end | Active before the exclusive end; expired at the end with a persistent nonblocking warning. |
| LM-04 | The same evaluation is reinstalled, restored or subjected to client clock rollback | Original grant bounds persist or verification uncertainty is explicit; no new 14-day grant is invented. |
| LM-05 | A valid purchased key matching company, scope and term is registered | Verified commercial details replace evaluation status without deleting billing history. |
| LM-06 | A key payload is modified, an unknown signing key is supplied, or a valid key is out of scope | Invalid, unverifiable and scope-mismatch outcomes remain distinct; no unauthorized activation or replacement of a valid key. |
| LM-07 | A key is expired versus its verification network being unavailable | Expiry and unavailable states differ; neither produces silent renewal or a fresh verification claim. |
| LM-08 | An expired evaluation user browses historical invoices and exports data | Operations remain technically available; warning and registration/contact actions remain visible, with no implied commercial-rights extension. |
| LM-09 | A corporate email/IP is supplied or the user changes the declared mode | No inferred commercial purpose, automatic contract, trial reset or retroactive rights change. |
| LM-10 | An offline installation imports a signed entitlement | Authenticity can be reported only after verification; time/revocation uncertainty follows the approved offline design, without embedding signing secrets. |
| LM-11 | Expiry or a verification error occurs | No automatic email, destructive data action, tenant-authorization bypass or invoice-amount change. |

# Evidence and remaining decisions

The owner's accepted conversation is the source of the three modes, 14-day evaluation, purchased key and nonblocking expiry/contact decision. The [official PolyForm text](https://polyformproject.org/licenses/noncommercial/1.0.0) establishes public permitted purposes; it does not supply a separate commercial trial. [Keygen's offline-license documentation](https://keygen.sh/docs/choosing-a-licensing-model/offline-licenses/) illustrates signed entitlement files and offline authenticity verification, without selecting Keygen as a dependency.

Before runtime work, independently review exact evaluation terms, contracted identity/scope, authoritative time and persistence, online/offline behavior, key custody/rotation and state transitions. Actual commercial terms require the rightsholder's agreement; pricing, payment integration, sponsorship and deployment are outside HB-37. Document checks establish requirement consistency only, not legal enforceability or implementation success.

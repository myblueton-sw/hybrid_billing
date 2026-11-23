---
type: ReviewReport
title: Unkeyed evidence digests as an offline confirmation oracle
description: Read-only assessment of whether digests returned by the issuance setup reads let a reader confirm values the same reads withhold.
status: draft
---

# Scope and method

HYB-342, machine:mac_mini. Read-only source and contract inspection of main at 4871cc5 by an independent reviewer (role-based, model Fable 5.2), requested by root. No test, build or command was executed and no digest was recomputed; every statement below rests on reading or on inference as marked. No code was changed. The hypothesis came from the design review of the [protected setup discovery contract](../HYB-341/issuance-protected-setup-discovery-contract.md).

# Verdict: partly confirmed

- Confirmed by reading: `financialEvidenceDigest` is SHA-256 over deterministic JSON of schema, kind and value, with no key or salt (`apps/api/server/financialsubject.go`, `approvalnormalized.go`). Every retained digest and page snapshot of the setup surface is built with it, and several cover fields the same response omits.
- For a protected reference the oracle is conditional. Every digest that covers a hidden reference (channel, change) also covers at least one free-text rationale of up to 2000 bytes that no setup read returns. A reader can confirm a guessed reference only if they also know or guess that rationale exactly. The rationale acts as an accidental salt, not as a control.
- For the protected rules, payment term and term binding digests the only hidden input is the rationale; those digests are a direct oracle for the rationale text and nothing else.
- Attestation digests are not an oracle: every input is returned to the same reader or obtainable from a sibling read.
- Nothing of secret value is exposed in the present boundary. The hidden reference is one of two synthetic codes per use, the code list is returned to the same reader, and the fingerprint is a public constant.
- A finding outside the hypothesis: the reference fingerprint is itself an unkeyed, unsalted SHA-256 of the reference identifier, and the draft inspection read returns it directly together with the reference identifier (`financialissuancedraftprojection.go`). With real payment account values this would matter more than the digest oracle.

# Disclosed digests

Readers are humans in the boot workspace with a live session, an undelegated gate decision and an owned applied `inspect` grant under a console ceiling, in the sample installation only; receipts go to the holder of the matching write capability. "Rationale" is the command rationale retained as the record's reason.

| Digest | Returned by | Hidden inputs for that reader | Verdict |
| --- | --- | --- | --- |
| Configuration digest and scope digest | `configurations`, the head of every scoped read, `context`, `options` | Organization scope detail (identifiers, small integers, enums); the reason when the configuration has no setup record | Not a meaningful oracle: it confirms organization facts only |
| Configurations page snapshot | `configurations` | The same, jointly across records | Not a meaningful oracle |
| Protected rules digest | `protected`, rules receipts | Rationale | Confirmed oracle for the rules rationale only |
| Channel digest | `protected_channels`, channel receipts | Reference (two codes per subject kind) and rationale | Conditional: confirms rationale and reference together |
| Change digest | `protected_changes`, change receipts | After and before references, instruction channel and sender (all tiny closed sets), the change's rationale and the embedded channel's rationale | Conditional: the enumerable part is tens of combinations, gated by two rationales |
| Channel and change page snapshots | the two protected pages | Everything above for each record on the page; for changes also the four-valued rejection reason | No additional oracle beyond the record digests; the protected module's approval digest input was not read |
| Attestation digest and page snapshot | `attestations`, attestation receipts | None except a withdrawal instant in the snapshot | Not an oracle |
| Term digest | `terms`, term receipts | Rationale | Confirmed oracle for the term rationale only |
| Binding digest and terms page snapshot | `terms`, binding receipts | The term's rationale | Confirmed oracle for the same rationale |

Receipts never give a caller a digest over inputs they did not supply or already had to present. Derived identifiers that hash only public identifiers are not oracles. The `references` read returns codes, use and applicability only.

# Reference fingerprint

- Derivation: the reference identifier is the literal prefix `synthetic-reference:` plus a catalogue code, and the fingerprint is the SHA-256 of that identifier. No key, salt, workspace or installation input. The same construction serves recipient, template, payment account, channel references and the unregistered sender.
- Preimage space now: thirteen catalogue codes, returned to every protected reader. The fingerprint represents no secret.
- Preimage space in production: no production reference source exists or is designed. If the same pattern were applied to a real value, the space would be an account number, a phone number or an email address, all enumerable or dictionary-guessable offline.
- Disclosure: not returned by the setup reads or receipts; returned directly by the draft inspection read; bound into the channel and change digests, both protected page snapshots, the protected module's request digest and the issuance issue encoding. The reader authority of the draft inspection read and the downstream exposure of the issue encoding were not traced.
- Constraint on any fix: the protected module compares the sender's fingerprint with channel fingerprints for equality, so the fingerprint must stay deterministic per value within a workspace. A per-record random salt would break that control; a keyed derivation would not.

# Severity

Present boundary: informational to low. Sample installation only, in-memory, synthetic references, readers with a workspace-wide `inspect` grant. The only non-public hidden input is an operator's rationale text. The real defect today is one of statement: the contracts say certain fields are "not returned" while the returned digest is a deterministic commitment to them. No contract states a confidentiality guarantee, so this is a wording and expectation gap.

With real payment account references: medium, and higher under a payment-fraud threat model. A directly returned fingerprint of a real account value would allow offline recovery or confirmation with no digest involved. A reader could also confirm which known reference a pending change targets once the rationale is known or guessed, and rationales are often short or templated. Free-text rationales and attestation statements may themselves hold contact detail.

# Options

| Option | Effect | Cost |
| --- | --- | --- |
| a. Keyed digest with an installation key | Closes offline confirmation for anyone without the key; field shapes unchanged | Every retained digest value changes; needs a separate function for setup records because the shared one also serves approval subjects, ERP final evidence and derived identifiers; adds key storage, backup, restore and rotation to the evidence path; an auditor can no longer recompute a digest; does not fix a directly returned unkeyed fingerprint |
| b. Public digest over disclosed fields plus a private integrity digest that is never returned | No key; disclosed content stays independently verifiable | Two digests per record and a choice at every equality site (retry identity, attestation target digest, embedded-record comparison, standing checks); the public token no longer commits to hidden fields, so contract wording changes and snapshots detect only visible change; larger change than a; leaves the fingerprint unaddressed |
| c. Accept with a recorded reason | No code or identity change | Defensible only for the synthetic boundary; needs corrected contract wording and a recorded production gate |

# Recommendation

1. Now, option c: record the acceptance for the synthetic boundary, correct the wording of the four setup contracts to say that omitted fields are bound into the returned digest, and record a gate: no production reference source and no real statements or rationales until items 2 and 3 are decided.
2. Before any production reference source, independent of the choice above: derive fingerprints with an installation-keyed deterministic function over the real value, never an unkeyed hash, and review the direct return of the reference identifier and fingerprint by the draft inspection read. A keyed fingerprint also removes value recovery through the record digests and leaves only a linkage oracle (which known reference a change uses).
3. For that residual linkage and the rationale oracle: option b for page snapshots is cheap and needs no key; the choice between a and b for retained record digests should be made together with the persistence design, because it fixes retained identity.

# Decisions that belong to the owner

- Accepting the exposure for the synthetic boundary and its recorded reason.
- Whether a holder of `inspect` is meant to be barred from knowing which reference a channel or change uses and from reading rationales. If not, most of this is moot.
- Introducing an installation key into evidence digests or fingerprints, with its rotation and audit-verifiability consequences.
- Any change to public contract wording or digest semantics, which is a public-contract change and needs independent review.

# Confirmed and not confirmed

Confirmed by reading: the digest function has no key; the input structure of every digest in the table; which fields each row, head and receipt returns; the caller checks; the fingerprint derivation and the catalogue; the fingerprint's return by the draft inspection read; the module digest inputs for the protected and receivables modules.

Inferred or unverified: byte-exact offline reproducibility; that organization scope facts are known to the reader; that no other read or journal returns rationales; the protected module's approval digest derivation; the reader authority and downstream exposure of the draft inspection read and of the issue encoding; that no retained digest is persisted yet.

# Limits

This is an assessment, not a fix and not a penetration test. No remediation is implemented or scheduled by it; the ticket stays open for the owner's decisions.

---
type: SecurityAssessment
title: Retained and snapshot digests as confirmation oracles for hidden setup inputs
description: Reading-based assessment of whether digests returned by issuance setup reads let a reader confirm guessed values that the same reads withhold, with options for the owner.
status: draft
---

# Outcome and ownership

HYB-342, machine:mac_mini. This document answers the first step of the ticket: confirm or refute the hypothesis by reading the code. It changes no code and decides nothing. The choice among the options at the end belongs to the owner, and any change to a digest needs design and independent security review first, because retained digests are part of record identity and of public contracts.

Read from main at 6f79b39 on 2026-11-24. Nothing was executed; no digest was recomputed outside the service.

# Finding

The hypothesis is confirmed as a structural property and is of low consequence in the current pre-launch build.

- Facts: the digests are unkeyed, they cover values the reads withhold, and the reads return the digests.
- Assessment: a reader who holds the protected settings read permission and can reproduce the remaining inputs can test guesses of the withheld values offline. Today those values are free-text reasons and choices among synthetic samples. The search is easiest for the rules digest (one free text) and harder for the others, which withhold several values at once.
- Not established: that every remaining input of each digest is in fact available to such a reader. This assessment assumes it is, because the format is in the source and most inputs are returned by reads; the assumption was not proven field by field.

# Facts read from the code

| Fact | Where |
| --- | --- |
| `financialEvidenceDigest` wraps the value with a schema and kind and calls `normalizedDigest` | `apps/api/server/financialsubject.go:49` |
| `normalizedDigest` is SHA-256 over the JSON encoding, with no key and no per-record random value | `apps/api/server/approvalnormalized.go:127` |
| The rules digest covers identifier, workspace, proposer, reason and the rule values | `financialissuanceprotectedrules.go:18` |
| The channel digest covers the whole channel record except acceptance fields, state and module view; its specification holds kind, evidence, reason, reference and selector, and the record also holds the subject with its scope records and the rules digest | `financialissuancechannel.go:12`, `:27`, `financialissuancechannelsubject.go:23` |
| The change digest covers the change record except digest, phase, module view, verification and decisions; it therefore covers the specification (field, channel, channel digest, provenance, reason, the new reference, the instruction with its channel and sender, effective time) and also the whole nested channel record, the configuration, the profile and the reference in force before | `financialissuancechange.go:14`, `:31`, `:64`, `financialissuancechangewrites.go:124` |
| The configuration digest covers the configuration record except state, confirmer and activation | `financialissuanceconfiguration.go:86` |
| The channel row returns the channel digest and does not return the reference, the evidence identifier or the reason | `financialissuancesetupprotected.go:262` |
| The change row returns the change digest and the instruction kind; it does not return the new reference, the reason, the instruction's channel or sender, or the reference in force before | `financialissuancesetupprotected.go:392` |
| The rules head returns the rules digest and the rule values and does not return the reason | `financialissuancesetupprotected.go:508` |
| The page snapshot of the channel and change pages is a digest over a binding (the reader and the whole rules record with its reason) and the page, which contains the complete retained channel or change records and not only the returned rows; the snapshot is returned to the reader. The configurations and statements snapshots have the same shape | `financialissuancesetupprotected.go:226`, `:315`, `:441`, `:466` to `:497`, `financialissuancesetupconfigurations.go:74`, `financialissuanceattestation.go:294` |
| The configurations read returns the proposal reason for configurations proposed through the public command | `financialissuancesetupconfigurations.go:21`, `financialissuancesetupmutation.go:62` |
| Configuration, channel and change digests all cover the organisation scope records of the configuration (workspace, contract revision, billing profile, parties, document key); the setup context read returns only identifiers and currency, and a test asserts that the channel subject is not returned | `financialissuanceconfiguration.go:20`, `financialissuancesetupcontext.go:214`, `financialissuancesetupprotected_test.go:119` |
| In the public writes the evidence identifier of a channel is the statement identifier and the attestation field is the statement digest; the statements read returns both | `financialissuanceprotectedwrites.go:274`, `:331`, `financialissuanceattestation.go:110` |
| In the public writes the reason is the free-text rationale of the command (not blank, at most 2000 bytes); the receipts carry digests and no reason | `financialissuanceprotectedwrites.go:208`, `:274`, `:331`, `financialdocumentgovernance.go:33` |
| A reference in the public layer is one of a closed list of synthetic samples, two per use and one for an unregistered sender; its fingerprint is SHA-256 of its public identifier, and the sender fingerprint is copied into the instruction | `financialissuancesetupreferences.go:13` to `:28` |

# What a reader could confirm

For the records made through the public writes, the inputs of each returned digest that the setup reads do not return are listed below. The table is conditional on the reader reproducing the organisation scope records through other reads; whether another read returns them was not checked.

| Digest | Inputs not returned by setup reads | Size of the guess |
| --- | --- | --- |
| Rules digest | reason of the rules proposal | one free text |
| Channel digest | reason; which sample reference; scope records | one free text times two, given the scope records |
| Change digest | change reason; new sample; the channel's reason and sample; the reference in force before; the instruction's channel and sender; scope records | two free texts times small enumerations, or the change reason times small enumerations once the channel digest has been solved |
| Configuration digest | scope records (the reason is returned) | no withheld free text identified |
| Page snapshots | the rules reason together with every withheld field of every record on the page | joint; reduces to the rules reason only when the page is empty |

So the choice of a sample is protected by the unpredictability of the reasons hashed with it. For a channel, a short or conventional reason can be found by trying candidates offline with only two candidates for the sample, after which the sample is known. For a change the search is larger, because the channel's withheld values are hashed in as well; it becomes the same kind of search once the channel digest has been solved.

The reader in question already holds the protected settings read permission. The reads withhold these values from that reader on purpose (the discovery contract states that reasons and evidence identifiers are not returned), so confirming them is a disclosure beyond the contract, not an escalation to another permission.

# Consequence

- Today: the references are synthetic samples and the installation is a pre-launch service that states it holds no real customer data. The reason text is the only withheld value with possible business meaning. Consequence is low.
- If real references are introduced (a real payment account or recipient reference in place of a sample): the reference identifier would become an input of digests that are returned to readers, and the fingerprint of a reference is itself an unkeyed hash of its identifier. A reader could then confirm a guessed account or recipient reference. For identifiers with a known format and partly known content this is a realistic search. This is the case that matters and it should be closed before real references exist.
- Integrity is not affected: the digests still detect change; the weakness is confidentiality only.

# Options

1. Keyed digest. Compute disclosed digests with a secret held by the installation (for example HMAC-SHA-256). Closes the oracle for every record at once. Costs: the key must persist with the installation data, be covered by recovery and have a rotation rule; every retained digest changes, so existing records need a migration or a versioned digest domain; `financialEvidenceDigest` has about forty call sites across the financial domains, so a change at function level reaches beyond issuance setup; clients that only echo digests are unaffected.
2. Two digests per record. Keep the full digest internal for identity, and disclose a second digest computed only over values the reader is also given. Closes the oracle without a key. Costs: every read and every write contract that carries a digest must name which one it carries, including the digests that clients echo as preconditions (configuration, rules, channel and change digests and snapshots) and the digests embedded in other records (the channel digest and rules digest inside a change, the target digest of a statement, the digest of a configuration reference); more surface to keep consistent.
3. Per-record random value. Add an undisclosed random value to each record's digest input. Closes the oracle for new records with no installation key. Costs: the random value must be retained with the record; existing records keep their digests or are migrated.
Options 1 and 3 must also cover the fingerprint of a reference, which is a separate unkeyed hash.

4. Accept for the pre-launch build with a recorded reason, and make one of options 1 to 3 a precondition of introducing real references. No change now; the residual stays on the owner-visible list.

# Recommendation

Option 4 now, with option 1 or 3 decided before any real reference can be stored. The present exposure is a free-text reason and a choice between two synthetic samples; changing retained digests is a public-contract and migration change that should not be made without its own design and review. This recommendation is root's reading and is not a decision.

# Not checked

- Whether every remaining digest input is reproducible by a reader, field by field (time encoding, nested profile and configuration records), and whether organisation or contract reads under other permissions return the scope records.
- Whether the command journal exposes a rationale or a payload digest that covers it.
- Digests outside the issuance setup reads: payment terms, bindings, drafts, reviews, statements, permission proposals, and the financial approval domains that use the same function.
- Whether the statements read withholds anything that a statement digest covers.
- Logs, traces and other receipts as additional places where digests or withheld values appear.
- Any measurement: no digest was recomputed to demonstrate the comparison.

# Review

Independent security review (role-based AI reviewer separate from the writer, by reading, nothing executed), 2026-11-24: FAIL with three findings, all in the table of what a reader could confirm, which overstated how easy the search is — the page snapshot covers the complete records on the page and not only the rules reason; the change digest also covers the nested channel record, the reference in force before and the instruction's channel and sender; and the scope records hashed into three of the digests are not returned by the setup reads examined (context, configurations, protected, statements, references). The facts table, the table and the options were corrected with the reviewer's wording and line references. The finding, the present consequence, the risk with real references and the recommendation were judged to hold. Corrective re-review of the three findings: PASS, with no false statement introduced. Not read by the reviewer: the terms, options and action reads, and the items under "Not checked".

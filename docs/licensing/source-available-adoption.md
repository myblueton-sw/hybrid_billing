---
type: Decision Record
title: Source-available noncommercial licensing transition
description: Records the owner's final noncommercial-plus-commercial-agreement decision and the boundary preserving prior MIT and third-party grants.
status: stable
sources:
  - resource: https://polyformproject.org/licenses/noncommercial/1.0.0
  - resource: https://polyformproject.org/licenses/noncommercial/1.0.0.txt
  - resource: https://spdx.org/licenses/PolyForm-Noncommercial-1.0.0.html
  - resource: https://opensource.org/osd
  - resource: ./mit-adoption.md
---

# Decision and ownership

Tracking: [HB-36](https://linear.app/hybrid-billing/issue/HB-36/adopt-source-available-noncommercial-terms-with-separate-commercial). Accountable owner: Seung Woo Park. Decision date: 2026-10-28. Working machine: mac_mini. Root owns policy integration and independent review; an isolated implementer writes the license/document changes.

The owner's final instruction is to permit noncommercial use and require a separate agreement for commercial use. It supersedes the intervening request to retain an unrestricted open-source policy. Adopt unmodified PolyForm Noncommercial License 1.0.0 for the current project-authored original material under the applicability below. Its SPDX identifier is `PolyForm-Noncommercial-1.0.0`; the authoritative text is [LICENSE](../../LICENSE). The license's named personal and organizational permissions are preserved, not narrowed by this summary.

This is source-available licensing, not an OSI open-source license. MIT and Apache 2.0 both permit commercial use and therefore do not implement the chosen restriction. The earlier custom commercial/evaluation draft is not reinstated.

# Effective scope and preserved rights

The owner reports that he is the sole developer and that no external party has downloaded or received a distribution. This is owner-reported evidence, not an independently verified download log, recipient audit or clearance of every imported asset. The owner explicitly selected the current-project transition on that basis; the change is not limited to future lines or newly created files.

The transition is identified by the first revision introducing this decision and the associated PolyForm LICENSE. The current project-authored original material in that revision is offered under PolyForm, as are subsequent original revisions covered by this policy. The historical MIT file is not an alternative license for the current transition revision. Specific third-party/component grants and contributions without sufficient authority remain outside any blanket ownership claim.

Any rights already obtained for an earlier copy under MIT remain unaffected. The prior complete license text and collective copyright notice are preserved byte-for-byte in [LICENSES/MIT-legacy.txt](../../LICENSES/MIT-legacy.txt) for that historical scope. If a prior or parallel MIT copy was validly obtained, changing the current license does not impose new payment, registration or noncommercial conditions on that copy's granted rights. The owner statement does not justify a claim that such rights could be revoked if an earlier recipient is later identified.

Historical Git references carrying MIT include main `4f990f168790ad15b10bf6359927fadeefb8e6ef` and HB-35 branch commit `8ac6b5faae9ebb2076fbb63336515d094abcc968`. These are non-exhaustive provenance examples, not independently verified evidence of third-party receipt or a statement that the current version remains MIT. Future releases must state their applicable license and preserve component rights.

Third-party libraries, FOCUS data, imported assets, external contributions and their original notices keep their applicable terms. Neither the owner's instruction nor the collective project copyright notice proves copyright assignment, ownership clearance or commercial relicensing authority for others' work.

# Commercial permission and contributions

For uses outside PolyForm's permitted purposes, request a separate commercial agreement at [myblueton@gmail.com](mailto:myblueton@gmail.com). Commercial-purpose internal business operations, MSP offerings or resale are not exempt merely because the software is not sold as a standalone product. The license's explicit eligible organizational purposes remain permitted. No automatic violation finding follows solely from a company name or network connection.

An agreement must identify the actual authorized rightsholder, covered material/versions, allowed use and agreed terms. No price, signed contract, SLA or guaranteed right to license third-party contributions is established here. A new PolyForm contribution by itself does not grant the project a right to issue commercial licenses for that contribution; obtain separate written authority or exclude it from that offering, as described in [CONTRIBUTING.md](../../CONTRIBUTING.md).

This ticket changes license texts and explanatory documents only. It does not implement automatic external reporting, email transmission, registration, activation, remote locking, payments or deployment. Merge remains separately authorized through the repository workflow.

# Verification boundary

The exact official plain-text license is retrieved and compared without rewriting its clauses. Local checks preserve the previous MIT bytes, validate document links/frontmatter, inspect current policy references and restrict the diff to licensing documentation. These checks establish artifact consistency, not legal enforceability in every jurisdiction, a complete copyright audit or an executed commercial agreement. Independent change review and PR status are recorded in the [HB-36 report](../work/HB-36-noncommercial-license/verification.md).

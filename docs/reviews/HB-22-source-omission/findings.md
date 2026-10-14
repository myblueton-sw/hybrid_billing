---
type: Source Based Review
title: Full source omission review after HB-16
description: Records the 2026-10-14 source-first comparison of all 123 v1.16 chapters against the merged planning baseline and the 33 clause-level gaps it found.
status: draft
---

# Full source omission review

Tracking: Linear ticket [HB-22](https://linear.app/hybrid-billing/issue/HB-22). Accountable owner: Seung Woo Park. Machine: `mac_mini`. Date: 2026-10-14. Baseline: merged main `b5c65cf66992f703247b8e4713f86cef7835b767` plus authorized local planning companions. Unmerged HB-18 to HB-21 branches were treated as proposals, not baseline.

## Method

Root extracted every non-empty body paragraph and table row of the v1.16 DOCX (SHA-256 `e7772927…db84d`) into 1,973 locator lines and split them into three chapter packs: 1–40 (B0000–B0520), 41–80 (B0521–B1030) and 81–123 (B1031–B1545). Three separate read-only reviewers each read their whole pack, listed distinct obligations, searched the baseline in Korean and English, and classified gaps as LOST, WEAKENED, UNBOUND or DECISION-PENDING. They were told not to re-report HB-15 findings (FR-01–06, FA-01–14, SR-01–03, PA-*).

Approximate obligation counts: 105, 185 and 225 per pack; about 91, 150 and 205 fully traced. Counts are manual groupings, not mechanical totals. Root re-checked one claim per pack against the source and baseline: P1-02, P2-02 and P3-05 hold. For P3-05, the only "focus" matches in the screen specs are keyboard-focus text; none of them refers to FOCUS data.

## Result

No requirement group is lost. 33 clause-level gaps are new relative to HB-15: 21 MEDIUM, 12 LOW, 0 HIGH. The [source omission trace](../../planning/v1.16/source-omission-trace.json) holds the locators, requirement groups, contract sections, screen follow-ups and acceptance cases for each.

| ID | Sev | Class | Gap |
| --- | --- | --- | --- |
| P1-01 | MED | WEAKENED | Issuer change only for unissued scope from effective date; no automatic switch during ERP outage (B0155) |
| P1-02 | MED | WEAKENED | Automatic posting off by default at initial go-live; baseline allowed it when draft protection exists (B0219) |
| P1-03 | MED | UNBOUND | Contract-permitted estimate as a separate provisional statement with true-up (B0065) |
| P1-04 | MED | WEAKENED | VMware/DC host-and-VM double charge, VM identity, meter correction approval, read-only vCenter connector, no-history periods (B0060, B0319, B0324–B0327) |
| P1-05 | MED | WEAKENED | Asset inputs, shared-server split ratio, margin vs internal subsidy (B0309–B0310) |
| P1-06 | MED | WEAKENED | Templates never auto-activate; platform updates never change customer-approved policy (B0301–B0302) |
| P1-07 | MED | WEAKENED | Realized payment FX kept separate from original invoice (B0469) |
| P1-08 | MED | WEAKENED | Invoice combination key and separate consolidated report (B0129, B1219.R003, B1224) |
| P1-09 | LOW | WEAKENED | Derived attribution statement never shown as provider original (B0034, B0130) |
| P1-10 | LOW | UNBOUND | AI view remaining included quantity (B0078) |
| P1-11 | LOW | WEAKENED | Organization closure never moves cost arbitrarily (B0141) |
| P1-12 | LOW | WEAKENED | HawkEye KST calendar-month default (B0279) |
| P1-13 | LOW | LOST | Product scope exclusions (B0024, B0208) |
| P1-14 | LOW | WEAKENED | Capacity-basis idle example not retained as an expected value (B0314) |
| P2-01 | MED | WEAKENED | Selling methods, savings basis, hold without basis, no auto rate increase, no auto-billing of unattributed amounts (B0545–B0581) |
| P2-02 | MED | LOST | Separate prepay purchase-date and billing FX versions; FX margin decomposition (B0551, B0667.R009) |
| P2-03 | MED | WEAKENED | Provider-specific commitment double-count guards and null-not-zero rule (B0523, B0526–B0527) |
| P2-04 | MED | WEAKENED | Tax-integration verification as a separate go-live state (B0714) |
| P2-05 | MED | WEAKENED | Parallel run issues from one designated system; cutover package (B0902, B0950) |
| P2-06 | MED | UNBOUND | ID-less AI and ERP verification tables lacked acceptance IDs (B0653, B0662, B0873) |
| P2-07 | MED | WEAKENED | Post-posting ERP read-back (B0850) |
| P2-08 | LOW | LOST | ERP replacement traceability, master overwrite ban, Business Central/ERPNext draft-post split (B0846.R003, B0870–B0871) |
| P2-09 | MED | WEAKENED | Marketplace transaction type, MACC exclusion, discount inheritance, no auto-purchase (B0686–B0700) |
| P2-10 | LOW | WEAKENED | Worked oracle figures 108.50 and 158,625 KRW (B0566, B0571, B0572, B0574, B0576) |
| P2-11 | LOW | WEAKENED | No auto-guessing of financial onboarding values; diagnostic fields (B0930, B0938, B0940) |
| P2-12 | LOW | WEAKENED | Converted CSV never called the provider original (B0977) |
| P3-01 | MED | WEAKENED | Collection outage never free; overlap and unbilled interval checks; proration structure (B1469, B1472) |
| P3-02 | MED | UNBOUND | Contract pool participants, consumption order, carryover, join and leave (B1218–B1222) |
| P3-03 | MED | WEAKENED | Workbook summary and mapping sheets, unmapped-code block, customer file residue (B1032–B1040) |
| P3-04 | MED | WEAKENED | API-to-manual switch lock, designated approval, post-recovery sync, duplicate candidates (B1044–B1048) |
| P3-05 | MED | UNBOUND | FOCUS conformance and readiness surface (B1159–B1183) |
| P3-06 | LOW | WEAKENED | `/api/v1`, version independence and major-version rule demoted to decisions (B1410, B1452) |
| P3-07 | LOW | LOST | Customer consent, intercompany split, external-selling enablement check, difference cause, value metrics, policy-request content (B1145–B1488) |

## Relation to HB-15

Confirmed with fresh evidence: FA-03, FA-05, FA-06, FA-11, FA-13, SR-02, SR-03. None disputed. The HB-15 purpose review's "no infrastructure provisioning: preserved" is partly disputed: the source clauses exist, but no baseline document cited B0024 or B0208 (P1-13). FR-01–06, FA-01, FA-02, FA-04, FA-07–10 and FA-12 were not re-examined.

## Limitations

Matching used keyword search plus targeted reads; "none found" means no match for the search terms. Not every screen specification or JSON file was read in full. HB-18–21 were only searched. No runtime, provider, ERP or performance behavior was executed; all cases are NOT_RUN.

---
type: DesignPlan
title: P05 hardware records are separated by a rule
description: A neutral rule and spacing between the hardware records of the load view of P05, so that a list of several records can be scanned.
status: draft
---

# Scope and ownership

HYB-361, machine:mac_mini. Raised by the ui-designer check of HYB-305 increment 4d. One style rule; no data, command or contract change.

# Facts

Each hardware record of the load view (`release-evidence/LiveLoad.tsx` `Hardware`, `.p05-hw-item`) is a strip of five facts and a specification digest. No style separated one record from the next, so the digest of one ran straight into the identifier row of the next. A running service has one hardware record, as nothing registers hardware; the defect shows only with several.

# Built

Every record after the first has a neutral one-pixel rule above it, with 16 pixels on each side of the rule (`.p05-hw-item + .p05-hw-item`). The rule uses the row border colour the profiles of the same view use.

# Verification

`tools/hardware-qa.mjs` gained a check, in three languages at 1440 and 390, that the first record has no rule and each later one has a solid one-pixel rule of a neutral colour; 49 passed, with hardware beyond the sample's one written by the check. I looked at the English render at 1440. A designer did not review the result; the owner has not reviewed the screen.

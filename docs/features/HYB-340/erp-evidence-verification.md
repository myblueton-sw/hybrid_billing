---
type: VerificationRecord
title: Current ERP preparation evidence reads
description: Tenant-scoped mapping and survey readback required by the planned synthetic ERP preparation adapter.
status: draft
---

# Scope and result

HYB-340, machine:mac_mini. MappingOf validates workspace, connection, closed mapping kind and platform ID. SurveyOf validates workspace, connection, customer and canonical month. Malformed keys return ErrInvalid; valid absent or foreign-workspace keys return ErrNotFound. The registry lock protects current record lookup. Returned views contain only value fields, so callers cannot rewrite stored mapping/survey evidence. Normal enter clock-watermark advancement remains possible, but reads do not change records, revisions, mapping counts or submission collections.

Independent refusal_review passed the read contract before implementation and reviewed the concrete source/tests without blockers. It did not execute tests. Independent platform_refusals also passed the document/source/evidence comparison, explicitly retaining actual adapter and external-effect work as incomplete. Root ran the full erpledger race suite, passing 2.529s (/private/tmp/hyb340-erp-evidence-race.log), and go vet ./modules/erpledger, passing with no diagnostics (/private/tmp/hyb340-erp-evidence-vet.log). New tests check changed mapping revision/author/time, changed survey evidence/author/time, independent returned values, collection counts, malformed IDs/months and absent/foreign records. No server or console behavior changed, so their suites were not rerun for this prerequisite-only change.

# Remaining integration

These reads enable comparison with retained explicit document/connection evidence; they neither authorize nor create ERP preparation. The actual synthetic document basis, real erpledger.Prepare adapter, ownership recovery and service assembly remain the next implementation in [the ERP preparation plan](erp-preparation-plan.md). Neither tax/quantity defaults nor a production ERP compatibility claim is introduced. W-23 stays active; no external send/post, issuance, payment, owner FIX, merge or deployment occurred.

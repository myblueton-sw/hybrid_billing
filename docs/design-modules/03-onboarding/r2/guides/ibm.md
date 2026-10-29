---
type: Design
title: IBM account and enterprise report setup
description: Defines required connection fields, setup ownership, provider constraints and verification gates.
status: draft
---

HB-52 detailed planning; proposed product behavior, not implemented support. See [shared workflow](../setup-workflow.md). Official provider evidence was researched on 2026-10-29; real integration tests are NOT_RUN.

# Form and ownership

| Field | Control and requirement | Owner / verification |
| --- | --- | --- |
| Report route | Account / Enterprise beta | Billing owner; routes have different scopes and permissions |
| Source ID | Account ID or enterprise ID, never interchangeable | Billing owner |
| Reader identity | Service ID and protected API-key reference | IAM owner; token renewal by expires_in |
| Month / history | YYYY-MM and bounded requested interval | Billing operator; observed availability, no guessed retention |
| Format | Complete CSV preferred pending verified paginated API contract | Operator |
| Expected children / invoice source | Enterprise complete custody and independent billing evidence | Billing owner |

# Account route

Native FOCUS 1.2 console path: Manage → Billing and usage → Usage → Export → FOCUS → month. Guide requires Billing Administrator/Account Owner. A narrower recurring minimum role remains UNVERIFIED; do not claim Viewer works.

Documented endpoint: GET https://billing.cloud.ibm.com/v4/accounts/{account_id}/focus/{month}, IAM bearer token. Guide documents format=json by default with pagination, or csv for all data; _limit default 200/max 250. Exact continuation contract is unresolved. The x-focus-version description contradicts itself; never send boolean true as a version. Account version/header/pagination support is a release blocker pending current contract and sanitized actual-response validation. Manual CSV intake is an explicit alternative, not automatic recurring connection.

# Enterprise route

Enterprise FOCUS uses a separate beta API: GET https://enterprise.cloud.ibm.com/v1/focus-reports?enterprise_id=…&month=YYYY-MM&format=csv . Optional page; omission returns full compressed report. Whole enterprise coverage cannot be filtered to account groups. Enterprise Usage Reports Viewer/Editor/Administrator is assigned on Enterprise in the enterprise account. Child-account policy does not grant parent access. Require full source custody before download.

# Operations and recovery

Use an approved service identity and secret-store reference; IAM tokens expire and must renew, not be stored as permanent connection credentials. Read-only report retrieval is separate from creating identity or granting billing rights. Inspect existing identity and known-month scope before reuse.

401 → renew once then repair identity; 403 → scope/role repair; 400 → validate route/month/format/version; missing month → distinguish unavailable/error/true absence. Respect Retry-After when returned, bounded jitter for 429/5xx. Quarantine incomplete pages/archive and retain prior accepted revision. Record route/ID, month, response/content identity, complete receipt, actual schema, currencies and revision. Usage is not automatically final invoice evidence; account charges/discounts/adjustments require comparison. Fixed close SLA, history floor and FOCUS-specific rate quota remain unknown.

# Official evidence and acceptance

- [Account export guide](https://cloud.ibm.com/docs/account?topic=account-export-usage-focus), [API reference](https://cloud.ibm.com/docs/apis/metering-reporting).
- [Enterprise export](https://cloud.ibm.com/docs/enterprise-management?topic=enterprise-management-enterprise-export-focus), [beta API](https://cloud.ibm.com/docs/apis/enterprise-apis/resource-usage-reports).
- [IAM token lifecycle](https://cloud.ibm.com/docs/iam?topic=iam-iamtoken_from_apikey), [usage versus bill](https://cloud.ibm.com/docs/account?topic=account-viewingusage).

Live acceptance NOT_RUN: exact account header/pagination, minimum role, enterprise archive/scope, token rotation/renewal, month availability, full receipt, replay and invoice comparison. No working native platform connector is claimed.

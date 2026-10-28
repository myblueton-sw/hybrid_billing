---
type: SecurityImplementationPlan
title: Protected source usage export jobs and download
description: Defines bounded internal usage exports with fixed targets, current export authority and protected expiring artifacts.
---

# Scope and owners

HB-34 continues on PR31 after `9831a60c83817af26921df35e2f2d27df1e4013d`; accountable owner Seung Woo Park; machine mac_mini. Root owns implementation/tests in `modules/customeraccess/`, `modules/customerusage/`, `modules/focus_import/customer_usage.py`, scoped tests and HB-34 documents. Separate architecture/security review is required before implementation and on actual changes. Authentication remains deferred.

Implement a process-local queued job service with explicit Create, Run and Download operations. Run is callable independently of Create by a trusted application worker with a currently verified recipient context; no recipient credential is saved or minted. This is not a durable distributed scheduler or public HTTP route. Protected artifact bytes stay in private bounded memory, with explicit TTL and job/row/byte limits. Restart loses jobs and grants, never restores stale output. No public link, filesystem path, bucket credential or raw-store fallback is returned.

# Fixed authority and target

`usage.export` is an independent trusted-service action; read-only grants cannot export and export-only grants cannot use the existing read service. Extend customeraccess with an action-specific constructor and opaque checkpoints. Capture validates the current verified principal, exact scopes/fields and relevant store versions. A pinned query requires the same store, action, selection and stable installation/subject/credential epoch, and the captured scoped versions both before and after work. New credentials for the same unchanged epoch may supply a fresh validity deadline; changing subject epoch or relevant grants invalidates the old checkpoint. Revoke/regrant, equivalent relevant updates and availability ABA invalidate the job permanently. Unrelated records do not. No action/checkpoint can be selected or constructed from customer input.

Each job freezes the installed gateway profile revision, source snapshot, one company/tenant, range, six usage fields, maximum rows, recipient identity and expiry. Source snapshot/tenant/lifecycle validity is checked through the existing held-source bridge at generation and every download. A wrong or unknown recipient/job produces uniform denied behavior without invalidating another recipient's job. No partial export follows a reduction in grants; a new job is required.

# Complete bounded export and protected release

Add a server-selected export bridge mode. Python validates all selected source rows under the existing source locks and returns the full selected projection only if its count is within the explicit export maximum (up to the existing 10,000-row source ceiling) and the byte budget. Page-mode behavior stays bounded-page. Export mode returns an explicit full-selection marker and row count; Go rejects a missing/wrong marker or inconsistent count. Source semantics remain delivery-window overlap with contained original charge intervals, no proration, hot slices only, fixed snapshot and no monetary fields. Exceeding any ceiling rejects the whole export and cannot claim completeness.

Use an internal JSON package (`usage-export-v1`), with exact quantities/identifiers/time values encoded as strings. This selects no public CSV/XLSX/PDF offering. It avoids spreadsheet formula execution; future formats need independent serialization review. The manifest includes profile/snapshot/range/scope, field list, usage-only basis, no applicable currency, supported full-selection row count, payload SHA256 and completion. It contains no supplier amounts, source roles/locators or hidden totals. Download metadata includes a fixed safe filename, content type, total bytes and package SHA256. Quantity totals across incompatible units and monetary totals are not invented.

Create captures authority before queueing. Run atomically claims Pending→Running, obtains all source rows through a pinned export-authorized query, serializes while the child still holds source protection, then performs final Go authority validation and the existing release acknowledgment/exit protocol. It publishes Ready only if the same job remains live and unexpired. Failure makes it Invalid with no bytes; duplicate/concurrent Run cannot create a second artifact. This is bounded one-shot generation rather than a resumable chunk scheduler.

Download requires the same current recipient and original authority checkpoint. It re-reads the full fixed source projection under source protection and compares the canonical package/digest with the saved artifact before the final Go authorization check and release handshake. It returns copied bytes only on an exact match and unexpired job. Source/policy/snapshot change, corruption, timeout or failed handshake returns no bytes. Revalidation is deliberately expensive in this small internal increment; there is no unsafe cache fallback. Grant changes before the final check deny; changes after that internal linearization point cannot recall output already released.

No export/job mutex or Go authority lock is held during source subprocess work. Expiry is checked before and after work; expired/invalid jobs discard artifact bytes. An explicit purge operation removes expired jobs; job admission also purges them, preserving bounded memory. The trusted server clock must be concurrency-safe and reliable. Direct non-spawning subprocess and source-lock delay limits from the usage projection design remain applicable.

# Verification

Test read/export separation, forged checkpoint/recipient/selection, queue and worker current authority, revoke/regrant between stages, revocation during held source read, expiry and invalid cleanup, concurrent Run, row/byte/job limits, exact JSON quantities/zero/missing and inert formula-like labels, hidden data exclusion, complete export beyond a page, wrong/full-selection marker, corrupted artifact and source mismatch. Run Go race tests, Python projection/regression tests and actual synthetic PostgreSQL/ClickHouse bridge export/download tests. Preserve existing databases/bindings/control evidence. Record independent review and actual-versus-synthetic evidence separately. Public authentication, production scale, durable artifacts and supported public download formats remain open.

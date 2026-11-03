# S05 actual local source evidence

Open a received row in S04 and choose **View source evidence**. S05 verifies the original CSV against the pinned manifest and displays an allowlisted text extract alongside normalization recomputed from that original. This does not verify current warehouse values or authorize billing/publication.

The route `/evidence/{release}?service=local&snapshot={manifest}&row={ordinal}&after={page-cursor}` carries identity only. Ordinals count logical CSV records, including quoted multiline fields. Return, browser history and Escape preserve the selected S04 detail and focus. Invalid context fails closed; an empty S05 entry links to S04.

Use the existing console build and `tools/console_service.py --local-operator --actor approver-a --port 8798` with isolated synthetic service configuration. Open `/#/source-data?lang=ko&service=local` to exercise the complete flow. No original download, public-customer access, OCR or restoration is implemented.

From `apps/console`, run `npm test`, `npm run build`, and `node tools/evidence-qa.mjs --url http://127.0.0.1:8798/ --playwright-module <module> --chromium <executable>`. The Python suite is `tests/console_service/test_evidence.py`; actual service tests require the explicit isolated-fixture gate. See `docs/features/HB-94/evidence.md` for review evidence and limits.

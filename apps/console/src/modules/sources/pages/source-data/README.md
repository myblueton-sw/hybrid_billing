# S04 actual local source inspection

This screen reads the loopback service's normalized synthetic inputs using a fixed approver process identity. It is not production cost authorization or billing confirmation. Rows remain separate by release, currency and cost meaning.

`/source-data/{release}?service=local&snapshot={manifest}&after={ordinal}` binds a bounded 50-row page. Optional `row` selects a release-relative row detail; `focus` restores the originating row. Every request rechecks authority. Failed loads clear prior protected data. The actual backend and deferred planning features are documented in `docs/features/HB-93/source-data.md`.

Use the console build and `tools/console_service.py --local-operator --actor approver-a --port 8797`, then open `/#/source-data?lang=ko&service=local`. Existing isolated services and environment configuration are prerequisites; do not initialize or replace shared data as part of a preview.

Run `npm test`, `npm run build`, and `node tools/source-data-qa.mjs --url http://127.0.0.1:8797/ --playwright-module <module> --chromium <executable>` from `apps/console`. Python source-service tests use `tests/console_service/test_source_data.py`; actual isolated tests require the explicit `HB_CONSOLE_INTEGRATION=isolated` test gate. Browser tests never confirm owner acceptance.

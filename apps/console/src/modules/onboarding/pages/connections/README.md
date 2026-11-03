# S01 local connection metadata

The canonical S01 screen belongs to the onboarding module. Its live variant reads the fixed process connection from PostgreSQL and installed local CSV contracts from the pinned registry. It exposes no input rows, amounts, original paths or credentials.

Use `/#/connections?lang=ko&service=local`, or `/connections/{connection-id}` for an exact detail. A loader process links to S03 intake; an approver process links to the latest S04/S06 candidate. Destinations reauthorize. Refresh reads metadata and does not perform external diagnostics. Configured period, latest registered candidate, quarantine and unknown completeness remain distinct.

Run the existing console build, then `tools/console_service.py --local-operator --actor approver-a --port 8799` or a separate loader process on port 8800 with the isolated synthetic service configuration. Run `npm test`, `npm run build`, and `node tools/connections-qa.mjs --url http://127.0.0.1:8799/ --loader-url http://127.0.0.1:8800/ --playwright-module <module> --chromium <executable>` from `apps/console`.

Backend tests are `tests/console_service/test_connections.py`; actual service tests require the explicit isolated-fixture gate. See `docs/features/HB-90/connections.md` for scope, independent reviews and limitations. Configuration editing, provider authentication, diagnostics, revocation and S02 go-live are not implemented by this screen.

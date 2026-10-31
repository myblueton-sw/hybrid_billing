# Design preview tooling

Run commands from repository root as documented in [the design build guide](../../docs/design-build-guide.md).

- `npm ci --ignore-scripts --no-audit --no-fund --prefix tools/design` prepares locked local font packages.
- `npm run check --prefix tools/design` verifies canonical source hashes and core brand/style rules.
- `npm run build --prefix tools/design` creates the standalone preview under output/designs/HB-63-console with local fonts, licenses and a deterministic hash manifest.

No production React, backend, publishing or deployment build is provided here. Browser verification remains the separate verify.cjs workflow. Do not commit node_modules or generated output.

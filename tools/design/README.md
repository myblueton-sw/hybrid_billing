# Design preview tooling

The static preview installs and runs through portable Node/npm commands without a required Python interpreter, POSIX shell, Homebrew, service manager or machine-specific path. Node.js 22 or newer and npm are prerequisites for these tools. The generated HTML, JavaScript, CSS, fonts and notices can also be served by an ordinary HTTP server; browser operation does not require Node on the viewer's device.

Run from repository root:

```sh
npm ci --ignore-scripts --no-audit --no-fund --prefix tools/design
npm run check --prefix tools/design
npm run build --prefix tools/design
npm start --prefix tools/design
```

Open the printed loopback URL. The default is `http://127.0.0.1:8763/?lang=ko#contents`; stop the server with Ctrl+C. These commands have the same form in Windows PowerShell, macOS and Linux shells. Windows/Linux execution remains unverified until run on those hosts; source portability is not universal OS certification. The server tests were executed on the current macOS host.

`check` verifies canonical source hashes and brand/style rules. `build` creates `output/designs/HB-63-console` with local fonts/licenses and a deterministic manifest. Build failures and missing output must be resolved before serving.

## Local serving options

```sh
npm start --prefix tools/design -- --port 8764
npm start --prefix tools/design -- --directory "path to built preview" --host 127.0.0.1 --port 8764
npm start --prefix tools/design -- --help
```

The default build directory is resolved from the tool location, not the working directory. An explicitly supplied relative directory is resolved from the process working directory. `HB_PREVIEW_DIRECTORY`, `HB_PREVIEW_HOST` and `HB_PREVIEW_PORT` are optional environment equivalents; command arguments take precedence. Port `0` requests an available local port and prints the selected URL.

Only loopback hosts (`127.0.0.1`, `::1`, `localhost`) are accepted. The server allows GET/HEAD for files owned by the current build manifest, with browser MIME types, no directory listing and no caching. Unknown files, traversal paths, dotfiles and symlinks are rejected. It rereads the manifest so a subsequent build is visible without adding unowned files. During an incomplete build it reports 503 rather than claiming the preview is available. This is a local review server, not authenticated production hosting. Keep the build directory under trusted local control.

## Verification

```sh
npm test --prefix tools/design
npm run qa --prefix tools/design -- --help
npm run qa --prefix tools/design -- --suite generic --playwright-module "path to installed playwright"
npm run qa --prefix tools/design -- --suite journeys --playwright-module "path to installed playwright" --chromium "path to chromium executable"
```

The Node suite also tests exact-scope local review persistence/failures and revision determinism across checkout line endings and dependency changes. Server tests use synthetic temporary builds and an automatically assigned loopback port. It verifies manifest ownership, browser file types, HEAD, rejected methods/paths, symlinks and invalid/missing builds. Symlink tests explicitly skip on hosts that prohibit creating test symlinks; a skip is not a pass for that host.

Browser QA uses a separately installed Playwright package and its installed browser, or an explicit `--chromium` executable. The runner accepts `--suite all|generic|journeys|review` and `--url`; unavailable suites or tools fail explicitly. `all` requires all three suite files in the current source revision. Existing suite output remains under `output/validation`. The runner passes options through Node without shell-specific environment assignment or a hard-coded browser path. Installing or downloading browser engines remains a separate test-environment step, not an installation prerequisite for the static preview.

No production backend, publishing or deployment build is provided. Do not commit node_modules, generated output or local review evidence. See [the design build guide](../../docs/design-build-guide.md) for design ownership and review boundaries.

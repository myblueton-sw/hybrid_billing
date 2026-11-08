# OS-independent installation and operation

The owner requires all designs to support installation and operation without dependence on a particular operating system (2026-11-01, HB-63). This is an architectural constraint, not a claim that every OS or device has already been tested.

- Use portable runtime/browser interfaces, relative paths and explicit configuration. Do not require an OS-specific shell, package manager, service manager, registry, keychain, path layout or native GUI to install or operate the product or its frontend preview.
- Keep platform integrations behind optional adapters. An integration must not become the only way to install, start, configure, back up, upgrade or operate the baseline product. Containers may be an option, not an automatic excuse for a mandatory host OS.
- Provide equivalent installation/start/stop and verification entry points using supported portable tools. Declare their version/runtime requirements and offline/network boundaries; OS neutrality does not mean no runtime dependencies.
- Browser screens must use standards-based layouts, input/file/download APIs, local assets and fonts. Do not rely on OS-specific dialogs, filesystem access APIs or font installation for core tasks. Handle unsupported capability and storage failure visibly.
- Track compatibility by actual host OS/architecture, runtime and browser version. Separate design targets, static portability review and successful executions. Untested combinations remain UNVERIFIED; do not claim universal OS support from one Mac test or from use of a container.
- Apply this principle to new screen, installer, deployment and operational designs. Existing OS-dependent development helpers are not product runtime requirements and must not appear as the required user installation path. Assess existing product dependencies before claiming full product compliance.

# Canonical design foundations

Adopted subset of the owner-supplied design-system archive for HB-63. Canonical SVGs and tokens are byte-identical to the supplied files recorded in provenance.json. Original metadata is retained. console.css contains the explicitly reviewed console typography overrides.

This directory exports no React components. Do not infer that Money, Icon, AppShell or the original Tailwind integration has been installed or validated. See [design rules](../../docs/design-system.md) and [build/work guide](../../docs/design-build-guide.md).

Build integration lives under tools/design; generated local fonts and their licenses are copied to output/designs/HB-63-console. Fontsource dependencies are locked there and retain OFL-1.1. The original reference archive remains separate evidence, not a second editable source of the active foundations.
The React build verifies every provenance-recorded SHA-256 before bundling. Preserve original SVG metadata, logo geometry and notices. These assets do not establish new distribution rights. See [licensing boundaries](../../LICENSING.md).

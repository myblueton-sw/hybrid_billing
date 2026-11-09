---
type: VerificationRecord
title: Historical worktree directory cleanup
description: Records the owner-requested removal of redundant hb worktree directories after preserving commits, evidence and active dependencies.
status: draft
---

# Scope and outcome

[HB-148](https://linear.app/hybrid-billing/issue/HB-148/remove-historical-hb-worktree-directories-after-preserving-local) follows the owner's explicit request to clean the many `~/project/hb*` directories. Root performed the cleanup; a separate read-only reviewer inspected the inventory and guarded preservation/deletion procedure before execution. This is a one-time cleanup record, not a new benchmark or automatic retention rule.

All20 matching historical worktree directories were removed. None had tracked modifications. Every original HEAD remains reachable from main or the previously preserved unmerged work branches. The active HB-147 integration and isolated writers remain under the system temporary directory, outside `~/project`.

# Preservation and verification

| Item | Actual result |
| --- | --- |
| Historical directories |20 removed through Git worktree removal after exact-path/status/HEAD/reachability checks |
| Generated verification evidence |15 distinct runs,892 files preserved under the canonical repository's `output/validation`; source/destination relative paths, file hashes and symlink identities were compared |
| Shared console dependencies |2693 file/symlink entries moved once into `.local/dependencies/console/node_modules` with an identical tree manifest |
| Active dependency consumers |4 active temporary-worktree links retargeted and verified resolved to the preserved installation |
| Canonical data/environment |Old `.local` and `.env` entries were symlinks; removal did not follow them into canonical data or environment files |
| Source/commit safety |Zero tracked changes; original HEADs rechecked for reachability immediately before each removal; unmerged work branches retained |
| Remaining ignored/untracked items |Reclassified immediately before removal; unexpected paths or newly created/unpreserved output would abort rather than be deleted |
| Process/destination safety |Home-owned process cwd inventory checked for users of retired worktrees; canonical destination parents checked for symlinks before moves/deletion |

The initial independent review found a gap between the preservation snapshot and forced worktree removal: new untracked output could otherwise have been lost. Root added deletion-time classification, source and destination rechecks, process-cwd checks and destination-parent symlink guards. Targeted independent re-review passed for this actual20-directory plan. Generated build directories and interpreter/tool caches were discarded only after classification; original verification evidence was preserved separately.

The detailed private preservation manifest remains local at `.local/runs/HB-148-worktree-cleanup/preservation-manifest.json`; credentials, environment contents, raw retained data and dependency/vendor files are not included in this Git record. Existing Graphify results were not benchmarked, modified or removed by this task.

After dependency relocation, the HB-147 frontend writer reran its13 contract tests and whole-console TypeScript check successfully. The root's integrated verification is recorded with HB-147 separately. Other operating systems, dependency reinstallation and production operation were not tested by this filesystem cleanup.

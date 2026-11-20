---
type: VerificationRecord
title: Durable license registry and verification-time state
description: Single-installation persistence with fail-closed commit handling and explicit authority uncertainty.
status: draft
---

# Delivered behavior

HYB-328 now adds an isolated durable facade around the existing license domain registry. `CreateDurable` creates a new private state directory and binds one installation/product selected by the host service; `OpenDurable` requires the same expected identity and an existing valid checkpoint. No implicit initialization or additional installation registration is exposed. The original in-memory Registry and current API remain unchanged.

Every facade operation serializes access and writes the complete state before returning a result, including Status, history/key/proposal reads and domain refusals whose clock watermarks may have changed. It preserves modes, histories, grants, accepted claims, key compromise/retirement, pending/applied proposals, counters, local and authority watermarks, last verified status and sticky expiry. Restored activation/document references share the canonical grant/document objects.

Storage failure suppresses the domain result and poisons all further operations until close/reopen. The outcome is unknown: a rename can succeed before directory sync fails, so reopening validates whichever old/new complete checkpoint exists. No failed save is advertised as successful or automatically retried. Missing state after initialization is not silently reconstructed. A full proposal counter returns ErrLimit instead of wrapping.

The state format is strict canonical JSON with schema version, expected installation/product binding and SHA-256 corruption detection. Unknown/duplicate fields, trailing bytes, invalid references/claims/proofs/indexes and malformed intervals are rejected. File reads/writes are bounded at 64 MiB; the checkpoint limit is operational capacity, not a product usage metric. Invalid submitted payloads are not persisted; the existing outcome/digest history is retained.

# File and time boundary

The adapter creates a new 0700 directory, uses an exclusive 0600 ownership marker, writes a private temporary file, synchronizes it, atomically replaces the checkpoint within the directory, and synchronizes directory metadata before acknowledgment. Creation also synchronizes the parent directory. Final-path symlinks, nonregular checkpoints and group/other-accessible files/directories are rejected. The caller must select a trusted parent directory; same-user hostile filesystem manipulation is outside this adapter's security claim.

Ownership is acquired before reading state and retained until Close. A stale ownership marker is never automatically removed. After an interrupted process, an operator must establish that no writer remains, preserve the directory as recovery evidence, and explicitly release the stale marker before Open. Do not invoke Create against retained state, delete checkpoints to bypass errors or restore an old directory as a new evaluation.

Time remains the explicitly injected Authority. Unavailable or backward authoritative time yields unverified state under the existing domain rules; the local clock is not substituted. Tests use synthetic authorities and synthetic grants. This implementation does not select a network time provider, prove real issuer authority or prevent an operator who controls disk/code from restoring an entire historical store. Checksums are not authentication. Signed-input verification remains in the domain; restored accepted state assumes the trusted private store.

# Actual verification

- `go test -race ./modules/licensemode`: PASS after final fixes, 8.606 seconds in the recorded root run.
- `go test -json ./modules/licensemode`: 42 top-level tests passed, 67 passes including subtests, no failures.
- `go vet ./modules/licensemode`: PASS.
- Linux amd64 and Windows amd64 `go test -c`: PASS. These are compilation checks only, not filesystem/runtime verification.
- Host: Go 1.26.4, macOS arm64. Filesystem execution is verified on this host; physical power-loss behavior and other OS/filesystem guarantees remain unverified. Unsupported filesystem synchronization/replacement behavior returns errors.
- Covered: evaluation interval and repeat across restart; observed expiry survives restart/reset; unavailable/backward authority; commercial replay, history, compromise and shared references; commit faults at create/write/file-sync/close/rename/directory-sync; old versus new outcome on reopen; result suppression and poisoned reads/writes; single writer; wrong binding; private modes and final-path symlinks; stale owner; 64 MiB read bound; corrupt/schema/duplicate/trailing/unknown input; invalid grant/document/proposal relationships; missing-state refusal; exhausted proposal counter; concurrent Status.

# Independent review and remaining scope

Independent design review passed with commit-outcome, shared-reference, identity, strict-validation and filesystem constraints. First actual-change review independently ran the race suite and requested two hardening fixes: refuse disappearance of an existing checkpoint and prevent proposal-counter wrap. Both were implemented with regression tests. Final independent re-review PASS with no blockers. Reviewer independently ran `go test -race -count=1 ./modules/licensemode` successfully in 11.638 seconds; the earlier truncated reviewer message established no test count.

This is a persistence prerequisite, not complete P07 service binding. API authorization, stable installer identity integration, trusted production authority/issuer configuration, original plugin notice artifacts, registration/activation routes and live screen integration remain pending. No running preview, real grant, billing amount, permission, owner acceptance or deployment changed. Existing frontend verification remains in [verification.md](verification.md); the current plan is [plan.md](plan.md). Owner Seung Woo Park; root writer; machine mac_mini; continue the existing HYB-328 branch and PR.

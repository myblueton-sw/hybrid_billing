---
type: VerificationRecord
title: Version-bound frontend notice artifacts
description: Original frontend license bytes bound to installed and locked versions with deterministic distribution metadata.
status: draft
---

# Contract

The seven existing frontend notice files now have `licenses/manifest.json`, schema 1, scope `frontend-notice-set`. Each entry carries package name, installed version checked against package-lock.json, package-relative `source_file` (`LICENSE`), output-root-relative public `path`, byte length and SHA-256. Assets and metadata derive from one capture. Missing identity/version evidence, mismatched installed package identity/version, missing original files and zero-byte originals fail generation rather than omit a package. Development serves JSON as application/json with nosniff and GET/HEAD handling; original text remains byte-identical.

This is neither a complete SBOM nor a plugin inventory. Matching installed metadata to lock versions does not verify the installed files against registry tarball integrity. Hashes describe emitted originals, not legal clearance or issuer provenance. No filesystem paths or registry URLs are published.

# Verification

- Root targeted tests: 5/5 PASS, 624.96225 ms. Coverage: original-byte development serving, manifest GET/HEAD/POST, missing evidence, package/version mismatch, empty originals, deterministic metadata, digest/length and production-emission equality.
- Independent reviewer reran the same five tests: PASS, 728.0 ms; actual-change review PASS with no blocking findings.
- Full console TypeScript/build: PASS. Existing large-chunk warning remains.
- Actual built distribution: all seven output files match manifest lengths, SHA-256 and installed original bytes.
- Latest main FOCUS operational records integrated without conflict before this increment. No runtime or data store was replaced.

Host verification used macOS arm64 and Node 25.9.0. Other host combinations remain unverified. There is no changed screen layout, interaction or translated copy; prior rendered P07 reviews remain scoped to their own verification records, and no new visual review is claimed here.

# Remaining dependencies

External authoritative time requires a selected provider, trust configuration and freshness/offline policy. Legal issuer provenance needs actual authorization and covered-material/grant evidence. Plugin notices need version-bound original sources; support and Marketplace need real producer contracts. These remain explicit dependencies and are not resolved by the frontend manifest. Owner acceptance and merge remain pending.

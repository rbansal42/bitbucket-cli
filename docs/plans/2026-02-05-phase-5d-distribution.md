# Phase 5d: Distribution & Releases Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up automated releases and distribution for the bb CLI.

**Architecture:** Use GoReleaser for building and releasing, GitHub Actions for CI/CD, and Homebrew tap for macOS/Linux distribution.

**Tech Stack:** GoReleaser v2, GitHub Actions, Homebrew

---

## Completed Tasks

### Task 1: CI Workflow
**Files:** `.github/workflows/ci.yml`

Created GitHub Actions workflow for:
- Running tests on push to main/dev and PRs
- Running go vet
- Building for all platforms (linux, darwin, windows) x (amd64, arm64)

### Task 2: Release Workflow
**Files:** `.github/workflows/release.yml`

Created GitHub Actions workflow for:
- Triggering on version tags (v*)
- Running tests
- Running GoReleaser to build and publish releases

### Task 3: GoReleaser Configuration
**Files:** `.goreleaser.yml`

Updated existing config to include:
- Homebrew tap support
- All existing build configurations preserved

### Task 4: License File
**Files:** `LICENSE`

Added MIT license.

### Task 5: README
**Files:** `README.md`

Created comprehensive README with:
- Installation instructions (Homebrew, binary download, source)
- Quick start guide
- Full command reference
- Shell completion instructions
- Configuration info
- Contributing guide

---

## Manual Setup Required

### Homebrew Tap Repository

To enable Homebrew installation, create a new repository:
1. Create `rbansal42/homebrew-tap` on GitHub
2. Add `HOMEBREW_TAP_GITHUB_TOKEN` secret to bb repository with write access to the tap repo

### Creating a Release

```bash
# Tag the release
git tag v0.1.0
git push origin v0.1.0

# The release workflow will automatically:
# 1. Run tests
# 2. Build binaries for all platforms
# 3. Create GitHub release with binaries
# 4. Update Homebrew formula (if tap repo exists)
```

---

## Summary

| Task | Files | Status |
|------|-------|--------|
| CI Workflow | .github/workflows/ci.yml | Done |
| Release Workflow | .github/workflows/release.yml | Done |
| GoReleaser Config | .goreleaser.yml | Updated |
| LICENSE | LICENSE | Done |
| README | README.md | Done |

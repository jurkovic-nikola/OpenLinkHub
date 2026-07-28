# LumenForge Maintenance and Reliability Backlog

## Purpose and working principles

This document tracks cautious maintenance, reliability, testing, and targeted safety work. It is not a broad architectural roadmap, and its entries are not automatically release blockers.

- Prefer small, inspection-first branches tied to concrete bugs or features.
- Do not refactor stable hardware-control code merely to reduce duplication.
- Add deterministic RGB output tests before broad RGB rendering changes.
- Treat repeated device code as potentially device-specific until tests and hardware evidence show otherwise.
- Introduce narrow helpers only when the relevant persistence or device code is already changing.
- Do not create generalized migration infrastructure without a real migration.

## Completed and validated

- Local-only networking.
- Localhost API and browser-origin protection.
- Immutable application and mutable runtime-state separation.
- User and system service path resolution.
- Installer hardening and rollback.
- Real user-service installation, upgrade, reboot, udev, memory, OpenRGB, cooling, RGB, and runtime-state validation.
- Runtime-path separation merged through PR #6 and installed from merge commit `72defaa7b313d5dc11b535fc404edfb6d70bfdac`.

## Immediate reliability follow-up

These are bounded follow-ups with specific failure modes or validation gaps. They should remain separate, reviewable changes.

### 1. Repository formatting baseline

Run `gofmt` on:

- `src/rgb/cyberpunkglitch.go`
- `src/systray/devices_tray.go`

This is behavior-neutral cleanup to restore a completely green all-files formatting baseline.

### 2. Transactional LCD upload replacement

Current LCD upload handling may overwrite or truncate the destination before activation and sibling cleanup fully succeed.

- Preserve and restore the previous on-disk destination if activation or cleanup fails.
- Add tests covering overwrite rollback and sibling-cleanup failure.
- Keep the change limited to LCD upload persistence; do not refactor unrelated LCD rendering.

### 3. OpenRGB-import profile persistence errors

`saveDeviceProfile` currently ignores some directory-creation, JSON-marshalling, and file-write failures.

- Propagate or report failures instead of continuing as though persistence succeeded.
- Add failure-injection tests using temporary directories.
- Preserve current filenames, profile schema, and successful behavior.

### 4. Metadata reader cleanup

Ensure opened metadata files are closed when JSON decoding fails in the `cpro`, `ccxt`, `lnpro`, and `lsh` loaders. Preserve each loader's existing fallback and fatal-error policy.

## Targeted lifecycle investigation

Similar `activeRgb` pointer ownership and goroutine lifecycle code exists across many device modules, while the current targeted race tests do not execute most real hardware loops.

- Do not perform a repository-wide conversion.
- Begin with representative devices that are physically available for testing.
- Add lifecycle tests covering start, update, replacement, stop, and shutdown.
- Establish one proven ownership and synchronization pattern before applying it to additional devices.
- Treat this as post-security reliability work unless a real crash, race report, or shutdown bug appears.

## Longer-term maintenance

These items should begin only when their stated prerequisite or a concrete maintenance need exists.

### Deterministic RGB output tests

Before broad RGB rendering work, add byte-exact tests for representative rainbow, wave, static, gradient, temperature-based, inverted, AIO, and LCD-masked output. Use fixed start times and deterministic random seeds. These tests are the prerequisite for broad output assembly, timing, dispatch, or buffer-finalization changes.

### Thin profile and path helpers

When profile persistence is already changing, consider small helpers for category paths, typed JSON loading, existing save behavior, and directory creation. Preserve filenames, directory layout, fallbacks, error behavior, and overwrite semantics; test with temporary directories.

### Atomic persistence primitives

When related persistence code changes, consider narrow temporary-file, sync, and atomic-replacement primitives. Preserve existing schemas and successful behavior, and add failure-path tests before adopting a helper elsewhere.

### Effect timing cleanup

Only after deterministic timing tests exist, a small helper may centralize elapsed time, speed factors, or default-speed handling for effects already being modified. Preserve animation speed and direction exactly; do not perform a repository-wide conversion.

### Cluster dispatch cleanup

Simplify cluster effect dispatch only when a concrete change makes the existing switch materially difficult to maintain. Keep special cases explicit, preserve arguments and side effects, and compare outputs before and after.

### Careful OpenRGB handler sharing

Share OpenRGB handler code only where behavior is demonstrably compatible. Persistence, error handling, live versus durable state, speed overrides, rollback, controller requirements, and asynchronous lifecycles differ; prefer narrow helpers over a generic wrapper.

### Versioned migrations

Create versioned migration infrastructure only for a real incompatible profile or configuration schema change. Any such work must include backups, clear version detection, idempotence, temporary-directory integration tests, and protection against overwriting user data.

### Unsupported SysV init script

Retire the unsupported SysV init script or modernize it only when that installation path has a confirmed user and can be validated. Do not treat it as equivalent to the supported user and system service paths.

## Active security work tracked separately

The following are active security changes, not general maintenance backlog items:

- External Source Registry redesign.
- Backup restore hardening.

Their implementation and completion status are tracked separately.

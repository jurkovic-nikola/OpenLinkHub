# LumenForge Maintenance and Reliability Backlog

## Purpose and working principles

This document tracks cautious maintenance, reliability, testing, and targeted safety work. It is not a broad architectural roadmap, and its entries are not automatically release blockers.

LumenForge is a local, alpha-stage RGB and fan controller maintained by one developer. Evaluate maintenance and audit findings proportionally to that purpose, maturity, threat model, and actual usage. A technically possible edge case is not automatically a bug that must be fixed; rare or theoretical conditions should normally be recorded rather than scheduled.

- Prioritize realistic security vulnerabilities, hardware-safety risks, ordinary user-data loss, crashes, hangs, or regressions during plausible normal use, and reproducible alpha-tester reports.
- Prefer small fixes whose complexity is proportionate to the problem.
- Prefer small, inspection-first branches tied to concrete bugs or features.
- Do not refactor stable hardware-control code merely to reduce duplication.
- Add deterministic RGB output tests before broad RGB rendering changes.
- Treat repeated device code as potentially device-specific until tests and hardware evidence show otherwise.
- Introduce narrow helpers only when the relevant persistence or device code is already changing.
- Do not create generalized migration infrastructure without a real migration.
- Do not introduce broad rewrites, enterprise-grade transaction systems, generalized abstractions, or large maintenance burdens solely for contrived concurrency, filesystem, power-loss, or rollback scenarios.
- Weigh regression risk and ongoing maintenance cost against the benefit of a proposed fix.
- Give significant weight to real hardware validation and alpha-testing evidence.
- Documented findings may remain observations without becoming release blockers or planned work.

## Completed and validated

- Local-only networking.
- Localhost API and browser-origin protection.
- Immutable application and mutable runtime-state separation.
- User and system service path resolution.
- Installer hardening and rollback.
- Real user-service installation, upgrade, reboot, udev, memory, OpenRGB, cooling, RGB, and runtime-state validation.
- Runtime-path separation merged through PR #6 and installed from merge commit `72defaa7b313d5dc11b535fc404edfb6d70bfdac`.
- Post-audit reliability cleanup merged through commit `77b8fa961a309389b48854bea5004a4a43b81497`:
  - Restored the complete tracked-Go formatting baseline.
  - Made mutable LCD uploads transactional.
  - Serialized LCD upload transactions so an older failed upload cannot roll back a newer successful upload.
  - Added focused LCD tests for replacement, rollback, cleanup, cache and live-state preservation, and concurrent transactions.
  - Ensured metadata files are closed on decode failures in `cpro`, `ccxt`, `lnpro`, and `lsh`.
  - Validated with the full Go test suite, Go vet, focused race tests, frontend tests, installer checks, a successful build, and installation and normal operation on real hardware.

## Observations to revisit only if reproduced

### OpenRGB profile persistence failure handling

Some OpenRGB profile persistence paths may handle unusual filesystem failures imperfectly. Normal effect changes, cluster-mode changes, profile persistence, service restart, and state restoration have been manually validated successfully.

A large transactional redesign was investigated but intentionally not retained because its complexity and regression risk were disproportionate to the rare theoretical failure modes. Do not redesign this subsystem pre-emptively. Revisit it only when alpha testing produces a reproducible failure under realistic conditions, or when related persistence code must change for a concrete feature or bug.

## Conditional reliability investigations

### `activeRgb` ownership and goroutine lifecycle

Similar `activeRgb` pointer ownership and goroutine lifecycle patterns exist across many device modules. No normal-use crash, shutdown failure, or race has been demonstrated.

Do not begin a repository-wide or representative-device investigation solely because the repeated pattern exists. Revisit it only after a reproducible crash, an actual race report, a shutdown or effect-lifecycle failure, or an alpha-tester report tied to this area. Any future work must begin with one physically available device and focused lifecycle tests, not a broad sweep.

## Deferred maintenance — trigger required

These subjects are not scheduled merely because they are listed. Begin one only when activated by a concrete feature, a reproducible bug, an alpha-testing report, a real schema migration, or a demonstrated maintenance obstacle, while preserving its stated prerequisites and safety guidance.

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

## Current project priorities

1. External Source Registry redesign.
2. Backup restore hardening.
3. Release-readiness validation.
4. Next alpha release and tester feedback.

## Active security work tracked separately

The following remain active because they address realistic trust-boundary and untrusted-input concerns rather than theoretical maintenance edge cases:

- External Source Registry redesign.
- Backup restore hardening.

Their implementation and completion status are tracked separately.

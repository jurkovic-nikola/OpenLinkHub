# LumenForge Core Refactor Backlog

## Purpose

This document tracks optional code-quality and safety work that may be worth revisiting after the current alpha has been tested in normal use.

It is not a release blocker or an active implementation roadmap. Completed refactors and bug fixes have been removed. New work should begin only when a concrete maintenance need, bug, or feature change justifies it.

## Priority 1: Add deterministic RGB output tests

Before making broad changes to RGB effect rendering, add deterministic tests for representative effects.

Recommended coverage:

- Rainbow
- Wave
- Static
- Gradient
- Temperature-based effects
- Inverted output
- AIO and LCD masking behaviour
- Fixed start times and deterministic random seeds

The tests should compare exact output bytes for known inputs.

### Why this matters

RGB rendering is highly visible and sensitive to byte ordering, offsets, device-specific masking, and timing. Golden tests would make later cleanup safer and help catch regressions that may otherwise appear only on physical hardware.

### When to do it

Do this before any broad refactor of effect output assembly, timing, dispatch, or buffer finalization.

---

## Priority 2: Inspect lifecycle and resource safety when touching device modules

Several targeted lifecycle and file-safety issues have already been fixed. Do not perform a broad sweep solely for consistency.

When modifying a device module, inspect nearby code for:

- goroutines starting before required state is assigned
- shared `activeRgb` access without a stable local reference
- `os.Open` or ZIP entry readers used before checking errors
- missing `Close` calls
- duplicate HTTP responses after an error response
- persistence operations that silently replace or discard user data

### Why this matters

These issues can cause panics, races, descriptor leaks, or damaged runtime state. Targeted inspection keeps the risk lower than a repository-wide refactor.

### When to do it

Perform this review as part of work already touching the relevant module or when a concrete bug points to that area.

---

## Priority 3: Introduce thin profile-path helpers when persistence code next changes

Profile and runtime-state code still contains repeated path construction and file operations across modules.

A future cleanup may introduce small helpers for:

- constructing category and profile paths
- loading typed JSON from a known path
- saving profile data through the existing persistence behaviour
- ensuring directories exist without changing overwrite semantics

### Constraints

- Preserve current filenames and directory layout.
- Preserve existing fallback and error behaviour.
- Use temporary directories in tests.
- Do not introduce automatic migrations as part of this work.
- Do not overwrite or normalize existing user files unexpectedly.

### Why this matters

Thin helpers could reduce repetition without creating a large persistence framework.

### When to do it

Only when profile persistence is already being changed for a feature or bug fix.

---

## Priority 4: Normalize effect timing only when modifying effects

Many effects repeat elapsed-time and speed calculations.

A small helper could eventually centralize:

- elapsed seconds from a start time
- speed-factor calculation
- default-speed handling

### Constraints

- Add deterministic timing tests first.
- Preserve current animation speed and direction exactly.
- Migrate only the effects already being modified.
- Avoid a repository-wide conversion in one branch.

### Why this matters

This could reduce repeated calculations, but it has little user-visible value by itself and can subtly change animation behaviour.

---

## Priority 5: Simplify cluster effect dispatch if it becomes difficult to maintain

The cluster effect dispatcher contains repeated patterns that call an effect and then return its output.

A small helper may reduce repetition while keeping special cases explicit.

### Constraints

- Preserve effect-specific arguments and side effects.
- Keep temperature, gradient, and other special cases readable.
- Compare outputs before and after the change.
- Do not replace the dispatcher with a generic map if that hides meaningful differences.

### When to do it

Only when adding or changing cluster effects makes the existing switch materially harder to maintain.

---

## Priority 6: Revisit OpenRGB handler sharing only for a concrete need

The OpenRGB handlers already share request decoding where behaviour is compatible.

Further consolidation should be approached cautiously because the endpoints differ in:

- persistence behaviour
- profile-save error handling
- live versus durable state
- speed override behaviour
- layout rollback behaviour
- active-controller requirements
- asynchronous lifecycle operations

### Recommendation

Prefer small, narrowly scoped helpers over one generic handler wrapper.

Do not consolidate endpoints merely to reduce line count.

---

## Deferred unless required by a real migration

### Versioned profile migration system

Do not introduce a generalized migration manager pre-emptively.

Create migration infrastructure only when LumenForge has an actual profile or configuration schema change that cannot be handled safely by the existing compatibility logic.

Any future migration system must include:

- backups before destructive changes
- temporary-directory integration tests
- idempotent migration behaviour
- clear version detection
- protection against overwriting user data

---

## Not currently recommended

### Full RGB `OutputBuilder` migration

A broad conversion of all RGB effects to a shared output builder is not currently justified.

It would touch runtime-critical rendering behaviour involving:

- byte ordering
- offsets
- inverted output
- AIO and LCD masking
- temporary buffers
- device-specific layout assumptions

A small finalization helper may be considered later, but only after deterministic RGB tests exist and only when there is a concrete maintenance benefit.

### Repository-wide `activeRgb` sweep

Do not modify every remaining device module simply to match previously fixed modules.

Inspect and correct lifecycle handling only when:

- a race or nil-pointer bug is observed
- tests expose unsafe ordering
- the module is already being changed

### Broad file-loading abstraction

Do not replace existing file-loading code across the repository with a generic framework.

Target concrete safety issues and preserve current module-specific behaviour.

---

## Current status

The previous low-risk helper cleanup and targeted safety work are complete, including:

- shared JSON request decoding for compatible handlers
- shared fixed-template execution handling
- trimmed integer parsing helper
- targeted systray menu-layout cleanup
- cluster and selected device `activeRgb` lifecycle fixes
- HTTP double-response prevention
- file-open error handling and descriptor cleanup
- backup integrity-reader safety
- OpenRGB lifecycle and persistence fixes completed in later focused branches

No item in this document is required before testing or releasing the current alpha.

## Working rule

Prefer small, inspection-first branches tied to a concrete bug or feature.

Do not start broad refactors solely to remove duplication from stable hardware-control paths.

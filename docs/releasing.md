# Release Process

This document is for LumenForge maintainers preparing an authorized release.
It is not needed for ordinary source builds or installation.

## Development and Release Versions

Ordinary tracked builds use the development fallback version
`0.2.0-alpha-dev`. Both development and release builds use the shared build
script:

```bash
./scripts/build.sh
```

A release build must explicitly provide `VERSION`. The script removes one
optional leading lowercase `v` before embedding the version. It does not
normalize an uppercase `V` or arbitrary prefixes. Creating a Git tag alone does
not inject that tag into the build.

The current release-build form is:

```bash
VERSION=v0.2.0-alpha ./scripts/build.sh
```

## Alpha Release Checklist

1. Complete and merge focused feature and documentation branches into
   `LumenForge-Dev`.
2. Run one combined validation pass on `LumenForge-Dev`.
3. Review the accumulated changes as an alpha release candidate.
4. Change the changelog heading from `Unreleased` to the release version and
   date.
5. Build with the authorized release version supplied through `VERSION`.
6. Confirm the binary reports the intended release version.
7. Merge the approved release into `main`.
8. Create the matching Git tag, such as `v0.2.0-alpha`.
9. Push the approved branch and tag.
10. Move the tracked development fallback to the next selected development
    version after release.

The exact release process may change as release automation is finalized.

## Safety Notes

- Perform release work from a clean working tree and index.
- Validate the exact commit that will be tagged.
- Do not assume that creating a tag changes the embedded binary version.
- Preserve runtime and personal device data outside release artifacts.

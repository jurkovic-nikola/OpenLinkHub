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
4. Decide whether Docker publishing is included or deliberately withheld.
5. Change the changelog heading from `Unreleased` to the release version and
   date.
6. Build with the authorized release version supplied through `VERSION`.
7. Confirm the binary reports the intended release version.
8. Merge the approved release into `main`.
9. Create the matching Git tag, such as `v0.2.0-alpha`.
10. Push the approved branch and tag.
11. Move the tracked development fallback to the next selected development
    version after release.

The exact release process may change as Docker support and release automation
are finalized.

## Docker Status

- Docker publishing is not yet authorized as part of the alpha release
  process.
- The inherited Docker build path currently requires separate validation.
- Docker version injection and publishing controls will be documented after
  the `LumenForge-Docker-Publish-Guard` work is completed.

Do not invent or infer a Docker release command before that work defines and
validates the publishing path.

## Safety Notes

- Perform release work from a clean working tree and index.
- Validate the exact commit that will be tagged.
- Do not assume that creating a tag changes the embedded binary version.
- Do not publish Docker images until that path is explicitly validated and
  authorized.
- Preserve runtime and personal device data outside release artifacts.

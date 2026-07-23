#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "$script_dir/.." && pwd)"
output_path="${OUTPUT:-$repository_root/LumenForge}"
version_override="${VERSION:-}"

build_args=(build -o "$output_path")

if [[ -n "$version_override" ]]; then
  normalized_version="${version_override#v}"
  if [[ -z "$normalized_version" ]]; then
    echo "VERSION must not be empty after removing an optional leading v." >&2
    exit 1
  fi
  if [[ "$normalized_version" =~ [[:space:]] ]]; then
    echo "VERSION must not contain whitespace." >&2
    exit 1
  fi
  build_args+=(-ldflags "-X LumenForge/src/version.Version=$normalized_version")
  version_message="explicit VERSION override $normalized_version"
else
  version_message="tracked development fallback"
fi

if [[ -z "${CGO_CFLAGS_ALLOW+x}" ]]; then
  export CGO_CFLAGS_ALLOW="-fno-strict-overflow"
fi

printf 'Building LumenForge to %s using %s.\n' "$output_path" "$version_message"

cd -- "$repository_root"
go "${build_args[@]}" .

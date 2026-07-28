# External Temperature Sources

LumenForge temperature sensor type `7` reads local command output through a
trusted External Source Registry. Temperature profiles store only an opaque
registry `id`; executable paths and arguments never come from a profile,
browser request, or dashboard field.

## Registry location

LumenForge reads exactly one `external-sources.json` file for the active service
mode:

| Mode | Registry path |
| --- | --- |
| User service | resolved user configuration directory + `external-sources.json`, normally `$XDG_CONFIG_HOME/lumenforge/external-sources.json` or `~/.config/lumenforge/external-sources.json` |
| System service | `/etc/lumenforge/external-sources.json` |
| Direct development run | `external-sources.json` in the repository/current working directory |

The file is optional. A missing file means that no external sources are
available and does not prevent LumenForge from starting. LumenForge and its web
UI never create or edit this file; create and maintain it locally with an
appropriate text editor. The installers do not create the file.

For a user service, the registry must be a regular, non-symlink file owned by
the service user and must not be group- or world-writable. For a system service,
it must be a regular, non-symlink file owned by root and must not be group- or
world-writable. Development mode still requires a regular, non-symlink file,
valid JSON, and safe executables, but does not impose installed-service
ownership or permission checks.

## JSON format

The registry has one `sources` array. Each source requires a unique `id`, a
human-readable `name`, an absolute `executable`, and a fixed `args` string
array:

```json
{
  "sources": [
    {
      "id": "gpu-temperature",
      "name": "GPU Temperature",
      "executable": "/usr/local/bin/read-gpu-temperature",
      "args": ["--gpu", "0"]
    }
  ]
}
```

An `id` is 1-64 letters, numbers, dots, underscores, or hyphens. Profiles use
this stable `id`; `name` is only the dashboard label. The executable must
resolve through symlinks to an existing regular file with an execute bit. The
same executable may be used by multiple ids with different fixed arguments.

Malformed JSON, unknown fields, duplicate or invalid ids, missing fields, an
insecure registry file, or an unsafe executable makes the registry unavailable.
The error is logged locally and reported without executable details by the
read-only `/api/external-sources` endpoint. Other LumenForge device, RGB,
cooling, and temperature-source behavior remains available.

## Execution and output

LumenForge reloads the small registry on demand and revalidates the selected
canonical executable immediately before every execution. It starts that path
directly with exactly the registered argument array. It does not invoke a shell
and performs no substitution, expansion, interpolation, or profile-supplied
argument handling.

Every command has a hard two-second timeout. Standard output and captured
standard error are each limited to 4 KiB. Standard output, after ordinary
surrounding whitespace is removed, must contain exactly one finite numeric
value. Empty output, extra text or values, malformed numbers, `NaN`, infinity,
oversized output, timeouts, and process failures are rejected.

Old sensor-type-7 profile fields containing arbitrary executable paths are not
migrated and are never executed. Recreate those profiles by selecting a trusted
registry id in the dashboard.

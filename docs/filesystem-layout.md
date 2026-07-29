# Filesystem Layout and Ownership

LumenForge separates installed application resources from configuration and
runtime-owned data. This layout applies to fresh installations and allows an
installer rerun to replace the application without altering user-created state.

## Installed paths

| Purpose | User service | System service |
| --- | --- | --- |
| Application root | `/opt/LumenForge` | `/opt/LumenForge` |
| Configuration directory | `$XDG_CONFIG_HOME/lumenforge/`, or `~/.config/lumenforge/` | `/var/lib/lumenforge/` |
| Configuration file | configuration directory + `config.json` | `/var/lib/lumenforge/config.json` |
| Mutable data root | `$XDG_DATA_HOME/lumenforge/`, or `~/.local/share/lumenforge/` | `/var/lib/lumenforge/` |
| Mutable database | data root + `database/` | `/var/lib/lumenforge/database/` |
| External Source Registry | configuration directory + `external-sources.json` | `/etc/lumenforge/external-sources.json` |

Custom `XDG_CONFIG_HOME` and `XDG_DATA_HOME` values must be absolute. The user
installer resolves them when it creates the service unit.

## Immutable application tree

`/opt/LumenForge` is owned by `root:root`. Directories are normally `0755`, the
binary is `0755`, and ordinary shipped files are `0644`. The system account and
desktop user can read this tree but cannot modify it.

The tree contains the binary, HTML templates, JavaScript, CSS, fonts, images,
documentation, API and OpenRGB documentation, shipped device definitions,
language data, RGB definitions, and bundled default LCD media. It does not
contain `config.json`, mutable profiles, uploads, generated state, logs, or a
runtime-owned database.

Shipped database content includes `external/`, `keyboard/`, `language/`,
`motherboard/`, `nexus/`, `xeneon/`, `rgb.json`, `lcd/background.jpg`, and
bundled `lcd/images/`.

## Mutable data

The user service owns its XDG configuration and data roots; both are normally
`0700`. The system service uses the `lumenforge:lumenforge`-owned
`/var/lib/lumenforge` directory, normally `0750`, with stricter mutable
subdirectories. Both service units use `UMask=0077`, and `config.json` is
written with mode `0600`.

Mutable database content includes:

- `profiles/` for device and cluster profiles;
- `rgb/` for per-device RGB state;
- `temperatures/` for cooling and temperature profiles;
- `macros/`, `key-assignments/`, and `led/`;
- `lcd/` for editable LCD modes and `lcd/images/` for uploads;
- `audio.json`, `scheduler.json`, and `openrgbimport-zones.json`.

Dashboard and display state live at the mutable data root. Bundled LCD media is
read from the application tree, while uploaded media with the same name takes
precedence from the mutable LCD directory.

## Configuration behavior

The dashboard is not a general `config.json` editor. LumenForge may create the
file, add newly supported defaults, and persist limited runtime-managed fields,
including supported-device exclusions. Internal application, configuration,
and data roots are never written to `config.json` or exposed as dashboard
settings.

Installed service paths are absolute, cleaned, and independent of the process
working directory. A direct run without `LUMENFORGE_SERVICE_MODE` is
development mode: it uses the current working directory for shipped resources,
`config.json`, mutable data, and `external-sources.json`. When started from the
repository root, the registry is `<repository>/external-sources.json`.

## Logging and backups

An empty `logFile` writes to standard error, which systemd records in the
system or user journal. `"-"` has the same behavior. A relative explicit path
is resolved beneath the mutable data root. An explicit destination inside
`/opt/LumenForge` is rejected.

Dashboard backup creation includes the resolved `config.json`, mutable
`database/`, `dashboard.json`, and `display.json` when present. It excludes all
immutable application content. Restore maps `config.json` to the configuration
directory and mutable entries to the data root; it never restores into
`/opt/LumenForge`. Restore validates and stages the complete snapshot before
replacement, preserves the current `logFile` and `amdsmiPath`, and requires an
immediate service restart. See [Backup and Restore](backup-restore.md) for the
accepted ZIP structure, limits, and snapshot semantics.

The optional `/etc/lumenforge/external-sources.json` file is root-controlled
administrator configuration for the system service. The current installers do
not create it. The user-service equivalent is stored in that user's resolved
configuration directory. See [External Source Registry](external-sources.md)
for ownership, permissions, schema, and execution rules.

# Backup and Restore

LumenForge's Settings page downloads and restores a ZIP snapshot of its
configuration and mutable runtime state. Restore is an alpha feature: make a
separate copy of important state before testing it, and treat every uploaded
backup as untrusted input.

## Backup format

A LumenForge backup contains:

```text
config.json
database/
database/**                 optional files and subdirectories
dashboard.json              optional
display.json                optional
_hash.txt
```

`config.json` and `_hash.txt` must each appear exactly once. Directories are
accepted only beneath `database/`. Every other entry must be a regular file at
one of the paths shown above. Absolute paths, traversal, backslashes,
non-canonical or duplicate paths, symbolic links, devices, FIFOs, sockets, and
other special entries are rejected.

The compressed HTTP upload limit is 5 MiB. After opening the ZIP, restore also
enforces these uncompressed limits:

| Limit | Maximum |
| --- | ---: |
| Archive entries, including directories | 4,096 |
| Path depth | 16 components |
| One regular file | 32 MiB |
| All restored regular files combined | 128 MiB |
| `_hash.txt` | 128 bytes |

Both ZIP metadata and the bytes produced by decompression are bounded. These
limits are intentionally above LumenForge's current profiles and 5 MiB media
upload limit while preventing a small compressed upload from expanding without
bound.

`_hash.txt` is the SHA-256 digest of the concatenated regular-file contents in
archive order, excluding `_hash.txt` itself. It detects accidental corruption;
it is not a signature and does not prove who created the backup. Restore also
checks ZIP decompression and CRC errors and validates the JSON syntax of
`config.json`, `dashboard.json`, `display.json`, and every `.json` file beneath
`database/`. Non-JSON mutable files, such as LCD media, remain supported.

External Source Registry files are not part of this format and are never
backed up or restored.

## Restore behavior

Restore validates the complete archive before creating private staging
directories beside the configured destinations. Staging directories use mode
`0700`; staged files use mode `0600`. Nothing at a live restore target is
changed until structure, limits, the corruption hash, decompression, and JSON
syntax have all passed.

At commit time LumenForge briefly renames each current target to a unique
sibling name, renames the staged replacement into place, and then removes the
temporary originals after all targets have been installed. If a later rename
fails, it attempts to put every original back. A validation or staging failure
therefore leaves live state unchanged, and ordinary commit failures roll back.

This is a small local rename transaction, not a filesystem snapshot. A sudden
power loss during the bounded rename sequence can still leave an intermediate
state, and a filesystem failure can prevent rollback. Check the local service
log if restore reports a commit or rollback failure.

Restore uses snapshot semantics:

- `database/` replaces the complete live mutable database; files present only
  in the old database are removed.
- A present `dashboard.json` or `display.json` replaces the live file.
- If either optional file is absent from the backup, its live copy is removed.
- The archived `logFile` and `amdsmiPath` values never replace the current
  host's values. All other archived configuration fields, including unknown
  fields, are retained.

Installed user services restore `config.json` beneath
`$XDG_CONFIG_HOME/lumenforge/` (falling back to `~/.config/lumenforge/`) and
mutable data beneath `$XDG_DATA_HOME/lumenforge/` (falling back to
`~/.local/share/lumenforge/`). The system service restores both beneath
`/var/lib/lumenforge/`. Restore never targets `/opt/LumenForge` or
`/etc/lumenforge`.

A direct development run still uses its working directory for configuration
and mutable data, but restore accepts only `config.json`, `database/**`,
`dashboard.json`, and `display.json`. It cannot restore source files, static
assets, templates, `go.mod`, installer scripts, or other application content.

Backup creation and restore are serialized with each other. Ordinary runtime
profile or device writes are not globally paused, so restart immediately after
a successful restore and do not make further dashboard changes first.

## Required restart

LumenForge does not restart itself. A successful response means that files were
replaced, not that the running process reloaded them.

For a user-service installation:

```bash
systemctl --user restart LumenForge.service
```

For a system-service installation:

```bash
sudo systemctl restart LumenForge.service
```

## Troubleshooting

Inspect the user-service log and status:

```bash
systemctl --user status LumenForge.service
journalctl --user -u LumenForge.service -n 100 --no-pager
```

Inspect the system-service log and status:

```bash
sudo systemctl status LumenForge.service
sudo journalctl -u LumenForge.service -n 100 --no-pager
```

If restore rejects a backup, keep the current live data in place. Do not edit
`_hash.txt` to force acceptance: recreate the backup from a known-good running
installation or inspect the ZIP offline for the reported structure, size,
corruption, or JSON problem.

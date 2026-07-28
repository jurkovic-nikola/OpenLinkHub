# External Source Registry

External Source lets a cooling profile obtain a numeric temperature from an
administrator-approved local executable. A profile stores only an opaque
`externalSourceId`. Executable paths and arguments are defined in a trusted,
locally administered registry; the browser and API cannot submit arbitrary
paths or arguments. LumenForge starts the registered executable directly
without a shell.

## Registry location

LumenForge reads exactly one `external-sources.json` file for the active service
mode:

| Mode | Registry path |
| --- | --- |
| User service | `$XDG_CONFIG_HOME/lumenforge/external-sources.json`, falling back to `~/.config/lumenforge/external-sources.json` when `XDG_CONFIG_HOME` is unset |
| System service | `/etc/lumenforge/external-sources.json` |
| Direct development run | `<current working directory>/external-sources.json`; when started from the repository root, this is `<repository>/external-sources.json` |

The registry is optional and is not created by LumenForge or either installer.
A missing file leaves the External Source dropdown empty and does not prevent
LumenForge from starting.

## Registry format

Create a JSON object with one `sources` array:

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

Each entry has four required fields:

- `id` is the stable value stored as `externalSourceId` in temperature
  profiles. It must be unique and contain 1-64 letters, numbers, dots,
  underscores, or hyphens.
- `name` is the human-readable label shown in the registry dropdown.
- `executable` is an absolute path to the local program.
- `args` is the fixed, ordered array of string arguments passed to the program.

Multiple IDs may use the same executable with different fixed argument arrays.
Unknown fields, duplicate IDs, missing fields, and malformed JSON invalidate
the registry.

## Ownership and permissions

The registry must be a regular file and must not be a symbolic link. The
executable path may resolve through symbolic links, but its resolved target
must be a regular file with at least one execute bit.

For a user service:

- the service user must own the registry;
- the registry must not be group- or world-writable;
- the current service user or root must own the executable; and
- the executable must not be group- or world-writable.

For example, using the default user configuration fallback and a user-owned
executable:

```bash
mkdir -p ~/.config/lumenforge
chmod 600 ~/.config/lumenforge/external-sources.json
chmod 700 ~/.local/bin/example-temperature-source
```

If `XDG_CONFIG_HOME` is set, apply the registry mode to
`$XDG_CONFIG_HOME/lumenforge/external-sources.json` instead.

For a system service, root must own both the registry and executable, and
neither may be group- or world-writable. For existing files:

```bash
sudo install -d -o root -g root -m 755 /etc/lumenforge
sudo chown root:root /etc/lumenforge/external-sources.json
sudo chmod 600 /etc/lumenforge/external-sources.json
sudo chown root:root /usr/local/bin/read-gpu-temperature
sudo chmod 755 /usr/local/bin/read-gpu-temperature
```

Development mode still requires a regular, non-symlink registry containing
valid JSON. It does not check the registry owner or its group/world write bits.
Its executable policy is the same as user-service mode: the resolved executable
must be a regular executable file owned by the current user or root and must
not be group- or world-writable.

Do not use mode `777` or place trusted executables in shared writable
locations.

## Executable output contract

The executable must finish within two seconds and write exactly one finite
numeric value to standard output. Surrounding whitespace and a final newline
are allowed, but units, labels, or any other text are not. Standard output must
not exceed 4 KiB. Captured standard error is independently limited to 4 KiB.

Valid output:

```text
42
42.5
-5.25
```

Invalid output:

```text
42.5 C
temperature=42.5
NaN
Infinity
42.5 43
```

Positive and negative infinity are rejected, as are empty output, process
failures, timeouts, and values that cannot be represented as a finite
temperature.

## User-service setup

1. Confirm that the user service is running with
   `systemctl --user status LumenForge.service`.
2. Create or install a trusted executable that follows the output contract.
3. Verify the executable owner and mode, for example with
   `stat -c 'owner=%U group=%G mode=%a path=%n' ~/.local/bin/example-temperature-source`.
4. Create `external-sources.json` at the user-service path listed above.
5. Set the registry permissions, normally with
   `chmod 600 ~/.config/lumenforge/external-sources.json`.
6. Restart LumenForge with
   `systemctl --user restart LumenForge.service`.
7. Hard-refresh the browser so it fetches the current registry list.
8. Open the temperature page and create a cooling profile.
9. Select `External Source` as the sensor type.
10. Select the registry entry by its display `name`.
11. Save the profile and assign it to the appropriate channel as needed.

For a system-service installation, use the root-owned registry rules above and
restart with:

```bash
sudo systemctl restart LumenForge.service
```

Configure cooling curves conservatively and test them on your own hardware.

### Testing-only fixed value

This user-service registry entry uses the commonly root-owned
`/usr/bin/printf` executable to return a fixed `42.5`:

```json
{
  "sources": [
    {
      "id": "fixed-42-5",
      "name": "Fixed 42.5 (testing only)",
      "executable": "/usr/bin/printf",
      "args": ["42.5\\n"]
    }
  ]
}
```

This confirms registry loading and profile selection; it is not a real
hardware sensor. LumenForge passes `42.5\n` as a fixed argument directly to
`printf`. No shell or interpolation is involved.

## Reload and profile behavior

The registry is loaded and validated on demand when the API lists sources,
when a profile selection is validated, and whenever a selected source is
executed. A service restart is therefore not required for the server to notice
a saved registry change. The browser dropdown is populated by an API request,
so hard-refresh the page after changing the file. The selected executable is
also revalidated immediately before execution.

Only `External Source` (sensor type `7`) displays the registry dropdown. Other
sensor types retain their existing device and sensor selectors. The dropdown
shows each entry's `name`, while a saved profile persists only its
`externalSourceId`; executable paths and fixed arguments are never profile
fields. A missing or unknown ID is rejected. Legacy type-7 fields containing
arbitrary executable paths are not migrated and are never executed; recreate
those profiles by selecting a registry entry.

The read-only `GET /api/external-sources` endpoint returns only each entry's
`id` and `name`. There is no API or browser registry editor.

## Troubleshooting and failure behavior

Check the service and recent logs:

```bash
systemctl --user status LumenForge.service
journalctl --user -u LumenForge.service --since "10 minutes ago"
```

For a system service:

```bash
sudo systemctl status LumenForge.service
sudo journalctl -u LumenForge.service --since "10 minutes ago"
```

Inspect ownership and permissions with:

```bash
stat -c 'owner=%U group=%G mode=%a path=%n' <path>
```

Expected failure cases include:

- a missing registry, which produces an empty dropdown;
- malformed JSON, invalid entries, or insecure registry ownership or modes;
- a missing or unknown `externalSourceId`;
- an executable that was removed, replaced, or made unsafe;
- execution exceeding two seconds;
- empty, non-numeric, non-finite, or extra standard output; and
- standard output or captured standard error exceeding its 4 KiB limit.

Registry and execution errors are logged. An external-source failure does not
intentionally stop unrelated RGB, cooling, or device control, but the failed
reading follows LumenForge's existing temperature failure/fallback behavior.
No particular fan speed is guaranteed. Use conservative curves and validate
failure behavior with the hardware and channels you intend to control.

## Security boundaries

The registry is locally administered and has no API or browser editor. Commands
run directly without a shell, profile data cannot supply arguments, and the
resolved executable is revalidated before execution. External Source is a
bounded temperature-input mechanism, not a general plugin framework.
Administrators remain responsible for the executable's contents, dependencies,
side effects, and safety.

## Verified user-service behavior

Manual testing on a real user-service installation confirmed:

- a user-owned mode-`600` registry and root-owned mode-`755`
  `/usr/bin/printf` were accepted;
- the entry appeared in the External Source dropdown;
- the saved profile's External Source selection contained only
  `"externalSourceId": "fixed-42-5"`, and the profile and selection survived a
  service restart;
- a user-owned mode-`700` test executable was invoked by device polling;
- valid `42.5` output was accepted;
- malformed output and an execution longer than two seconds were rejected and
  logged; and
- LumenForge and unrelated controls remained operational during those
  failures.

This test used one ordinary fan channel with a deliberately flat curve. It did
not validate every supported device family. A system-service installation with
a root-owned registry has not yet been tested manually.

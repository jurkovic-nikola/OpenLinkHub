# Configuration Reference

LumenForge reads `config.json` once during startup. The user service uses
`$XDG_CONFIG_HOME/lumenforge/config.json`, falling back to
`~/.config/lumenforge/config.json`. The system service uses
`/var/lib/lumenforge/config.json`. A direct source run uses `config.json` in the
process working directory. Stop LumenForge before editing the file, keep a
backup, and restart it after saving.

Dashboard backup restore preserves the current host's `logFile` and
`amdsmiPath` values while restoring other archived configuration fields. See
[Backup and Restore](backup-restore.md) for the accepted archive structure,
limits, snapshot behavior, and required post-restore restart.

The file is JSON. Field names are case-sensitive, JSON types must match the
types below, and malformed JSON prevents startup. LumenForge currently ignores
unknown fields and performs little general range validation, so an accepted
JSON value can still make a subsystem fail to initialize.

Users normally do not need to change a key unless its row identifies a
hardware, integration, diagnostic, or port-conflict reason to do so.

## Complete Safe Default Configuration

This is the configuration generated for a new installation:

```json
{
  "debug": false,
  "manual": false,
  "frontend": true,
  "metrics": false,
  "listenPort": 27003,
  "logFile": "",
  "logLevel": "info",
  "enableSystemTray": false,
  "enableGamepad": true,
  "enableMotherboard": false,
  "motherboardBiosOnExit": false,
  "checkDevicePermission": false,
  "graphProfiles": true,
  "resumeDelay": 15000,
  "temperatureOffset": 0,
  "cpuSensorChip": "",
  "cpuTempFile": "",
  "memory": false,
  "memoryType": 5,
  "memorySmBus": "i2c-0",
  "memorySku": "",
  "memoryRegisterOverride": "",
  "ramTempViaHwmon": true,
  "enhancementKits": "",
  "amdGpuIndex": 0,
  "amdsmiPath": "",
  "nvidiaGpuIndex": [
    0
  ],
  "defaultNvidiaGPU": 0,
  "openRGBPort": 6742,
  "exclude": []
}
```

`enableOpenRGBTargetServer` is absent because its default is `false` and its
JSON field uses `omitempty`, so the generated configuration omits that false
zero value. It is serialized when enabled. Internal filesystem roots are not
configuration fields and are never serialized.

## Server and Network

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `listenPort` | integer, `27003` | `1`-`65535` starts the HTTP server; `0` or a negative value disables it | Port for the web UI and HTTP API on `127.0.0.1`. Users normally change it only to resolve a local port conflict. An out-of-range positive port makes server startup fail. | Yes |
| `frontend` | boolean, `true` | `true`, `false` | Registers the HTML interface routes. Setting it to `false` does not disable the HTTP API or static-file route. Most users should leave it enabled. | Yes |
| `metrics` | boolean, `false` | `true`, `false` | Registers the `/api/metrics` endpoint. Enable only when a local monitoring client needs it. It uses the same listener and security boundary as the rest of the API. | Yes |

LumenForge is local-only. The dashboard and HTTP API always listen on IPv4
loopback at `127.0.0.1:<listenPort>`. No configuration value can expose this
listener on a wildcard, LAN, Tailscale, or other network interface, and remote
dashboard/API access is unsupported.

Dashboard and API requests may identify the listener only as
`127.0.0.1:<listenPort>` or `localhost:<listenPort>`. Browser mutations must be
same-origin and carry LumenForge's local request proof. CORS, reverse proxies,
forwarded hosts, and remote web clients are unsupported. See the
[HTTP API guide](../api/README.md#local-request-protection) for command-line
mutation examples.

`listenAddress` is a deprecated compatibility field. Existing configurations
containing it still load, and LumenForge does not rewrite a config merely to
remove it, but the value is ignored by every listener and is no longer written
to new configurations. A configured non-loopback legacy value produces a
startup warning explaining that LumenForge is restricted to `127.0.0.1`.

The server settings are independent of OpenRGB's SDK server. `listenPort` is
LumenForge's HTTP port; `openRGBPort` is the local OpenRGB SDK port.

## Logging and Diagnostics

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `logLevel` | string, `"info"` | `"info"`, `"warn"`, `"error"`, `"fatal"`, or `"silent"`; matching is case-insensitive | Minimum emitted log severity. Any unrecognized value silently behaves as `"info"`. Use `"silent"` only when loss of diagnostic output is acceptable. | Yes |
| `logFile` | string, `""` | Empty, `"-"`, or a filesystem path | Empty or `"-"` writes to standard error, which an installed service records in journald. A relative path is resolved beneath the mutable data root; a safe absolute path selects that file. Destinations beneath `/opt/LumenForge` are rejected. At startup an existing file log is archived as a timestamped `.tar.gz`. | Yes |
| `debug` | boolean, `false` | `true`, `false` | Enables additional device and inherited OpenRGB-compatible target-server diagnostics. It does not change `logLevel`; normally leave it disabled because output can be verbose. | Yes |

## Temperature, GPU, and Profile Selection

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `cpuSensorChip` | string, `""` | Empty for automatic detection, or an exact Linux hwmon `name` value | Selects the CPU hwmon device when automatic matching chooses the wrong sensor. Advanced; for example, `"k10temp"`. A nonmatching name produces no CPU reading. | Yes |
| `cpuTempFile` | string, `""` | Empty for `temp1_input`, or a hwmon filename | Overrides the temperature file within the selected CPU hwmon directory, for example `"temp2_input"`. The value is a filename, not a complete path. A missing or unreadable file produces no reading. | Yes |
| `temperatureOffset` | integer, `0` | Any JSON integer; use a small positive or negative Celsius correction | Added to CPU temperatures read through the hwmon path. It does not calibrate GPU or RAM readings. Most users should leave it at zero. | Yes |
| `defaultNvidiaGPU` | integer, `0` | A nonnegative `nvidia-smi` device index, or `-1` to disable NVIDIA temperature/model use | Selects the default NVIDIA GPU. If its temperature cannot be read, the general GPU-temperature path falls back to AMD. Change only on multi-GPU systems or to disable NVIDIA probing. | Yes |
| `nvidiaGpuIndex` | array of integers, `[0]` | NVIDIA indices; practical values are nonnegative indices accepted by `nvidia-smi` | Supplies the NVIDIA GPU choices used by multi-GPU temperature profiles and device controls. At least two entries are required before the API accepts the multi-GPU sensor type. Invalid indices yield missing/zero readings. The system-information summary currently iterates list positions, so non-contiguous values may not be represented there consistently. | Yes |
| `amdGpuIndex` | integer, `0` | An `amd-smi` GPU index; practical values are nonnegative | Selects the AMD GPU used for model, temperature, and utilization queries. Change only on a multi-GPU AMD system. | Yes |
| `amdsmiPath` | string, `""` | Empty for `amd-smi` from `PATH`, or an executable name/path | Selects the AMD SMI executable. Use only when `amd-smi` is installed outside the service's `PATH`; an unavailable value disables AMD SMI data. | Yes |
| `graphProfiles` | boolean, `true` for a new configuration | `true`, `false` | Uses graph-based cooling-profile editing and control where supported; `false` retains the legacy profile form/control path. Existing configurations first upgraded from versions without this key receive `false` for compatibility. | Yes |

## Device Discovery and Runtime Behaviour

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `manual` | boolean, `false` | `true`, `false` | Disables automatic cooling-curve and temperature-based fan/pump speed adjustment for supported cooling devices, and enables the manual-speed API. Device telemetry, CPU/GPU temperature refresh, device temperature readings, and RPM refresh continue. This is an advanced mode: LumenForge will not apply normal cooling curves while it is enabled. | Yes |
| `exclude` | array of unsigned integers, `[]` | USB product IDs from `0` through `65535` | Omits matching supported devices from LumenForge discovery. The Settings UI maintains this field automatically; manual editing normally is not needed. UI changes are saved immediately, but restart LumenForge to ensure already initialized devices are re-enumerated consistently. | Yes for reliable re-enumeration |
| `checkDevicePermission` | boolean, `false` | `true`, `false` | Requires a discovered HID device's mode bits to match one of LumenForge's expected permission modes before initialization. This is an advanced diagnostic guard; enabling it can cause otherwise usable devices to be skipped. | Yes |
| `resumeDelay` | integer, `15000` | Milliseconds; use `0` or a positive integer | Delay after resume before LumenForge requests process exit so the service manager can restart it. Negative values effectively remove the delay. Change only to accommodate slow device or session recovery. | Yes |
| `enableGamepad` | boolean, `true` | `true`, `false` | Creates the virtual gamepad input path used by supported assignments. Disable it if virtual gamepad creation is unwanted; keyboard and mouse input handling are separate. | Yes |
| `enableMotherboard` | boolean, `false` | `true`, `false` | Enables supported motherboard PWM/fan control. This requires compatible hardware and access; see [Motherboard PWM](motherboard-pwm.md). Leave disabled unless deliberately configuring motherboard control. | Yes |
| `motherboardBiosOnExit` | boolean, `false` | `true`, `false` | Returns a managed motherboard controller to BIOS mode when LumenForge stops. It has no effect unless `enableMotherboard` is enabled. | Yes |
| `enableSystemTray` | boolean, `false` | `true`, `false` | Starts the StatusNotifierItem/D-Bus tray integration when LumenForge runs as a user service inside a compatible desktop session. The system service cannot expose the tray because it runs outside the logged-in user's graphical and session D-Bus environment. **Open Dashboard** opens the configured local dashboard in the default browser, normally at `http://127.0.0.1:27003`. | Yes |

## Memory

Memory control is disabled by default and requires a correctly selected SMBus
device plus suitable device permissions. See [Memory DDR4 / DDR5](memory-configuration.md)
before enabling it.

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `memory` | boolean, `false` | `true`, `false` | Enables SMBus memory-device discovery and control. Leave disabled unless the bus, memory generation, and permissions have been verified. | Yes |
| `memorySmBus` | string, `"i2c-0"` | An exact Linux I2C device basename such as `"i2c-15"` | Chooses the SMBus device. It is used only when `memory` is `true`. Selecting the wrong bus can probe unintended I2C devices, so verify it with the memory guide. | Yes |
| `memoryType` | integer, `5` | `4` for DDR4 or `5` for DDR5 | Selects the memory protocol. Other values cause memory initialization to stop. It is used only when `memory` is `true`. | Yes |
| `memorySku` | string, `""` | Empty or a DIMM SKU/part-number fallback | Supplies a fallback when automatic DDR5 SKU decoding produces no usable value. It does not enable or disable decoding, and it does not override a non-empty decoded SKU. Normally leave it empty. | Yes |
| `ramTempViaHwmon` | boolean, `true` for a new configuration | `true`, `false` | Reads supported DIMM temperatures through the Linux `spd5118` (DDR5) or `jc42` (DDR4) hwmon driver. Existing configurations first upgraded from versions without this key receive `false` for compatibility. Disable if that hwmon path is unavailable or undesirable. | Yes |
| `enhancementKits` | base64 string representing bytes, `""` | Bytes for addresses `0x58`-`0x5f`; for example `"WA=="` represents `[0x58]` | Forces listed memory RGB addresses to be treated as Corsair Light Enhancement Kits. This is an advanced, unsafe override used only with `memory: true`; an incorrect address can misclassify a real DIMM. JSON integer arrays are not accepted by the current decoder. | Yes |
| `memoryRegisterOverride` | base64 string representing bytes, `""` | Bytes for addresses `0x58`-`0x5f`; for example `"WFk="` represents `[0x58, 0x59]` | Forces probing to continue when the initial register read fails. This is an advanced, unsafe hardware-probing override used only with `memory: true`. JSON integer arrays are not accepted by the current decoder. | Yes |

The unusual base64 representation of the last two fields follows Go's JSON
encoding for a byte slice. Leave both empty unless a known hardware workaround
requires them.

## OpenRGB Integration

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `openRGBPort` | integer, `6742` | `1`-`65535` | Port used by the selected OpenRGB workflow. The importer connects to a local OpenRGB SDK Server at `127.0.0.1` on this port; the optional target server listens on `127.0.0.1` on this port. An invalid port prevents the applicable connection or listener. | Yes |
| `enableOpenRGBTargetServer` | boolean, `false`; optional and omitted from generated defaults | `true`, `false` | Enables LumenForge's inherited OpenRGB-compatible target server, which exposes supported LumenForge-managed native devices to an OpenRGB client. This is the reverse of the importer workflow. Leave it `false` unless deliberately using the target server. | Yes |

The target-server workflow remains functional inherited OpenLinkHub behavior.
Here, inherited and secondary describe its origin and relationship to the newer
importer, not deprecation or reduced functionality. See
[OpenRGB target server](../openrgb/README.md) for setup and supported devices.

For importer mode, configure OpenRGB's own SDK Server Host as `127.0.0.1` and
match its port to `openRGBPort`. LumenForge does not change OpenRGB's
configuration. The target server and a local OpenRGB SDK Server cannot listen
on the same `openRGBPort`, and the importer manager does not run while the
local target server is enabled. Importer and target-server workflows may be
used only when their port and device-ownership arrangements do not conflict.

Imported-controller identities, layouts, disabled membership, RGB profiles,
cluster membership, and other files below the mutable database root are runtime
state. They are managed by the application and are not `config.json` options.
See [OpenRGB device import](openrgb-import.md) for that lifecycle.

## Generated and Internal Values

Application, configuration, and mutable-data roots are internal runtime values.
They are centrally resolved, are not `config.json` fields, and cannot be
changed through the dashboard or API. The optional
[External Source Registry](external-sources.md) is separate from `config.json`.
It uses `$XDG_CONFIG_HOME/lumenforge/external-sources.json` (falling back to
`~/.config/lumenforge/external-sources.json`) for a user service and
`/etc/lumenforge/external-sources.json` for a system service.

LumenForge may add newly introduced keys when it loads an older configuration.
That upgrade rewrite preserves known and unknown JSON fields. Two compatibility
defaults differ from a newly generated file: a missing `graphProfiles` or
`ramTempViaHwmon` key is added as `false`, while new files use `true`.

## Environment Variables, Flags, and Precedence

There are no command-line configuration flags and no environment-variable
overrides for the JSON keys above. For those settings, the value loaded from
`config.json` is the only configured value.

The installed service units set runtime path environment values. They are not
JSON settings and are validated before use:

| Variable | Effect |
| --- | --- |
| `LUMENFORGE_SERVICE_MODE` | `system` selects installed system-service behavior; `user` or `desktop` selects installed user-service behavior. Empty selects direct development mode. Other values are rejected. |
| `LUMENFORGE_APPLICATION_ROOT` | Absolute immutable application root. Installed units set `/opt/LumenForge`. |
| `LUMENFORGE_CONFIG_ROOT` | Absolute directory containing `config.json`. |
| `LUMENFORGE_DATA_ROOT` | Absolute mutable data root. |

Installed configuration and data roots are rejected if relative or if they are
equal to or nested beneath the application root. The user installer resolves
custom absolute XDG homes and records the resulting roots in its unit. Direct
development mode uses the working directory only when no installed service mode
is selected. See [Filesystem Layout and Ownership](filesystem-layout.md).

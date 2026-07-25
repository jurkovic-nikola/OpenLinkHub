# Configuration Reference

LumenForge reads `config.json` once during startup. Installer-managed system and
user services both use `/opt/LumenForge/config.json`; a direct source run uses
`config.json` in the process working directory. Stop LumenForge before editing
the file, keep a backup, and restart it after saving.

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
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "",
  "manual": false,
  "frontend": true,
  "metrics": false,
  "memory": false,
  "memorySmBus": "i2c-0",
  "memoryType": 5,
  "exclude": [],
  "memorySku": "",
  "resumeDelay": 15000,
  "logFile": "",
  "logLevel": "info",
  "enhancementKits": "",
  "temperatureOffset": 0,
  "amdGpuIndex": 0,
  "amdsmiPath": "",
  "checkDevicePermission": false,
  "graphProfiles": true,
  "cpuTempFile": "",
  "ramTempViaHwmon": true,
  "nvidiaGpuIndex": [
    0
  ],
  "defaultNvidiaGPU": 0,
  "openRGBPort": 6742,
  "enableGamepad": true,
  "enableMotherboard": false,
  "motherboardBiosOnExit": false,
  "memoryRegisterOverride": "",
  "enableSystemTray": false
}
```

`enableOpenRGBTargetServer` is intentionally absent because it is deprecated
and its default is `false`. The internal `ConfigPath` field is also omitted.

## Server and Network

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `listenPort` | integer, `27003` | `1`-`65535` starts the HTTP server; `0` or a negative value disables it | Port for the web UI and HTTP API. Users normally change it only to resolve a port conflict. An out-of-range positive port makes server startup fail. | Yes |
| `listenAddress` | string, `"127.0.0.1"` | An address accepted by Go's TCP listener; an empty string binds all interfaces | Address for the web UI and HTTP API. Keep the loopback default unless the host is otherwise secured: the API has no built-in authentication and includes state-changing endpoints. | Yes |
| `frontend` | boolean, `true` | `true`, `false` | Registers the HTML interface routes. Setting it to `false` does not disable the HTTP API or static-file route. Most users should leave it enabled. | Yes |
| `metrics` | boolean, `false` | `true`, `false` | Registers the `/api/metrics` endpoint. Enable only when a local monitoring client needs it. It uses the same listener and security boundary as the rest of the API. | Yes |

The server settings are independent of OpenRGB's SDK server. `listenPort` is
LumenForge's HTTP port; `openRGBPort` is the local OpenRGB SDK port.

## Logging and Diagnostics

| Key | Type and default | Accepted values | Purpose and guidance | Restart |
| --- | --- | --- | --- | --- |
| `logLevel` | string, `"info"` | `"info"`, `"warn"`, `"error"`, `"fatal"`, or `"silent"`; matching is case-insensitive | Minimum emitted log severity. Any unrecognized value silently behaves as `"info"`. Use `"silent"` only when loss of diagnostic output is acceptable. | Yes |
| `logFile` | string, `""` | Empty, `"-"`, or a filesystem path | Empty writes to `stdout.log` beside `config.json`; `"-"` writes to standard error; another value selects that file. At startup an existing log is archived as a timestamped `.tar.gz`. The service user must be able to create the file and archive in its parent directory. | Yes |
| `debug` | boolean, `false` | `true`, `false` | Enables additional device and legacy OpenRGB target-server diagnostics. It does not change `logLevel`; normally leave it disabled because output can be verbose. | Yes |

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
| `enableSystemTray` | boolean, `false` | `true`, `false` | Starts the StatusNotifierItem/D-Bus tray integration. Enable only in a compatible desktop session; it is normally appropriate for the experimental user-service mode, not a headless system service. | Yes |

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
| `openRGBPort` | integer, `6742` | `1`-`65535` | Port of the OpenRGB SDK server on `127.0.0.1`. Imported controllers require an OpenRGB server at this port. An invalid port prevents OpenRGB client connection. | Yes |
| `enableOpenRGBTargetServer` | boolean, `false`; deprecated and omitted from new files | `true`, `false` | Enables LumenForge's legacy OpenRGB-compatible target listener on `listenAddress:openRGBPort`. This conflicts with importing devices from OpenRGB on the same port and is unsupported for normal importer use. Leave it absent or `false`; retain it only for deliberate legacy compatibility testing. | Yes |

LumenForge always connects to the importer SDK endpoint on loopback; changing
`listenAddress` does not change that client address. `enableOpenRGBTargetServer`
uses the same `openRGBPort`, so it and OpenRGB importer operation are mutually
exclusive.

Imported-controller identities, layouts, disabled membership, RGB profiles,
cluster membership, and other files below `database/` are runtime state. They
are managed by the application and are not `config.json` options. See
[OpenRGB device import](openrgb-import.md) for that lifecycle.

## Generated and Internal Values

`ConfigPath` is an internal field populated from the process working directory
after `config.json` is loaded. It may appear with that exact capitalized name
after LumenForge rewrites the configuration, but its file value is ignored and
replaced at every startup. Do not add or edit it.

If an `atomic` marker exists in the working directory, LumenForge internally
uses `/etc/LumenForge/config.json` and `/etc/LumenForge` for configuration data.
The current installers do not create that marker; it is an internal legacy
deployment mode rather than an ordinary user setting.

LumenForge may add newly introduced keys when it loads an older configuration.
That upgrade rewrite preserves known and unknown JSON fields. Two compatibility
defaults differ from a newly generated file: a missing `graphProfiles` or
`ramTempViaHwmon` key is added as `false`, while new files use `true`.

## Environment Variables, Flags, and Precedence

There are no command-line configuration flags and no environment-variable
overrides for the JSON keys above. For those settings, the value loaded from
`config.json` is the only configured value.

`LUMENFORGE_SERVICE_MODE` is a separate runtime-context override, not a JSON
setting:

| Value | Effect |
| --- | --- |
| `system` | Forces system-service behaviour. |
| `user` or `desktop` | Forces desktop/user-service behaviour. |
| Empty or unrecognized | LumenForge infers the mode from UID, desktop-session environment, and home directory. |

Values are trimmed and compared case-insensitively. The installed service units
set this variable explicitly, so its recognized environment value takes
precedence over automatic service-mode detection. It does not override any
`config.json` field.

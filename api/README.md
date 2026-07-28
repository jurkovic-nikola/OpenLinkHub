# API Documentation and Examples

LumenForge's dashboard and HTTP API are local-only and listen on
`127.0.0.1:<listenPort>`. Requests must identify the service through a Host of
`127.0.0.1` or `localhost`, optionally followed by the configured
`listenPort`. Wildcard, LAN, Tailscale, arbitrary-domain, reverse-proxy, and
remote API access are unsupported.

## Local Request Protection

GET routes are read-only. Browser requests that change state use POST, PUT,
PATCH, or DELETE and must come from the same origin as the dashboard. They also
require the process-local request proof in `X-LumenForge-Request-Proof`.
LumenForge does not enable CORS, and remote web clients and cross-origin
preflights are unsupported.

The dashboard obtains and submits this proof automatically. A local
command-line client must fetch it after each LumenForge restart and include it
on every mutation. The helpers below are used by all mutation examples in this
document:

```bash
LUMENFORGE_URL='http://127.0.0.1:27003'
LUMENFORGE_TOKEN="$(curl --silent "$LUMENFORGE_URL/api/security/token" | jq -r '.token')"

lfcurl() {
  curl --header "X-LumenForge-Request-Proof: $LUMENFORGE_TOKEN" \
    --header 'Content-Type: application/json' "$@"
}

lfcurl_empty() {
  curl --header "X-LumenForge-Request-Proof: $LUMENFORGE_TOKEN" "$@"
}
```

Use `lfcurl` for JSON mutations and `lfcurl_empty` for documented mutations
that have no body. Multipart clients must include the same proof header but
must let curl generate the multipart Content-Type and boundary, for example
with `--form`. The temperature-profile form endpoint retains
`application/x-www-form-urlencoded`. A missing, stale, or invalid proof is
rejected with HTTP `403`; fetch a new token or reload the dashboard. An
unsupported mutation Content-Type is rejected with HTTP `415`.

## OpenRGB Importer API

These endpoints manage controllers imported from a separately running OpenRGB
SDK server. LumenForge connects to that server at
`127.0.0.1:<openRGBPort>`; see the
[configuration reference](../docs/configuration.md#openrgb-integration).

The lifecycle is:

1. **Discover** obtains a status-neutral preview and selection keys that import
   revalidates against a fresh scan.
2. **Import** re-discovers and enrolls only explicitly selected controllers.
3. An imported and currently registered controller is **active** and can use
   the live color, brightness, effect, speed, and layout endpoints.
4. **Remove** disables importer membership. It is not the same as a temporary
   OpenRGB disconnection.
5. **Refresh** asks the manager to asynchronously discover current OpenRGB SDK
   state and reconcile controllers already represented by configured imports.
   It does not enroll unknown controllers, create imports, or change saved
   layouts.

For installation and UI guidance, see
[OpenRGB device import](../docs/openrgb-import.md).

### Response and Request Conventions

Successful and application-level error responses from the endpoints in this
section use HTTP `200` and this JSON envelope:

```json
{
  "code": 200,
  "status": 1,
  "message": "Operation-specific message",
  "data": {}
}
```

`status` is `1` for success and `0` for an application error. `message` and
`data` are omitted when the handler has no value for them. Clients must inspect
`status`; HTTP `200` by itself does not indicate application success. Using the
wrong HTTP method returns HTTP `405` with a plain-text method-not-allowed
message instead of this JSON envelope.

The import and remove endpoints accept at most 64 KiB, reject unknown JSON
fields and trailing JSON values, and limit each batch to 64 entries. Discovery
and refresh need no request body; they reject a declared body larger than
64 KiB but otherwise ignore it. The live-control and layout endpoints use the
older permissive JSON decoder: it decodes the first JSON value, ignores unknown
fields, and does not check for or reject trailing JSON or additional values
after the first decoded object. Omitted fields receive their Go zero values.

Common handler-level error bodies are:

| Condition | HTTP and JSON response |
| --- | --- |
| Malformed body, wrong JSON type, strict-body unknown field, or strict-body trailing value | HTTP `200`; `{"code":200,"status":0,"message":"Invalid request body"}` |
| Declared discovery/refresh body over 64 KiB | HTTP `200`; `{"code":200,"status":0,"message":"Request body is too large"}` |
| Empty import batch | HTTP `200`; `{"code":200,"status":0,"message":"At least one OpenRGB selection key is required"}` |
| More than 64 import keys | HTTP `200`; `{"code":200,"status":0,"message":"Too many OpenRGB selections; maximum batch size is 64"}` |
| Empty remove batch | HTTP `200`; `{"code":200,"status":0,"message":"At least one OpenRGB import serial is required"}` |
| More than 64 remove serials | HTTP `200`; `{"code":200,"status":0,"message":"Too many OpenRGB imports; maximum batch size is 64"}` |

### OpenRGB Connection Status

`GET /api/openrgb/status` reports LumenForge's current global OpenRGB client
state. It takes no query parameters or request body and performs no connection,
discovery, device, or persistence operation.

```bash
curl --silent http://127.0.0.1:27003/api/openrgb/status
```

Success:

```json
{
  "code": 200,
  "status": 1,
  "data": {
    "state": "Connected"
  }
}
```

`data.state` is exactly `"Connected"`, `"Offline"`, or `"Not Configured"`.
When the OpenRGB state has an associated error, `data.error` contains its text.
The endpoint still returns `status: 1`; the state and optional error are the
result being queried.

`"Not Configured"` means no enabled importer is being managed. `"Offline"`
means configured imports currently cannot use the SDK server. Temporary
offline status does not remove imports or delete saved data.

### Discover Controllers

`POST /api/openrgbimport/discover` performs a deliberate, status-neutral scan
of the OpenRGB SDK server and compares the result with the saved importer
store. It acts on discovered controllers, not active devices, and does not
import, enable, remove, or persist anything.

```bash
lfcurl_empty --silent -X POST \
  http://127.0.0.1:27003/api/openrgbimport/discover
```

Representative success:

```json
{
  "code": 200,
  "status": 1,
  "message": "OpenRGB controller discovery completed",
  "data": {
    "discoveryState": "available",
    "configured": [
      {
        "serial": "openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "product": "Example Controller"
      }
    ],
    "controllers": [
      {
        "key": "orgb-v1-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "identityKind": "external-serial",
        "product": "Example Controller",
        "vendor": "Example Vendor",
        "displaySerial": "ABC123",
        "displaySerialLabel": "OpenRGB serial",
        "zoneCount": 1,
        "ledCount": 12,
        "zones": [
          {
            "name": "Main",
            "ledCount": 12
          }
        ],
        "state": "selectable"
      }
    ]
  }
}
```

`data` uses these fields:

| Field | Meaning |
| --- | --- |
| `discoveryState` | `"available"` when the scan completed, `"offline"` when OpenRGB could not be reached, or `"conflict"` when LumenForge's inherited local OpenRGB target server prevents importer use. |
| `error` | Optional discovery error text. |
| `configured` | Saved importer entries. Each has `serial`, `product`, and optional `disabled: true`. This list can include controllers not present in the current scan. |
| `controllers` | Current OpenRGB controller previews. |

Each controller preview always contains `product`, `displaySerial`,
`displaySerialLabel`, `zoneCount`, `ledCount`, `zones`, and `state`. It may
also contain `key`, `identityKind`, `vendor`, `version`, `description`,
`configuredSerial`, `reasonCode`, or `reason`. Zone objects contain `name` and
`ledCount`, plus optional `classification`.

Controller `state` is:

| State | Meaning |
| --- | --- |
| `selectable` | A stable, unique identity was derived and `key` can be passed to import. This includes a previously removed/disabled import that can be reimported. |
| `imported` | The controller already matches an enabled configured import; `configuredSerial` identifies it. |
| `ambiguous` | Available metadata cannot uniquely identify the controller. No usable selection key is returned. |
| `invalid` | Metadata, layout, or saved identity conflicts make import unsafe. `reasonCode` and `reason` describe the rejection. |

`identityKind`, when present, is `"external-serial"`,
`"location-product-vendor"`, or `"product-vendor-name"`. A controller that
reports zero LEDs may receive a conservative fallback layout in the preview;
that is initial import configuration, not a change to OpenRGB.

On failure the handler returns HTTP `200`, `status: 0`, the underlying error in
`message`, and the same discovery-shaped `data` when available. This lets
clients display saved imports even while OpenRGB is offline.

### Import Controllers

`POST /api/openrgbimport/import` imports explicitly selected discovery
candidates. It requires:

| Field | Type | Rules |
| --- | --- | --- |
| `keys` | array of strings | Required; 1-64 entries. Each value must be the `orgb-v1-` key returned by discovery (the prefix followed by 64 hexadecimal characters). Whitespace is trimmed and duplicates are coalesced. |

The controller keys, import serials, and zone values in the import, remove,
control, and layout curl examples below are placeholders. Replace them with a
`key` returned by discovery, a `serial` returned by import or
`configuredSerial` returned by discovery, and valid zone names and LED counts
from your own OpenRGB environment.

```bash
lfcurl --silent -X POST \
  -d '{"keys":["orgb-v1-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"]}' \
  http://127.0.0.1:27003/api/openrgbimport/import
```

Success:

```json
{
  "code": 200,
  "status": 1,
  "message": "OpenRGB controllers imported",
  "data": {
    "configuredSerials": [
      "openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    ],
    "configured": [
      {
        "serial": "openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "product": "Example Controller"
      }
    ],
    "controllers": [
      {
        "serial": "openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "product": "Example Controller"
      }
    ]
  }
}
```

The endpoint performs a fresh status-neutral discovery and validates each key
against that result. A stale, missing, ambiguous, or invalid candidate is
rejected rather than inferred from the earlier preview. Import is idempotent
when the selected controller is already imported with consistent live
membership.

A successful import writes or enables its entry in
`openrgbimport-zones.json` beneath the mutable database root, registers the
device, joins it to importer management, and creates default device-profile/RGB
files only when they do not already exist. Reimporting a disabled controller
reuses its stable LumenForge identity, saved layout, profiles, RGB artifacts,
and preserved ordering state.
The lifecycle is transactional and attempts to roll back partial changes when
activation fails.

Errors returned by the import lifecycle use HTTP `200`, `status: 0`, an exact
implementation message, and an import-shaped `data` object whose result arrays
are empty. Those arrays are populated only after a successful import, including
a successful idempotent request for controllers that are already imported.
Handler-level malformed/oversized input and batch-limit errors use the common
responses above and omit `data`. Other error categories include an invalid or
stale key, unavailable OpenRGB, local target-server conflict, and activation or
persistence failure.

### Remove Imports

`POST /api/openrgbimport/remove` disables enabled importer memberships. It
requires:

| Field | Type | Rules |
| --- | --- | --- |
| `serials` | array of strings | Required; 1-64 configured LumenForge import serials. Values are trimmed and deduplicated; each may contain only letters, numbers, and `-`. |

Use the `serial` returned by import or `configuredSerial` returned by
discovery:

```bash
lfcurl --silent -X POST \
  -d '{"serials":["openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"]}' \
  http://127.0.0.1:27003/api/openrgbimport/remove
```

Success:

```json
{
  "code": 200,
  "status": 1,
  "message": "OpenRGB imports removed",
  "data": {
    "removedSerials": [
      "openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    ]
  }
}
```

Removal detaches the active wrapper from the registry, cluster, and importer
manager and saves the importer entry as disabled. It preserves the stable
identity, zone layout, profiles, RGB files, dashboard/cluster ordering, and
other reimport artifacts. Reimport is therefore different from importing a new
controller.

An OpenRGB outage is not removal. During a temporary disconnection an enabled
import remains registered and configured, is marked unavailable, and is
reconciled by the manager when possible.

Errors use HTTP `200`, `status: 0`, `message`, and a remove-shaped `data`
object. Missing imports, already-disabled imports, inconsistent manager or
registry membership, malformed input, more than 64 serials, and transactional
detach/persistence failures are rejected without silently deleting artifacts.

### Refresh Configured Imports

`POST /api/openrgbimport/refresh` requests one coalesced importer-manager
reconciliation for already enabled imports. It takes no fields:

```bash
lfcurl_empty --silent -X POST \
  http://127.0.0.1:27003/api/openrgbimport/refresh
```

Success:

```json
{
  "code": 200,
  "status": 1,
  "message": "OpenRGB import reconciliation requested",
  "data": {
    "queued": true
  }
}
```

The request wakes or installs the manager and returns after the reconciliation
request is accepted and queued; it does not wait for reconciliation to finish.
The ensuing asynchronous reconciliation discovers current OpenRGB SDK
controllers and matches them only to configured imports. It may reconnect or
rebind known controllers, change their availability, update active
presentation/runtime metadata, persist backfilled identity metadata in
`openrgbimport-zones.json` in the mutable database root, and replay desired lighting state after a
reconnection.

Reconciliation does not enroll unknown controllers, create new imports, change
importer membership, or modify saved layouts. Errors include
`"OpenRGB imports are not configured"`, a local target-server conflict, or
request-context/manager failure and use HTTP `200`, `status: 0`.

### Active-Device Control Conventions

The remaining endpoints act only on an imported controller currently present
in LumenForge's active device registry. Merely appearing in discovery is not
enough. Every body has an optional string `serial`; when omitted or empty it
uses the legacy default `openrgb-mobo-1`. New clients should always send the
actual serial returned by import.

If no registry entry matches, the response is HTTP `200` with `status: 0` and
`message: "Device not found"`. A matching non-import device returns
`"Invalid device instance"`. A temporarily unavailable import can remain in
the registry, but operations that require OpenRGB output can return lifecycle,
controller, or transport error text. Devices controlled by RGB Cluster reject
direct color and effect operations with
`"device is controlled by RGB cluster"`.

Color, brightness, and effect operations update live or in-memory state and,
when an active device profile exists, LumenForge attempts to save that profile.
The profile-save helper suppresses directory-creation, serialization, and file
write errors, so a successful endpoint response does not guarantee that the
profile change reached durable storage.

#### Set Color

`POST /api/openrgbimport/color` stops the current animation, selects the
`static` effect, applies one RGB color to all configured zones, and sends it to
the active controller.

| Field | Type | Rules |
| --- | --- | --- |
| `serial` | string | Optional legacy default; actual import serial recommended. |
| `r`, `g`, `b` | integer | The decoder accepts any Go `int`; use `0`-`255`. Omitted values become `0`. Values outside the byte range are converted modulo 256 rather than rejected. |

```bash
lfcurl --silent -X POST \
  -d '{"serial":"openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","r":255,"g":96,"b":0}' \
  http://127.0.0.1:27003/api/openrgbimport/color
```

Success is:

```json
{"code":200,"status":1,"message":"Color set"}
```

The static effect and zone colors are updated in the active device profile when
one exists, and LumenForge attempts the profile save described above. This is a
live output operation even if durable profile persistence fails. It does not
change importer identity or zone layout. Output, inactive-lifecycle,
missing-controller, and RGB Cluster errors return `status: 0`.

#### Set Brightness

`POST /api/openrgbimport/brightness` changes the active import's brightness:

| Field | Type | Rules |
| --- | --- | --- |
| `serial` | string | Optional legacy default; actual import serial recommended. |
| `brightness` | unsigned integer | JSON decoder accepts `0`-`255`; the device clamps `101`-`255` to `100`. Negative, fractional, or greater-than-255 values make the body invalid. Omitted is `0`. |

```bash
lfcurl --silent -X POST \
  -d '{"serial":"openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","brightness":75}' \
  http://127.0.0.1:27003/api/openrgbimport/brightness
```

Success is:

```json
{"code":200,"status":1,"message":"Brightness set"}
```

Brightness is updated in the active device profile when one exists, and
LumenForge attempts the profile save described above. A running effect reads
the new value on subsequent frames; otherwise LumenForge sends an updated frame
immediately. This does not change importer configuration.

#### Set Effect

`POST /api/openrgbimport/effect` stops the previous effect and starts or applies
the requested effect:

| Field | Type | Rules |
| --- | --- | --- |
| `serial` | string | Optional legacy default; actual import serial recommended. |
| `effect` | string | The handler does not validate this field. Implemented names are listed below; clients should use them. An omitted string is empty. |

Implemented effect names are `circle`, `circleshift`, `colorpulse`,
`colorshift`, `colorwarp`, `cpu-temperature`, `flickering`, `flame`, `aurora`,
`cyberpunkglitch`, `gpu-temperature`, `gradient`, `off`, `rainbow`,
`pastelrainbow`, `pastelspiralrainbow`, `rotator`, `spinner`,
`spiralrainbow`, `static`, `storm`, `watercolor`, and `wave`.

```bash
lfcurl --silent -X POST \
  -d '{"serial":"openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","effect":"rainbow"}' \
  http://127.0.0.1:27003/api/openrgbimport/effect
```

Success is:

```json
{"code":200,"status":1,"message":"Effect set"}
```

The effect name is updated in the active device profile before output starts,
and LumenForge attempts the profile save described above. `off` sends black,
`static` sends the current in-memory color, and other implemented names run their
animation. Because validation is absent, an unknown name remains in active
state and may reach persistent storage even though it has no defined public
effect behaviour; do not rely on that fallback. This endpoint does not change
layout or importer membership.

#### Set Effect Speed

`POST /api/openrgbimport/speed` changes the in-memory timing used by the active
importer's fallback effect runner:

| Field | Type | Rules |
| --- | --- | --- |
| `serial` | string | Optional legacy default; actual import serial recommended. |
| `speed` | string | `"slow"` selects a 4.0 timing value, `"fast"` selects 0.8, and every other string (including `"normal"` or omitted) selects 2.0. |

```bash
lfcurl --silent -X POST \
  -d '{"serial":"openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","speed":"fast"}' \
  http://127.0.0.1:27003/api/openrgbimport/speed
```

Success is:

```json
{"code":200,"status":1,"message":"Speed set"}
```

This endpoint sends no immediate OpenRGB frame and does not persist the speed.
It also does not report an inactive lifecycle from `SetSpeed`; after a valid
registry lookup the handler returns success even if the device became detached.
Saved effect-profile speed or an enabled RGB override supersedes this in-memory
fallback value. The effect loop reloads saved profile speed on each frame, so a
successful request may have little or no visible effect for profile-backed or
shipped modes. Effects without a saved profile can still use the fallback
value. Use the effect/profile APIs for persistent profile speed behaviour.

#### Save Zone Layout

`POST /api/openrgbimport/config` validates and persists the imported
controller's zone layout:

| Field | Type | Rules |
| --- | --- | --- |
| `serial` | string | Optional legacy default; actual import serial recommended. It must match the active device. |
| `zones` | array of objects | Required; 1-128 zones and 1-4096 LEDs total. |
| `zones[].name` | string | Sanitized for persistence; an empty value becomes `"Zone N"`. |
| `zones[].ledCount` | integer | Required operationally; `1`-`1024` per zone. |

```bash
lfcurl --silent -X POST \
  -d '{"serial":"openrgb-hash-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","zones":[{"name":"Main","ledCount":12},{"name":"Accent","ledCount":8}]}' \
  http://127.0.0.1:27003/api/openrgbimport/config
```

Success is:

```json
{"code":200,"status":1,"message":"Config saved"}
```

Unlike the color, brightness, effect, and speed endpoints, this changes
persistent importer configuration in `openrgbimport-zones.json` beneath the
mutable database root. It preserves the saved product, external serial,
location, vendor, and disabled membership fields, updates the active
device/profile/cluster representation, and sends a frame using the proposed
layout when connected.

If any LED count increases beyond the previously saved working value,
LumenForge performs four OpenRGB health checks after the test frame. The first
check occurs immediately, with 500 ms between subsequent attempts. If frame
delivery or health validation fails, the previous in-memory layout is restored
and the new config is not saved.

A persistent store-write failure happens later: the proposed layout is already
installed in memory and may already have been sent to OpenRGB. That failure
returns HTTP `200`, `status: 0`, but does not restore the previous in-memory
layout. Invalid zone counts and totals above 4096 are rejected before applying
the layout; inactive-lifecycle and other implementation errors also return
HTTP `200`, `status: 0` with their error text in `message`.

The identifiers, labels, profile names, and runtime values in the inherited
examples below are illustrative placeholders. Replace them with values reported
by your own LumenForge installation.

### Get all data
```bash
$ curl -X GET http://127.0.0.1:27003/api/ --silent | jq
{
  "code": 200,
  "status": 0,
  "device": {
    "EXAMPLE-DEVICE-001": {
      "ProductType": 0,
      "Product": "iCUE LINK System Hub",
      "Serial": "EXAMPLE-DEVICE-001",
      "Firmware": "0.0.0",
      "Image": "icon-device.svg"
    }
  }
}
```
### Get CPU temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/cpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "45.0 °C"
}
```
```bash
$ curl -X GET http://127.0.0.1:27003/api/cpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 45
}
```
### Get GPU temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/gpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "40.0 °C"
}
```
```bash
$ curl -X GET http://127.0.0.1:27003/api/gpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 40
}
```
### Get storage temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/storageTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": [
    {
      "Key": "example-sensor-1",
      "Model": "EXAMPLE-STORAGE-001",
      "Temperature": 40,
      "TemperatureString": "40.0 °C"
    },
    {
      "Key": "example-sensor-2",
      "Model": "EXAMPLE-STORAGE-002",
      "Temperature": 35,
      "TemperatureString": "35.0 °C"
    },
    {
      "Key": "example-sensor-3",
      "Model": "EXAMPLE-STORAGE-003",
      "Temperature": 35,
      "TemperatureString": "35.0 °C"
    }
  ]
}
```
### Get battery status
```bash
$ curl -X GET http://127.0.0.1:27003/api/batteryStats --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "EXAMPLE-DEVICE-005": {
      "Device": "VIRTUOSO MAX WIRELESS",
      "Level": 60,
      "DeviceType": 2
    }
  }
}
```
### Get all devices
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices/ --silent | jq
{
  "code": 200,
  "status": 0,
  "devices": {
    "EXAMPLE-DEVICE-001": {
      "ProductType": 0,
      "Product": "iCUE LINK System Hub",
      "Serial": "EXAMPLE-DEVICE-001",
      "Firmware": "",
      "Image": "",
      "GetDevice": {
        "Debug": false,
        "manufacturer": "Corsair",
        "product": "iCUE LINK System Hub",
        "serial": "EXAMPLE-DEVICE-001",
        "firmware": "0.0.0",
        "aio": false,
        "devices": {
          "1": {
            "channelId": 1,
            "type": 1,
            "deviceId": "EXAMPLE-COMPONENT-001",
            "name": "iCUE LINK QX RGB",
            "rpm": 600,
            "temperature": 25,
            "temperatureString": "25.0 °C",
            "description": "Fan",
            "profile": "example-profile",
            "rgb": "static",
            "label": "Example Fan",
            "portId": 0,
            "IsTemperatureProbe": true,
            "IsLinkAdapter": false,
            "IsCpuBlock": false,
            "HasSpeed": true,
            "HasTemps": true,
            "AIO": false,
            "Position": 0,
            "ExternalAdapter": 0,
            "LCDSerial": ""
          }
        }
      }
    }
  }
}
```
### Get specific device
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices/EXAMPLE-DEVICE-001 --silent | jq
{
  "code": 200,
  "status": 0,
  "device": {
    "Debug": false,
    "manufacturer": "Corsair",
    "product": "iCUE LINK System Hub",
    "serial": "EXAMPLE-DEVICE-001",
    "firmware": "0.0.0",
    "aio": false,
    "devices": {
      "1": {
        "channelId": 1,
        "type": 1,
        "deviceId": "EXAMPLE-COMPONENT-001",
        "name": "iCUE LINK QX RGB",
        "rpm": 600,
        "temperature": 25,
        "temperatureString": "25.0 °C",
        "description": "Fan",
        "profile": "example-profile",
        "rgb": "static",
        "label": "Example Fan",
        "portId": 0,
        "IsTemperatureProbe": true,
        "IsLinkAdapter": false,
        "IsCpuBlock": false,
        "HasSpeed": true,
        "HasTemps": true,
        "AIO": false,
        "Position": 0,
        "ExternalAdapter": 0,
        "LCDSerial": ""
      }
    }
  }
}
```
### Get devices RGB data
```bash
$ curl -X GET http://127.0.0.1:27003/api/color/ --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "EXAMPLE-DEVICE-003": {
      "device": "MM700 RGB",
      "defaultColor": {
        "red": 255,
        "green": 100,
        "blue": 0,
        "brightness": 1,
        "Hex": ""
      },
      "profiles": {}
    },
    "EXAMPLE-DEVICE-004": {
      "device": "ST100 RGB",
      "defaultColor": {
        "red": 255,
        "green": 100,
        "blue": 0,
        "brightness": 1,
        "Hex": ""
      },
      "profiles": {}
    }
  }
}
```
### Get device RGB data
```bash
$ curl -X GET http://127.0.0.1:27003/api/color/EXAMPLE-DEVICE-003 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "device": "MM700 RGB",
    "defaultColor": {
      "red": 255,
      "green": 100,
      "blue": 0,
      "brightness": 1,
      "Hex": ""
    },
    "profiles": {}
  }
}
```
### Get temperature profiles
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/ --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "example-profile": {
      "sensor": 2,
      "zeroRpm": false,
      "profiles": [
        {
          "id": 1,
          "min": 0,
          "max": 30,
          "mode": 0,
          "fans": 30,
          "pump": 50
        }
      ],
      "points": {
        "0": [
          {
            "x": 0,
            "y": 50
          }
        ],
        "1": [
          {
            "x": 0,
            "y": 25
          }
        ]
      },
      "device": "",
      "channelId": 0,
      "linear": false,
      "Hidden": false
    }
  }
}
```
### Get registered external temperature sources

This read-only endpoint returns only the opaque id stored by a sensor-type-7
profile and its dashboard label. Executable paths and fixed arguments are never
returned. See [External Temperature Sources](../docs/external-sources.md) for
the local registry format and trust rules.

```bash
$ curl -X GET http://127.0.0.1:27003/api/external-sources --silent | jq
{
  "code": 200,
  "status": 1,
  "data": [
    {
      "id": "gpu-temperature",
      "name": "GPU Temperature"
    }
  ]
}
```

### Get temperature profile
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/example-profile --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "sensor": 2,
    "zeroRpm": false,
    "profiles": [
      {
        "id": 1,
        "min": 0,
        "max": 30,
        "mode": 0,
        "fans": 30,
        "pump": 50
      }
    ],
    "points": {
      "0": [
        {
          "x": 0,
          "y": 50
        }
      ],
      "1": [
        {
          "x": 0,
          "y": 25
        }
      ]
    },
    "device": "",
    "channelId": 0,
    "linear": false,
    "Hidden": false
  }
}
```
### Get temperature graph profile
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/graph/example-profile --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "0": {
      "sensor": 2,
      "points": [
        {
          "x": 0,
          "y": 50
        }
      ]
    },
    "1": {
      "sensor": 2,
      "points": [
        {
          "x": 0,
          "y": 25
        }
      ]
    }
  }
}
```
### Get input media keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/media --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "1": {
      "Name": "Volume Up",
      "CommandCode": 115,
      "Media": true,
      "Mouse": false
    },
    "2": {
      "Name": "Volume Down",
      "CommandCode": 114,
      "Media": true,
      "Mouse": false
    }
  }
}
```
### Get input keyboard keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/keyboard --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "0": {
      "Name": "None",
      "CommandCode": 0,
      "Media": false,
      "Mouse": false
    },
    "10": {
      "Name": "Number 3",
      "CommandCode": 4,
      "Media": false,
      "Mouse": false
    },
    "11": {
      "Name": "Number 4",
      "CommandCode": 5,
      "Media": false,
      "Mouse": false
    }
  }
}
```
### Get input mouse keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/mouse --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "91": {
      "Name": "(Mouse) Left Click",
      "CommandCode": 272,
      "Media": false,
      "Mouse": true
    },
    "92": {
      "Name": "(Mouse) Right Click",
      "CommandCode": 273,
      "Media": false,
      "Mouse": true
    },
    "93": {
      "Name": "(Mouse) Middle Click",
      "CommandCode": 274,
      "Media": false,
      "Mouse": true
    },
    "94": {
      "Name": "(Mouse) Back",
      "CommandCode": 275,
      "Media": false,
      "Mouse": true
    },
    "95": {
      "Name": "(Mouse) Forward",
      "CommandCode": 276,
      "Media": false,
      "Mouse": true
    }
  }
}
```
### Get all LED data
```bash
$ curl -X GET http://127.0.0.1:27003/api/led/ --silent | jq
{
  "code": 200,
  "status": 1,
  "data": [
    {
      "serial": "EXAMPLE-DEVICE-004",
      "deviceName": "ST100 RGB",
      "devices": {
        "0": {
          "ledChannels": 1,
          "pump": false,
          "aio": false,
          "fan": false,
          "stand": true,
          "tower": false,
          "channels": {
            "0": {
              "red": 0,
              "green": 255,
              "blue": 255,
              "brightness": 0,
              "Hex": "#00ffff"
            }
          }
        }
      }
    }
  ]
}
```
### Get device LED data
```bash
$ curl -X GET http://127.0.0.1:27003/api/led/EXAMPLE-DEVICE-004 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "serial": "EXAMPLE-DEVICE-004",
    "deviceName": "ST100 RGB",
    "devices": {
      "0": {
        "ledChannels": 1,
        "pump": false,
        "aio": false,
        "fan": false,
        "stand": true,
        "tower": false,
        "channels": {
          "0": {
            "red": 0,
            "green": 255,
            "blue": 255,
            "brightness": 0,
            "Hex": "#00ffff"
          }
        }
      }
    }
  }
}
```
### Get macros
```bash
$ curl -X GET http://127.0.0.1:27003/api/macro/ --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "1": {
      "id": 1,
      "name": "example-macro",
      "actions": {
        "0": {
          "actionType": 3,
          "actionCommand": 20,
          "actionDelay": 0
        },
        "1": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        },
        "2": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        },
        "3": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        }
      }
    }
  }
}
```
### Get macro
```bash
$ curl -X GET http://127.0.0.1:27003/api/macro/1 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "id": 1,
    "name": "example-macro",
    "actions": {
      "0": {
        "actionType": 3,
        "actionCommand": 20,
        "actionDelay": 0
      },
      "1": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      },
      "2": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      },
      "3": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      }
    }
  }
}
```
### Get dashboard settings
```bash
$ curl -X GET http://127.0.0.1:27003/api/dashboard --silent | jq
{
  "code": 200,
  "status": 1,
  "dashboard": {
    "showCpu": true,
    "showDisk": true,
    "showGpu": true,
    "showDevices": true,
    "verticalUi": false,
    "celsius": true,
    "showLabels": true,
    "showBattery": true
  }
}
```
### Get media playback (requires LumenForge to run in a user context)
```bash
$ curl -X GET http://127.0.0.1:27003/api/media/playback --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "playing": true,
    "service": "org.mpris.MediaPlayer2.example",
    "playbackStatus": "Playing",
    "title": "Example Title",
    "artists": [
      "Example Artist"
    ],
    "album": "Example Album",
    "length-us": 120000000,
    "position-us": 30000000,
    "length": 120,
    "position": 30,
    "track-id": "/org/mpris/MediaPlayer2/track/example"
  }
}

```
### Control media playback (requires LumenForge to run in a user context)

Media control is state-changing and therefore uses POST. Supported actions are
`previous`, `stop`, `play`, `next`, `volumeDown`, `volumeUp`, and `mute`.

```bash
$ lfcurl_empty -X POST http://127.0.0.1:27003/api/media/play --silent | jq
```

The former GET form is no longer supported.

### Create temperature profile - CPU
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"example-cpu-profile", "sensor":0}' --silent | jq
```
### Create temperature profile - GPU
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"example-gpu-profile", "sensor":1}' --silent | jq
```
### Create temperature profile - Liquid
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"example-liquid-profile", "sensor":2}' --silent | jq
```
### Create temperature profile - Static
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"example-static-profile", "sensor":2, "static":true}' --silent | jq
```
### Create temperature profile - registered external source

The `externalSourceId` must match an entry returned by
`GET /api/external-sources`. Executable paths and arguments are not accepted.

```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"example-external-profile", "sensor":7, "externalSourceId":"gpu-temperature"}' --silent | jq
```

### Set device speed profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/speed -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId":1, "profile":"example-profile"}' --silent | jq
```
### Set device speed profile on all channels
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/speed -d '{ "deviceId":"EXAMPLE-DEVICE-002", "channelId":0, "profile":"example-profile" }' --silent | jq
```
### Set device speed
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/speed/manual -d '{ "deviceId":"EXAMPLE-DEVICE-001", "channelId":1, "value":50 }' --silent | jq
```
### Set device RGB profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/color -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId":1, "profile":"rainbow"}' --silent | jq
```
### Set device RGB profile on all channels
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/color -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId":-1, "profile":"rainbow"}' --silent | jq
```
### Set LED Strip type (Link System Hub)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/hub/strip -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 3, "stripId": 1}' --silent | jq
```
### Set external LED device type (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/hub/type -d '{"deviceId":"EXAMPLE-DEVICE-001", "portId": 1, "deviceType": 2}' --silent | jq
```
### Set external LED device amount (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/hub/amount -d '{"deviceId":"EXAMPLE-DEVICE-001", "portId": 1, "deviceAmount": 2}' --silent | jq
```
### Set device label
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/label -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "deviceType": 0, "label": "Example Fan"}' --silent | jq
```
### Setup multiple LCD devices (Link System Hub)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/device -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "deviceType": 0, "lcdSerial": "EXAMPLE-LCD-001"}' --silent | jq
```
### Change LCD animation image
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/image -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "image": "example-animation"}' --silent | jq
```
### Change LCD rotation - default
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "rotation": 0}' --silent | jq
```
### Change LCD rotation - 90 degrees
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "rotation": 1}' --silent | jq
```
### Change LCD rotation - 180 degrees
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "rotation": 2}' --silent | jq
```
### Change LCD rotation - 270 degrees
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"EXAMPLE-DEVICE-001", "channelId": 1, "rotation": 3}' --silent | jq
```
### Change LCD profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/lcd/profile -d '{"deviceId":"EXAMPLE-DEVICE-001", "profile": "100"}' --silent | jq
```
### Change brightness - Dropdown
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/brightness -d '{"deviceId":"EXAMPLE-DEVICE-001", "brightness": 1}' --silent | jq
```
### Change brightness - Slider
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/brightness/gradual -d '{"deviceId":"EXAMPLE-DEVICE-001", "brightness": 75}' --silent | jq
```
### Change device position (Link System Hub) - To Left
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/position/update -d '{"deviceId":"EXAMPLE-DEVICE-001", "position": 3, "deviceIdString": "EXAMPLE-COMPONENT-002", "direction": 0}' --silent | jq
```
### Change device position (Link System Hub) - To Right
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/position/update -d '{"deviceId":"EXAMPLE-DEVICE-001", "position": 3, "deviceIdString": "EXAMPLE-COMPONENT-002", "direction": 1}' --silent | jq
```
### Set dashboard settings
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/dashboard/update -d '{"showCpu": true, "showGpu": true, "showDisk": true, "showDevices": true, "showLabels": true, "celsius": true}' --silent | jq
```
### Set custom ARGB device (Commander Core, Commander Core XT)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/argb -d '{"deviceId":"EXAMPLE-DEVICE-002", "portId": 6, "deviceType": 2}' --silent | jq
```
### Set keyboard color - Single key
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyId": 15, "keyOption": 0, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set keyboard color - Single row
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyId": 15, "keyOption": 1, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set keyboard color - All keys
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyId": 15, "keyOption": 2, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - Current area (MM700, ST100)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "areaId": 1, "areaOption": 0, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - Current row (MM700, ST100)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "areaId": 1, "areaOption": 1, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - All rows (MM700, ST100)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"EXAMPLE-DEVICE-002", "areaId": 1, "areaOption": 2, "color": {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Change user profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/userProfile/change -d '{"deviceId":"EXAMPLE-DEVICE-002", "userProfileName": "example-profile"}' --silent | jq
```
### Change keyboard profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/profile/change -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardProfileName": "example-profile"}' --silent | jq
```
### Save current keyboard profile
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/profile/save -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardProfileName": "example-profile", "new": false}' --silent | jq
```
### Change keyboard layout
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/layout -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardLayout": "US"}' --silent | jq
```
### Change keyboard dial option
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/dial -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardControlDial": 0}' --silent | jq
```
### Change keyboard sleep mode
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 3}' --silent | jq
```
### Change keyboard polling rate
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/keyboard/pollingRate -d '{"deviceId":"EXAMPLE-DEVICE-002", "pollingRate": 3}' --silent | jq
```
### Change rgb scheduler
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/scheduler/rgb -d '{"rgbControl":true, "rgbOff": "time-value", "rgbOn": "time-value"}' --silent | jq
```
### Set PSU fan speed
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/psu/speed -d '{"deviceId":"EXAMPLE-DEVICE-002", "fanMode": 7}' --silent | jq
```
### Set mouse DPI values
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/dpi -d '{"deviceId":"EXAMPLE-DEVICE-002", "stages":{"0":100,"1":200,"2":300,"3":400,"4":500}}' --silent | jq
```
### Set mouse DPI colors
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/dpiColors -d '{"deviceId":"EXAMPLE-DEVICE-002", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}}}' --silent | jq
```
### Set mouse Zone colors
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/zoneColors -d '{"deviceId":"EXAMPLE-DEVICE-002", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}}}' --silent | jq
```
### Set mouse Sleep mode - 1 minute
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 1}' --silent | jq
```
### Set mouse Sleep mode - 5 minutes
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 5}' --silent | jq
```
### Set mouse Sleep mode - 10 minutes
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 10}' --silent | jq
```
### Set mouse Sleep mode - 15 minutes
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 15}' --silent | jq
```
### Set mouse Sleep mode - 30 minutes
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 30}' --silent | jq
```
### Set mouse Sleep mode - 1 hour
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 60}' --silent | jq
```
### Set mouse Polling Rate - 125 Hz / 8 msec (check your device polling rates)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"EXAMPLE-DEVICE-002", "pollingRate": 1}' --silent | jq
```
### Set mouse Polling Rate - 250 Hz / 4 msec (check your device polling rates)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"EXAMPLE-DEVICE-002", "pollingRate": 2}' --silent | jq
```
### Set mouse Polling Rate - 500 Hz / 2 msec (check your device polling rates)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"EXAMPLE-DEVICE-002", "pollingRate": 3}' --silent | jq
```
### Set mouse Polling Rate - 1000 Hz / 1 msec (check your device polling rates)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"EXAMPLE-DEVICE-002", "pollingRate": 4}' --silent | jq
```
### Set mouse Angle Snapping
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/angleSnapping -d '{"deviceId":"EXAMPLE-DEVICE-002", "angleSnapping": 1}' --silent | jq
```
### Set mouse Button Optimization
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/buttonOptimization -d '{"deviceId":"EXAMPLE-DEVICE-002", "buttonOptimization": 1}' --silent | jq
```
### Set mouse Key Assignment
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/updateKeyAssignment -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyIndex": 10, "enabled":true,"pressAndHold":false,"keyAssignmentType": 3,"keyAssignmentValue":55}' --silent | jq
```
### Set headset Zone colors
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/mouse/zoneColors -d '{"deviceId":"EXAMPLE-DEVICE-002", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}}}' --silent | jq
```
### Change headset sleep mode
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sleep -d '{"deviceId":"EXAMPLE-DEVICE-002", "sleepMode": 3}' --silent | jq
```
### Change headset mute indicator
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/muteIndicator -d '{"deviceId":"EXAMPLE-DEVICE-002", "muteIndicator": 1}' --silent | jq
```
### Create new macro value
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/macro/newValue -d '{"macroId":1, "macroType": 1, "macroValue": 13, "macroDelay":200}' --silent | jq
```
### Update temperature graph - Fans
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/temperatures/updateGraph -d '{"profile": "example-profile", "updateType": 1,"points": [{"x": 0,"y": 25}]}' --silent | jq
```
### Update temperature graph - Pump
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/temperatures/updateGraph -d '{"profile": "example-profile", "updateType": 0,"points": [{"x": 0,"y": 25}]}' --silent | jq
```

### Headset Active Noise Cancellation - Off (require Sidetone Off)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "EXAMPLE-DEVICE-002", "noiseCancellation": 0}' --silent | jq
```
### Headset Active Noise Cancellation - On (require Sidetone Off)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "EXAMPLE-DEVICE-002", "noiseCancellation": 1}' --silent | jq
```
### Headset Active Noise Cancellation - Transparency (require Sidetone Off)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "EXAMPLE-DEVICE-002", "noiseCancellation": 2}' --silent | jq
```
### Headset Sidetone - Off (require Active Noise Cancellation Off)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sidetone -d '{"deviceId": "EXAMPLE-DEVICE-002", "sideTone": 0}' --silent | jq
```
### Headset Sidetone - On (require Active Noise Cancellation Off)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sidetone -d '{"deviceId": "EXAMPLE-DEVICE-002", "sideTone": 1}' --silent | jq
```
### Headset Sidetone Value - 0 (require Sidetone On)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "EXAMPLE-DEVICE-002", "sideToneValue": 0}' --silent | jq
```
### Headset Sidetone Value - 50 (require Sidetone On)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "EXAMPLE-DEVICE-002", "sideToneValue": 50}' --silent | jq
```
### Headset Sidetone Value - 100 (require Sidetone On)
```bash
$ lfcurl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "EXAMPLE-DEVICE-002", "sideToneValue": 100}' --silent | jq
```
### Save new user profile
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/userProfile -d '{"deviceId":"EXAMPLE-DEVICE-002", "userProfileName": "example-profile"}' --silent | jq
```
### Save new keyboard profile
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/keyboard/profile/new -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardProfileName": "example-profile", "new": true}' --silent | jq
```
### Save new macro profile
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/macro/new -d '{"macroName":"example-macro"}' --silent | jq
```
### Save device RGB profile
```bash
$ lfcurl -X PUT http://127.0.0.1:27003/api/color/change -d '{"deviceId":"EXAMPLE-DEVICE-002", "profile":"static", "startColor":{"red":255, "green":255, "blue":255}, "endColor":{"red":255, "green":255, "blue":255}, "speed":4}' --silent | jq
```
### Delete keyboard profile
```bash
$ lfcurl -X DELETE http://127.0.0.1:27003/api/keyboard/profile/delete -d '{"deviceId":"EXAMPLE-DEVICE-002", "keyboardProfileName": "example-profile"}' --silent | jq
```
### Delete macro value
```bash
$ lfcurl -X DELETE http://127.0.0.1:27003/api/macro/value -d '{"macroId":1, "macroIndex": 3}' --silent | jq
```
### Delete temperature profile (If any of your devices are using given profile, they will be reset to Normal profile)
```bash
$ lfcurl -X DELETE http://127.0.0.1:27003/api/temperatures/delete -d '{"profile":"example-profile"}' --silent | jq
```
### Delete macro profile
```bash
$ lfcurl -X DELETE http://127.0.0.1:27003/api/macro/profile -d '{"macroId":1}' --silent | jq
```

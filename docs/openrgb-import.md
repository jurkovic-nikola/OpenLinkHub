# OpenRGB Device Import

LumenForge can use a running OpenRGB SDK Server as a bridge to devices supported by OpenRGB. In this primary LumenForge workflow, OpenRGB provides device access and LumenForge acts as an SDK client/importer:

```text
OpenRGB-supported device -> OpenRGB SDK Server -> LumenForge -> Dashboard / RGB Cluster
```

Imported devices can appear alongside LumenForge's native devices and participate in dashboard and RGB Cluster workflows where the available OpenRGB metadata and LumenForge's importer support it.

OpenRGB support alone does not guarantee complete LumenForge support. Device names, zones, LED counts, effects, and control behavior vary by vendor and by the metadata exposed through the OpenRGB SDK protocol.

## OpenRGB Prerequisites and Ownership

OpenRGB is a separate external dependency. It must already be running with its SDK Server enabled and reachable before LumenForge can discover or control OpenRGB-backed devices. LumenForge does not install, launch, stop, restart, configure, or otherwise manage OpenRGB itself.

LumenForge connects only to the local OpenRGB SDK Server at `127.0.0.1` on the port configured by `openRGBPort`. New LumenForge configurations default to port `6742`. Remote OpenRGB hosts are unsupported. In OpenRGB itself, configure the SDK Server Host as `127.0.0.1`; LumenForge documents this setting but does not change OpenRGB's configuration.

## Discover and Import Controllers

1. Start the OpenRGB application, or a local headless OpenRGB instance, with its SDK Server enabled.
2. Set OpenRGB's SDK Server Host to `127.0.0.1` and confirm its SDK Server port.
3. If OpenRGB is not using port `6742`, set `openRGBPort` in LumenForge's `config.json` to the same port.
4. Start LumenForge.
5. Open **Settings** in LumenForge.
6. Locate **OpenRGB SDK Integration** and select **Discover & Manage Controllers**.
7. Review the controllers reported by the local SDK Server and explicitly select the controllers you want to import.
8. Select **Import Selected**.
9. Verify the imported controllers and their layouts in the dashboard and, where supported, RGB Cluster.

Use **Discover Again** in the management dialog to refresh the available-controller list. **Refresh Imported Controllers** reconciles controllers that are already imported; it does not import newly discovered controllers.

Discovery, import, removal, reimport, and refresh of configured imports are live operations and normally do not require restarting LumenForge.

## Discovery and Import Are Separate

A discovered controller is available from the OpenRGB SDK Server, but it is not automatically imported into LumenForge. Import is an explicit user action.

Rediscovery compares the available controllers with saved imported-controller identities. An existing import is reconciled with its saved record instead of being duplicated. If multiple controllers cannot be distinguished safely, LumenForge leaves them unavailable for selection rather than assigning an unstable identity.

OpenRGB device support alone does not guarantee that a controller exposes enough usable metadata for complete LumenForge control. A controller may be listed while still being unavailable, ambiguous, only partially usable, or dependent on manual layout correction.

## Import Configuration and Runtime State

LumenForge stores imported-controller identity and saved layout information in
`openrgbimport-zones.json` beneath the mutable database root:

| Mode | Path |
| --- | --- |
| User service | `$XDG_DATA_HOME/lumenforge/database/openrgbimport-zones.json`, or `~/.local/share/lumenforge/database/openrgbimport-zones.json` |
| System service | `/var/lib/lumenforge/database/openrgbimport-zones.json` |
| Direct development | `database/openrgbimport-zones.json` beneath the working directory |

This is generated runtime state and is not shipped with personal device information. Discovery alone does not add an unimported controller to this file. The store is written when an import or later configuration lifecycle change saves an imported-controller record.

For each imported device, the saved configuration is the source of truth:

- A new import starts from usable discovered metadata or an eligible conservative fallback.
- Saved zone names and LED counts persist across restarts.
- Saved user-edited layouts take precedence over incomplete newly discovered metadata.
- Routine rediscovery does not overwrite an existing saved layout.
- Users can correct zone names and LED counts through the imported controller's LumenForge configuration interface.

Device profiles and per-device RGB state are stored separately from this import store and are reused when the same preserved controller is reimported.

## Stable Identity, Removal, and Reimport

Imported controllers receive a stable LumenForge identity based on the strongest usable identity information reported by OpenRGB. Refreshing or rediscovering the same controller should reconcile the existing import rather than create a duplicate.

Some OpenRGB `Location` values describe a transient mode or state rather than a physical location. Values such as `Direct`, `Dir`, and `Off` are not treated as reliable stable controller identity.

Removing an imported controller through Settings disables it and removes it from active LumenForge use. Its saved import record, device profiles, and RGB configuration artifacts remain available for later reimport. When the same controller is discovered and explicitly imported again, LumenForge reuses its existing stable identifier and preserves its saved:

- Stable LumenForge identity
- Zone layout
- Device profiles
- RGB state

## Incomplete and Zero-LED Metadata

Some OpenRGB controllers report incomplete zone information or initially report zero LEDs. When the metadata meets the importer's safety limits but is insufficient to form a usable layout, LumenForge may offer a conservative, editable fallback layout.

The fallback is intended to make an eligible controller selectable and configurable. It is not a guarantee of the controller's physical zone structure or LED counts, and not every incomplete controller qualifies for a fallback.

Verify the real zone structure and LED counts in OpenRGB, then correct the imported controller's layout in LumenForge as needed. Saved manual corrections take precedence over incomplete rediscovery data and are retained across restart, removal, rediscovery, and reimport.

## LED Count Safety

An incorrect LED count may cause OpenRGB to ignore lighting updates. The device may stop responding until the correct zone and LED counts are restored.

Confirm the real values in OpenRGB before saving a layout in LumenForge. LumenForge validates supported configuration limits, but it cannot determine the correct physical layout when OpenRGB reports incomplete or inaccurate metadata.

## Resetting Saved Import Records

Deleting `openrgbimport-zones.json` from the applicable mutable database root
resets all saved OpenRGB import identity and layout records contained in that
file. This can discard manual zone-name and LED-count corrections and is not a
routine troubleshooting step.

If a full import-record reset is necessary:

1. Stop LumenForge.
2. Back up `openrgbimport-zones.json` from the mutable database root.
3. Remove the original file.
4. Start LumenForge and explicitly discover and import the desired controllers again.

Deleting the import store does not itself remove the separate device-profile or per-device RGB files. Do not assume, however, that those artifacts can be associated with a controller again after its saved identity record has been deleted.

## Two OpenRGB Directions

LumenForge contains two distinct OpenRGB integrations:

1. **Import into LumenForge (primary):** LumenForge connects as an SDK client to a separate, locally running OpenRGB SDK Server and explicitly imports selected OpenRGB-backed controllers into the LumenForge UI.
2. **Expose LumenForge devices to OpenRGB (inherited/secondary):** inherited OpenLinkHub functionality can run an optional OpenRGB-compatible target listener on `127.0.0.1` so a local OpenRGB client can control supported native devices.

The inherited target listener is separate from the importer. It does not discover or import external OpenRGB-backed controllers and is not involved in the Settings import-management workflow.

The external SDK importer is the primary workflow documented here. Older target-listener screenshots and instructions are retained separately in [`openrgb/README.md`](../openrgb/README.md) for inherited compatibility and are explicitly labeled as such.

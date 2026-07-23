# Changelog

This changelog covers LumenForge development beginning with the fork from
OpenLinkHub. Earlier upstream history remains available through the Git history
and the OpenLinkHub repository.

## Unreleased

### Added

- OpenRGB controller discovery and import management in Settings.
- Live OpenRGB import, removal, refresh, and reimport without restarting LumenForge.
- Stable OpenRGB controller identities and duplicate prevention.
- Conservative, editable fallback layouts for controllers reporting zero LED metadata.
- Experimental user-service installer support for validation.
- Coordinated graceful shutdown.

### Changed

- Installer upgrades now preserve runtime-owned configuration and user data.
- System and user installers detect conflicting service modes.
- OpenRGB layouts and RGB/profile state persist across restart, removal, and reimport.
- The LED-count warning now describes ignored lighting updates instead of a presumed OpenRGB crash.

### Fixed

- Prevented previously imported controllers with incomplete metadata from becoming invalid or duplicated.
- Prevented mode-like OpenRGB locations such as `Direct`, `Dir`, and `Off` from destabilizing controller identity.
- Preserved disabled imports’ stable identity, saved layouts, profiles, and RGB artifacts for later reimport.
- Improved OpenRGB connection and import lifecycle resilience.

## 0.1.0-alpha

- Initial LumenForge alpha release.
- Added OpenRGB device import.
- Added RGB Cluster workflows and physical device ordering.
- Added built-in themes.
- Added optional system tray integration.

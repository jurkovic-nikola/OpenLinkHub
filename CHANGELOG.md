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
- Per-user systemd service installation for desktop use and system-tray support.
- Coordinated graceful shutdown.

### Changed

- Installed application files are now root-owned and immutable under
  `/opt/LumenForge`, while user and system configuration, profiles, uploads,
  generated state, and logs use separate XDG or `/var/lib/lumenforge` paths.
- Installer upgrades preserve external runtime-owned configuration and data.
- Installation and upgrade now use the same mode-specific installer without a separate upgrade wrapper.
- System and user installers detect conflicting service modes.
- Installer upgrades remove obsolete maintenance-file copies from the installed directory.
- Fresh configuration files now use a stable, grouped field order.
- OpenRGB layouts and RGB/profile state persist across restart, removal, and reimport.
- The LED-count warning now describes ignored lighting updates instead of a presumed OpenRGB crash.
- Development builds now report `0.2.0-alpha-dev`, and the shared build path supports an explicit build-time version override for future releases.

### Fixed

- Prevented previously imported controllers with incomplete metadata from becoming invalid or duplicated.
- Prevented mode-like OpenRGB locations such as `Direct`, `Dir`, and `Off` from destabilizing controller identity.
- Preserved disabled imports’ stable identity, saved layouts, profiles, and RGB artifacts for later reimport.
- Improved OpenRGB connection and import lifecycle resilience.
- Corrected fresh RGB animation defaults and calibrated Flame, Cyberpunk Glitch, Storm, and software Rain timing.
- Made cluster and individual animation speed controls consistently run from Slow to Fast, left to right.
- Hid irrelevant speed controls for static and temperature-based RGB profiles.
- Fixed the user-service tray action that opens the configured dashboard in the default browser.
- Restored AMD SMI GPU reporting for current `gpu_data` responses.
- Restored K65 Plus Wireless control-dial press actions and included configured labels in temperature-probe selectors.
- Corrected the K60 RGB PRO G-key lighting packet mapping and Link Adapter success feedback.
- Moved supported Corsair memory metadata out of hard-coded device logic while retaining safe built-in defaults.
- Updated the image-processing dependency to include current WebP and font parsing fixes.

## 0.1.0-alpha

- Initial LumenForge alpha release.
- Added OpenRGB device import.
- Added RGB Cluster workflows and physical device ordering.
- Added built-in themes.
- Added optional system tray integration.

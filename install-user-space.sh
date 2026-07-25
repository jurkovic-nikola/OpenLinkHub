#!/bin/bash

# Recommended for desktop users and required for LumenForge system-tray support.
# Use install.sh for headless or desktop-independent system-service operation.
# Run this script as the intended desktop user, not root.
# Reboot after first installation to acquire hardware-access group membership.

set -e

PRODUCT="LumenForge"
INSTALL_DIR="/opt/LumenForge"
PERMISSION_FILE="99-lumenforge.rules"
PERMISSION_TARGET="/etc/udev/rules.d/99-lumenforge.rules"
DEVICE_GROUP="lumenforge"

fail() {
  echo "Error: $*" >&2
  exit 1
}

SCRIPT_PATH="$(readlink -f -- "$0")" || fail "Unable to resolve the installer path."
SOURCE_DIR="$(dirname -- "$SCRIPT_PATH")"
CANONICAL_INSTALL_DIR="$(readlink -m -- "$INSTALL_DIR")" || fail "Unable to resolve $INSTALL_DIR."

if [ "$SOURCE_DIR" = "$CANONICAL_INSTALL_DIR" ]; then
  fail "Refusing to run from $INSTALL_DIR. Run this installer from a fresh source checkout or extracted release directory."
fi

if [ "$EUID" -eq 0 ]; then
  fail "install-user-space.sh must be run as the intended desktop user, not root, because it needs that user's systemd manager."
fi

command -v getent >/dev/null 2>&1 || fail "getent is required to resolve the current user's account."
command -v systemctl >/dev/null 2>&1 || fail "systemctl is required to install the user service."

passwd_entry="$(getent passwd "$(id -u)")"
[ -n "$passwd_entry" ] || fail "Unable to resolve the current user through getent passwd."
IFS=: read -r TARGET_USER _ _ _ _ USER_HOME _ <<< "$passwd_entry"
[ -n "$TARGET_USER" ] || fail "The current user's account has no username."
case "$USER_HOME" in
  /*) ;;
  *) fail "The home directory for $TARGET_USER is not an absolute path: $USER_HOME" ;;
esac

SYSTEMD_DIR="$USER_HOME/.config/systemd/user"
SYSTEMD_FILE="$SYSTEMD_DIR/$PRODUCT.service"

[ -f "$SOURCE_DIR/$PRODUCT" ] || fail "Binary not found at $SOURCE_DIR/$PRODUCT"
[ -f "$SOURCE_DIR/$PERMISSION_FILE" ] || fail "Udev rule not found at $SOURCE_DIR/$PERMISSION_FILE"

for path in \
  web static docs api openrgb \
  database/external database/keyboard database/language database/motherboard \
  database/nexus database/xeneon database/lcd/images; do
  [ -d "$SOURCE_DIR/$path" ] || fail "Required release directory not found at $SOURCE_DIR/$path"
done

for path in database/lcd/background.jpg database/rgb.json; do
  [ -f "$SOURCE_DIR/$path" ] || fail "Required release file not found at $SOURCE_DIR/$path"
done

systemctl --user show-environment >/dev/null 2>&1 || \
  fail "Unable to reach $TARGET_USER's systemd user manager. Run this installer from that user's desktop login."

if systemctl is-active --quiet "$PRODUCT.service" || systemctl is-enabled --quiet "$PRODUCT.service"; then
  fail "A system-level $PRODUCT.service is active or enabled. Stop and disable it explicitly before installing the user service; this installer will not alter it."
fi

session_has_device_group=false
if id -nG | tr ' ' '\n' | grep -Fxq "$DEVICE_GROUP"; then
  session_has_device_group=true
fi

user_in_device_group=false
if getent group "$DEVICE_GROUP" >/dev/null 2>&1 && \
  id -nG "$TARGET_USER" | tr ' ' '\n' | grep -Fxq "$DEVICE_GROUP"; then
  user_in_device_group=true
fi

if command -v sudo >/dev/null 2>&1; then
  PRIVILEGED_CMD=(sudo)
elif command -v run0 >/dev/null 2>&1; then
  echo "sudo not found. Falling back to run0."
  PRIVILEGED_CMD=(run0)
else
  fail "Neither sudo nor run0 is available for the required system changes."
fi

if ! grep -Fq 'OWNER="lumenforge"' "$SOURCE_DIR/$PERMISSION_FILE"; then
  fail "The source udev rule does not contain OWNER=\"lumenforge\" and cannot be safely transformed."
fi

temporary_rule="$(mktemp "${TMPDIR:-/tmp}/lumenforge-udev.XXXXXX")"
trap 'rm -f "$temporary_rule"' EXIT
sed 's/OWNER="lumenforge"/GROUP="lumenforge"/g' \
  "$SOURCE_DIR/$PERMISSION_FILE" > "$temporary_rule"

if grep -Fq 'OWNER="lumenforge"' "$temporary_rule"; then
  fail "Unable to transform the temporary udev rule for group-based device access."
fi
if ! grep -Fq 'GROUP="lumenforge"' "$temporary_rule"; then
  fail "The transformed temporary udev rule does not grant access to GROUP=\"lumenforge\"."
fi

user_service_exists=false
if [ -f "$SYSTEMD_FILE" ] || systemctl --user cat "$PRODUCT.service" >/dev/null 2>&1; then
  user_service_exists=true
fi

if [ "$user_service_exists" = true ]; then
  echo "Stopping the existing $PRODUCT user service before upgrade..."
  systemctl --user stop "$PRODUCT.service"
fi

echo "Installing release assets and device access configuration..."
"${PRIVILEGED_CMD[@]}" bash -s -- \
  "$SOURCE_DIR" "$INSTALL_DIR" "$PRODUCT" "$TARGET_USER" \
  "$DEVICE_GROUP" "$temporary_rule" "$PERMISSION_TARGET" "$user_in_device_group" <<'PRIVILEGED_SCRIPT'
set -e

SOURCE_DIR=$1
INSTALL_DIR=$2
PRODUCT=$3
TARGET_USER=$4
DEVICE_GROUP=$5
TEMPORARY_RULE=$6
PERMISSION_TARGET=$7
USER_IN_DEVICE_GROUP=$8

copy_release_assets() {
  echo "Installing release assets without replacing existing runtime data..."

  mkdir -p "$INSTALL_DIR"
  install -m 755 "$SOURCE_DIR/$PRODUCT" "$INSTALL_DIR/$PRODUCT"

  for directory in web static docs api openrgb; do
    rm -rf "$INSTALL_DIR/$directory"
    cp -a "$SOURCE_DIR/$directory" "$INSTALL_DIR/$directory"
  done

  mkdir -p "$INSTALL_DIR/database"
  for directory in external keyboard language motherboard nexus xeneon; do
    rm -rf "$INSTALL_DIR/database/$directory"
    cp -a "$SOURCE_DIR/database/$directory" "$INSTALL_DIR/database/$directory"
  done

  mkdir -p "$INSTALL_DIR/database/lcd"
  install -m 644 "$SOURCE_DIR/database/lcd/background.jpg" "$INSTALL_DIR/database/lcd/background.jpg"
  # LCD media includes user uploads. Add missing shipped defaults without replacing user content.
  mkdir -p "$INSTALL_DIR/database/lcd/images"
  cp -an "$SOURCE_DIR/database/lcd/images/." "$INSTALL_DIR/database/lcd/images/"
  install -m 644 "$SOURCE_DIR/database/rgb.json" "$INSTALL_DIR/database/rgb.json"

  # Runtime-owned directories and files are retained and never cleared on upgrade.
  mkdir -p \
    "$INSTALL_DIR/database/key-assignments" \
    "$INSTALL_DIR/database/led" \
    "$INSTALL_DIR/database/macros" \
    "$INSTALL_DIR/database/profiles" \
    "$INSTALL_DIR/database/rgb" \
    "$INSTALL_DIR/database/temperatures"

  # Install and upgrade only from a fresh source or release directory. Remove
  # legacy maintenance copies that cannot be run from the installed directory.
  rm -f \
    "$INSTALL_DIR/install.sh" \
    "$INSTALL_DIR/install-user-space.sh" \
    "$INSTALL_DIR/upgrade.sh" \
    "$INSTALL_DIR/99-lumenforge.rules"

  for file in README.md LICENSE CHANGELOG.md; do
    if [ -f "$SOURCE_DIR/$file" ]; then
      install -m 644 "$SOURCE_DIR/$file" "$INSTALL_DIR/$file"
    fi
  done
}

if ! getent group "$DEVICE_GROUP" >/dev/null 2>&1; then
  echo "Creating device-access group $DEVICE_GROUP..."
  groupadd -r "$DEVICE_GROUP"
fi

if [ "$USER_IN_DEVICE_GROUP" = false ]; then
  echo "Adding $TARGET_USER to $DEVICE_GROUP..."
  usermod -aG "$DEVICE_GROUP" "$TARGET_USER"
fi

copy_release_assets

echo "Setting ownership and permissions under $INSTALL_DIR..."
chown -R "$TARGET_USER:$DEVICE_GROUP" "$INSTALL_DIR"
chmod -R u+rwX,go+rX,go-w "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR/$PRODUCT"

echo "Installing group-based udev device permissions..."
install -m 644 "$TEMPORARY_RULE" "$PERMISSION_TARGET"
udevadm control --reload-rules
udevadm trigger
PRIVILEGED_SCRIPT

echo "Writing systemd user service..."
mkdir -p "$SYSTEMD_DIR"
cat > "$SYSTEMD_FILE" <<'EOM'
[Unit]
Description=LumenForge unified Linux RGB, cooling, and device control hub
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
Environment=LUMENFORGE_SERVICE_MODE=user
WorkingDirectory=/opt/LumenForge
ExecStart=/opt/LumenForge/LumenForge
Restart=on-failure
RestartSec=10
TimeoutStopSec=15

[Install]
WantedBy=default.target
EOM

systemctl --user daemon-reload
systemctl --user enable "$PRODUCT.service"

if [ "$session_has_device_group" = true ]; then
  echo "Starting $PRODUCT user service..."
  systemctl --user start "$PRODUCT.service"
  echo "Done. You can access the WebUI at: http://127.0.0.1:27003/"
else
  echo "$PRODUCT.service is installed and enabled, but it was not started."
  if [ "$user_in_device_group" = false ]; then
    echo "$TARGET_USER was newly added to the $DEVICE_GROUP group."
  else
    echo "The current login session has not acquired the $DEVICE_GROUP group."
  fi
  echo "Log out and back in, or reboot. The enabled user service will then start automatically."
fi

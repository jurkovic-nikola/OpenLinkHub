#!/bin/bash

set -e
USER_TO_CHECK="lumenforge"
SYSTEMD_FILE="/etc/systemd/system/LumenForge.service"
LEGACY_SYSTEMD_FILE="/usr/lib/systemd/system/LumenForge.service"
PRODUCT="LumenForge"
INSTALL_DIR="/opt/$PRODUCT"

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

  # Runtime-owned directories are created when absent and never cleared on upgrade.
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

check_user_service_conflict() {
  local candidate passwd_entry desktop_uid desktop_home

  for candidate in \
    "/etc/systemd/user/$PRODUCT.service" \
    "/etc/systemd/user/default.target.wants/$PRODUCT.service" \
    "/usr/local/lib/systemd/user/$PRODUCT.service" \
    "/usr/lib/systemd/user/$PRODUCT.service"; do
    if [ -e "$candidate" ] || [ -L "$candidate" ]; then
      fail "A system-wide user unit is installed or enabled at $candidate. Disable or remove that user service explicitly before installing the system service."
    fi
  done

  if [ -z "${SUDO_USER:-}" ] || [ "$SUDO_USER" = "root" ]; then
    echo "Warning: SUDO_USER does not identify an invoking desktop user, so a per-user $PRODUCT.service cannot be checked. Ensure every LumenForge user service is stopped and disabled before continuing."
    return
  fi

  if ! passwd_entry="$(getent passwd "$SUDO_USER")" || [ -z "$passwd_entry" ]; then
    echo "Warning: Unable to resolve SUDO_USER=$SUDO_USER, so that user's $PRODUCT.service cannot be checked. Ensure it is stopped and disabled before continuing."
    return
  fi
  IFS=: read -r _ _ desktop_uid _ _ desktop_home _ <<< "$passwd_entry"
  case "$desktop_uid" in
    '' | *[!0-9]*)
      echo "Warning: The account data for SUDO_USER=$SUDO_USER is invalid, so that user's $PRODUCT.service cannot be checked. Ensure it is stopped and disabled before continuing."
      return
      ;;
  esac
  case "$desktop_home" in
    /*) ;;
    *)
      echo "Warning: The account data for SUDO_USER=$SUDO_USER is invalid, so that user's $PRODUCT.service cannot be checked. Ensure it is stopped and disabled before continuing."
      return
      ;;
  esac

  for candidate in \
    "$desktop_home/.config/systemd/user/$PRODUCT.service" \
    "$desktop_home/.config/systemd/user/default.target.wants/$PRODUCT.service" \
    "$desktop_home/.local/share/systemd/user/$PRODUCT.service" \
    "/run/user/$desktop_uid/systemd/user/$PRODUCT.service" \
    "/run/user/$desktop_uid/systemd/transient/$PRODUCT.service"; do
    if [ -e "$candidate" ] || [ -L "$candidate" ]; then
      fail "A $PRODUCT user service for $SUDO_USER is installed, enabled, or transient at $candidate. Stop and disable it explicitly before installing the system service."
    fi
  done

  echo "Warning: the live systemd user-manager state for $SUDO_USER cannot be queried portably from this root installer. Known unit and enablement paths are clear; ensure no already-loaded $PRODUCT user service is still active."
}

if [ ! -f "$SOURCE_DIR/$PRODUCT" ]; then
  echo "Binary not found at $SOURCE_DIR/$PRODUCT"
  exit 1
fi

check_user_service_conflict

already_installed=false
if [ -f "$SYSTEMD_FILE" ] || [ -f "$LEGACY_SYSTEMD_FILE" ]; then
  already_installed=true
fi

echo "Checking if application user $USER_TO_CHECK exists..."
if ! getent group "$USER_TO_CHECK" >/dev/null; then
  echo "Creating application group $USER_TO_CHECK..."
  groupadd -r "$USER_TO_CHECK"
fi

if id "$USER_TO_CHECK" &>/dev/null; then
  echo "Application user $USER_TO_CHECK found."
else
  echo "Creating application user $USER_TO_CHECK..."
  useradd -r -g "$USER_TO_CHECK" -d "$INSTALL_DIR" -s /bin/false "$USER_TO_CHECK"
fi

if [ "$already_installed" = true ]; then
  echo "$PRODUCT is already installed. Performing upgrade..."
  systemctl stop "$PRODUCT" || true
else
  echo "Installing $PRODUCT..."
fi

copy_release_assets

echo "Setting permissions..."
chmod -R 755 "$INSTALL_DIR"
chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" "$INSTALL_DIR"

echo "Writing systemd service..."
cat > "$SYSTEMD_FILE" <<- EOM
[Unit]
Description=LumenForge unified Linux RGB, cooling, and device control hub
After=sleep.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
Environment=LUMENFORGE_SERVICE_MODE=system
User=$USER_TO_CHECK
Group=$USER_TO_CHECK
WorkingDirectory=/opt/$PRODUCT
ExecStart=/opt/$PRODUCT/$PRODUCT
Restart=on-failure
RestartSec=10
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
EOM

echo "Setting udev device permissions..."
install -m 644 "$SOURCE_DIR/99-lumenforge.rules" /etc/udev/rules.d/99-lumenforge.rules

echo "Reloading udev..."
udevadm control --reload-rules
udevadm trigger

echo "Reloading systemd and enabling service..."
systemctl daemon-reload
systemctl enable "$PRODUCT"

echo "Starting $PRODUCT..."
systemctl start "$PRODUCT"

echo "Done. You can access WebUI console via: http://127.0.0.1:27003/"
exit 0

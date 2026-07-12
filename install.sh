#!/bin/bash

set -e
USER_TO_CHECK="lumenforge"
DIST="/etc/lsb-release"
SYSTEMD_FILE="/etc/systemd/system/LumenForge.service"
PRODUCT="LumenForge"
SOURCE_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/opt/$PRODUCT"

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
  rm -rf "$INSTALL_DIR/database/lcd/images"
  cp -a "$SOURCE_DIR/database/lcd/images" "$INSTALL_DIR/database/lcd/images"
  install -m 644 "$SOURCE_DIR/database/rgb.json" "$INSTALL_DIR/database/rgb.json"

  # Runtime-owned directories are created when absent and never cleared on upgrade.
  mkdir -p \
    "$INSTALL_DIR/database/key-assignments" \
    "$INSTALL_DIR/database/led" \
    "$INSTALL_DIR/database/macros" \
    "$INSTALL_DIR/database/profiles" \
    "$INSTALL_DIR/database/rgb" \
    "$INSTALL_DIR/database/temperatures"

  # Remove the old standalone upgrader; upgrades now run install.sh from a new source checkout.
  rm -f "$INSTALL_DIR/upgrade.sh"

  for file in 99-lumenforge.rules install.sh; do
    if [ -f "$SOURCE_DIR/$file" ]; then
      install -m 755 "$SOURCE_DIR/$file" "$INSTALL_DIR/$file"
    fi
  done
  for file in README.md LICENSE CHANGELOG.md; do
    if [ -f "$SOURCE_DIR/$file" ]; then
      install -m 644 "$SOURCE_DIR/$file" "$INSTALL_DIR/$file"
    fi
  done
}

if [ ! -f "$SOURCE_DIR/$PRODUCT" ]; then
  echo "Binary not found at $SOURCE_DIR/$PRODUCT"
  exit 1
fi

if [ -f "$DIST" ]; then
  SYSTEMD_FILE="/etc/systemd/system/LumenForge.service"
else
  SYSTEMD_FILE="/usr/lib/systemd/system/LumenForge.service"
fi

already_installed=false
if [ -f "$SYSTEMD_FILE" ]; then
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

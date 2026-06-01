#!/bin/bash

set -e
CURRENT_DIR=$(pwd)
PRODUCT="OpenLinkHub"
PERMISSION_FILE="99-openlinkhub.rules"
PERMISSION_DIR="/etc/udev/rules.d/"
SEARCH_FOR="openlinkhub"
USER_TO_CHECK=$USER
SEARCH_FOR_GROUP='OWNER="openlinkhub"'
REPLACE_WITH='GROUP="openlinkhub"'

if [ ! -f "$PRODUCT" ]; then
  echo "No binary file. Exit"
  exit 0
fi

echo "Checking if username $USER_TO_CHECK exists..."
if id "$USER_TO_CHECK" &>/dev/null; then
    echo "Username $USER_TO_CHECK found."
else
    echo "Username $USER_TO_CHECK not found. Exit"
    exit 0
fi

GROUP_EXISTS=false
if getent group "$SEARCH_FOR" >/dev/null; then
    echo "Group '$SEARCH_FOR' already exists, skipping creation."
    GROUP_EXISTS=true
fi

USER_IN_GROUP=false
if id -nG "$USER_TO_CHECK" | grep -qw "$SEARCH_FOR"; then
    echo "$USER_TO_CHECK is already in $SEARCH_FOR"
    USER_IN_GROUP=true
fi

SYSTEMD_FILE="/home/$USER_TO_CHECK/.config/systemd/user/OpenLinkHub.service"

echo "Creating User systemd file..."
mkdir -p "/home/$USER_TO_CHECK/.config/systemd/user"

cat > "$SYSTEMD_FILE" <<- EOM
[Unit]
Description=Open source interface for iCUE LINK System Hub, Corsair AIOs and Hubs
After=default.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
WorkingDirectory=$CURRENT_DIR
ExecStart=$CURRENT_DIR/$PRODUCT
ExecReload=/bin/kill -s HUP \$MAINPID
RestartSec=5
Restart=always

[Install]
WantedBy=default.target
EOM

if [ -f "$SYSTEMD_FILE" ]; then
  echo "User systemd file created"
  systemctl --user daemon-reload
  systemctl --user enable "$PRODUCT"
fi

echo "Preparing permission file..."
sed -i -e "s/$SEARCH_FOR_GROUP/$REPLACE_WITH/g" "$PERMISSION_FILE"

# Check if sudo is available
if command -v sudo &>/dev/null; then
    PRIVILEGED_CMD="sudo"
else
    echo "sudo not found. Falling back to run0"
    PRIVILEGED_CMD="run0 -i"
fi

echo "Running privileged operations in one batch..."

$PRIVILEGED_CMD bash <<EOF
set -e

# Create group if needed
if [ "$GROUP_EXISTS" = false ]; then
    for gid in \$(seq 999 -1 900); do
        if ! getent group "\$gid" >/dev/null; then
            groupadd -g "\$gid" "$SEARCH_FOR"
            echo "Created group '$SEARCH_FOR' with GID \$gid"
            break
        fi
    done
fi

# Add user to group if needed
if [ "$USER_IN_GROUP" = false ]; then
    usermod -aG "$SEARCH_FOR" "$USER_TO_CHECK"
    echo "Added $USER_TO_CHECK to $SEARCH_FOR"
fi

# Copy udev rule
cp "$CURRENT_DIR/$PERMISSION_FILE" "$PERMISSION_DIR"
chmod -x "$PERMISSION_DIR"

# Reload udev rules
udevadm control --reload-rules
udevadm trigger

# Fix ownership and permissions
chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" "$CURRENT_DIR"
chmod -R 755 "$CURRENT_DIR"

EOF

echo "Done. Reboot your system with: systemctl reboot"

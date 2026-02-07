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

if [ ! -f $PRODUCT ]; then
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

if getent group "$SEARCH_FOR" >/dev/null; then
    echo "Group '$SEARCH_FOR' already exists, skipping creation."
else
    # Loop through system GIDs to find a free one
    for gid in $(seq 999 -1 900); do
        if ! getent group "$gid" >/dev/null; then
            sudo groupadd -g "$gid" "$SEARCH_FOR"
            echo "Created group '$SEARCH_FOR' with GID $gid"
            break
        fi
    done
fi

if id -nG "$USER_TO_CHECK" | grep -qw "$SEARCH_FOR"; then
    echo "$USER_TO_CHECK is already in $SEARCH_FOR"
else
    sudo usermod -aG "$SEARCH_FOR" "$USER_TO_CHECK"
    echo "Added $USER_TO_CHECK to $SEARCH_FOR"
fi

SYSTEMD_FILE="/home/$USER_TO_CHECK/.config/systemd/user/OpenLinkHub.service"

echo "Creating User systemd file..."
mkdir -p ~/.config/systemd/user
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
  echo "User systemd file: $SYSTEMD_FILE created"
  systemctl --user daemon-reload
  systemctl --user enable $PRODUCT
fi

echo "Copy device permission file $PERMISSION_FILE to $PERMISSION_DIR"
sed -i -e "s/$SEARCH_FOR_GROUP/$REPLACE_WITH/g" $PERMISSION_FILE
sudo cp "$CURRENT_DIR/$PERMISSION_FILE" $PERMISSION_DIR
sudo chmod -x $PERMISSION_DIR

echo "Reloading permissions..."
sudo udevadm control --reload-rules
sudo udevadm trigger

echo "Setting folder permissions..."
chmod -R 755 "$CURRENT_DIR"
chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" "$CURRENT_DIR"

echo "Done. Reboot your system with systemctl reboot"
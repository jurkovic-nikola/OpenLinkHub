#!/bin/bash

set -e
USER_TO_CHECK="openlinkhub"
DIST="/etc/lsb-release"
SYSTEMD_FILE="/etc/systemd/system/OpenLinkHub.service"
PRODUCT="OpenLinkHub"

if [ ! -f $PRODUCT ]; then
  echo "No binary file. Exit"
  exit 0
fi

if [ -f $DIST ]; then
  SYSTEMD_FILE="/etc/systemd/system/OpenLinkHub.service"
else
  SYSTEMD_FILE="/usr/lib/systemd/system/OpenLinkHub.service"
fi

echo "Checking if application username $USER_TO_CHECK exists..."
if id "$USER_TO_CHECK" &>/dev/null; then
    echo "Application username $USER_TO_CHECK found. Skipping username creation..."
else
    echo "Application username $USER_TO_CHECK not found. Creating application username..."
    sudo useradd -r "$USER_TO_CHECK" --shell=/bin/false
    if id "$USER_TO_CHECK" &>/dev/null; then
      echo "Application username $USER_TO_CHECK created."
      if [ -f $SYSTEMD_FILE ]; then
        cat > $SYSTEMD_FILE <<- EOM
[Unit]
Description=Open source interface for iCUE LINK System Hub, Corsair AIOs and Hubs
After=sleep.target

[Service]
User=$USER_TO_CHECK
Group=$USER_TO_CHECK
WorkingDirectory=/opt/$PRODUCT
ExecStart=/opt/$PRODUCT/$PRODUCT
ExecReload=/bin/kill -s HUP \$MAINPID
RestartSec=5

[Install]
WantedBy=multi-user.target
EOM
        sudo systemctl daemon-reload
      fi
    fi
fi

if [ -f $SYSTEMD_FILE ]; then
  echo "$PRODUCT is already installed. Performing upgrade"
  sudo systemctl stop $PRODUCT
  cp -r ../OpenLinkHub /opt
  chmod -R 755 /opt/$PRODUCT/
  chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" /opt/$PRODUCT/
  cp /opt/$PRODUCT/99-openlinkhub.rules /etc/udev/rules.d/
  echo "Reloading udev..."
  sudo udevadm control --reload-rules
  sudo udevadm trigger
  sudo systemctl start $PRODUCT
  echo "Done"
  exit 0
fi

echo "Installation is running..."
cp -r ../OpenLinkHub /opt
# Permissions
echo "Setting permissions..."
chmod -R 755 /opt/$PRODUCT/
chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" /opt/$PRODUCT/

# systemd file
echo "Creating systemd file..."
cat > $SYSTEMD_FILE <<- EOM
[Unit]
Description=Open source interface for iCUE LINK System Hub, Corsair AIOs and Hubs
After=sleep.target

[Service]
User=$USER_TO_CHECK
Group=$USER_TO_CHECK
WorkingDirectory=/opt/$PRODUCT
ExecStart=/opt/$PRODUCT/$PRODUCT
ExecReload=/bin/kill -s HUP \$MAINPID
RestartSec=10

[Install]
WantedBy=multi-user.target
EOM

echo "Running systemctl daemon-reload"
sudo systemctl daemon-reload

echo "Setting udev device permissions..."
sudo rm -f /etc/udev/rules.d/99-corsair*.rules
sudo cp 99-openlinkhub.rules /etc/udev/rules.d/

echo "Reloading udev..."
sudo udevadm control --reload-rules
sudo udevadm trigger

echo "Setting service to state: enabled"
sudo systemctl enable $PRODUCT

echo "Starting $PRODUCT..."
sudo systemctl start $PRODUCT

echo "Done. You can access WebUI console via: http://127.0.0.1:27003/"
exit 0
#!/bin/bash
echo "Cleanup..."
sudo rm -f /etc/udev/rules.d/99-corsair-*.rules

echo "Setting udev device permissions..."
APP_USERNAME="openlinkhub"
lsusb -d 1b1c: | while read -r line; do
ids=$(echo "$line" | awk '{print $6}')
vendor_id=$(echo "$ids" | cut -d':' -f1)
device_id=$(echo "$ids" | cut -d':' -f2)

match=false
usb_array=("1bc5" "2b10" "2b07" "1bfd" "1bfe" "1be3" "1bdb" "1bdc")
for hex in "${usb_array[@]}"; do
    if [ "$hex" == "$device_id" ]; then
        match=true
        break
    fi
done

if [ "$match" = true ]; then
cat > /etc/udev/rules.d/99-corsair-"$device_id".rules <<- EOM
SUBSYSTEMS=="usb", ATTRS{idVendor}=="$vendor_id", ATTRS{idProduct}=="$device_id", MODE="0600", OWNER="$APP_USERNAME"
EOM
else
cat > /etc/udev/rules.d/99-corsair-"$device_id".rules <<- EOM
KERNEL=="hidraw*", SUBSYSTEMS=="usb", ATTRS{idVendor}=="$vendor_id", ATTRS{idProduct}=="$device_id", MODE="0600", OWNER="$APP_USERNAME"
EOM
fi
done

echo "Reloading udev..."
sudo udevadm control --reload-rules
sudo udevadm trigger
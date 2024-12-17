#!/bin/sh

set -e
CURRENT_USER=$SUDO_USER
PRODUCT="OpenLinkHub"
PRODUCT_FILE="/opt/$PRODUCT/$PRODUCT"

if [ ! -f $PRODUCT ]; then
  echo "No binary file. Exit"
  exit 0
fi

if [ -f $PRODUCT_FILE ]; then
  echo "Performing upgrade..."
  sudo systemctl stop $PRODUCT
  cp OpenLinkHub "/opt/$PRODUCT/"
  cp -r web/ "/opt/$PRODUCT/"
  cp -r static/ "/opt/$PRODUCT/"
  cp -r database/ "/opt/$PRODUCT/"
  chmod -R 755 /opt/$PRODUCT
  chown -R "$CURRENT_USER":root /opt/$PRODUCT
  sudo systemctl start $PRODUCT
  exit 0
fi
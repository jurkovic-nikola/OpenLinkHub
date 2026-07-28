#!/usr/bin/env bash

# Recommended for desktop users and required for system-tray support.
# Run as the intended desktop user, never as root.

set -euo pipefail
umask 077

PRODUCT="LumenForge"
INSTALL_DIR="/opt/LumenForge"
PERMISSION_FILE="99-lumenforge.rules"
PERMISSION_TARGET="/etc/udev/rules.d/99-lumenforge.rules"
DEVICE_GROUP="lumenforge"

fail() {
  echo "Error: $*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required."
}

if [[ $EUID -eq 0 ]]; then
  fail "install-user-space.sh must be run as the intended desktop user, not root."
fi

for command in readlink getent id systemctl install cp find mktemp mv rm stat \
  cat sed grep tr dirname; do
  require_command "$command"
done

SCRIPT_PATH="$(readlink -f -- "$0")" || fail "Unable to resolve the installer path."
SOURCE_DIR="$(readlink -f -- "$(dirname -- "$SCRIPT_PATH")")" ||
  fail "Unable to resolve the release source directory."
CANONICAL_INSTALL_DIR="$(readlink -m -- "$INSTALL_DIR")" ||
  fail "Unable to resolve $INSTALL_DIR."
[[ $CANONICAL_INSTALL_DIR == "$INSTALL_DIR" ]] ||
  fail "The installation path did not resolve to $INSTALL_DIR."
[[ $SOURCE_DIR != "$CANONICAL_INSTALL_DIR" ]] ||
  fail "Refusing to run from $INSTALL_DIR. Run this installer from a fresh source checkout or extracted release directory."
[[ ! -L $INSTALL_DIR ]] || fail "Refusing symlinked application destination $INSTALL_DIR."

passwd_entry="$(getent passwd "$(id -u)")"
[[ -n $passwd_entry ]] || fail "Unable to resolve the current user through getent passwd."
IFS=: read -r TARGET_USER _ TARGET_UID _ _ USER_HOME _ <<<"$passwd_entry"
[[ -n $TARGET_USER && $TARGET_UID =~ ^[0-9]+$ ]] ||
  fail "The current user's account data is invalid."
[[ $USER_HOME == /* ]] ||
  fail "The home directory for $TARGET_USER is not absolute: $USER_HOME."

CONFIG_HOME="${XDG_CONFIG_HOME:-$USER_HOME/.config}"
DATA_HOME="${XDG_DATA_HOME:-$USER_HOME/.local/share}"
[[ $CONFIG_HOME == /* ]] || fail "XDG_CONFIG_HOME must be absolute when set: $CONFIG_HOME."
[[ $DATA_HOME == /* ]] || fail "XDG_DATA_HOME must be absolute when set: $DATA_HOME."
CONFIG_HOME="$(readlink -m -- "$CONFIG_HOME")" || fail "Unable to resolve $CONFIG_HOME."
DATA_HOME="$(readlink -m -- "$DATA_HOME")" || fail "Unable to resolve $DATA_HOME."
CONFIG_ROOT="$(readlink -m -- "$CONFIG_HOME/lumenforge")" ||
  fail "Unable to resolve the LumenForge configuration directory."
DATA_ROOT="$(readlink -m -- "$DATA_HOME/lumenforge")" ||
  fail "Unable to resolve the LumenForge data directory."

case "$CONFIG_ROOT" in
  "$INSTALL_DIR" | "$INSTALL_DIR"/*) fail "Configuration must not be stored under $INSTALL_DIR." ;;
esac
case "$DATA_ROOT" in
  "$INSTALL_DIR" | "$INSTALL_DIR"/*) fail "Mutable data must not be stored under $INSTALL_DIR." ;;
esac

SYSTEMD_DIR="$CONFIG_HOME/systemd/user"
SYSTEMD_FILE="$SYSTEMD_DIR/$PRODUCT.service"

required_directories=(
  web static docs api openrgb
  database/external database/keyboard database/language database/motherboard
  database/nexus database/xeneon database/lcd/images
)
required_files=(
  "$PRODUCT"
  "$PERMISSION_FILE"
  database/lcd/background.jpg
  database/rgb.json
)
for relative_path in "${required_directories[@]}"; do
  [[ -d $SOURCE_DIR/$relative_path ]] ||
    fail "Required release directory not found at $SOURCE_DIR/$relative_path."
done
for relative_path in "${required_files[@]}"; do
  [[ -f $SOURCE_DIR/$relative_path ]] ||
    fail "Required release file not found at $SOURCE_DIR/$relative_path."
done
for relative_path in "${required_directories[@]}" "${required_files[@]}"; do
  [[ ! -L $SOURCE_DIR/$relative_path ]] ||
    fail "Required release asset must not be a symlink: $SOURCE_DIR/$relative_path."
done
if [[ -n $(find -P "$SOURCE_DIR/web" "$SOURCE_DIR/static" "$SOURCE_DIR/docs" \
  "$SOURCE_DIR/api" "$SOURCE_DIR/openrgb" "$SOURCE_DIR/database/external" \
  "$SOURCE_DIR/database/keyboard" "$SOURCE_DIR/database/language" \
  "$SOURCE_DIR/database/motherboard" "$SOURCE_DIR/database/nexus" \
  "$SOURCE_DIR/database/xeneon" "$SOURCE_DIR/database/lcd/images" \
  -type l -print -quit) ]]; then
  fail "Release assets must not contain symlinks."
fi

systemctl --user show-environment >/dev/null 2>&1 ||
  fail "Unable to reach $TARGET_USER's systemd user manager. Run this installer from that user's desktop login."
if systemctl is-active --quiet "$PRODUCT.service" ||
  systemctl is-enabled --quiet "$PRODUCT.service"; then
  fail "A system-level $PRODUCT.service is active or enabled. Stop and disable it before installing the user service."
fi

session_has_device_group=false
if id -nG | tr ' ' '\n' | grep -Fxq "$DEVICE_GROUP"; then
  session_has_device_group=true
fi
user_was_in_device_group=false
if id -nG "$TARGET_USER" | tr ' ' '\n' | grep -Fxq "$DEVICE_GROUP"; then
  user_was_in_device_group=true
fi

if command -v sudo >/dev/null 2>&1; then
  PRIVILEGED_CMD=(sudo)
elif command -v run0 >/dev/null 2>&1; then
  PRIVILEGED_CMD=(run0)
else
  fail "Neither sudo nor run0 is available for the required privileged changes."
fi

if ! "${PRIVILEGED_CMD[@]}" bash -c '
  for command in bash getent id groupadd usermod udevadm chown install cp find \
    mktemp mv rm rmdir stat chmod; do
    command -v "$command" >/dev/null 2>&1 || {
      echo "Error: privileged command $command is required." >&2
      exit 1
    }
  done
'; then
  fail "The privileged environment is missing a required administration command."
fi

grep -Fq 'OWNER="lumenforge"' "$SOURCE_DIR/$PERMISSION_FILE" ||
  fail "The source udev rule cannot be safely transformed for group access."
temporary_rule=""
PREVIOUS_DIR=""
SWAP_MARKER=""
UNIT_TEMP=""
UNIT_READY=""
UNIT_BACKUP=""
UNIT_HAD_PREVIOUS=false
UNIT_REPLACED=false
UNIT_ENABLE_ATTEMPTED=false
USER_SERVICE_WAS_ACTIVE=false
USER_SERVICE_WAS_ENABLED=false
APPLICATION_SWAPPED=false
INSTALLATION_COMPLETE=false

rollback_error() {
  echo "Rollback error: $*" >&2
}

rollback_application_tree() {
  "${PRIVILEGED_CMD[@]}" bash -s -- \
    "$INSTALL_DIR" "$PRODUCT" "$PREVIOUS_DIR" "$SWAP_MARKER" <<'ROLLBACK_SCRIPT'
set -euo pipefail

INSTALL_DIR=$1
PRODUCT=$2
PREVIOUS_DIR=$3
SWAP_MARKER=$4
FAILED_DIR=""

previous_has_content=false
if [[ -n $PREVIOUS_DIR && -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR &&
  -n $(find "$PREVIOUS_DIR" -mindepth 1 -print -quit) ]]; then
  previous_has_content=true
fi

if [[ $previous_has_content == true ]]; then
  if [[ -e $INSTALL_DIR || -L $INSTALL_DIR ]]; then
    FAILED_DIR="$(mktemp -d /opt/.LumenForge.failed.XXXXXX)"
    rmdir -- "$FAILED_DIR"
    mv -- "$INSTALL_DIR" "$FAILED_DIR"
  fi
  mv -- "$PREVIOUS_DIR" "$INSTALL_DIR"
  if [[ -n $FAILED_DIR && -e $FAILED_DIR ]]; then
    rm -rf -- "$FAILED_DIR"
  fi
elif [[ -e $SWAP_MARKER || -L $SWAP_MARKER ]]; then
  if [[ -e $INSTALL_DIR || -L $INSTALL_DIR ]]; then
    rm -rf -- "$INSTALL_DIR"
  fi
  if [[ -n $PREVIOUS_DIR && -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR ]]; then
    rmdir -- "$PREVIOUS_DIR"
  fi
elif [[ -n $PREVIOUS_DIR && -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR ]]; then
  rmdir -- "$PREVIOUS_DIR"
fi

rm -f -- "$SWAP_MARKER"
ROLLBACK_SCRIPT
}

cleanup() {
  local original_status=$?
  local unit_restored=false

  trap - EXIT
  set +e

  for temporary_file in "$temporary_rule" "$UNIT_TEMP" "$UNIT_READY"; do
    if [[ -n $temporary_file && ( -e $temporary_file || -L $temporary_file ) ]]; then
      rm -f -- "$temporary_file" ||
        rollback_error "unable to remove temporary file $temporary_file"
    fi
  done

  if [[ $original_status -ne 0 && $INSTALLATION_COMPLETE == false ]]; then
    echo "Installation failed; attempting to restore the previous application and user service state." >&2

    if [[ $UNIT_ENABLE_ATTEMPTED == true && $USER_SERVICE_WAS_ENABLED == false ]]; then
      systemctl --user disable "$PRODUCT.service" >/dev/null 2>&1 ||
        rollback_error "unable to undo newly enabled user service $PRODUCT.service"
    fi

    if [[ -n $PREVIOUS_DIR ]]; then
      rollback_application_tree ||
        rollback_error "unable to restore the previous application tree; inspect $PREVIOUS_DIR and $INSTALL_DIR"
    fi

    if [[ $UNIT_REPLACED == true ]]; then
      if [[ $UNIT_HAD_PREVIOUS == true && -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
        if mv -fT -- "$UNIT_BACKUP" "$SYSTEMD_FILE"; then
          UNIT_BACKUP=""
          unit_restored=true
        else
          rollback_error "unable to restore the previous user unit at $SYSTEMD_FILE"
        fi
      else
        if rm -f -- "$SYSTEMD_FILE"; then
          unit_restored=true
        else
          rollback_error "unable to remove the newly installed user unit at $SYSTEMD_FILE"
        fi
      fi
    fi

    if [[ $UNIT_REPLACED == true && $unit_restored == true ]]; then
      systemctl --user daemon-reload ||
        rollback_error "user systemd daemon reload failed after restoring the previous unit"
    fi

    if [[ $USER_SERVICE_WAS_ACTIVE == true ]]; then
      systemctl --user start "$PRODUCT.service" ||
        rollback_error "unable to restore the previously active user service $PRODUCT.service"
    fi
  elif [[ -n $PREVIOUS_DIR && $INSTALLATION_COMPLETE == false ]]; then
    rollback_application_tree ||
      rollback_error "unable to clean up unused rollback directory $PREVIOUS_DIR"
  fi

  if [[ -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
    if [[ $original_status -eq 0 || $UNIT_REPLACED == false || $unit_restored == true ]]; then
      rm -f -- "$UNIT_BACKUP" ||
        rollback_error "unable to remove temporary user-unit backup $UNIT_BACKUP"
    else
      rollback_error "previous user-unit backup retained at $UNIT_BACKUP after restore failure"
    fi
  fi

  exit "$original_status"
}
trap cleanup EXIT

temporary_rule="$(mktemp "${TMPDIR:-/tmp}/lumenforge-udev.XXXXXX")"
sed 's/OWNER="lumenforge"/GROUP="lumenforge"/g' \
  "$SOURCE_DIR/$PERMISSION_FILE" >"$temporary_rule"
grep -Fq 'OWNER="lumenforge"' "$temporary_rule" &&
  fail "Unable to remove owner-only access from the temporary udev rule."
grep -Fq 'GROUP="lumenforge"' "$temporary_rule" ||
  fail "The transformed udev rule does not grant group access."

install -d -m 0700 "$SYSTEMD_DIR"
if [[ -e $SYSTEMD_FILE || -L $SYSTEMD_FILE ]]; then
  [[ ! -L $SYSTEMD_FILE && -f $SYSTEMD_FILE ]] ||
    fail "Refusing to replace non-regular or symlinked user unit destination $SYSTEMD_FILE."
  UNIT_BACKUP="$(mktemp "$SYSTEMD_DIR/.LumenForge.service.previous.XXXXXX")"
  cp --preserve=mode,ownership,timestamps -- "$SYSTEMD_FILE" "$UNIT_BACKUP" ||
    fail "Unable to preserve the existing user unit before replacement."
  UNIT_HAD_PREVIOUS=true
fi

if systemctl --user is-active --quiet "$PRODUCT.service"; then
  USER_SERVICE_WAS_ACTIVE=true
  echo "Stopping the existing $PRODUCT user service before replacing application files..."
  systemctl --user stop "$PRODUCT.service"
fi
if systemctl --user is-enabled --quiet "$PRODUCT.service"; then
  USER_SERVICE_WAS_ENABLED=true
fi

install -d -m 0700 "$CONFIG_ROOT" "$DATA_ROOT"
for directory in key-assignments led macros profiles rgb temperatures lcd lcd/images; do
  install -d -m 0700 "$DATA_ROOT/database/$directory"
done

PREVIOUS_DIR="$("${PRIVILEGED_CMD[@]}" mktemp -d /opt/.LumenForge.previous.XXXXXX)" ||
  fail "Unable to allocate a root-controlled rollback directory."
[[ $PREVIOUS_DIR == /opt/.LumenForge.previous.* && -n $PREVIOUS_DIR ]] ||
  fail "Privileged rollback directory has an unexpected path: $PREVIOUS_DIR."
SWAP_MARKER="$PREVIOUS_DIR.swapped"

"${PRIVILEGED_CMD[@]}" bash -s -- \
  "$SOURCE_DIR" "$INSTALL_DIR" "$PRODUCT" "$TARGET_USER" "$DEVICE_GROUP" \
  "$temporary_rule" "$PERMISSION_TARGET" "$PREVIOUS_DIR" "$SWAP_MARKER" \
  <<'PRIVILEGED_SCRIPT'
set -euo pipefail
umask 077

SOURCE_DIR=$1
INSTALL_DIR=$2
PRODUCT=$3
TARGET_USER=$4
DEVICE_GROUP=$5
TEMPORARY_RULE=$6
PERMISSION_TARGET=$7
PREVIOUS_DIR=$8
SWAP_MARKER=$9

[[ ! -L $INSTALL_DIR ]] || {
  echo "Error: refusing symlinked application destination $INSTALL_DIR." >&2
  exit 1
}
[[ -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR &&
  $(stat -c '%U:%G:%a' "$PREVIOUS_DIR") == "root:root:700" ]] || {
  echo "Error: invalid root-controlled rollback directory $PREVIOUS_DIR." >&2
  exit 1
}
[[ ! -e $SWAP_MARKER && ! -L $SWAP_MARKER ]] || {
  echo "Error: rollback marker already exists at $SWAP_MARKER." >&2
  exit 1
}

validate_device_group_entry() {
  local entry=$1
  local group_name group_password group_gid group_members

  [[ $entry != *$'\n'* && $entry =~ ^[^:]+:[^:]*:[^:]+:[^:]*$ ]] || {
    echo "Error: existing $DEVICE_GROUP group data is incomplete or malformed; refusing to modify it." >&2
    exit 1
  }
  IFS=: read -r group_name group_password group_gid group_members <<<"$entry"
  [[ $group_name == "$DEVICE_GROUP" && $group_gid =~ ^[0-9]+$ ]] || {
    echo "Error: existing $DEVICE_GROUP group data is incomplete or malformed; refusing to modify it." >&2
    exit 1
  }
  [[ $group_gid -ne 0 ]] || {
    echo "Error: existing $DEVICE_GROUP group uses privileged GID 0; refusing to modify it." >&2
    exit 1
  }
  [[ $group_gid -gt 0 ]] || {
    echo "Error: existing $DEVICE_GROUP group must use a nonzero GID; refusing to modify it." >&2
    exit 1
  }
}

group_entry=""
if group_entry="$(getent group "$DEVICE_GROUP")"; then
  validate_device_group_entry "$group_entry"
else
  groupadd -r "$DEVICE_GROUP"
  group_entry="$(getent group "$DEVICE_GROUP")" || {
    echo "Error: unable to resolve the newly created $DEVICE_GROUP group." >&2
    exit 1
  }
  validate_device_group_entry "$group_entry"
fi

target_groups="$(id -nG "$TARGET_USER")"
if [[ " $target_groups " != *" $DEVICE_GROUP "* ]]; then
  usermod -aG "$DEVICE_GROUP" "$TARGET_USER"
fi

STAGING_DIR=""
PRIVILEGED_COMPLETE=false
cleanup() {
  local original_status=$?
  local failed_dir=""

  trap - EXIT
  set +e

  if [[ -n $STAGING_DIR && -e $STAGING_DIR ]]; then
    rm -rf -- "$STAGING_DIR" ||
      echo "Rollback error: unable to remove staging directory $STAGING_DIR." >&2
  fi

  if [[ $original_status -ne 0 && $PRIVILEGED_COMPLETE == false ]]; then
    if [[ -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR &&
      -n $(find "$PREVIOUS_DIR" -mindepth 1 -print -quit) ]]; then
      if [[ -e $INSTALL_DIR || -L $INSTALL_DIR ]]; then
        failed_dir="$(mktemp -d /opt/.LumenForge.failed.XXXXXX)"
        if rmdir -- "$failed_dir" && mv -- "$INSTALL_DIR" "$failed_dir"; then
          :
        else
          echo "Rollback error: unable to move failed application tree from $INSTALL_DIR." >&2
        fi
      fi
      if [[ ! -e $INSTALL_DIR && ! -L $INSTALL_DIR ]]; then
        if mv -- "$PREVIOUS_DIR" "$INSTALL_DIR"; then
          if [[ -n $failed_dir && -e $failed_dir ]]; then
            rm -rf -- "$failed_dir" ||
              echo "Rollback error: unable to remove failed tree $failed_dir." >&2
          fi
        else
          echo "Rollback error: unable to restore $PREVIOUS_DIR to $INSTALL_DIR." >&2
        fi
      fi
    elif [[ -e $SWAP_MARKER || -L $SWAP_MARKER ]]; then
      rm -rf -- "$INSTALL_DIR" ||
        echo "Rollback error: unable to remove failed fresh application tree $INSTALL_DIR." >&2
      if [[ -d $PREVIOUS_DIR && ! -L $PREVIOUS_DIR ]]; then
        rmdir -- "$PREVIOUS_DIR" ||
          echo "Rollback error: unable to remove empty rollback directory $PREVIOUS_DIR." >&2
      fi
    fi
    rm -f -- "$SWAP_MARKER" ||
      echo "Rollback error: unable to remove rollback marker $SWAP_MARKER." >&2
  fi

  exit "$original_status"
}
trap cleanup EXIT

STAGING_DIR="$(mktemp -d /opt/.LumenForge.stage.XXXXXX)"
install -d -o root -g root -m 0755 "$STAGING_DIR"
install -o root -g root -m 0755 "$SOURCE_DIR/$PRODUCT" "$STAGING_DIR/$PRODUCT"
for directory in web static docs api openrgb; do
  cp -a -- "$SOURCE_DIR/$directory" "$STAGING_DIR/$directory"
done
install -d -o root -g root -m 0755 "$STAGING_DIR/database"
for directory in external keyboard language motherboard nexus xeneon; do
  cp -a -- "$SOURCE_DIR/database/$directory" "$STAGING_DIR/database/$directory"
done
install -d -o root -g root -m 0755 "$STAGING_DIR/database/lcd"
cp -a -- "$SOURCE_DIR/database/lcd/images" "$STAGING_DIR/database/lcd/images"
install -o root -g root -m 0644 \
  "$SOURCE_DIR/database/lcd/background.jpg" "$STAGING_DIR/database/lcd/background.jpg"
install -o root -g root -m 0644 \
  "$SOURCE_DIR/database/rgb.json" "$STAGING_DIR/database/rgb.json"
for file in README.md LICENSE CHANGELOG.md; do
  if [[ -f $SOURCE_DIR/$file && ! -L $SOURCE_DIR/$file ]]; then
    install -o root -g root -m 0644 "$SOURCE_DIR/$file" "$STAGING_DIR/$file"
  fi
done

chown -R root:root "$STAGING_DIR"
find "$STAGING_DIR" -type d -exec chmod 0755 {} +
find "$STAGING_DIR" -type f -exec chmod 0644 {} +
chmod 0755 "$STAGING_DIR/$PRODUCT"

[[ ! -L $INSTALL_DIR ]] || {
  echo "Error: refusing symlinked application destination $INSTALL_DIR." >&2
  exit 1
}
if [[ -e $INSTALL_DIR ]]; then
  rmdir -- "$PREVIOUS_DIR"
  mv -- "$INSTALL_DIR" "$PREVIOUS_DIR"
else
  rmdir -- "$PREVIOUS_DIR"
fi
mv -- "$STAGING_DIR" "$INSTALL_DIR"
STAGING_DIR=""
install -o root -g root -m 0600 /dev/null "$SWAP_MARKER"

install -o root -g root -m 0644 "$TEMPORARY_RULE" "$PERMISSION_TARGET"
udevadm control --reload-rules
udevadm trigger

PRIVILEGED_COMPLETE=true
PRIVILEGED_SCRIPT
APPLICATION_SWAPPED=true

escape_systemd_value() {
  local value=$1
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//%/%%}
  printf '%s' "$value"
}

escaped_config_root="$(escape_systemd_value "$CONFIG_ROOT")"
escaped_data_root="$(escape_systemd_value "$DATA_ROOT")"
UNIT_TEMP="$(mktemp "$SYSTEMD_DIR/.LumenForge.service.write.XXXXXX")"
exec {unit_fd}>"$UNIT_TEMP" ||
  fail "Unable to open the temporary user unit for writing."
if ! cat >&"$unit_fd" <<EOF
[Unit]
Description=LumenForge unified Linux RGB, cooling, and device control hub
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
Environment=LUMENFORGE_SERVICE_MODE=user
Environment=LUMENFORGE_APPLICATION_ROOT=/opt/LumenForge
Environment="LUMENFORGE_CONFIG_ROOT=$escaped_config_root"
Environment="LUMENFORGE_DATA_ROOT=$escaped_data_root"
UMask=0077
ExecStart=/opt/LumenForge/LumenForge
Restart=on-failure
RestartSec=10
TimeoutStopSec=15

[Install]
WantedBy=default.target
EOF
then
  exec {unit_fd}>&- || true
  fail "Unable to write the complete temporary user unit."
fi
if ! exec {unit_fd}>&-; then
  fail "Unable to close the temporary user unit after writing."
fi

UNIT_READY="$(mktemp "$SYSTEMD_DIR/.LumenForge.service.ready.XXXXXX")"
install -m 0600 "$UNIT_TEMP" "$UNIT_READY" ||
  fail "Unable to prepare the final user unit."
[[ $(stat -c '%u:%a' "$UNIT_READY") == "$TARGET_UID:600" ]] ||
  fail "Prepared user unit is not owned by $TARGET_USER with mode 0600."
mv -fT -- "$UNIT_READY" "$SYSTEMD_FILE" ||
  fail "Unable to atomically replace $SYSTEMD_FILE."
UNIT_READY=""
UNIT_REPLACED=true
rm -f -- "$UNIT_TEMP"
UNIT_TEMP=""

systemctl --user daemon-reload
UNIT_ENABLE_ATTEMPTED=true
systemctl --user enable "$PRODUCT.service"

if [[ $session_has_device_group == true ]]; then
  echo "Starting $PRODUCT user service..."
  systemctl --user start "$PRODUCT.service"
else
  echo "$PRODUCT.service is installed and enabled, but it was not started."
  if [[ $user_was_in_device_group == false ]]; then
    echo "$TARGET_USER was added to the $DEVICE_GROUP group."
  else
    echo "The current login session has not acquired the $DEVICE_GROUP group."
  fi
  echo "Log out and back in, or reboot. The enabled user service will start in the refreshed session."
fi

INSTALLATION_COMPLETE=true
if ! "${PRIVILEGED_CMD[@]}" rm -rf -- "$PREVIOUS_DIR" "$SWAP_MARKER"; then
  echo "Warning: installation succeeded, but rollback artifacts remain at $PREVIOUS_DIR or $SWAP_MARKER." >&2
else
  PREVIOUS_DIR=""
  SWAP_MARKER=""
fi
if [[ -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
  if rm -f -- "$UNIT_BACKUP"; then
    UNIT_BACKUP=""
  else
    echo "Warning: installation succeeded, but the previous user-unit backup remains at $UNIT_BACKUP." >&2
  fi
fi

if [[ $session_has_device_group == true ]]; then
  echo "Done. You can access the WebUI at http://127.0.0.1:27003/"
fi

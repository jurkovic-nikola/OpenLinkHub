#!/usr/bin/env bash

set -euo pipefail
umask 077

PRODUCT="LumenForge"
RUNTIME_USER="lumenforge"
RUNTIME_GROUP="lumenforge"
INSTALL_DIR="/opt/LumenForge"
STATE_DIR="/var/lib/lumenforge"
SYSTEMD_FILE="/etc/systemd/system/LumenForge.service"
LEGACY_SYSTEMD_FILE="/usr/lib/systemd/system/LumenForge.service"
UDEV_TARGET="/etc/udev/rules.d/99-lumenforge.rules"

fail() {
  echo "Error: $*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required."
}

if [[ $EUID -ne 0 ]]; then
  fail "install.sh must be run as root."
fi

for command in readlink getent install cp find mktemp mv rm rmdir chmod chown \
  stat cat systemctl udevadm groupadd useradd nologin; do
  require_command "$command"
done
NOLOGIN_SHELL="$(command -v nologin)"

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

required_directories=(
  web static docs api openrgb
  database/external database/keyboard database/language database/motherboard
  database/nexus database/xeneon database/lcd/images
)
required_files=(
  "$PRODUCT"
  99-lumenforge.rules
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

check_user_service_conflict() {
  local candidate passwd_entry desktop_uid desktop_home

  for candidate in \
    "/etc/systemd/user/$PRODUCT.service" \
    "/etc/systemd/user/default.target.wants/$PRODUCT.service" \
    "/usr/local/lib/systemd/user/$PRODUCT.service" \
    "/usr/lib/systemd/user/$PRODUCT.service"; do
    if [[ -e $candidate || -L $candidate ]]; then
      fail "A system-wide user unit is installed or enabled at $candidate. Disable or remove that user service explicitly before installing the system service."
    fi
  done

  if [[ -z ${SUDO_USER:-} || ${SUDO_USER:-} == root ]]; then
    echo "Warning: SUDO_USER does not identify an invoking desktop user, so a per-user $PRODUCT.service cannot be checked. Ensure every LumenForge user service is stopped and disabled before continuing."
    return
  fi
  if ! passwd_entry="$(getent passwd "$SUDO_USER")" || [[ -z $passwd_entry ]]; then
    echo "Warning: Unable to resolve SUDO_USER=$SUDO_USER, so that user's $PRODUCT.service cannot be checked."
    return
  fi
  IFS=: read -r _ _ desktop_uid _ _ desktop_home _ <<<"$passwd_entry"
  [[ $desktop_uid =~ ^[0-9]+$ && $desktop_home == /* ]] ||
    fail "Invalid account data for SUDO_USER=$SUDO_USER."

  for candidate in \
    "$desktop_home/.config/systemd/user/$PRODUCT.service" \
    "$desktop_home/.config/systemd/user/default.target.wants/$PRODUCT.service" \
    "$desktop_home/.local/share/systemd/user/$PRODUCT.service" \
    "/run/user/$desktop_uid/systemd/user/$PRODUCT.service" \
    "/run/user/$desktop_uid/systemd/transient/$PRODUCT.service"; do
    if [[ -e $candidate || -L $candidate ]]; then
      fail "A $PRODUCT user service for $SUDO_USER is installed, enabled, or transient at $candidate. Stop and disable it before installing the system service."
    fi
  done
}

check_user_service_conflict

echo "Creating or validating the system runtime identity..."
group_entry=""
passwd_entry=""

validate_runtime_group_entry() {
  local entry=$1
  local group_name group_password group_gid group_members

  [[ $entry != *$'\n'* && $entry =~ ^[^:]+:[^:]*:[^:]+:[^:]*$ ]] ||
    fail "Existing $RUNTIME_GROUP group data is incomplete or malformed; refusing to modify it."
  IFS=: read -r group_name group_password group_gid group_members <<<"$entry"
  [[ $group_name == "$RUNTIME_GROUP" && $group_gid =~ ^[0-9]+$ ]] ||
    fail "Existing $RUNTIME_GROUP group data is incomplete or malformed; refusing to modify it."
  [[ $group_gid -ne 0 ]] ||
    fail "Existing $RUNTIME_GROUP group uses privileged GID 0; refusing to modify it."
  [[ $group_gid -gt 0 ]] ||
    fail "Existing $RUNTIME_GROUP group must use a nonzero GID; refusing to modify it."
  RUNTIME_GROUP_GID=$group_gid
}

RUNTIME_GROUP_GID=""
if group_entry="$(getent group "$RUNTIME_GROUP")"; then
  validate_runtime_group_entry "$group_entry"
fi

if passwd_entry="$(getent passwd "$RUNTIME_USER")"; then
  [[ -n $group_entry ]] ||
    fail "Existing $RUNTIME_USER account has no matching $RUNTIME_GROUP group; refusing to modify the account or create a conflicting group."
  IFS=: read -r account_name account_password account_uid account_gid account_gecos \
    account_home account_shell account_extra <<<"$passwd_entry"
  [[ $account_name == "$RUNTIME_USER" &&
    -n $account_password &&
    $account_uid =~ ^[0-9]+$ &&
    $account_uid -ne 0 &&
    $account_gid =~ ^[0-9]+$ &&
    $account_gid -eq "$RUNTIME_GROUP_GID" &&
    $account_home == "$STATE_DIR" &&
    $account_shell == "$NOLOGIN_SHELL" &&
    -z $account_extra ]] ||
    fail "Existing $RUNTIME_USER account is not the dedicated LumenForge service identity; expected non-root UID, primary group $RUNTIME_GROUP, home $STATE_DIR, and shell $NOLOGIN_SHELL. No account changes were made."
else
  if [[ -z $group_entry ]]; then
    groupadd -r "$RUNTIME_GROUP"
    group_entry="$(getent group "$RUNTIME_GROUP")" ||
      fail "Unable to resolve the newly created $RUNTIME_GROUP group."
    validate_runtime_group_entry "$group_entry"
  fi
  useradd -r -g "$RUNTIME_GROUP" -d "$STATE_DIR" -s "$NOLOGIN_SHELL" \
    -M -p '!' "$RUNTIME_USER"
fi

STAGING_DIR=""
PREVIOUS_DIR=""
FAILED_DIR=""
UNIT_TEMP=""
UNIT_READY=""
UNIT_BACKUP=""
UNIT_HAD_PREVIOUS=false
UNIT_REPLACED=false
UNIT_ENABLE_ATTEMPTED=false
SERVICE_WAS_ACTIVE=false
SERVICE_WAS_ENABLED=false
APPLICATION_SWAP_STARTED=false
APPLICATION_SWAPPED=false
INSTALLATION_COMPLETE=false

rollback_error() {
  echo "Rollback error: $*" >&2
}

cleanup() {
  local original_status=$?
  local unit_restored=false

  trap - EXIT
  set +e

  if [[ -n $STAGING_DIR && -e $STAGING_DIR ]]; then
    rm -rf -- "$STAGING_DIR" ||
      rollback_error "unable to remove staging directory $STAGING_DIR"
  fi

  for temporary_file in "$UNIT_TEMP" "$UNIT_READY"; do
    if [[ -n $temporary_file && ( -e $temporary_file || -L $temporary_file ) ]]; then
      rm -f -- "$temporary_file" ||
        rollback_error "unable to remove temporary unit file $temporary_file"
    fi
  done

  if [[ $original_status -ne 0 && $INSTALLATION_COMPLETE == false ]]; then
    echo "Installation failed; attempting to restore the previous application and service state." >&2

    if [[ $UNIT_ENABLE_ATTEMPTED == true && $SERVICE_WAS_ENABLED == false ]]; then
      systemctl disable "$PRODUCT.service" >/dev/null 2>&1 ||
        rollback_error "unable to undo newly enabled $PRODUCT.service"
    fi

    if [[ $APPLICATION_SWAP_STARTED == true ]]; then
      if [[ -n $PREVIOUS_DIR && -e $PREVIOUS_DIR ]]; then
        if [[ -e $INSTALL_DIR || -L $INSTALL_DIR ]]; then
          FAILED_DIR="$(mktemp -d /opt/.LumenForge.failed.XXXXXX)"
          if rmdir -- "$FAILED_DIR" && mv -- "$INSTALL_DIR" "$FAILED_DIR"; then
            :
          else
            rollback_error "unable to move the failed application tree away from $INSTALL_DIR"
          fi
        fi
        if [[ ! -e $INSTALL_DIR && ! -L $INSTALL_DIR ]]; then
          if mv -- "$PREVIOUS_DIR" "$INSTALL_DIR"; then
            if [[ -n $FAILED_DIR && -e $FAILED_DIR ]]; then
              rm -rf -- "$FAILED_DIR" ||
                rollback_error "unable to remove failed replacement tree $FAILED_DIR"
            fi
          else
            rollback_error "unable to restore $PREVIOUS_DIR to $INSTALL_DIR"
          fi
        else
          rollback_error "cannot restore $PREVIOUS_DIR because $INSTALL_DIR is still occupied"
        fi
      elif [[ $APPLICATION_SWAPPED == true && ( -e $INSTALL_DIR || -L $INSTALL_DIR ) ]]; then
        rm -rf -- "$INSTALL_DIR" ||
          rollback_error "unable to remove the failed fresh application tree at $INSTALL_DIR"
      fi
    fi

    if [[ $UNIT_REPLACED == true ]]; then
      if [[ $UNIT_HAD_PREVIOUS == true && -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
        if mv -fT -- "$UNIT_BACKUP" "$SYSTEMD_FILE"; then
          UNIT_BACKUP=""
          unit_restored=true
        else
          rollback_error "unable to restore the previous system unit at $SYSTEMD_FILE"
        fi
      else
        if rm -f -- "$SYSTEMD_FILE"; then
          unit_restored=true
        else
          rollback_error "unable to remove the newly installed system unit at $SYSTEMD_FILE"
        fi
      fi
    fi

    if [[ $UNIT_REPLACED == true && $unit_restored == true ]]; then
      systemctl daemon-reload ||
        rollback_error "systemd daemon reload failed after restoring the previous unit"
    fi

    if [[ $SERVICE_WAS_ACTIVE == true ]]; then
      systemctl start "$PRODUCT.service" ||
        rollback_error "unable to restore the previously active $PRODUCT.service"
    fi
  fi

  if [[ -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
    if [[ $original_status -eq 0 || $UNIT_REPLACED == false || $unit_restored == true ]]; then
      rm -f -- "$UNIT_BACKUP" ||
        rollback_error "unable to remove temporary unit backup $UNIT_BACKUP"
    else
      rollback_error "previous unit backup retained at $UNIT_BACKUP after restore failure"
    fi
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

if [[ -e $SYSTEMD_FILE || -L $SYSTEMD_FILE ]]; then
  [[ ! -L $SYSTEMD_FILE && -f $SYSTEMD_FILE ]] ||
    fail "Refusing to replace non-regular or symlinked system unit destination $SYSTEMD_FILE."
  UNIT_BACKUP="$(mktemp /etc/systemd/system/.LumenForge.service.previous.XXXXXX)"
  cp --preserve=mode,ownership,timestamps -- "$SYSTEMD_FILE" "$UNIT_BACKUP" ||
    fail "Unable to preserve the existing system unit before replacement."
  UNIT_HAD_PREVIOUS=true
fi

if systemctl is-active --quiet "$PRODUCT.service"; then
  SERVICE_WAS_ACTIVE=true
fi
if systemctl is-enabled --quiet "$PRODUCT.service"; then
  SERVICE_WAS_ENABLED=true
fi
if [[ $SERVICE_WAS_ACTIVE == true ]]; then
  echo "Stopping the running $PRODUCT system service..."
  systemctl stop "$PRODUCT.service"
fi

if [[ -e $STATE_DIR || -L $STATE_DIR ]]; then
  [[ ! -L $STATE_DIR && -d $STATE_DIR ]] ||
    fail "Refusing symlinked or non-directory state root $STATE_DIR."
  [[ $(stat -c '%U:%G:%a' "$STATE_DIR") == "$RUNTIME_USER:$RUNTIME_GROUP:750" ]] ||
    fail "Existing state root $STATE_DIR must be owned by $RUNTIME_USER:$RUNTIME_GROUP with mode 0750; refusing to normalize service-owned state."
else
  install -d -o "$RUNTIME_USER" -g "$RUNTIME_GROUP" -m 0750 "$STATE_DIR"
fi

[[ ! -L $INSTALL_DIR ]] || fail "Refusing symlinked application destination $INSTALL_DIR."
APPLICATION_SWAP_STARTED=true
if [[ -e $INSTALL_DIR ]]; then
  PREVIOUS_DIR="$(mktemp -d /opt/.LumenForge.previous.XXXXXX)"
  rmdir -- "$PREVIOUS_DIR"
  mv -- "$INSTALL_DIR" "$PREVIOUS_DIR"
fi
mv -- "$STAGING_DIR" "$INSTALL_DIR"
STAGING_DIR=""
APPLICATION_SWAPPED=true

UNIT_TEMP="$(mktemp /etc/systemd/system/.LumenForge.service.write.XXXXXX)"
exec {unit_fd}>"$UNIT_TEMP" ||
  fail "Unable to open the temporary system unit for writing."
if ! cat >&"$unit_fd" <<EOF
[Unit]
Description=LumenForge unified Linux RGB, cooling, and device control hub
After=sleep.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
Environment=LUMENFORGE_SERVICE_MODE=system
Environment=LUMENFORGE_APPLICATION_ROOT=/opt/LumenForge
Environment=LUMENFORGE_CONFIG_ROOT=/var/lib/lumenforge
Environment=LUMENFORGE_DATA_ROOT=/var/lib/lumenforge
User=$RUNTIME_USER
Group=$RUNTIME_GROUP
UMask=0077
ExecStart=/opt/LumenForge/LumenForge
Restart=on-failure
RestartSec=10
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
EOF
then
  exec {unit_fd}>&- || true
  fail "Unable to write the complete temporary system unit."
fi
if ! exec {unit_fd}>&-; then
  fail "Unable to close the temporary system unit after writing."
fi

UNIT_READY="$(mktemp /etc/systemd/system/.LumenForge.service.ready.XXXXXX)"
install -o root -g root -m 0644 "$UNIT_TEMP" "$UNIT_READY" ||
  fail "Unable to prepare the final system unit."
[[ $(stat -c '%U:%G:%a' "$UNIT_READY") == "root:root:644" ]] ||
  fail "Prepared system unit does not have root:root ownership and mode 0644."
mv -fT -- "$UNIT_READY" "$SYSTEMD_FILE" ||
  fail "Unable to atomically replace $SYSTEMD_FILE."
UNIT_READY=""
UNIT_REPLACED=true
rm -f -- "$UNIT_TEMP"
UNIT_TEMP=""

install -o root -g root -m 0644 "$SOURCE_DIR/99-lumenforge.rules" "$UDEV_TARGET"
udevadm control --reload-rules
udevadm trigger

systemctl daemon-reload
UNIT_ENABLE_ATTEMPTED=true
systemctl enable "$PRODUCT.service"

echo "Starting $PRODUCT system service..."
systemctl start "$PRODUCT.service"

INSTALLATION_COMPLETE=true
if [[ -n $PREVIOUS_DIR && -e $PREVIOUS_DIR ]]; then
  if rm -rf -- "$PREVIOUS_DIR"; then
    PREVIOUS_DIR=""
  else
    echo "Warning: installation succeeded, but the previous application tree remains at $PREVIOUS_DIR." >&2
  fi
fi
if [[ -n $UNIT_BACKUP && -e $UNIT_BACKUP ]]; then
  if rm -f -- "$UNIT_BACKUP"; then
    UNIT_BACKUP=""
  else
    echo "Warning: installation succeeded, but the previous unit backup remains at $UNIT_BACKUP." >&2
  fi
fi
echo "Done. You can access the WebUI at http://127.0.0.1:27003/"

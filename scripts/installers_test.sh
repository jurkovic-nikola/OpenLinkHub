#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
system_installer="$repository_root/install.sh"
user_installer="$repository_root/install-user-space.sh"
system_unit="$repository_root/LumenForge.service"

fail() {
  echo "installer test failed: $*" >&2
  exit 1
}

assert_contains() {
  local file=$1
  local text=$2
  grep -Fq -- "$text" "$file" ||
    fail "$file does not contain required text: $text"
}

assert_absent() {
  local file=$1
  local text=$2
  if grep -Fq -- "$text" "$file"; then
    fail "$file contains forbidden text: $text"
  fi
}

assert_absent_pattern() {
  local file=$1
  local pattern=$2
  if grep -Eq -- "$pattern" "$file"; then
    fail "$file contains forbidden pattern: $pattern"
  fi
}

line_number() {
  local file=$1
  local text=$2
  local line remainder

  while IFS=: read -r line remainder; do
    printf '%s' "$line"
    return 0
  done < <(grep -nF -- "$text" "$file")
  fail "$file does not contain ordered text: $text"
}

assert_order() {
  local file=$1
  local before=$2
  local after=$3
  local before_line after_line

  before_line="$(line_number "$file" "$before")"
  after_line="$(line_number "$file" "$after")"
  ((before_line < after_line)) ||
    fail "$file places '$after' before required predecessor '$before'"
}

for installer in "$system_installer" "$user_installer"; do
  assert_contains "$installer" '[[ ! -L $INSTALL_DIR ]]'
  assert_contains "$installer" 'mktemp -d /opt/.LumenForge.stage.XXXXXX'
  assert_contains "$installer" 'chown -R root:root "$STAGING_DIR"'
  assert_contains "$installer" 'find "$STAGING_DIR" -type d -exec chmod 0755'
  assert_contains "$installer" 'find "$STAGING_DIR" -type f -exec chmod 0644'
  assert_contains "$installer" 'database/lcd/background.jpg'
  assert_contains "$installer" 'database/rgb.json'
  assert_absent "$installer" 'WorkingDirectory='
  assert_absent "$installer" 'chown -R "$TARGET_USER'
  assert_absent "$installer" 'chown -R "$RUNTIME_USER'
  assert_absent "$installer" '$STAGING_DIR/database/profiles'
  assert_absent "$installer" '$STAGING_DIR/config.json'
  assert_absent "$installer" '$SOURCE_DIR/install.sh'
done

assert_contains "$system_installer" '[[ $EUID -ne 0 ]]'
assert_absent "$system_installer" 'usermod '
assert_contains "$system_installer" 'validate_runtime_group_entry'
assert_contains "$system_installer" '$entry =~ ^[^:]+:[^:]*:[^:]+:[^:]*$'
assert_contains "$system_installer" 'group_gid -ne 0'
assert_contains "$system_installer" 'group_gid -gt 0'
assert_contains "$system_installer" 'group uses privileged GID 0'
assert_contains "$system_installer" 'group data is incomplete or malformed'
assert_absent "$system_installer" '-z $group_members'
assert_contains "$system_installer" 'passwd_entry="$(getent passwd "$RUNTIME_USER")"'
assert_contains "$system_installer" 'account_uid -ne 0'
assert_contains "$system_installer" 'account_gid -eq "$RUNTIME_GROUP_GID"'
assert_contains "$system_installer" 'account_home == "$STATE_DIR"'
assert_contains "$system_installer" 'account_shell == "$NOLOGIN_SHELL"'
assert_contains "$system_installer" 'Existing $RUNTIME_USER account is not the dedicated LumenForge service identity'
assert_contains "$system_installer" "-M -p '!'"
assert_contains "$system_installer" 'STATE_DIR="/var/lib/lumenforge"'
assert_contains "$system_installer" 'install -d -o "$RUNTIME_USER" -g "$RUNTIME_GROUP" -m 0750 "$STATE_DIR"'
assert_contains "$system_installer" '[[ ! -L $STATE_DIR && -d $STATE_DIR ]]'
assert_contains "$system_installer" 'Refusing symlinked or non-directory state root'
assert_contains "$system_installer" '$(stat -c '\''%U:%G:%a'\'' "$STATE_DIR") == "$RUNTIME_USER:$RUNTIME_GROUP:750"'
assert_contains "$system_installer" 'refusing to normalize service-owned state'
assert_absent_pattern "$system_installer" 'chown[[:space:]]+-R.*STATE_DIR'
assert_absent_pattern "$system_installer" 'chmod[[:space:]]+-R.*STATE_DIR'
assert_absent "$system_installer" '"$STATE_DIR/database/'
assert_contains "$system_installer" 'Environment=LUMENFORGE_SERVICE_MODE=system'
assert_contains "$system_installer" 'Environment=LUMENFORGE_CONFIG_ROOT=/var/lib/lumenforge'
assert_contains "$system_installer" 'Environment=LUMENFORGE_DATA_ROOT=/var/lib/lumenforge'
assert_contains "$system_installer" 'UMask=0077'
assert_contains "$system_installer" 'check_user_service_conflict'
assert_contains "$system_installer" 'systemctl stop "$PRODUCT.service"'
assert_contains "$system_installer" 'Refusing to replace non-regular or symlinked system unit destination'
assert_contains "$system_installer" 'UNIT_TEMP="$(mktemp /etc/systemd/system/.LumenForge.service.write.XXXXXX)"'
assert_contains "$system_installer" 'exec {unit_fd}>"$UNIT_TEMP"'
assert_contains "$system_installer" 'if ! exec {unit_fd}>&-; then'
assert_contains "$system_installer" 'install -o root -g root -m 0644 "$UNIT_TEMP" "$UNIT_READY"'
assert_contains "$system_installer" 'mv -fT -- "$UNIT_READY" "$SYSTEMD_FILE"'
assert_absent "$system_installer" '>"$SYSTEMD_FILE"'
assert_contains "$system_installer" 'mv -- "$PREVIOUS_DIR" "$INSTALL_DIR"'
assert_contains "$system_installer" 'mv -fT -- "$UNIT_BACKUP" "$SYSTEMD_FILE"'
assert_contains "$system_installer" 'systemctl daemon-reload'
assert_contains "$system_installer" 'unable to restore the previously active $PRODUCT.service'
assert_order "$system_installer" 'echo "Starting $PRODUCT system service..."' 'if rm -rf -- "$PREVIOUS_DIR"; then'
assert_order "$system_installer" 'systemctl is-active --quiet "$PRODUCT.service"' 'if [[ -e $STATE_DIR || -L $STATE_DIR ]]; then'
assert_order "$system_installer" 'systemctl stop "$PRODUCT.service"' 'if [[ -e $STATE_DIR || -L $STATE_DIR ]]; then'

assert_contains "$user_installer" '[[ $EUID -eq 0 ]]'
unprivileged_validation="$(
  sed -n '/^for command in readlink /,/^done$/p' "$user_installer"
)"
[[ $unprivileged_validation != *groupadd* ]] ||
  fail "$user_installer requires groupadd through the desktop user's PATH"
[[ $unprivileged_validation != *usermod* ]] ||
  fail "$user_installer requires usermod through the desktop user's PATH"
assert_contains "$user_installer" '"${PRIVILEGED_CMD[@]}" bash -c'
assert_contains "$user_installer" 'for command in bash getent id groupadd usermod udevadm chown install cp find'
assert_contains "$user_installer" 'privileged command $command is required'
assert_order "$user_installer" '"${PRIVILEGED_CMD[@]}" bash -c' 'systemctl --user stop "$PRODUCT.service"'
assert_order "$user_installer" '"${PRIVILEGED_CMD[@]}" bash -c' 'groupadd -r "$DEVICE_GROUP"'
assert_contains "$user_installer" 'validate_device_group_entry'
assert_contains "$user_installer" '$entry =~ ^[^:]+:[^:]*:[^:]+:[^:]*$'
assert_contains "$user_installer" 'group_gid -ne 0'
assert_contains "$user_installer" 'group_gid -gt 0'
assert_contains "$user_installer" 'group uses privileged GID 0'
assert_contains "$user_installer" 'group data is incomplete or malformed'
assert_absent "$user_installer" '-z $group_members'
assert_contains "$user_installer" 'user_was_in_device_group=false'
assert_contains "$user_installer" 'user_was_in_device_group=true'
assert_contains "$user_installer" 'if [[ $user_was_in_device_group == false ]]; then'
assert_order "$user_installer" 'user_was_in_device_group=false' 'if [[ $user_was_in_device_group == false ]]; then'
assert_absent "$user_installer" 'user_in_device_group'
assert_contains "$user_installer" 'target_groups="$(id -nG "$TARGET_USER")"'
assert_contains "$user_installer" 'if [[ " $target_groups " != *" $DEVICE_GROUP "* ]]; then'
assert_order "$user_installer" 'validate_device_group_entry "$group_entry"' 'usermod -aG "$DEVICE_GROUP" "$TARGET_USER"'
assert_order "$user_installer" 'target_groups="$(id -nG "$TARGET_USER")"' 'usermod -aG "$DEVICE_GROUP" "$TARGET_USER"'
assert_contains "$user_installer" 'if [[ $session_has_device_group == true ]]; then'
assert_order "$user_installer" 'if [[ $session_has_device_group == true ]]; then' 'echo "Starting $PRODUCT user service..."'
assert_contains "$user_installer" 'CONFIG_HOME="${XDG_CONFIG_HOME:-$USER_HOME/.config}"'
assert_contains "$user_installer" 'DATA_HOME="${XDG_DATA_HOME:-$USER_HOME/.local/share}"'
assert_contains "$user_installer" '[[ $CONFIG_HOME == /* ]]'
assert_contains "$user_installer" '[[ $DATA_HOME == /* ]]'
assert_contains "$user_installer" 'install -d -m 0700 "$CONFIG_ROOT" "$DATA_ROOT"'
assert_contains "$user_installer" 'Environment=LUMENFORGE_SERVICE_MODE=user'
assert_contains "$user_installer" 'Environment="LUMENFORGE_CONFIG_ROOT=$escaped_config_root"'
assert_contains "$user_installer" 'Environment="LUMENFORGE_DATA_ROOT=$escaped_data_root"'
assert_contains "$user_installer" 'UMask=0077'
assert_contains "$user_installer" 'systemctl is-active --quiet "$PRODUCT.service"'
assert_contains "$user_installer" 'systemctl --user stop "$PRODUCT.service"'
assert_contains "$user_installer" 'GROUP="lumenforge"'
assert_contains "$user_installer" 'Refusing to replace non-regular or symlinked user unit destination'
assert_contains "$user_installer" 'UNIT_TEMP="$(mktemp "$SYSTEMD_DIR/.LumenForge.service.write.XXXXXX")"'
assert_contains "$user_installer" 'exec {unit_fd}>"$UNIT_TEMP"'
assert_contains "$user_installer" 'if ! exec {unit_fd}>&-; then'
assert_contains "$user_installer" 'install -m 0600 "$UNIT_TEMP" "$UNIT_READY"'
assert_contains "$user_installer" '$(stat -c '\''%u:%a'\'' "$UNIT_READY") == "$TARGET_UID:600"'
assert_contains "$user_installer" 'mv -fT -- "$UNIT_READY" "$SYSTEMD_FILE"'
assert_absent "$user_installer" '>"$SYSTEMD_FILE"'
assert_contains "$user_installer" 'rollback_application_tree'
assert_contains "$user_installer" 'mv -- "$PREVIOUS_DIR" "$INSTALL_DIR"'
assert_contains "$user_installer" 'mv -fT -- "$UNIT_BACKUP" "$SYSTEMD_FILE"'
assert_contains "$user_installer" 'systemctl --user daemon-reload'
assert_contains "$user_installer" 'unable to restore the previously active user service'
assert_order "$user_installer" 'systemctl --user enable "$PRODUCT.service"' '"${PRIVILEGED_CMD[@]}" rm -rf -- "$PREVIOUS_DIR" "$SWAP_MARKER"'
assert_order "$user_installer" 'echo "Starting $PRODUCT user service..."' '"${PRIVILEGED_CMD[@]}" rm -rf -- "$PREVIOUS_DIR" "$SWAP_MARKER"'

assert_contains "$system_unit" 'Environment=LUMENFORGE_SERVICE_MODE=system'
assert_contains "$system_unit" 'Environment=LUMENFORGE_APPLICATION_ROOT=/opt/LumenForge'
assert_contains "$system_unit" 'Environment=LUMENFORGE_CONFIG_ROOT=/var/lib/lumenforge'
assert_contains "$system_unit" 'Environment=LUMENFORGE_DATA_ROOT=/var/lib/lumenforge'
assert_contains "$system_unit" 'UMask=0077'
assert_absent "$system_unit" 'WorkingDirectory='

replace_unit_for_test() {
  local source=$1
  local destination=$2
  local mode=$3
  local directory temporary ready

  if [[ -e $destination || -L $destination ]]; then
    [[ ! -L $destination && -f $destination ]] || return 1
  fi

  directory="$(dirname -- "$destination")"
  temporary="$(mktemp "$directory/.unit.write.XXXXXX")"
  ready="$(mktemp "$directory/.unit.ready.XXXXXX")"
  if ! cp -- "$source" "$temporary" ||
    ! install -m "$mode" "$temporary" "$ready" ||
    ! mv -fT -- "$ready" "$destination"; then
    rm -f -- "$temporary" "$ready"
    return 1
  fi
  rm -f -- "$temporary"
}

unit_test_root="$(mktemp -d "${TMPDIR:-/tmp}/lumenforge-unit-test.XXXXXX")"
trap 'rm -rf -- "$unit_test_root"' EXIT
source_unit="$unit_test_root/source.service"
destination_unit="$unit_test_root/LumenForge.service"
printf '[Service]\nExecStart=/bin/true\n' >"$source_unit"

replace_unit_for_test "$source_unit" "$destination_unit" 0644 ||
  fail "temporary-directory unit replacement failed"
[[ $(stat -c '%a' "$destination_unit") == 644 ]] ||
  fail "normal unit replacement did not produce mode 0644"

symlink_target="$unit_test_root/symlink-target"
printf 'unchanged\n' >"$symlink_target"
rm -f -- "$destination_unit"
ln -s -- "$symlink_target" "$destination_unit"
if replace_unit_for_test "$source_unit" "$destination_unit" 0600; then
  fail "temporary-directory unit replacement accepted a symlink destination"
fi
[[ $(<"$symlink_target") == unchanged ]] ||
  fail "symlink rejection changed the symlink target"

rm -f -- "$destination_unit"
printf 'unsafe\n' >"$destination_unit"
chmod 0666 "$destination_unit"
replace_unit_for_test "$source_unit" "$destination_unit" 0600 ||
  fail "replacement of an unsafe-mode regular unit failed"
[[ $(stat -c '%a' "$destination_unit") == 600 ]] ||
  fail "replacement did not correct an unsafe pre-existing mode to 0600"

echo "installer static checks passed"

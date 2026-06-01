#!/usr/bin/env bash
set -euo pipefail

REPO_API="https://api.github.com/repos/jurkovic-nikola/OpenLinkHub/releases/latest"
PRODUCT="OpenLinkHub"
SERVICE_NAME="${PRODUCT}.service"
PERMISSION_FILE="99-openlinkhub.rules"
PERMISSION_DIR="/etc/udev/rules.d"
SEARCH_FOR="openlinkhub"
SEARCH_FOR_GROUP='OWNER="openlinkhub"'
REPLACE_WITH='GROUP="openlinkhub"'
USER_TO_CHECK="${SUDO_USER:-$USER}"
USER_HOME="$(getent passwd "$USER_TO_CHECK" | cut -d: -f6)"
SYSTEMD_DIR="$USER_HOME/.config/systemd/user"
SYSTEMD_FILE="$SYSTEMD_DIR/$SERVICE_NAME"
#PRIVILEGED_CMD="sudo"
TMP_DIR="$(mktemp -d)"

if [[ -t 1 ]]; then
  BOLD=$'\033[1m'
  DIM=$'\033[2m'
  RED=$'\033[31m'
  GREEN=$'\033[32m'
  YELLOW=$'\033[33m'
  BLUE=$'\033[34m'
  MAGENTA=$'\033[35m'
  CYAN=$'\033[36m'
  RESET=$'\033[0m'
else
  BOLD=""
  DIM=""
  RED=""
  GREEN=""
  YELLOW=""
  BLUE=""
  MAGENTA=""
  CYAN=""
  RESET=""
fi

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

section() {
  echo
  printf "${BOLD}${CYAN}>${RESET} ${BOLD}%s${RESET}\n" "$1"
}

info() {
  printf "${BLUE}[INFO]${RESET} %s\n" "$1"
}

success() {
  printf "${GREEN}[OK]${RESET} %s\n" "$1"
}

warn() {
  printf "${YELLOW}[WARN]${RESET} %s\n" "$1"
}

error() {
  printf "${RED}[ERROR]${RESET} %s\n" "$1" >&2
}

run_step() {
  local label="$1"
  shift

  printf "${MAGENTA}>${RESET} %s... " "$label"

  if "$@" >/dev/null 2>&1; then
    printf "${GREEN}done${RESET}\n"
  else
    printf "${RED}failed${RESET}\n"
    return 1
  fi
}

progress_bar() {
  local current="$1"
  local total="$2"
  local width=30

  local percent=$(( current * 100 / total ))
  local filled=$(( current * width / total ))
  local empty=$(( width - filled ))

  printf "\r["
  printf "%${filled}s" "" | tr ' ' '#'
  printf "%${empty}s" "" | tr ' ' '-'
  printf "] %3d%%" "$percent"
}

extract_with_progress() {
  local archive="$1"
  local destination="$2"

  local total
  total="$(tar -tzf "$archive" | wc -l | tr -d ' ')"

  if [[ "$total" -eq 0 ]]; then
    tar -xzf "$archive" -C "$destination"
    return
  fi

  local count=0

  while IFS= read -r file; do
    count=$((count + 1))
    progress_bar "$count" "$total"
  done < <(tar -xvzf "$archive" -C "$destination" 2>/dev/null)

  echo
}

section "Fetching latest release"

TAR_URL="$(
  curl -fsSL "$REPO_API" \
    | grep -Eo '"browser_download_url":\s*"[^"]+\.tar\.gz"' \
    | sed -E 's/"browser_download_url":\s*"([^"]+)"/\1/' \
    | head -n 1
)"

if [[ -z "${TAR_URL:-}" ]]; then
  error "Could not find a .tar.gz asset in the latest release."
  exit 1
fi

success "Found latest release archive"
info "$TAR_URL"

ARCHIVE="$TMP_DIR/openlinkhub.tar.gz"

section "Downloading archive"

curl -fL --progress-bar "$TAR_URL" -o "$ARCHIVE"
success "Download complete"

section "Checking systemd service"

SERVICE_EXISTS=false
if systemctl --user status "$SERVICE_NAME" >/dev/null 2>&1 || [[ -f "$SYSTEMD_FILE" ]]; then
  SERVICE_EXISTS=true
fi

if [[ "$SERVICE_EXISTS" == true ]]; then
  warn "Existing user service found. Performing upgrade..."
  run_step "Stopping $SERVICE_NAME" systemctl --user stop "$SERVICE_NAME" || true
else
  info "No existing user service found. Treating this as first install."
fi

section "Extracting files"

info "Extracting to $USER_HOME"
extract_with_progress "$ARCHIVE" "$USER_HOME"
success "Extraction complete"

CURRENT_DIR="$(
  find "$USER_HOME" -maxdepth 2 -type f -name "$PRODUCT" -executable \
    -printf '%h\n' \
    | head -n 1
)"

if [[ -z "${CURRENT_DIR:-}" ]]; then
  error "Could not find executable '$PRODUCT' after extraction."
  exit 1
fi

success "Application directory: $CURRENT_DIR"

if [[ ! -f "$CURRENT_DIR/$PERMISSION_FILE" ]]; then
  error "Could not find $PERMISSION_FILE in $CURRENT_DIR."
  exit 1
fi

GROUP_EXISTS=false
if getent group "$SEARCH_FOR" >/dev/null; then
  GROUP_EXISTS=true
fi

USER_IN_GROUP=false
if id -nG "$USER_TO_CHECK" | grep -qw "$SEARCH_FOR"; then
  USER_IN_GROUP=true
fi

section "Installing permissions and udev rules"
info "Processing $CURRENT_DIR/$PERMISSION_FILE"
sed -i -e "s/$SEARCH_FOR_GROUP/$REPLACE_WITH/g" "$CURRENT_DIR/$PERMISSION_FILE"

# Check if sudo is available
if command -v sudo &>/dev/null; then
    PRIVILEGED_CMD="sudo"
else
    echo "sudo not found. Falling back to run0"
    PRIVILEGED_CMD="run0 -i"
fi

$PRIVILEGED_CMD bash <<EOF
set -e

if [ "$GROUP_EXISTS" = false ]; then
    for gid in \$(seq 999 -1 900); do
        if ! getent group "\$gid" >/dev/null; then
            groupadd -g "\$gid" "$SEARCH_FOR"
            echo "Created group '$SEARCH_FOR' with GID \$gid"
            break
        fi
    done
fi

if [ "$USER_IN_GROUP" = false ]; then
    usermod -aG "$SEARCH_FOR" "$USER_TO_CHECK"
    echo "Added $USER_TO_CHECK to $SEARCH_FOR"
fi

cp "$CURRENT_DIR/$PERMISSION_FILE" "$PERMISSION_DIR/$PERMISSION_FILE"
chmod 644 "$PERMISSION_DIR/$PERMISSION_FILE"

udevadm control --reload-rules
udevadm trigger

chown -R "$USER_TO_CHECK":"$USER_TO_CHECK" "$CURRENT_DIR"
chmod -R 755 "$CURRENT_DIR"
EOF

success "Permissions and udev rules installed"

section "Configuring user systemd service"

if [[ "$SERVICE_EXISTS" == false ]]; then
  info "Creating user systemd file"

  mkdir -p "$SYSTEMD_DIR"

  cat > "$SYSTEMD_FILE" <<-EOM
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

  if [[ -f "$SYSTEMD_FILE" ]]; then
    success "User systemd file created"
    run_step "Reloading user systemd daemon" systemctl --user daemon-reload
    run_step "Enabling $SERVICE_NAME" systemctl --user enable "$SERVICE_NAME"
  fi
else
  run_step "Reloading user systemd daemon" systemctl --user daemon-reload
fi

section "Starting service"

run_step "Starting $SERVICE_NAME" systemctl --user start "$SERVICE_NAME"

echo
printf "${BOLD}${GREEN}Installation complete. You can access WebUI on: http://127.0.0.1:27003/${RESET}\n"

if [[ "$USER_IN_GROUP" == false ]]; then
  echo
  warn "$USER_TO_CHECK was added to group '$SEARCH_FOR'."
  warn "You need to log out and back in for group membership to apply, or reboot your machine."
fi

echo

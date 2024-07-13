# OpenLinkHub interface for Linux
Open source Linux interface for iCUE LINK Hub and other devices.

![Build](https://github.com/jurkovic-nikola/OpenLinkHub/actions/workflows/go.yml/badge.svg)

![Web UI](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/ui.png?raw=true)

# Info
- This project was created out of own necessity to control fans and pumps on workstations after switching everything to Linux.
- I take no responsibility for this code at all. Use this at your own risk. 
- Most of the devices are actually tested on live hardware.
- OpenLinkHub supports multiple devices.
- Take care and have fun!
## Supported devices

| Device                  | VID    | PID    | Sub Devices                                                                                                                                                            |
|-------------------------|--------|--------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| iCUE LINK System Hub    | `1b1c` | `0c3f` | QX Fan<br />RX Fan<br/>RX RGB Fan<br/>RX MAX Fan<br/>H100i<br/>H115i<br/>H150i<br/>H170i<br/>XC7 Elite<br/>XG7<br/>XD5 Elite<br/>XD5 Elite LCD <br/>VRM Cooling Module | |                                                                                                                                                                   |


## Installation
### 1. Requirements
- libudev-dev
- lm-sensors
- go 1.22.2 - https://go.dev/dl/
```bash
# Required packages (deb)
$ sudo apt-get install libudev-dev
$ sudo apt-get install lm-sensors
$ sudo sensors-detect

# Required packages (rpm)
$ sudo dnf install libudev-devel
$ sudo dnf install lm_sensors
$ sudo sensors-detect
```
### 2. Build
```bash
# Build
git clone https://github.com/jurkovic-nikola/OpenLinkHub.git
cd OpenLinkHub/
go build .
```
### 3. Installation
```bash
# Installation
# Make root directory
sudo mkdir /opt/OpenLinkHub

# Copy binary to a root directory
sudo cp OpenLinkHub /opt/OpenLinkHub

# Copy service configuration to a root directory
sudo cp config.json /opt/OpenLinkHub

# Copy database/ folder to to a root directory
sudo cp -r database/ /opt/OpenLinkHub

# Copy static/ folder to to a root directory
sudo cp -r static/ /opt/OpenLinkHub/

# Copy web/ folder to to a root directory
sudo cp -r web/ /opt/OpenLinkHub/

# Give your username permission to a root folder and everything bellow
# Replace your-username with your login name
sudo chown -R your-username:root /opt/OpenLinkHub/

# Allow hidraw communication as non-root
echo "KERNEL==\"hidraw*\", SUBSYSTEMS==\"usb\", ATTRS{idVendor}==\"1b1c\", ATTRS{idProduct}==\"0c3f\", MODE=\"0666\"" | sudo tee /etc/udev/rules.d/99-corsair-icuelink.rules

# Reload udev rules without reboot
sudo udevadm control --reload-rules && sudo udevadm trigger
```

### 4. Installation from compiled build
```bash
# Installation

# Make main folder
mkdir OpenLinkHub && cd OpenLinkHub

# Download latest build from https://github.com/jurkovic-nikola/OpenLinkHub/releases
wget https://github.com/jurkovic-nikola/OpenLinkHub/releases/download/0.0.3-beta/0.0.3-beta.zip

# Extract package
unzip -x 0.0.3-beta.zip

# Continue from 3. Installation section for next steps
```

### 5. Configuration
```json
{
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "k10temp-pci-00c3",
  "cpuPackageIdent": "Tctl",
  "manual": false,
  "frontend": true
}
```
- listenPort: HTTP server port.
- listenAddress: Address for HTTP server to listen on.
- cpuSensorChip: CPU sensor chip for temperature
- manual: set to true if you want to use your own UI for device control. Setting this to true will disable temperature monitoring and automatic device speed adjustments. 
- frontend: set to false if you do not need WebUI console, and you are making your own UI app. 
```bash
$ sensors
k10temp-pci-00c3
Adapter: PCI adapter
Tctl:         +35.4°C  
Tccd1:        +33.8°C  
Tccd2:        +31.2°C
```
- cpuPackageIdent: CPU package key (see sensors above)

## Device Dashboard
- Simple Device Dashboard is accessible by browser via link `http://127.0.0.1:27003/`
- Device Dashboard:
  - System Overview
  - Device Overview:
    - Change temperature profile per channel
    - Change RGB mode per device
  - RGB Overview:
    - Overview of existing RGB modes
  - Temperature Overview:
    - Create new profiles
    - Update existing profiles
    - Delete existing profile
  - General dashboard settings
- You should use Device Dashboard for any configuration
- Bootstrap 5 Dark Admin template

## RGB Modes
- RGB configuration is located at `database/rgb.json` file.
### Configuration
- profiles: Custom RGB mode data
  - key: RGB profile name
    - speed: RGB effect speed, from 1 to 10
    - brightness: Color brightness, from 0.1 to 1
    - smoothness: How smooth transition from one color to another is.
      - the smoothness is in range of 1 to 40
    - start: Custom starting color in (R, G, B, brightness format)
    - end: Custom ending color  (R, G, B, brightness format)
      - If you want random colors, remove data from start and end JSON block. `"start":{}` and `"end":{}`

## API
- OpenLinkHub ships with built-in HTTP server for device overview and control.
- Documentation is available at `http://127.0.0.1:27003/docs`
## Automatic startup
### systemd config
- This is a plain basic systemd example to get you running. You can create systemd service how you like. 
```
[Unit]
Description=Open source interface for iCUE LINK System Hub

[Service]
User=your-user
WorkingDirectory=/opt/OpenLinkHub
ExecStart=/opt/OpenLinkHub/OpenLinkHub
ExecReload=/bin/kill -s HUP $MAINPID
RestartSec=5

[Install]
WantedBy=multi-user.target
```
- Modify User= to match user who will run this
- Modify WorkingDirectory= to match where executable is
- Modify ExecStart= to match an executable path
- Save content to `/etc/systemd/system/OpenLinkHub.service` (deb based)
- Save content to `/usr/lib/systemd/system/OpenLinkHub.service` (rpm based)
- Run `systemctl enable --now OpenLinkHub`

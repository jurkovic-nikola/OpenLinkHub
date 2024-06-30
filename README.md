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

| Device                 | VID    | PID    | Sub Devices                                                                                                                                                                                                                                                                      |
|------------------------|--------|--------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| iCUE LINK System Hub   | `1b1c` | `0c3f` | QX Fan<br />RX Fan<br/>RX RGB Fan<br/>H100i<br/>H115i<br/>H150i<br/>H170i<br/>XC7 Elite<br/>XG7<br/>XD5 Elite<br/>XD5 Elite LCD                                                                                                                                                  | |

## Installation
### Requirements
- libudev-dev
- lm-sensors
- go 1.22.2 - https://go.dev/dl/
```bash
# Required packages
$ sudo apt-get install libudev-dev
$ sudo apt-get install lm-sensors
$ sudo sensors-detect
```
### Build
```bash
# Build
git clone https://github.com/jurkovic-nikola/OpenLinkHub.git
cd OpenLinkHub/
go build .
```
### Device permissions
To read/write to an HID device, you either need to run the application as root or define permission via custom udev files.
```bash
echo "KERNEL==\"hidraw*\", SUBSYSTEMS==\"usb\", ATTRS{idVendor}==\"1b1c\", ATTRS{idProduct}==\"0c3f\", MODE=\"0666\"" | sudo tee /etc/udev/rules.d/99-corsair-icuelink.rules

# Reload rules without reboot
sudo udevadm control --reload-rules && sudo udevadm trigger
```
## Configuration
```json
{
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "k10temp-pci-00c3",
  "cpuPackageIdent": "Tctl"
}
```
- listenPort: HTTP server port.
- listenAddress: Address for HTTP server to listen on.
- cpuSensorChip: CPU sensor chip for temperature
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
    - Update existing RGB modes
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
- defaultColor: Default RGB color for all devices in integer format
  - 255,255,255, 0.1 - White with 10 % of brightness
  - 255,255,255, 0.5 - White with 50 % of brightness
  - 255,255,255, 1 - White with 100 % of brightness
  - Note: Setting brightness to 0 will result in no color on a device
  - This mode will be ignored if rgbMode is defined
- rgbMode: This will enable custom RGB mode for all devices.
  - If this mode is enabled, REST API color modification is not possible.
- rgbModes: Custom RGB mode data
  - key: RGB mode name from `rgbMode`
    - speed: RGB effect speed, from 1 to 10
    - brightness: Color brightness, from 0.1 to 1
    - smoothness: How smooth transition from one color to another is.
      - the smoothness is in range of 1 to 40
    - start: Custom starting color in (R, G, B, brightness format)
    - end: Custom ending color  (R, G, B, brightness format)
      - If you want random colors, remove data from start and end JSON block. `"start":{}` and `"end":{}`
### List of supported RGB modes:
  - rainbow
  - watercolor
  - colorshift
  - colorpulse
  - circle
  - circleshift
  - flickering
  - colorwarp
  - snipper
- `rainbow` - Self explanatory, rainbow colors
- `watercolor` - Self explanatory, water colors
- `colorshift` - Shifts color from field `start` to `end`
- `colorpulse` - Self explanatory, color pulse
- `circle` - Cycles color for each device until an end of device list, then repeat
- `circleshift` - Cycles color for each device until an end of device list with second color shift, then repeat
- `flickering` - Will randomly turn LEDs off per cycle
- `colorwarp` - Will warp two colors on every device
- `snipper` - Single LED spinning around all devices

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
Group=your-user-group
WorkingDirectory=/path-tp-executable-directory/
ExecStart=/path-tp-executable-directory/OpenLinkHub
ExecReload=/bin/kill -s HUP $MAINPID
RestartSec=5

[Install]
WantedBy=multi-user.target
```
- Modify User= to match user who will run this
- Modify Group= to match a group from udev rule file
- Modify WorkingDirectory= to match where executable is
- Modify ExecStart= to match an executable path
- Save content to `/etc/systemd/system/OpenLinkHub.service`
- Run `systemctl enable --now OpenLinkHub`
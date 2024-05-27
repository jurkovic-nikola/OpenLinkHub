# OpenICUELinkHub interface for Linux
Open source interface for iCUE LINK Hub device and Linux with built-in API for overview and device control.

![Build](https://github.com/jurkovic-nikola/OpenICUELinkHub/actions/workflows/go.yml/badge.svg)

# Info
- This project was created out of own necessity to control fans and pumps on workstations after switching everything to Linux.
- I take no responsibility for this code at all. Use this at your own risk. 
- Most of the devices are actually tested on live hardware.
- Take care and have fun!
## Supported hubs

| Device        | VID    | PID    |
|---------------|--------|--------|
| iCUE LINK Hub | `1b1c` | `0c3f` |

## Operating Modes
### Software operating mode
OpenCUELink supports two operating modes:
- Standalone mode will automatically monitor your CPU temperature and adjust fan and pump speeds based on the defined temperature curve.
- Manual mode will allow you to use the REST to control devices. 
  - This mode can be used to integrate this into your custom UI application. 
  - You will have full control over fans and pumps.

### Fan operating modes
- Fans support 2 different operating modes:
  - Mode 0: Speed is controlled via percentage (from 20-100).
  - Mode 1: Speed is controlled via RPM (from 400-2400+).
### Pump operating modes:
- Pump speed can be controlled only via percentage.

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
git clone https://github.com/jurkovic-nikola/OpenICUELinkHub.git
cd OpenICUELinkHub/
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
  "vendorId": "1b1c",
  "productId": "0c3f",
  "standalone": true,
  "listenPort": 27003,
  "defaultFanValue": 50,
  "defaultPumpValue": 80,
  "listenAddress": "127.0.0.1",
  "pullingIntervalMs": 1000,
  "serial": "5C126A3EB51A39569ABADC4C3A1FCF54",
  "headers": [
    { "key":"Content-Type", "value": "application/json" }
  ],
  "cpuSensorChip": "k10temp-pci-00c3",
  "cpuPackageIdent": "Tctl",
  "templateList": "overview.html"
}
```
- vendorId: ID of device vendor
- productId: ID of a device
- standalone: The program will independently monitor CPU temperature and adjust fan and pump speed based on temperatureCurve array.
- listenPort: HTTP server port.
- defaultFanChannelSpeed: Default percentage of fan speed when the application is running.
- defaultPumpChannelSpeed: Default percentage of pump speed when the application is running.
- listenAddress: Address for HTTP server to listen on.
- pullingIntervalMs: How often data is pulled from a device (rpm, temperature) in milliseconds
- serial: Device serial number (iSerial bellow)
```bash
$ lsusb -d 1b1c:0c3f -v

Bus 003 Device 007: ID 1b1c:0c3f Corsair iCUE LINK System Hub
Device Descriptor:
  bLength                18
  bDescriptorType         1
  bcdUSB               2.00
  bDeviceClass            0 
  bDeviceSubClass         0 
  bDeviceProtocol         0 
  bMaxPacketSize0        64
  idVendor           0x1b1c Corsair
  idProduct          0x0c3f 
  bcdDevice            1.00
  iManufacturer           1 Corsair
  iProduct                2 iCUE LINK System Hub
  iSerial                 3 2EFBD3CFCADF4050A3557CBC5159A599
```
- headers: Default http headers
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
- templateList: HTML templates
## temperatures.json
- temperaturePullingIntervalMs: How ofter temperature is read from CPU sensor (in milliseconds)
- temperatureCurve: Array of temperature curves for fan and pump speed.
  - min / max: Range of CPU temperature
  - mode: 
    - 0: Percent
    - 1: RPM
  - fans: fan speed
  - pump: pump speed
  - channelIds: list of channel IDs for which a temperature curve applies. 
    - This is useful to ramp up other fans when a certain temperature is reached. If left empty, speed setting will be applied to all channels by default. 
    - You can use this ability to create positive pressure in case and reduce dust. 
    - For AIO, usually fans and pump
    - For custom loop, Pump/Res combo and fans
  - color: RGB color for specified temperature range. 
    - This is useful when the temperature reaches critical level on your system.
    - Making color array empty will disable RGB changes via temperature level.
    - It Can be used to modify your device color per temperature range
## custom.json
- useCustomChannelIdColor: if set to true, default color will be ignored, and you will be able to define color for each device
  - config example contains multiple channelIds, aka "1", "13", "14" and "15"
  - those IDs are from `curl http://127.0.0.1:27003 --silent | jq` under devices section.
  - Each ID is channelId connected to a iCUE Link Hub
  - This mode will be ignored if rgbMode is defined
- useCustomChannelIdSpeed: is set to true, each device will have a static speed defined in config. 
  - In this mode, a standalone flag is ignored and CPU monitoring is not enabled.
### How to identify channels?
- In most cases, your channel will have a name
- In cases of FANs, the best way to identify a channel is to set custom color on each channel. 
## RGB Modes
- RGB configuration is located at `rgb.json` file. 
### Configuration
- defaultColor: Default RGB color for all devices in integer format
  - 255,255,255, 0.1 - White with 10 % of brightness
  - 255,255,255, 0.5 - White with 50 % of brightness
  - 255,255,255, 1 - White with 100 % of brightness
  - Note: Setting brightness to 0 will result in no color on a device
  - This mode will be ignored if rgbMode is defined
- useRgbEffects: Trigger usage of custom RGB effects
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
  - heartbeat
- `rainbow` - Self explanatory, rainbow colors
- `watercolor` - Self explanatory, water colors
- `colorshift` - Shifts color from field `start` to `end`
- `colorpulse` - Self explanatory, color pulse
- `circle` - Cycles color for each device until an end of device list, then repeat
- `circleshift` - Cycles color for each device until an end of device list with second color shift, then repeat
- `flickering` - Will randomly turn LEDs off per cycle
- `colorwarp` - Will warp two colors on every device
- `snipper` - Single LED spinning around all devices
- `heartbeat` - Heartbeat

## Device Dashboard
- Simple Device Dashboard is accessible by browser via link `http://127.0.0.1:27003/ui`
- Device Dashboard is currently WIP (basic system and device overview)
- Bootstrap 5 Dark Admin template
## API
- OpenICUELinkHub ships with built-in HTTP server for device overview and control.
### Overview
```bash
$ curl http://127.0.0.1:27003 --silent | jq
{
  "code": 200,
  "device": {
    "Manufacturer": "Corsair",
    "Product": "iCUE LINK System Hub",
    "Serial": "2EFBD3CFCADF4050A3557CBC5159A599",
    "Firmware": "2.4.438",
    "devices": {
      "1": {
        "channelId": 1,
        "deviceId": "0100282F8203582BFB00018643",
        "name": "QX Fan",
        "rpm": 962,
        "temperature": 27.8
      },
      "2": {
        "channelId": 2,
        "deviceId": "0100136032035898E90000555B",
        "name": "QX Fan",
        "rpm": 967,
        "temperature": 28.9
      },
      "3": {
        "channelId": 3,
        "deviceId": "0292D70F00AEA4610336A80000",
        "name": "H150i",
        "rpm": 1564,
        "temperature": 27.9
      },
      "4": {
        "channelId": 4,
        "deviceId": "0100136032035898E900007609",
        "name": "QX Fan",
        "rpm": 966,
        "temperature": 29.3
      }
    }
  }
}
```
### List of devices
```bash
$ curl http://127.0.0.1:27003/devices --silent | jq
{
  "code": 200,
  "devices": {
    "1": {
      "channelId": 1,
      "deviceId": "0100282F8203582BFB00018643",
      "name": "QX Fan",
      "rpm": 1207,
      "temperature": 28.1
    },
    "2": {
      "channelId": 2,
      "deviceId": "0100136032035898E90000555B",
      "name": "QX Fan",
      "rpm": 1213,
      "temperature": 29
    },
    "3": {
      "channelId": 3,
      "deviceId": "0292D70F00AEA4610336A80000",
      "name": "H150i",
      "rpm": 1908,
      "temperature": 27.9
    },
    "4": {
      "channelId": 4,
      "deviceId": "0100136032035898E900007609",
      "name": "QX Fan",
      "rpm": 1210,
      "temperature": 29.4
    }
  }
}
```
### Modify device speed by percent
```bash
$ curl -X POST http://127.0.0.1:27003/speed -d '{"channelId":1,"mode":0,"value":80}' --silent | jq
{
  "code": 200,
  "message": "Device speed successfully changed"
}
```
### Modify device speed by RPM
```bash
$ curl -X POST http://127.0.0.1:27003/speed -d '{"channelId":1,"mode":1,"value":1800}' --silent | jq
{
  "code": 200,
  "message": "Device speed successfully changed"
}
```
### Modify device color - specific channel
```bash
$ curl -X POST http://127.0.0.1:27003/color -d '{"channelId":1,"color": {"red": 255, "green": 0, "blue": 0, "brightness": 1}}' --silent | jq
{
  "code": 200,
  "message": "Device color successfully changed"
}
```
### Modify device color - all channels
```bash
$ curl -X POST http://127.0.0.1:27003/color -d '{"channelId":0,"color": {"red": 255, "green": 0, "blue": 0, "brightness": 1}}' --silent | jq
{
  "code": 200,
  "message": "Device color successfully changed"
}
```

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
ExecStart=/path-tp-executable-directory/OpenICUELinkHub
ExecReload=/bin/kill -s HUP $MAINPID
RestartSec=5

[Install]
WantedBy=multi-user.target
```
- Modify User= to match user who will run this
- Modify Group= to match a group from udev rule file
- Modify WorkingDirectory= to match where executable is
- Modify ExecStart= to match an executable path
- Save content to `/etc/systemd/system/OpenICUELinkHub.service`
- Run `systemctl enable --now OpenICUELinkHub`
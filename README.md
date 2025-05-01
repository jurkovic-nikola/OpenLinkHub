# OpenLinkHub interface for Linux
An open-source Linux interface for iCUE LINK Hub and other Corsair AIOs, Hubs.
Manage RGB lighting, fan speeds, system metrics, as well as keyboards, mice, headsets via a web dashboard.

![Build](https://github.com/jurkovic-nikola/OpenLinkHub/actions/workflows/go.yml/badge.svg)
[![](https://dcbadge.limes.pink/api/server/https://discord.gg/mPHcasZRPy?style=flat)](https://discord.gg/mPHcasZRPy)

## Features

- Web-based UI accessible at `http://localhost:27003`
- Control AIO coolers, fans, hubs, pumps, LCDs and RGB lighting
- Manage keyboards, mice and headsets
- Support for DDR4 and DDR5 memory
- Custom fan profiles, and temperature sensors and RGB editor

![Web UI](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/ui.png?raw=true)

# Info
- This project was created out of own necessity to control fans and pumps on workstations after switching everything to Linux.
- I take no responsibility for this code at all. Use this at your own risk. 
- Most of the devices are actually tested on live hardware.
- Take care and have fun!
- This project does not accept any donations.
- This project is not an official Corsair product.

## Supported devices
| Device                        | PID                            | Sub Devices                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
|-------------------------------|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| iCUE LINK System Hub          | `0c3f`                         | <details><summary>Show</summary>iCUE LINK QX RGB<br />iCUE LINK RX<br/>iCUE LINK RX RGB<br/>iCUE LINK RX MAX<br/>iCUE LINK RX MAX RGB<br/>iCUE LINK LX RGB<br />iCUE LINK H100i<br/>iCUE LINK H115i<br/>iCUE LINK H150i<br/>iCUE LINK H170i<br/>XC7 Elite<br/>XG7<br/>XD5 Elite<br/>XD5 Elite LCD <br/>VRM Cooling Module<br />iCUE LINK TITAN H100i<br />iCUE LINK TITAN H150i<br />iCUE LINK TITAN H115i<br />iCUE LINK TITAN H170i<br />LCD Pump Cover<br />iCUE LINK XG3 HYBRID<br />iCUE LINK ADAPTER<br />iCUE LINK LS350 Aurora RGB<br />iCUE LINK LS430 Aurora RGB</details> |
| iCUE COMMANDER Core           | `0c32`<br />`0c1c`             | <details><summary>Show</summary>H100i ELITE CAPELLIX<br />H115i ELITE CAPELLIX<br />H150i ELITE CAPELLIX<br />H170i ELITE CAPELLIX<br />H100i ELITE LCD<br />H150i ELITE LCD<br />H170i ELITE LCD<br />H100i ELITE LCD XT<br />H115i ELITE LCD XT<br />H150i ELITE LCD XT<br />H170i ELITE LCD XT<br />H100i ELITE CAPELLIX XT<br />H115i ELITE CAPELLIX XT<br />H150i ELITE CAPELLIX XT<br />H170i ELITE CAPELLIX XT<br />1x Temperature Probe<br /> 4-LED RGB Fan<br /> 8-LED RGB Fan<br /> QL Fan Series<br /> LL Fan Series<br /> ML Fan Series<br />Any PWM Fan</details>       |
| iCUE COMMANDER Core XT        | `0c2a`                         | <details><summary>Show</summary>External RGB Hub<br />2x Temperature Probe<br /> 4-LED RGB Fan<br /> 8-LED RGB Fan<br /> QL Fan Series<br /> LL Fan Series<br /> ML Fan Series<br />Any PWM Fan<br />H55 RGB AIO<br />H100 RGB AIO<br />H150 RGB AIO<br />HYDRO X - XD7 Pump / Res Combo<br />HYDRO X - CX7 / XC9 CPU Block<br />HYDRO X - XC5 PRO / XC8 PRO CPU Block<br />HYDRO X - XD5 Pump / Res Combo<br />HYDRO X - GPU Block<br />HYDRO X - XD3 Pump / Res Combo</details>                                                                                                    |
| iCUE H100i RGB ELITE          | `0c35`<br />`0c40`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H115i RGB ELITE          | `0c36`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H150i RGB ELITE          | `0c37`<br />`0c41`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H100i RGB PRO XT         | `0c20`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H115i RGB PRO XT         | `0c21`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H150i RGB PRO XT         | `0c22`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H115i RGB PLATINUM            | `0c17`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H100i RGB PLATINUM            | `0c18`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H100i RGB PLATINUM SE         | `0c19`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H150i PLATINUM                | `0c12`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H115i PLATINUM                | `0c13`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| H100i PLATINUM                | `0c15`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Lighting Node CORE            | `0c1a`                         | <details><summary>Show</summary>HD RGB Series Fan<br />LL RGB Series Fan<br />ML PRO RGB Series Fan<br />QL RGB Series Fan<br />8-LED Series Fan<br />SP RGB Series Fan</details>                                                                                                                                                                                                                                                                                                                                                                                                    |
| Lighting Node PRO             | `0c0b`                         | <details><summary>Show</summary>2x External RGB Hub<br />HD RGB Series Fan<br />LL RGB Series Fan<br />ML PRO RGB Series Fan<br />QL RGB Series Fan<br />8-LED Series Fan<br />SP RGB Series Fan</details>                                                                                                                                                                                                                                                                                                                                                                           |
| Commander PRO                 | `0c10`                         | <details><summary>Show</summary>2x External RGB Hub<br />4x Temperature Probe<br />Any PWM Fan</details>                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| XC7 ELITE LCD CPU Water Block | `0c42`                         | <details><summary>Show</summary>RGB Control<br />LCD Control</details>                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| iCUE NEXUS                    | `1b8e`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB PRO             | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB PRO SL          | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB RT              | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB RS              | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM RGB        | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE LPX                 | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM            | DDR4                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE                     | DDR5                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB                 | DDR5                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM RGB        | DDR5                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR TITANIUM RGB        | DDR5                           |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Slipstream Wireless           | `1bdc`<br />`1ba6`<br />`2b00` | <details><summary>Show</summary>K100 AIR RGB<br />IRONCLAW RGB WIRELESS<br />NIGHTSABRE WIRELESS<br />SCIMITAR RGB ELITE WIRELESS<br />SCIMITAR ELITE WIRELESS SE<br />M55 WIRELESS<br />DARK CORE RGB PRO SE WIRELESS<br />DARK CORE RGB PRO<br />M75 AIR WIRELESS<br />HARPOON RGB WIRELESS<br />DARKSTAR WIRELESS<br />K70 CORE TKL WIRELESS<br />M65 RGB ULTRA WIRELESS</details>                                                                                                                                                                                                |
| K55 CORE RGB                  | `1bfe`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K55 PRO RGB                   | `1ba4`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K65 PRO MINI                  | `1bd7`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K70 CORE RGB                  | `1bfd`                         |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K70 PRO RGB                   | `1bc6`<br />`1bb3`<br />`1bd4` |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K65 PLUS                      | `2b10`<br />`2b07`             | USB<br />Wireless                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| K100 AIR RGB                  | `1bab`                         | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| K100                          | `1bc5`<br />`1b7c`<br />`1b7d` | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| K70 RGB MK.2                  | `1b55`<br />`1b49`<br />`1b6b` | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| K70 CORE TKL                  | `2b01`                         | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| K70 CORE TKL WIRELESS         | `2b02`                         | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| K70 RGB TKL CHAMPION SERIES   | `1bb9`<br />`1b73`             | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| KATAR PRO                     | `1b93`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| KATAR PRO XT                  | `1bac`                         | DPI Control<br />RGB Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| KATAR PRO WIRELESS            | `1b94`                         | DPI Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| IRONCLAW RGB                  | `1b5d`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| IRONCLAW RGB WIRELESS         | `1b4c`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| NIGHTSABRE WIRELESS           | `1bb8`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| SCIMITAR RGB ELITE            | `1be3`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| SCIMITAR RGB ELITE WIRELESS   | `1bdb`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| SCIMITAR ELITE WIRELESS SE    | `2b22`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| M55                           | `2b03`                         | DPI Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| M55 RGB PRO                   | `1b70`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| DARK CORE RGB PRO SE WIRELESS | `1b7e`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| DARK CORE RGB PRO             | `1b80`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| M75                           | `1bf0`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| M75 AIR WIRELESS              | `1bf2`                         | DPI Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| M65 RGB ULTRA                 | `1b9e`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| M65 RGB ULTRA WIRELESS        | `1bb5`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| HARPOON RGB PRO               | `1b75`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| HARPOON RGB WIRELESS          | `1b5e`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| DARKSTAR WIRELESS             | `1bb2`                         | DPI Control<br />RGB Control<br />Key Assignments<br />Macros                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| VIRTUOSO RGB WIRELESS XT      | `0a62`<br />`0a64`             | RGB Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| VIRTUOSO MAX WIRELESS         | `2a02`                         | RGB Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HS80 RGB WIRELESS             | `0a6a`                         | RGB Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HS80 MAX WIRELESS             | `0a97`                         | RGB Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| ST100 RGB                     | `0a34`                         | RGB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| MM700 RGB                     | `1b9b`<br />`1bc9`             | RGB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| LT100 Smart Lighting Tower    | `0c23`                         | RGB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| HX1000i                       | `1c07`<br />`1c1e`             | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HX1200i                       | `1c08`<br />`1c23`             | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HX1500i                       | `1c1f`                         | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HX750i                        | `1c05`                         | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| HX850i                        | `1c06`                         | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| RM850i                        | `1c0c`                         | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| RM1000i                       | `1c0d`                         | Fan Control                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |

## Installation (automatic)
1. Download either .deb or .rpm package from the latest Release, depends on your Linux distribution
2. Open terminal
3. Navigate to the folder where the package is downloaded
```bash
# Debian Based (deb)
$ sudo apt install ./OpenLinkHub_X.X.X_amd64.deb 

# RPM based (rpm)
$ sudo dnf install ./OpenLinkHub-X.X.X-1.x86_64.rpm
```

## Installation (manual)
### 1. Requirements
- libudev-dev
- usbutils
- go 1.23.8 - https://go.dev/dl/
```bash
# Required packages (deb)
$ sudo apt-get install libudev-dev
$ sudo apt-get install usbutils

# Required packages (rpm)
$ sudo dnf install libudev-devel
$ sudo dnf install usbutils
```
### 2. Build & install
```bash
$ git clone https://github.com/jurkovic-nikola/OpenLinkHub.git
$ cd OpenLinkHub/
$ go build .
$ chmod +x install.sh
$ sudo ./install.sh
```

### 3. Installation from compiled build
```bash
# Download latest build from https://github.com/jurkovic-nikola/OpenLinkHub/releases
$ wget https://github.com/jurkovic-nikola/OpenLinkHub/releases/download/0.2.0/OpenLinkHub_0.2.0_amd64.tar.gz
$ tar xvf OpenLinkHub_0.2.0_amd64.tar.gz
$ cd OpenLinkHub/
$ chmod +x install.sh
$ sudo ./install.sh
```
### 4. Immutable distributions (Bazzite OS, SteamOS, etc...)
```bash
# Do not install RPM or DEB packages on immutable distributions, they will not work. 
# Download latest tar.gz from Release page. URL will be different when new version is released. 
# To copy download link, go to https://github.com/jurkovic-nikola/OpenLinkHub/releases and right click on tar.gz file and Copy link address. 
# Open your terminal application and type wget and paste copied link to download package. 
$ wget https://github.com/jurkovic-nikola/OpenLinkHub/releases/download/0.4.7/OpenLinkHub_0.4.7_amd64.tar.gz

# Extract package to your home directory
$ tar xf OpenLinkHub_X.X.X_amd64.tar.gz -C /home/$USER/

# Go to extract folder
$ cd /home/$USER/OpenLinkHub

# Make install-immutable.sh executable
$ chmod +x install-immutable.sh

# Run install-immutable.sh. Enter your password for sudo when asked to copy 99-openlinkhub.rules file
$ ./install-immutable.sh

# Restart 
$ systemctl reboot
```

### 5. Configuration
```json
{
  "debug": false,
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "k10temp",
  "manual": false,
  "frontend": true,
  "metrics": true,
  "resumeDelay": 15000,
  "memory": false,
  "memorySmBus": "i2c-0",
  "memoryType": 4,
  "exclude": [],
  "decodeMemorySku": true,
  "memorySku": "",
  "logFile": "",
  "logLevel": "info",
  "enhancementKits": [],
  "temperatureOffset": 0,
  "amdGpuIndex": 0,
  "amdsmiPath": "",
  "cpuTempFile": ""
}
```
- listenPort: HTTP server port.
- listenAddress: Address for HTTP server to listen on.
- cpuSensorChip: CPU sensor chip for temperature. `k10temp` or `zenpower` for AMD and `coretemp` for Intel
- manual: set to true if you want to use your own UI for device control. Setting this to true will disable temperature monitoring and automatic device speed adjustments. 
- frontend: set to false if you do not need WebUI console, and you are making your own UI app. 
- metrics: enable or disable Prometheus metrics
- resumeDelay: amount of time in milliseconds for program to reinitialize all devices after sleep / resume
- memory: Enable overview / control over the memory
- memorySmBus: i2c smbus sensor id
- memoryType: 4 for DDR4. 5 for DDR5
- exclude: list of device ids in uint16 format to exclude from program control
- decodeMemorySku: set to false to manually define `memorySku` value.
- memorySku: Memory part number, e.g. (CMT64GX5M2B5600Z40)
- You can find memory part number by running the following command: `sudo dmidecode -t memory | grep 'Part Number'`
- logFile: custom location for logging. Default is empty.
  - Defining `-` for logFile will send all logs to standard console output.
  - If you change the location of logging, make sure the application username has permission to write to that folder.
- logLevel: log level to log in console or file.
- enhancementKits: DDR4/DDR5 Light Enhancement Kits addresses. 
- If your kit is installed in first and third slot, value would be: `"enhancementKits": [80,82],`. This value is byte value converted from hexadecimal output in `i2cdetect`
  - When kits are used, you need to set `decodeMemorySku` to `false` and define `memorySku`
- temperatureOffset: Temperature offset for AMD Threadripper CPUs
- amdGpuIndex: GPU device index. You can find your GPU index via `amd-smi static --asic --json`
- amdsmiPath: Manual path to amd-smi binary (not recommended). Better way is to define `amd-smi` path in `$PATH` variable if missing.
- cpuTempFile: custom hwmon temperature input file, e.g. tempX_input. Use in combination with `cpuSensorChip`.
## Uninstall
```bash
# Stop service
sudo systemctl stop OpenLinkHub.service

# Remove application directory
sudo rm -rf /opt/OpenLinkHub/

# Remove systemd file (file location can be found by running sudo systemctl status OpenLinkHub.service)
sudo rm /etc/systemd/system/OpenLinkHub.service
# or
sudo rm /usr/lib/systemd/system/OpenLinkHub.service

# Reload systemd
sudo systemctl daemon-reload

# Remove udev rules
sudo rm -f /etc/udev/rules.d/99-openlinkhub.rules
sudo rm -f /etc/udev/rules.d/98-corsair-memory.rules

# Reload udev
sudo udevadm control --reload-rules
sudo udevadm trigger
```
## Running in Docker
As an alternative, OpenLinkHub can be run in Docker, using the Dockerfile in this repository to build it locally. A configuration file has to be mounted to /opt/OpenLinkHub/config.json
```bash
$ docker build . -t openlinkhub
$ # To build a specific version you can use the GIT_TAG build argument
$ docker build --build-arg GIT_TAG=0.1.3-beta -t openlinkhub .

$ docker run --privileged -v ./config.json:/opt/OpenLinkHub/config.json openlinkhub

# For WebUI access, networking is required
$ docker run --network host --privileged -v ./config.json:/opt/OpenLinkHub/config.json openlinkhub
```

## LCD
- LCD images / animations are located in `/opt/OpenLinkHub/database/lcd/images/`
## Dashboard
- Device Dashboard is accessible by browser via link `http://127.0.0.1:27003/`
- Device Dashboard allows you to control your devices.
## RGB
- RGB configuration is located at `database/rgb/your-device-serial.json` file
- RGB can be configured via RGB Editor in Dashboard

## API
- OpenLinkHub ships with built-in HTTP server for device overview and control.
- Documentation is available at `http://127.0.0.1:27003/docs`

## Memory - DDR4 / DDR5
- By default, memory overview and RGB control are disabled in OpenLinkHub. 
- To enable it, you will need to switch `"memory":false` to `"memory":true` and set proper `memorySmBus` value. 
- Things to consider prior:
  - If you are using any other RGB software that can control your RAM, do not set `"memory":true`. 
  - Two programs cannot write to the same I2C address at the same time. 
  - If you do not know what `acpi_enforce_resources=lax` means, do not enable this.
  - If you see your DDR5 memory addresses as `UU`, this means they are locked by `spd5118` the kernel module. For DDR4, they are locked by the `ee1004` kernel module. You will need to unload the appropriate module.
  - If you're still eager to use this...continue reading
```bash
# Install tools
$ sudo apt-get install i2c-tools

# Enable loading of i2c-dev at boot and restart
echo "i2c-dev" | sudo tee /etc/modules-load.d/i2c-dev.conf

# List all i2c, this is AMD example! (AM4, X570 AORUS MASTER (F39d - 09/02/2024)
# If everything is okay, you should see something like this, especially first 3 lines. 
$ sudo i2cdetect -l
i2c-0	smbus     	SMBus PIIX4 adapter port 0 at 0b00	SMBus adapter
i2c-1	smbus     	SMBus PIIX4 adapter port 2 at 0b00	SMBus adapter
i2c-2	smbus     	SMBus PIIX4 adapter port 1 at 0b20	SMBus adapter
i2c-3	i2c       	NVIDIA i2c adapter 1 at c:00.0  	I2C adapter
i2c-4	i2c       	NVIDIA i2c adapter 2 at c:00.0  	I2C adapter
i2c-5	i2c       	NVIDIA i2c adapter 3 at c:00.0  	I2C adapter
i2c-6	i2c       	NVIDIA i2c adapter 4 at c:00.0  	I2C adapter
i2c-7	i2c       	NVIDIA i2c adapter 5 at c:00.0  	I2C adapter
i2c-8	i2c       	NVIDIA i2c adapter 6 at c:00.0  	I2C adapter
i2c-9	i2c       	NVIDIA i2c adapter 7 at c:00.0  	I2C adapter

# If you do not see any smbus devices, you will probably need to set acpi_enforce_resources=lax
# Before setting acpi_enforce_resources=lax please research pros and cons of this and decide on your own!

# In most of the cases, memory will be registered under SMBus PIIX4 adapter port 0 at 0b00 device, aka i2c-0. Lets validate that.
# DDR4 example:
$ sudo i2cdetect -y 0 # this is i2c-0 from i2cdetect -l command
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         08 -- -- -- -- -- -- -- 
10: 10 -- -- 13 -- 15 -- -- 18 19 -- -- -- -- -- -- 
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
30: 30 31 -- -- 34 35 -- -- -- -- 3a -- -- -- -- -- 
40: -- -- -- -- -- -- -- -- -- -- 4a -- -- -- -- -- 
50: 50 51 52 53 -- -- -- -- 58 59 -- -- -- -- -- -- 
60: -- -- -- -- -- -- -- -- 68 -- -- -- 6c -- -- -- 
70: 70 -- -- -- -- -- -- --

# Lets explain this wall of text. (DDR4)
# DDR4 memory uses i2c addresses from 0x50 to 0x57 for DIMM information.
# In this example, I have 4 DIMMs of DDR4 in the workstation, so addresses 50,51,52 and 53 are populated. 
# If you have 2 DIMMs, 50 and 51 will be populated. 
# DIMMs with RGB control will have addresses from 58 to 5f. In this example, I have 2 DIMMs with RGB, 2 are without. So 58 and 59 are populated.
# DIMMs with temperature reporting have addresses from 18 to 1f. In this example, I have 2 DIMMs with temperature probe, 2 are without. So 18 and 19 are populated. 

# To summarize:
# If you have 1 DIMM, 50 address will be populated, 58 if DIMM has RGB control, 18 if there is temperature probe on the DIMM.
# If you have 2 DIMM, 50,51 address will be populated, 58,59 if DIMM has RGB control, 18,19 if there is temperature probe on the DIMM.
# If you have 3 DIMM, 50,51,52 address will be populated, 58,59,5a if DIMM has RGB control, 18,19,1a if there is temperature probe on the DIMM.
# If you have 4 DIMM, 50,51,52,53 will be populated, 58,59,5a,5b if DIMM has RGB control, 18,19,1a,1b if there is temperature probe on the DIMM.
# If your result looks similar to this, you are on the right i2c sensor. If not, try different SMBus device ID.

# DDR5 example:
$ sudo i2cdetect -y 2 # this is i2c-0 from i2cdetect -l command
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- -- 
10: -- -- -- -- -- -- -- -- -- 19 -- 1b -- -- -- -- 
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
30: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
40: -- -- -- -- -- -- -- -- -- 49 -- 4b -- -- -- -- 
50: -- 51 -- 53 -- -- -- -- -- -- -- -- -- -- -- -- 
60: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
70: -- -- -- -- -- -- -- --

# Lets explain this wall of text. (DDR5)
# DDR5 memory uses i2c addresses from 0x50 to 0x57 for DIMM information.
# In this example, I have 2 DIMMs of DDR5 in the workstation, so addresses 51 and 53 are populated. 
# If you have 4 DIMMs, 50,51,52 and 53 will be populated. 
# DIMMs with RGB control will have addresses from 18 to 1f. In this example, I have 2 DIMMs with RGB, so 19 and 1b are populated.
# DIMMs with temperature reporting uses same DIMM info registers, from 50 to 57. Temperature data is stored in 0x31 address.

# To summarize:
# If you have 1 DIMM, 50 address will be populated, 18 if DIMM has RGB control.
# If you have 2 DIMM, 50,51 address will be populated, 18,19 if DIMM has RGB control.
# If you have 3 DIMM, 50,51,52 address will be populated, 18,19,1a if DIMM has RGB control.
# If you have 4 DIMM, 50,51,52,53 will be populated, 18,19,1a,1b if DIMM has RGB control.
# If your result looks similar to this, you are on the right i2c sensor. If not, try different SMBus device ID.

# Set I2C permission
$ echo 'KERNEL=="i2c-0", MODE="0600", OWNER="openlinkhub"' | sudo tee /etc/udev/rules.d/98-corsair-memory.rules
# Reload udev rules
$ sudo udevadm control --reload-rules
$ sudo udevadm trigger
```
- Modify `"memorySmBus": "i2c-0"` if needed.
- Set `"memory":true` in config.json file.
- Set `"memoryType"` in config.json
  - `4` if you have a DDR4 platform
  - `5` if you have a DDR5 platform
- Restart OpenLinkHub service.

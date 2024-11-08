# OpenLinkHub interface for Linux
Open source Linux interface for iCUE LINK Hub and other Corsair AIOs, Hubs.

![Build](https://github.com/jurkovic-nikola/OpenLinkHub/actions/workflows/go.yml/badge.svg)

![Web UI](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/ui.png?raw=true)

# Info
- This project was created out of own necessity to control fans and pumps on workstations after switching everything to Linux.
- I take no responsibility for this code at all. Use this at your own risk. 
- Most of the devices are actually tested on live hardware.
- OpenLinkHub supports multiple devices.
- Take care and have fun!
- This project does not accept any donations.

## Supported devices
| Device                        | VID    | PID                | Sub Devices                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
|-------------------------------|--------|--------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| iCUE LINK System Hub          | `1b1c` | `0c3f`             | <details><summary>Show</summary>iCUE LINK QX RGB<br />iCUE LINK RX<br/>iCUE LINK RX RGB<br/>iCUE LINK RX MAX<br/>iCUE LINK RX MAX RGB<br/>iCUE LINK LX RGB<br />iCUE LINK H100i<br/>iCUE LINK H115i<br/>iCUE LINK H150i<br/>iCUE LINK H170i<br/>XC7 Elite<br/>XG7<br/>XD5 Elite<br/>XD5 Elite LCD <br/>VRM Cooling Module<br />iCUE LINK TITAN H100i<br />iCUE LINK TITAN H150i<br />iCUE LINK TITAN H115i<br />iCUE LINK TITAN H170i<br />LCD Pump Cover<br />iCUE LINK XG3 HYBRID<br />iCUE LINK ADAPTER<br />iCUE LINK LS350 Aurora RGB<br />iCUE LINK LS430 Aurora RGB</details> |
| iCUE COMMANDER Core           | `1b1c` | `0c32`<br />`0c1c` | <details><summary>Show</summary>H100i ELITE CAPELLIX<br />H115i ELITE CAPELLIX<br />H150i ELITE CAPELLIX<br />H170i ELITE CAPELLIX<br />H100i ELITE LCD<br />H150i ELITE LCD<br />H170i ELITE LCD<br />H100i ELITE LCD XT<br />H115i ELITE LCD XT<br />H150i ELITE LCD XT<br />H170i ELITE LCD XT<br />H100i ELITE CAPELLIX XT<br />H115i ELITE CAPELLIX XT<br />H150i ELITE CAPELLIX XT<br />H170i ELITE CAPELLIX XT<br />1x Temperature Probe<br /> 4-LED RGB Fan<br /> 8-LED RGB Fan<br /> QL Fan Series<br /> LL Fan Series<br /> ML Fan Series<br />Any PWM Fan</details>       |
| iCUE COMMANDER Core XT        | `1b1c` | `0c2a`             | <details><summary>Show</summary>External RGB Hub<br />2x Temperature Probe<br /> 4-LED RGB Fan<br /> 8-LED RGB Fan<br /> QL Fan Series<br /> LL Fan Series<br /> ML Fan Series<br />Any PWM Fan<br />H55 RGB AIO<br />H100 RGB AIO<br />H150 RGB AIO</details>                                                                                                                                                                                                                                                                                                                       |
| iCUE H100i RGB ELITE          | `1b1c` | `0c35`<br />`0c40` |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H115i RGB ELITE          | `1b1c` | `0c36`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H150i RGB ELITE          | `1b1c` | `0c37`<br />`0c41` |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H100i RGB PRO XT         | `1b1c` | `0c20`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H115i RGB PRO XT         | `1b1c` | `0c21`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| iCUE H150i RGB PRO XT         | `1b1c` | `0c22`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| Lighting Node CORE            | `1b1c` | `0c1a`             | <details><summary>Show</summary>HD RGB Series Fan<br />LL RGB Series Fan<br />ML PRO RGB Series Fan<br />QL RGB Series Fan<br />8-LED Series Fan<br />SP RGB Series Fan</details>                                                                                                                                                                                                                                                                                                                                                                                                    |
| Lighting Node PRO             | `1b1c` | `0c0b`             | <details><summary>Show</summary>2x External RGB Hub<br />HD RGB Series Fan<br />LL RGB Series Fan<br />ML PRO RGB Series Fan<br />QL RGB Series Fan<br />8-LED Series Fan<br />SP RGB Series Fan</details>                                                                                                                                                                                                                                                                                                                                                                           |
| Commander PRO                 | `1b1c` | `0c10`             | <details><summary>Show</summary>2x External RGB Hub<br />4x Temperature Probe<br />Any PWM Fan</details>                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| XC7 ELITE LCD CPU Water Block | `1b1c` | `0c42`             | <details><summary>Show</summary>RGB Control<br />LCD Control</details>                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| VENGEANCE RGB PRO             | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB PRO SL          | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB RT              | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB RS              | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM RGB        | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE LPX                 | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM            | `1b1c` | DDR4               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE                     | `1b1c` | DDR5               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| VENGEANCE RGB                 | `1b1c` | DDR5               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR PLATINUM RGB        | `1b1c` | DDR5               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| DOMINATOR TITANIUM RGB        | `1b1c` | DDR5               |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K55 CORE RGB                  | `1b1c` | `1bfe`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K65 PRO MINI                  | `1b1c` | `1bd7`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K70 CORE RGB                  | `1b1c` | `1bfd`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K70 PRO RGB                   | `1b1c` | `1bc6`             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| K65 PLUS                      | `1b1c` | `2b10`             | USB                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |

## Installation (automatic)
1. Download either .deb or .rpm package from the latest Release, depends on your Linux distribution
2. Open terminal
3. Navigate to the folder where the package is downloaded
```bash
# Debian Based (deb)
$ sudo apt install ./OpenLinkHub_X.X.X_amd64.deb 

# RPM based (rpm)
$ sudo rpm -ivh OpenLinkHub-X.X.X-1.x86_64.rpm
```

## Installation (manual)
### 1. Requirements
- libudev-dev
- go 1.22.2 - https://go.dev/dl/
```bash
# Required packages (deb)
$ sudo apt-get install libudev-dev

# Required packages (rpm)
$ sudo dnf install libudev-devel
```
### 2. Build & install
```bash
$ git clone https://github.com/jurkovic-nikola/OpenLinkHub.git
$ cd OpenLinkHub/
$ go build .
$ chmod +x install.sh
$ sudo sh install.sh
```

### 3. Installation from compiled build
```bash
# Download latest build from https://github.com/jurkovic-nikola/OpenLinkHub/releases
$ wget https://github.com/jurkovic-nikola/OpenLinkHub/releases/download/0.2.0/OpenLinkHub_0.2.0_amd64.tar.gz
$ tar xvf OpenLinkHub_0.2.0_amd64.tar.gz
$ cd OpenLinkHub/
$ chmod +x install.sh
$ sudo sh install.sh
```

### 4. Configuration
```json
{
  "debug": false,
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "k10temp",
  "manual": false,
  "frontend": true,
  "metrics": true,
  "dbusMonitor": false
}
```
- listenPort: HTTP server port.
- listenAddress: Address for HTTP server to listen on.
- cpuSensorChip: CPU sensor chip for temperature. `k10temp` or `zenpower` for AMD and `coretemp` for Intel
- manual: set to true if you want to use your own UI for device control. Setting this to true will disable temperature monitoring and automatic device speed adjustments. 
- frontend: set to false if you do not need WebUI console, and you are making your own UI app. 
- metrics: enable or disable Prometheus metrics
- dbusMonitor: enable or disable iCUE Link System Hub suspend / resume via DBus. Set this to true of your hub is not recovering after sleep via normal method
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

## Device Dashboard
- Device Dashboard is accessible by browser via link `http://127.0.0.1:27003/`
- Device Dashboard allows you to control your devices. 

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

## Memory - DDR4 / DDR5
- By default, memory overview and RGB control are disabled in OpenLinkHub. 
- To enable it, you will need to switch `"memory":false` to `"memory":true` and set proper `memorySmBus` value. 
- Things to consider prior:
  - If you are using any other RGB software that can control your RAM, do not set `"memory":true`. 
  - Two programs cannot write to the same I2C address at the same time. 
  - If you do not know what `acpi_enforce_resources=lax` means, do not enable this.
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
$ echo 'KERNEL=="i2c-0", MODE="0666"' | sudo tee /etc/udev/rules.d/99-corsair-memory.rules
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

## Motherboard PWM Configuration

## Supported devices
| Motherboard             | Chip      | Driver (out-of-tree)                  |
|-------------------------|-----------|---------------------------------------|
| X870E AORUS ELITE WIFI7 | `it8696`  | https://github.com/frankcrawford/it87 |
| X570 AORUS MASTER       | `it8688`  | https://github.com/frankcrawford/it87 |
| MAG X670E TOMAHAWK WIFI | `nct6687` | https://github.com/Fred78290/nct6687d |

Before you begin:
1. Do you need headers to be SW-controlled instead of BIOS ?
2. Do you know what a Super I/O chip is ?
3. Do you know how to walk through the `hwmon` structure ?

### Get your motherboard name
```
cat /sys/class/dmi/id/board_name
X870E AORUS ELITE WIFI7
```

### Find your device (example)
```
for f in /sys/class/hwmon/hwmon*/name; do   printf '%s=%s ' "$(basename "$(dirname "$f")")" "$(cat "$f")"; done
hwmon0=acpitz hwmon10=spd5118 hwmon11=spd5118 hwmon12=mt7925_phy0 hwmon1=it8696 hwmon2=nvme hwmon3=nvme hwmon4=nvme hwmon5=amdgpu hwmon6=hidpp_battery_3 hwmon7=k10temp hwmon8=gigabyte_wmi hwmon9=r8169_0_1000:00
```

In this example, the motherboard is using the `it8696` chip on the hwmon sensor `hwmon1`. Your data / chip will be different.

### Find your fan headers
What to look for:
- `pwmX`
- `pwmX_enable`
- `fanX_input`

```
ls -l /sys/class/hwmon/hwmon1/ | grep 'pwm*'

-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_auto_start
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_enable
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm1_freq
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_auto_start
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_enable
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm2_freq
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_auto_start
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_enable
-r--rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm3_freq
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_enable
-r--rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm4_freq
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_enable
-r--rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm5_freq
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_channels_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_point1_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_point1_temp_hyst
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_point2_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_point3_temp
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_auto_slope
-rw-rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_enable
-r--rw-r--. 1 root openlinkhub 4096 Feb 28 08:26 pwm6_freq
```
`it8696` reports 6 available fan devices.

### Find your current PWM config
```
cat /sys/class/hwmon/hwmon1/pwm1_enable 
2
```
- `1` - PWM Mode
- `2` - BIOS Mode


### JSON Configuration
- `name` - Your motherboard name from `cat /sys/class/dmi/id/board_name`
- `displayName` - Name of the device, visible in the Dashboard WebUI
- `chip` - Your motherboard chip name, e.g (`it8696`)
- `headerName` - How the device will be named in the WebUI.
- `headerInput` - (`fan1_input`) This stores your RPM speed value
- `headerConfig` - (`pwm1_enable`) This changes your header mode
- `headerValue` - (`pwm1`) This changes actual fan speed (from 1 to 255)

### Add your motherboard into `database/motherboard/motherboard.json`
```json
    {
      "name": "X870E AORUS ELITE WIFI7",
      "chip": "it8696",
      "interval": 3000,
      "headers": {
        "1": {
          "id": 1,
          "headerName": "Fan 1",
          "headerInput": "fan1_input",
          "headerConfig": "pwm1_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm1",
          "defaultHeaderMode": 2
        },
        "2": {
          "id": 2,
          "headerName": "Fan 2",
          "headerInput": "fan2_input",
          "headerConfig": "pwm2_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm2",
          "defaultHeaderMode": 2
        },
        "3": {
          "id": 3,
          "headerName": "Fan 3",
          "headerInput": "fan3_input",
          "headerConfig": "pwm3_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm3",
          "defaultHeaderMode": 2
        },
        "4": {
          "id": 4,
          "headerName": "Fan 4",
          "headerInput": "fan4_input",
          "headerConfig": "pwm4_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm4",
          "defaultHeaderMode": 2
        },
        "5": {
          "id": 5,
          "headerName": "Fan 5",
          "headerInput": "fan5_input",
          "headerConfig": "pwm5_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm5",
          "defaultHeaderMode": 2
        },
        "6": {
          "id": 6,
          "headerName": "Fan 6",
          "headerInput": "fan6_input",
          "headerConfig": "pwm6_enable",
          "headerModes": {
            "1": "PWM",
            "2": "BIOS"
          },
          "headerValue": "pwm6",
          "defaultHeaderMode": 2
        }
      }
    }
```
### Setup udev rules for your chip
```
sudo tee /etc/udev/rules.d/99-motherboard-pwm.rules >/dev/null <<'EOF'
SUBSYSTEM=="hwmon", KERNEL=="hwmon*", ATTR{name}=="it8696", RUN+="/bin/sh -c 'chgrp -h openlinkhub /sys%p/pwm* 2>/dev/null || true; chmod g+rw /sys%p/pwm* 2>/dev/null || true'"
EOF
```
- You need to change `ATTR{name}=="it8696"` to match your chip name!
- If OpenLinkHub is running under a different group or user, you need to change `chgrp -h openlinkhub ...`

### Reload udev rules
```
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=hwmon
```

### Configure OpenLinkHub (config.json)
- Set `enableMotherboard` to `true` and restart the service.
- Set `motherboardBiosOnExit` to `true` to set headers to BIOS mode on program exit.

Test if everything works and create a Pull Request with the updated `motherboard.json` file, including your board in this file.
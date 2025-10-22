## API Documentation and Examples

### Get all data
```bash
$ curl -X GET http://127.0.0.1:27003/api/ --silent | jq
{
  "code": 200,
  "status": 0,
  "device": {
    "40027074EFEBF2568288ACE590128B30": {
      "ProductType": 0,
      "Product": "iCUE LINK System Hub",
      "Serial": "40027074EFEBF2568288ACE590128B30",
      "Firmware": "2.11.517",
      "Image": "icon-device.svg"
    },
    ...
  }
}
```
### Get CPU temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/cpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "45.2 °C"
}
```
```bash
$ curl -X GET http://127.0.0.1:27003/api/cpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 45.25
}
```
### Get GPU temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/gpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "41.0 °C"
}
```
```bash
$ curl -X GET http://127.0.0.1:27003/api/gpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 41
}
```
### Get storage temp
```bash
$ curl -X GET http://127.0.0.1:27003/api/storageTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": [
    {
      "Key": "hwmon1",
      "Model": "KINGSTON SA2000M81000G",
      "Temperature": 42,
      "TemperatureString": "42.0 °C"
    },
    {
      "Key": "hwmon2",
      "Model": "KINGSTON SNVS2000G",
      "Temperature": 33,
      "TemperatureString": "33.0 °C"
    },
    {
      "Key": "hwmon3",
      "Model": "KINGSTON SNVS2000G",
      "Temperature": 33,
      "TemperatureString": "33.0 °C"
    }
  ]
}
```
### Get battery status
```bash
$ curl -X GET http://127.0.0.1:27003/api/batteryStats --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "A9SVC43300IDO4W": {
      "Device": "VIRTUOSO MAX WIRELESS",
      "Level": 62,
      "DeviceType": 2
    }
  }
}
```
### Get all devices
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices/ --silent | jq
{
  "code": 200,
  "status": 0,
  "devices": {
    "40027074EFEBF2568288ACE590128B30": {
      "ProductType": 0,
      "Product": "iCUE LINK System Hub",
      "Serial": "40027074EFEBF2568288ACE590128B30",
      "Firmware": "",
      "Image": "",
      "GetDevice": {
        "Debug": false,
        "manufacturer": "Corsair",
        "product": "iCUE LINK System Hub",
        "serial": "40027074EFEBF2568288ACE590128B30",
        "firmware": "2.11.517",
        "aio": false,
        "devices": {
          "1": {
            "channelId": 1,
            "type": 1,
            "deviceId": "0100282F8203582BFB00018643",
            "name": "iCUE LINK QX RGB",
            "rpm": 603,
            "temperature": 22.5,
            "temperatureString": "22.5 °C",
            "description": "Fan",
            "profile": "Liquid",
            "rgb": "static",
            "label": "GPU Intake 1",
            "portId": 0,
            "IsTemperatureProbe": true,
            "IsLinkAdapter": false,
            "IsCpuBlock": false,
            "HasSpeed": true,
            "HasTemps": true,
            "AIO": false,
            "Position": 0,
            "ExternalAdapter": 0,
            "LCDSerial": ""
          }
          ...
        }
      }
    }
  }
}
```
### Get specific device
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices/40027074EFEBF2568288ACE590128B30 --silent | jq
{
  "code": 200,
  "status": 0,
  "device": {
    "Debug": false,
    "manufacturer": "Corsair",
    "product": "iCUE LINK System Hub",
    "serial": "40027074EFEBF2568288ACE590128B30",
    "firmware": "2.11.517",
    "aio": false,
    "devices": {
      "1": {
        "channelId": 1,
        "type": 1,
        "deviceId": "0100282F8203582BFB00018643",
        "name": "iCUE LINK QX RGB",
        "rpm": 603,
        "temperature": 22.5,
        "temperatureString": "22.5 °C",
        "description": "Fan",
        "profile": "Liquid",
        "rgb": "static",
        "label": "GPU Intake 1",
        "portId": 0,
        "IsTemperatureProbe": true,
        "IsLinkAdapter": false,
        "IsCpuBlock": false,
        "HasSpeed": true,
        "HasTemps": true,
        "AIO": false,
        "Position": 0,
        "ExternalAdapter": 0,
        "LCDSerial": ""
      }
      ...
    }
  }
}
```
### Get devices RGB data
```bash
$ curl -X GET http://127.0.0.1:27003/api/color/ --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "0F023027AFAD89046268BA11F5001C05": {
      "device": "MM700 RGB",
      "defaultColor": {
        "red": 255,
        "green": 100,
        "blue": 0,
        "brightness": 1,
        "Hex": ""
      },
      "profiles": {
        ...
      }
    },
    "23B0010C27A0B8AF9D5FFA62051C00F5": {
      "device": "ST100 RGB",
      "defaultColor": {
        "red": 255,
        "green": 100,
        "blue": 0,
        "brightness": 1,
        "Hex": ""
      },
      "profiles": {
        ...
      }
    }
  }
}
```
### Get device RGB data
```bash
$ curl -X GET http://127.0.0.1:27003/api/color/0F023027AFAD89046268BA11F5001C05 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "device": "MM700 RGB",
    "defaultColor": {
      "red": 255,
      "green": 100,
      "blue": 0,
      "brightness": 1,
      "Hex": ""
    },
    "profiles": {
     ...
    }
  }
}
```
### Get temperature profiles
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/ --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "Liquid": {
      "sensor": 2,
      "zeroRpm": false,
      "profiles": [
        {
          "id": 1,
          "min": 0,
          "max": 30,
          "mode": 0,
          "fans": 30,
          "pump": 50
        },
        ...
      ],
      "points": {
        "0": [
          {
            "x": 0,
            "y": 50
          },
          ...
        ],
        "1": [
          {
            "x": 0,
            "y": 25
          },
          ...
        ]
      },
      "device": "",
      "channelId": 0,
      "linear": false,
      "Hidden": false
    }
  }
}
```
### Get temperature profile
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/Liquid --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "sensor": 2,
    "zeroRpm": false,
    "profiles": [
      {
        "id": 1,
        "min": 0,
        "max": 30,
        "mode": 0,
        "fans": 30,
        "pump": 50
      },
      ...
    ],
    "points": {
      "0": [
        {
          "x": 0,
          "y": 50
        },
        ...
      ],
      "1": [
        {
          "x": 0,
          "y": 25
        },
        ...
      ]
    },
    "device": "",
    "channelId": 0,
    "linear": false,
    "Hidden": false
  }
}
```
### Get temperature graph profile
```bash
$ curl -X GET http://127.0.0.1:27003/api/temperatures/graph/Liquid --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "0": {
      "sensor": 2,
      "points": [
        {
          "x": 0,
          "y": 50
        },
        ...
      ]
    },
    "1": {
      "sensor": 2,
      "points": [
        {
          "x": 0,
          "y": 25
        },
        ...
      ]
    }
  }
}
```
### Get input media keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/media --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "1": {
      "Name": "Volume Up",
      "CommandCode": 115,
      "Media": true,
      "Mouse": false
    },
    "2": {
      "Name": "Volume Down",
      "CommandCode": 114,
      "Media": true,
      "Mouse": false
    },
    ...
  }
}
```
### Get input keyboard keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/keyboard --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "0": {
      "Name": "None",
      "CommandCode": 0,
      "Media": false,
      "Mouse": false
    },
    "10": {
      "Name": "Number 3",
      "CommandCode": 4,
      "Media": false,
      "Mouse": false
    },
    "11": {
      "Name": "Number 4",
      "CommandCode": 5,
      "Media": false,
      "Mouse": false
    },
    ...
  }
}
```
### Get input mouse keys
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/mouse --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "91": {
      "Name": "(Mouse) Left Click",
      "CommandCode": 272,
      "Media": false,
      "Mouse": true
    },
    "92": {
      "Name": "(Mouse) Right Click",
      "CommandCode": 273,
      "Media": false,
      "Mouse": true
    },
    "93": {
      "Name": "(Mouse) Middle Click",
      "CommandCode": 274,
      "Media": false,
      "Mouse": true
    },
    "94": {
      "Name": "(Mouse) Back",
      "CommandCode": 275,
      "Media": false,
      "Mouse": true
    },
    "95": {
      "Name": "(Mouse) Forward",
      "CommandCode": 276,
      "Media": false,
      "Mouse": true
    }
  }
}
```
### Get all LED data
```bash
$ curl -X GET http://127.0.0.1:27003/api/led/ --silent | jq
{
  "code": 200,
  "status": 1,
  "data": [
    {
      "serial": "23B0010C27A0B8AF9D5FFA62051C00F5",
      "deviceName": "ST100 RGB",
      "devices": {
        "0": {
          "ledChannels": 1,
          "pump": false,
          "aio": false,
          "fan": false,
          "stand": true,
          "tower": false,
          "channels": {
            "0": {
              "red": 0,
              "green": 255,
              "blue": 255,
              "brightness": 0,
              "Hex": "#00ffff"
            }
          }
        },
        ...
      }
    }
  }
}
```
### Get device LED data
```bash
$ curl -X GET http://127.0.0.1:27003/api/led/23B0010C27A0B8AF9D5FFA62051C00F5 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "serial": "23B0010C27A0B8AF9D5FFA62051C00F5",
    "deviceName": "ST100 RGB",
    "devices": {
      "0": {
        "ledChannels": 1,
        "pump": false,
        "aio": false,
        "fan": false,
        "stand": true,
        "tower": false,
        "channels": {
          "0": {
            "red": 0,
            "green": 255,
            "blue": 255,
            "brightness": 0,
            "Hex": "#00ffff"
          }
        }
      },
      ...
    }
  }
}
```
### Get macros
```bash
$ curl -X GET http://127.0.0.1:27003/api/macro/ --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "1": {
      "id": 1,
      "name": "Test",
      "actions": {
        "0": {
          "actionType": 3,
          "actionCommand": 20,
          "actionDelay": 0
        },
        "1": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        },
        "2": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        },
        "3": {
          "actionType": 3,
          "actionCommand": 21,
          "actionDelay": 0
        }
      }
    }
  }
}
```
### Get macro
```bash
$ curl -X GET http://127.0.0.1:27003/api/macro/1 --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "id": 1,
    "name": "Nikola",
    "actions": {
      "0": {
        "actionType": 3,
        "actionCommand": 20,
        "actionDelay": 0
      },
      "1": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      },
      "2": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      },
      "3": {
        "actionType": 3,
        "actionCommand": 21,
        "actionDelay": 0
      }
    }
  }
}
```
### Get dashboard settings
```bash
$ curl -X GET http://127.0.0.1:27003/api/dashboard --silent | jq
{
  "code": 200,
  "status": 1,
  "dashboard": {
    "showCpu": true,
    "showDisk": true,
    "showGpu": true,
    "showDevices": true,
    "verticalUi": false,
    "celsius": true,
    "showLabels": true,
    "showBattery": true
  }
}
```
### Create temperature profile - CPU
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"CPU", "sensor":0}' --silent | jq
```
### Create temperature profile - GPU
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"GPU", "sensor":1}' --silent | jq
```
### Create temperature profile - Liquid
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"GPU", "sensor":2}' --silent | jq
```
### Create temperature profile - Static
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/new -d '{"profile":"GPU", "sensor":2, "static":true}' --silent | jq
```
### Set device speed profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/speed -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId":1, "profile":"Liquid"}' --silent | jq
```
### Set device speed profile on all channels
```bash
$ curl -X POST http://127.0.0.1:27003/api/speed -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":0, "profile":"Liquid" }' --silent | jq
```
### Set device speed
```bash
$ curl -X POST http://127.0.0.1:27003/api/speed/manual -d '{ "deviceId":"40027074EFEBF2568288ACE590128B30", "channelId":1, "value":50 }' --silent | jq
```
### Set device RGB profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/color -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId":1, "profile":"rainbow"}' --silent | jq
```
### Set device RGB profile on all channels
```bash
$ curl -X POST http://127.0.0.1:27003/api/color -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId":-1, "profile":"rainbow"}' --silent | jq
```
### Set LED Strip type (Link System Hub)
```bash
$ curl -X POST http://127.0.0.1:27003/api/hub/strip -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 3, "stripId": 1}' --silent | jq
```
### Set external LED device type (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ curl -X POST http://127.0.0.1:27003/api/hub/type -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "portId": 1, "deviceType": 2}' --silent | jq
```
### Set external LED device amount (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ curl -X POST http://127.0.0.1:27003/api/hub/amount -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "portId": 1, "deviceAmount": 2}' --silent | jq
```
### Set device label
```bash
$ curl -X POST http://127.0.0.1:27003/api/label -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "deviceType": 0, "label": "Top Fan"}' --silent | jq
```
### Setup multiple LCD devices (Link System Hub)
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/device -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "deviceType": 0, "lcdSerial": "1234567890"}' --silent | jq
```
### Change LCD animation image
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/image -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "image": "mySuperGif"}' --silent | jq
```
### Change LCD rotation - default
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "rotation": 0}' --silent | jq
```
### Change LCD rotation - 90 degrees
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "rotation": 1}' --silent | jq
```
### Change LCD rotation - 180 degrees
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "rotation": 2}' --silent | jq
```
### Change LCD rotation - 270 degrees
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "channelId": 1, "rotation": 3}' --silent | jq
```
### Change LCD profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/lcd/profile -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "profile": "100"}' --silent | jq
```
### Change brightness - Dropdown
```bash
$ curl -X POST http://127.0.0.1:27003/api/brightness -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "brightness": 1}' --silent | jq
```
### Change brightness - Slider
```bash
$ curl -X POST http://127.0.0.1:27003/api/brightness/gradual -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "brightness": 75}' --silent | jq
```
### Change device position (Link System Hub) - To Left
```bash
$ curl -X POST http://127.0.0.1:27003/api/position -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "position": 3, "deviceIdString": "0100136032035898E900007609", "direction": 0}' --silent | jq
```
### Change device position (Link System Hub) - To Right
```bash
$ curl -X POST http://127.0.0.1:27003/api/position -d '{"deviceId":"40027074EFEBF2568288ACE590128B30", "position": 3, "deviceIdString": "0100136032035898E900007609", "direction": 1}' --silent | jq
```
## Set dashboard settings
```bash
$ curl -X POST http://127.0.0.1:27003/api/dashboard/update -d '{"showCpu": true, "showGpu": true, "showDisk": true, "showDevices": true, "showLabels": true, "celsius": true}' --silent | jq
```
### Set custom ARGB device (Commander Core, Commander Core XT)
```bash
$ curl -X POST http://127.0.0.1:27003/api/argb -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "portId": 6, "deviceType": 2}' --silent | jq
```
### Set keyboard color - Single key
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyId": 15, "keyOption": 0, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set keyboard color - Single row
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyId": 15, "keyOption": 1, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set keyboard color - All keys
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyId": 15, "keyOption": 2, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - Current area (MM700, ST100)
```bash
$ curl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "areaId": 1, "areaOption": 0, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - Current row (MM700, ST100)
```bash
$ curl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "areaId": 1, "areaOption": 1, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Set misc color - All rows (MM700, ST100)
```bash
$ curl -X POST http://127.0.0.1:27003/api/misc/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "areaId": 1, "areaOption": 2, "color:" {"red":255, "green":255, "blue":255}}' --silent | jq
```
### Change user profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/userProfile/change -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "userProfileName": "myProfile"}' --silent | jq
```
### Change keyboard profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/profile/change -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test22"}' --silent | jq
```
### Save current keyboard profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/profile/save -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "0", "new": false}' --silent | jq
```
### Change keyboard layout
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/layout -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardLayout": "US"}' --silent | jq
```
### Change keyboard dial option
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/dial -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardControlDial": 0}' --silent | jq
```
### Change keyboard sleep mode
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 3}' --silent | jq
```
### Change keyboard polling rate
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/pollingRate -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "pollingRate": 3}' --silent | jq
```
### Change rgb scheduler
```bash
$ curl -X POST http://127.0.0.1:27003/api/scheduler/rgb -d '{"rgbControl":true, "rgbOff": "time-value", "rgbOn": "time-value"}' --silent | jq
```
### Set PSU fan speed
```bash
$ curl -X POST http://127.0.0.1:27003/api/psu/speed -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "fanMode": 7}' --silent | jq
```
### Set mouse DPI values
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/dpi -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "stages":{"0":100,"1":200,"2":300,"3":400,"4":500}}' --silent | jq
```
### Set mouse DPI colors
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/dpiColors -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}...}}' --silent | jq
```
### Set mouse Zone colors
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/zoneColors -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}...}}' --silent | jq
```
### Set mouse Sleep mode - 1 minute
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 1}' --silent | jq
```
### Set mouse Sleep mode - 5 minutes
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 5}' --silent | jq
```
### Set mouse Sleep mode - 10 minutes
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 10}' --silent | jq
```
### Set mouse Sleep mode - 15 minutes
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 15}' --silent | jq
```
### Set mouse Sleep mode - 30 minutes
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 30}' --silent | jq
```
### Set mouse Sleep mode - 1 hour
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 60}' --silent | jq
```
### Set mouse Polling Rate - 125 Hz / 8 msec (check your device polling rates)
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "pollingRate": 1}' --silent | jq
```
### Set mouse Polling Rate - 250 Hu / 4 msec (check your device polling rates)
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "pollingRate": 2}' --silent | jq
```
### Set mouse Polling Rate - 500 Hz / 2 msec (check your device polling rates)
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "pollingRate": 3}' --silent | jq
```
### Set mouse Polling Rate - 1000 Hz / 1 msec (check your device polling rates)
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/pollingRate -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "pollingRate": 4}' --silent | jq
```
### Set mouse Angle Snapping
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/angleSnapping -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "angleSnapping": 1}' --silent | jq
```
### Set mouse Button Optimization
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/buttonOptimization -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "buttonOptimization": 1}' --silent | jq
```
### Set mouse Key Assignment
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/updateKeyAssignment -d '{"deviceId":"5C126A3EB51A", "keyIndex": 10, "enabled":true,"pressAndHold":false,"keyAssignmentType": 3,"keyAssignmentValue":55}' --silent | jq
```
### Set headset Zone colors
```bash
$ curl -X POST http://127.0.0.1:27003/api/mouse/zoneColors -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "colorZones":{"0":{"red":255, "green":255, "blue":255},"1":{"red":255, "green":255, "blue":255}...}}' --silent | jq
```
### Change headset sleep mode
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 3}' --silent | jq
```
### Change headset mute indicator
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/muteIndicator -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "muteIndicator": 1}' --silent | jq
```
### Create new macro value
```bash
$ curl -X POST http://127.0.0.1:27003/api/macro/newValue -d '{"macroId":1, "macroType": 1, "macroValue": 13, "macroDelay":200}' --silent | jq
```
### Update temperature graph - Fans
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/updateGraph -d '{"profile": "Liquid", "updateType": 1,"points": [{"x": 0,"y": 25}...]}' --silent | jq
```
### Update temperature graph - Pump
```bash
$ curl -X POST http://127.0.0.1:27003/api/temperatures/updateGraph -d '{"profile": "Liquid", "updateType": 0,"points": [{"x": 0,"y": 25}...]}' --silent | jq
```

### Headset Active Noise Cancellation - Off (require Sidetone Off)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "noiseCancellation": 0}' --silent | jq
```
### Headset Active Noise Cancellation - On (require Sidetone Off)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "noiseCancellation": 1}' --silent | jq
```
### Headset Active Noise Cancellation - Transparency (require Sidetone Off)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/anc -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "noiseCancellation": 2}' --silent | jq
```
### Headset Sidetone - Off (require Active Noise Cancellation Off)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sidetone -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "sideTone": 0}' --silent | jq
```
### Headset Sidetone - On (require Active Noise Cancellation Off)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sidetone -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "sideTone": 1}' --silent | jq
```
### Headset Sidetone Value - 0 (require Sidetone On)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "sideToneValue": 0}' --silent | jq
```
### Headset Sidetone Value - 50 (require Sidetone On)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "sideToneValue": 50}' --silent | jq
```
### Headset Sidetone Value - 100 (require Sidetone On)
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sidetoneValue -d '{"deviceId": "5C126A3EB51A39569ABADC4C3A1FCF54", "sideToneValue": 100}' --silent | jq
```
### Save new user profile
```bash
$ curl -X PUT http://127.0.0.1:27003/api/userProfile -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "userProfileName": "myProfile"}' --silent | jq
```
### Save new keyboard profile
```bash
$ curl -X PUT http://127.0.0.1:27003/api/keyboard/profile/new -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test22", "new": true}' --silent | jq
```
### Save new macro profile
```bash
$ curl -X PUT http://127.0.0.1:27003/api/macro/new -d '{"macroName":"NewMacro"}' --silent | jq
```
### Save device RGB profile
```bash
$ curl -X PUT http://127.0.0.1:27003/api/macro/new -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "profile":"static", "startColor":{"red":255, "green":255, "blue":255}, "endColor":{"red":255, "green":255, "blue":255}, "speed":4}' --silent | jq
```
### Delete keyboard profile
```bash
$ curl -X DELETE http://127.0.0.1:27003/api/keyboard/profile/delete -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test"}' --silent | jq
```
### Delete macro value
```bash
$ curl -X DELETE http://127.0.0.1:27003/api/macro/value -d '{"macroId":1, "macroIndex": 3}' --silent | jq
```
### Delete temperature profile (If any of your devices are using given profile, they will be reset to Normal profile)
```bash
$ curl -X DELETE http://127.0.0.1:27003/api/temperatures/delete -d '{"profile":"CPU"}' --silent | jq
```
### Delete macro profile
```bash
$ curl -X DELETE http://127.0.0.1:27003/api/macro/profile -d '{"macroId":1}' --silent | jq
```

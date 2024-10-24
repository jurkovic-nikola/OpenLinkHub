## API Documentation and Examples

### Speed Control - Modify all channel
```bash
$ curl -X POST http://127.0.0.1:27003/api/speed -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":0, "profile":"Performance" }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Device speed successfully changed"
}
```
### Speed Control - Modify specific channel
```bash
$ curl -X POST http:/127.0.0.1:27003/api/speed -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":13, "profile":"Performance" }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Device speed successfully changed"
}
```
### Speed Control - Modify specific channel
```bash
$ curl -X POST http:/127.0.0.1:27003/api/speed/manual -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":13, "value":50 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Device speed successfully changed"
}
```
### Temperature Profile - Create CPU Profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":0 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Profile is successfully saved"
}
```
### Temperature Profile - Create GPU Profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":1 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Profile is successfully saved"
}
```
### Temperature Profile - Create Liquid Temperature Profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":2 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Profile is successfully saved"
}
```
### Temperature Profile - Create static CPU Profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":true, "sensor":0 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Profile is successfully saved"
}
```
### Temperature Profile - Create static GPU Profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":true, "sensor":1 }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Profile is successfully saved"
}
```
### Temperature Profile - Delete Profile
```bash
$ curl -X DELETE http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example" }' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Speed profile is successfully deleted"
}
```
### Temperature Profile - Get all profiles
```bash
$ curl -X GET http:/127.0.0.1:27003/api/temperatures --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "CPU": {
      "sensor": 0,
      "profiles": [
        {
          "id": 1,
          "min": 0,
          "max": 30,
          "mode": 0,
          "fans": 30,
          "pump": 70
        },
        {
          "id": 2,
          "min": 30,
          "max": 40,
          "mode": 0,
          "fans": 40,
          "pump": 70
        },
        {
          "id": 3,
          "min": 40,
          "max": 50,
          "mode": 0,
          "fans": 40,
          "pump": 70
        },
        {
          "id": 4,
          "min": 50,
          "max": 60,
          "mode": 0,
          "fans": 45,
          "pump": 70
        },
        {
          "id": 5,
          "min": 60,
          "max": 70,
          "mode": 0,
          "fans": 55,
          "pump": 70
        },
        {
          "id": 6,
          "min": 70,
          "max": 80,
          "mode": 0,
          "fans": 70,
          "pump": 80
        },
        {
          "id": 7,
          "min": 80,
          "max": 90,
          "mode": 0,
          "fans": 90,
          "pump": 90
        },
        {
          "id": 8,
          "min": 90,
          "max": 200,
          "mode": 0,
          "fans": 100,
          "pump": 100
        }
      ]
    }
    ...
  }
}
```
### Temperature Profile - Get specific profile
```bash
$ curl -X GET http:/127.0.0.1:27003/api/temperatures/CPU  --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "sensor": 0,
    "profiles": [
      {
        "id": 1,
        "min": 0,
        "max": 30,
        "mode": 0,
        "fans": 30,
        "pump": 70
      },
      {
        "id": 2,
        "min": 30,
        "max": 40,
        "mode": 0,
        "fans": 40,
        "pump": 70
      },
      {
        "id": 3,
        "min": 40,
        "max": 50,
        "mode": 0,
        "fans": 40,
        "pump": 70
      },
      {
        "id": 4,
        "min": 50,
        "max": 60,
        "mode": 0,
        "fans": 45,
        "pump": 70
      },
      {
        "id": 5,
        "min": 60,
        "max": 70,
        "mode": 0,
        "fans": 55,
        "pump": 70
      },
      {
        "id": 6,
        "min": 70,
        "max": 80,
        "mode": 0,
        "fans": 70,
        "pump": 80
      },
      {
        "id": 7,
        "min": 80,
        "max": 90,
        "mode": 0,
        "fans": 90,
        "pump": 90
      },
      {
        "id": 8,
        "min": 90,
        "max": 200,
        "mode": 0,
        "fans": 100,
        "pump": 100
      }
    ]
  }
}
```
### RGB - Get specific profile
```bash
$ curl -X GET http:/127.0.0.1:27003/api/color/rainbow --silent | jq
{
  "code": 200,
  "status": 1,
  "data": {
    "speed": 4,
    "brightness": 1,
    "smoothness": 0,
    "start": {
      "red": 0,
      "green": 0,
      "blue": 0,
      "brightness": 0
    },
    "end": {
      "red": 0,
      "green": 0,
      "blue": 0,
      "brightness": 0
    }
  }
}
```
### RGB - Get profiles
```bash
$ curl -X GET http:/127.0.0.1:27003/api/color --silent | jq
{
  "code": 200,
  "status": 0,
  "data": {
    "circle": {
      "speed": 1,
      "brightness": 1,
      "smoothness": 20,
      "start": {
        "red": 255,
        "green": 0,
        "blue": 0,
        "brightness": 1
      },
      "end": {
        "red": 255,
        "green": 0,
        "blue": 0,
        "brightness": 1
      }
    },
    "circleshift": {
      "speed": 1,
      "brightness": 1,
      "smoothness": 20,
      "start": {
        "red": 255,
        "green": 0,
        "blue": 0,
        "brightness": 1
      },
      "end": {
        "red": 0,
        "green": 255,
        "blue": 0,
        "brightness": 1
      }
    },
    ...
  }
}
```
### RGB - Change device RGB profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", channelId:13, "profile":"rainbow"}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Device RGB profile is successfully changed"
}
```
### Labels - Change device label
```bash
$ curl -X POST http:/127.0.0.1:27003/api/label -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", channelId:13, "label":"Front Intake 1"}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "Device label is successfully applied"
}
```
### LCD - Change device LCD mode - Liquid Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 0}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Pump Speed
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 1}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - CPU Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 2}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - GPU Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 3}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Combined Metrics (Liquid, CPU, Pump Speed)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 4}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Combined Metrics (Liquid, CPU)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 5}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Combined Metrics (CPU, GPU Temp)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 6}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Combined Metrics (CPU, GPU Load)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 7}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD mode - Combined Metrics (CPU, GPU Load/Temp)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 7}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD rotation - Normal
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 0}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD rotation - 90 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 1}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD rotation - 180 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 2}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### LCD - Change device LCD rotation - 270 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 3}' --silent | jq
{
  "code": 200,
  "status": 1,
  "message": "LCD mode successfully changed"
}
```
### Temperature - Get CPU temperature - string
```bash
$ curl -X GET http:/127.0.0.1:27003/api/cpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "37.8 °C"
}
```
### Temperature - Get CPU temperature - float
```bash
$ curl -X GET http:/127.0.0.1:27003/api/cpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 38
}
```
### Temperature - Get GPU temperature - string
```bash
$ curl -X GET http:/127.0.0.1:27003/api/gpuTemp --silent | jq
{
  "code": 200,
  "status": 1,
  "data": "37.8 °C"
}
```
### Temperature - Get GPU temperature - float
```bash
$ curl -X GET http:/127.0.0.1:27003/api/gpuTemp/clean --silent | jq
{
  "code": 200,
  "status": 1,
  "data": 38
}
```
### Temperature - Get Storage temperature
```bash
$ curl -X GET http:/127.0.0.1:27003/api/storageTemp --silent | jq
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
      "Temperature": 32,
      "TemperatureString": "32.0 °C"
    },
    {
      "Key": "hwmon3",
      "Model": "KINGSTON SNVS2000G",
      "Temperature": 33,
      "TemperatureString": "33.0 °C"
    },
    {
      "Key": "hwmon6",
      "Model": "Samsung SSD 860",
      "Temperature": 21,
      "TemperatureString": "21.0 °C"
    }
  ]
}
```
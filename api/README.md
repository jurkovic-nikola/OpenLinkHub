## API Documentation and Examples

### Modify speed on all channel
```bash
$ curl -X POST http://127.0.0.1:27003/api/speed -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":0, "profile":"Performance" }' --silent | jq
```

### Modify speed on specific channel
```bash
$ curl -X POST http:/127.0.0.1:27003/api/speed -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":13, "profile":"Performance" }' --silent | jq
```

### Modify speed on specific channel
```bash
$ curl -X POST http:/127.0.0.1:27003/api/speed/manual -d '{ "deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId":13, "value":50 }' --silent | jq
```

### Create CPU temperature profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":0 }' --silent | jq
```

### Create GPU temperature profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":1 }' --silent | jq
```

### Create Liquid temperature profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":false, "sensor":2 }' --silent | jq
```

### Create static CPU temperature profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":true, "sensor":0 }' --silent | jq
```

### Create static GPU temperature profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/temperatures -d '{ "profile":"Example", "static":true, "sensor":1 }' --silent | jq
```

### Delete temperature profile
```bash
$ curl -X DELETE http:/127.0.0.1:27003/api/temperatures/delete -d '{ "profile":"Example" }' --silent | jq
```

### Get all temperature profiles
```bash
$ curl -X GET http:/127.0.0.1:27003/api/temperatures --silent | jq
```

### Get specific temperature profile
```bash
$ curl -X GET http:/127.0.0.1:27003/api/temperatures/CPU  --silent | jq
```

### Get specific RGB profile
```bash
$ curl -X GET http:/127.0.0.1:27003/api/color/rainbow --silent | jq
```

### Get RGB profiles
```bash
$ curl -X GET http:/127.0.0.1:27003/api/color --silent | jq
```

### Change device RGB profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/color -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", channelId:13, "profile":"rainbow"}' --silent | jq
```

### Change device label
```bash
$ curl -X POST http:/127.0.0.1:27003/api/label -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", channelId:13, "label":"Front Intake 1"}' --silent | jq
```

### Change device LCD mode - Liquid Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 0}' --silent | jq
```

### Change device LCD mode - Pump Speed
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 1}' --silent | jq
```

### Change device LCD mode - CPU Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 2}' --silent | jq
```

### Change device LCD mode - GPU Temperature
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 3}' --silent | jq
```

### Change device LCD mode - Combined Metrics (Liquid, CPU, Pump Speed)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 4}' --silent | jq
```

### Change device LCD mode - Combined Metrics (Liquid, CPU)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 5}' --silent | jq
```

### Change device LCD mode - Combined Metrics (CPU, GPU Temp)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 6}' --silent | jq
```

### Change device LCD mode - Combined Metrics (CPU, GPU Load)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 7}' --silent | jq
```

### Change device LCD mode - Combined Metrics (CPU, GPU Load/Temp)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "mode": 7}' --silent | jq
```

### Change device LCD rotation - Normal
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 0}' --silent | jq
```

### Change device LCD rotation - 90 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 1}' --silent | jq
```

### Change device LCD rotation - 180 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 2}' --silent | jq
```

### Change device LCD rotation - 270 clockwise
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/rotation -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "rotation": 3}' --silent | jq
```

### Get CPU temperature - string
```bash
$ curl -X GET http:/127.0.0.1:27003/api/cpuTemp --silent | jq
```

### Get CPU temperature - float
```bash
$ curl -X GET http:/127.0.0.1:27003/api/cpuTemp/clean --silent | jq
```

### Get GPU temperature - string
```bash
$ curl -X GET http:/127.0.0.1:27003/api/gpuTemp --silent | jq
```

### Get GPU temperature - float
```bash
$ curl -X GET http:/127.0.0.1:27003/api/gpuTemp/clean --silent | jq
```

### Get Storage temperature
```bash
$ curl -X GET http:/127.0.0.1:27003/api/storageTemp --silent | jq
```

### Get all devices
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices --silent | jq
```

### Get specific devices
```bash
$ curl -X GET http://127.0.0.1:27003/api/devices/23B0010C27A0B8AF9D5FFA62051C00F5 --silent | jq
```

### Get media key codes
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/media --silent | jq
```

### Get keyboard codes
```bash
$ curl -X GET http://127.0.0.1:27003/api/input/keyboard --silent | jq
```

### Set LED Strip type (Link System Hub)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/hub/strip -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId": 3, "stripId": 1}' --silent | jq
```

### Set LED Strip type (Link System Hub)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/hub/strip -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId": 3, "stripId": 1}' --silent | jq
```

### Set external LED device type (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/hub/type -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "portId": 1, "deviceType": 2}' --silent | jq
```

### Set external LED device amount (CC XT, Commander Pro, Lightning Node Pro)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/hub/amount -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "portId": 1, "deviceAmount": 2}' --silent | jq
```

### Set device label
```bash
$ curl -X POST http:/127.0.0.1:27003/api/label -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId": 1, "deviceType": 0, "label": "Top Fan"}' --silent | jq
```

### Setup multiple LCD devices (Link System Hub)
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/device -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId": 1, "deviceType": 0, "lcdSerial": "1234567890"}' --silent | jq
```

### Change LCD animation image
```bash
$ curl -X POST http:/127.0.0.1:27003/api/lcd/device -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "channelId": 1, "deviceType": 0, "image": "mySuperGif"}' --silent | jq
```

### Save new user profile
```bash
$ curl -X PUT http:/127.0.0.1:27003/api/userProfile -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "userProfileName": "myProfile"}' --silent | jq
```

### Change user profile
```bash
$ curl -X POST http:/127.0.0.1:27003/api/userProfile -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "userProfileName": "myProfile"}' --silent | jq
```

### Change brightness - Dropdown
```bash
$ curl -X POST http:/127.0.0.1:27003/api/brightness -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "brightness": 1}' --silent | jq
```

### Change brightness - Slider
```bash
$ curl -X POST http:/127.0.0.1:27003/api/brightness/gradual -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "brightness": 75}' --silent | jq
```

### Change device position (Link System Hub) - To Left
```bash
$ curl -X POST http:/127.0.0.1:27003/api/brightness/gradual -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "position": 3, "deviceIdString": "0100136032035898E900007609", "direction": 0}' --silent | jq
```

### Change device position (Link System Hub) - To Right
```bash
$ curl -X POST http:/127.0.0.1:27003/api/brightness/gradual -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "position": 3, "deviceIdString": "0100136032035898E900007609", "direction": 1}' --silent | jq
```

### Get dashboard settings
```bash
$ curl -X GET http://127.0.0.1:27003/api/dashboard --silent | jq
```

### Set dashboard settings
```bash
$ curl -X POST http://127.0.0.1:27003/api/dashboard -d '{"showCpu": true, "showGpu": true, "showDisk": true, "showDevices": true, "showLabels": true, "celsius": true}' --silent | jq
```

### Set custom ARGB device (Commander Core, Commander Core XT)
```bash
$ curl -X POST http://127.0.0.1:27003/api/argb -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "portId": 6, "deviceType": 2}' --silent | jq
```

### Save new keyboard profile
```bash
$ curl -X PUT http://127.0.0.1:27003/api/keyboard/profile/new -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test22", "new": true}' --silent | jq
```

### Change keyboard profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/profile/change -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test22"}' --silent | jq
```

### Save current keyboard profile
```bash
$ curl -X POST http://127.0.0.1:27003/api/keyboard/profile/save -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "0", "new": false}' --silent | jq
```

### Delete keyboard profile
```bash
$ curl -X DELETE http://127.0.0.1:27003/api/keyboard/profile/delete -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "keyboardProfileName": "Test22"}' --silent | jq
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

### Set PSU fan speed
```bash
$ curl -X POST http://127.0.0.1:27003/api/psu/speed -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "fanMode": 7}' --silent | jq
```

### Change headset sleep mode
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/sleep -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "sleepMode": 3}' --silent | jq
```

### Change headset mute indicator
```bash
$ curl -X POST http://127.0.0.1:27003/api/headset/muteIndicator -d '{"deviceId":"5C126A3EB51A39569ABADC4C3A1FCF54", "muteIndicator": 1}' --silent | jq
```
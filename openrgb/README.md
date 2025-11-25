# OpenRGB Server
With the release of the 0.6.2 version of OpenLinkHub or the latest commit, OpenLinkHub supports native OpenRGB Client / Server communication.

This effectively resolves any device communication issue that occurs when two programs attempt to communicate with the same device. 

With this implementation, you can utilize OpenRGB for your RGB effects and use OpenLinkHub for temperature monitoring, fan control, pump control, LCD control, and everything else the program offers you. 

## How to configure
### Step 1
```json
{
  "enableOpenRGBTargetServer": true,
  "openRGBPort": 6743
}
```

- `enableOpenRGBTargetServer` This will enable TCP listener
- `openRGBPort` TCP port to listen on. This is OpenRGB native server port + 1

### Step 2
```bash
systemctl stop OpenLinkHub
```

### Step 3
Disable device in the OpenRGB application, so it's not processed there and click Apply button. After that you can either Rescan Devices or restart OpenRGB application. 
For each device you want integration, you'll have to disable it in OpenRGB. 

![OpenRGB Device](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/openrgb-device.png?raw=true)

### Step 4
```bash
systemctl start OpenLinkHub
```

### Step 5
- Toggle OpenRGB Integration

![OpenRGB Integration](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/openrgb.png?raw=true)

### Step 6
In OpenRGB, click on Client tab connect to 6743 port. 

![OpenRGB Client](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/openrgb-client.png?raw=true)

## Supported devices
| Device                 |
|------------------------|
| iCUE LINK System Hub   | 
| iCUE COMMANDER Core    |
| iCUE COMMANDER Core XT |
| iCUE COMMANDER DUO     |
| ELITE AIOs             |
| PLATINUM AIOs          |
| HYDRO AIOs             |
| Memory                 |
| MM700                  |
| MM800                  |

As new releases are rolled out, more devices will be added to the integration.
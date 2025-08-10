# OpenRGB Server
With the release of the 0.6.2 version of OpenLinkHub or the latest commit, OpenLinkHub supports native OpenRGB Client / Server communication.

This effectively resolves any device communication issue that occurs when two programs attempt to communicate with the same device. 

With this implementation, you can utilize OpenRGB for your RGB effects and use OpenLinkHub for temperature monitoring, fan control, pump control, LCD control, and everything else the program offers you. 

## How to configure
```json
{
  "enableOpenRGBTargetServer": true,
  "openRGBPort": 6743
}
```

- `enableOpenRGBTargetServer` This will enable TCP listener
- `openRGBPort` TCP port to listen on. This is OpenRGB native server port + 1

Each device that has OpenRGB integration will have a small toggle to enable/disable integration.

For each device you toggle the integration, you'll need to disable that device in the OpenRGB application, so it's not processed there.

When you toggle the integration, your RGB will go off, and it's ready to receive data from OpenRGB.

![OpenRGB Integration](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/openrgb.png?raw=true)


## Supported devices for 0.6.2 release
| Device                        |
|-------------------------------|
| iCUE LINK System Hub          | 
| iCUE COMMANDER Core           |
| iCUE COMMANDER Core XT        |

As new releases are rolled out, more devices will be added to the integration.
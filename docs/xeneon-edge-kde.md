## XENEON EDGE KDE

- If you don't want to use terminal, open KDE Settings, search for Touchscreen and change Target display from Automatic to XENEON EDGE. Click Apply.

- From terminal:

Run `kscreen-doctor -o` to get output UUID of the display.
```
Output: 3 HDMI-A-4 ed36747c-ceb1-402c-af08-f963d94bec99
        enabled
        connected
        priority 3
        HDMI
        replication source:0
        Modes:  57:2560x720@60.27*!  58:1920x1080@59.94  59:1920x1080@50.00  60:1280x1024@75.03  61:1280x720@60.00  62:1280x720@59.94  63:1280x720@50.00  64:1024x768@60.00  65:720x576@50.00  66:720x480@59.94  67:640x480@59.94  68:640x480@59.93 
        Custom modes: None
        Geometry: 5120,0 2560x720
        Scale: 1
        Rotation: 1
        Overscan: 0
        Vrr: incapable
        RgbRange: unknown
        HDR: incapable
        Wide Color Gamut: incapable
        ICC profile: none
        Color profile source: sRGB
        Color power preference: prefer efficiency and performance
        Brightness control: supported, set to 95% and dimming to 100%
        DDC/CI: allowed
        Color resolution: unknown
        Allow EDR: unsupported
        Sharpness control: unsupported
        Automatic brightness: unsupported
```

Configure touch input to the actual display
```
kwriteconfig6 \
  --file kcminputrc \
  --group Libinput \
  --group 10176 \
  --group 2137 \
  --group "wch.cn TouchScreen" \
  --key Enabled true

kwriteconfig6 \
  --file kcminputrc \
  --group Libinput \
  --group 10176 \
  --group 2137 \
  --group "wch.cn TouchScreen" \
  --key OutputUuid ed36747c-ceb1-402c-af08-f963d94bec99
```
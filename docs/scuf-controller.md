## SCUF Audio Configuration

By default, SCUF controllers have audio output levels set to 16,16 via ALSA. This provides 50% of the actual audio when audio is routed through SCUF. To fix this, the ALSA mixer settings need to be updated for this device. 

### Find your device - Cable
```bash
$ cat /proc/asound/cards
 0 [NVidia         ]: HDA-Intel - HDA NVidia
                      HDA NVidia at 0xde080000 irq 163
 1 [Generic        ]: HDA-Intel - HD-Audio Generic
                      HD-Audio Generic at 0xdf188000 irq 165
 2 [Generic_1      ]: HDA-Intel - HD-Audio Generic
                      HD-Audio Generic at 0xdf180000 irq 166
 3 [Output         ]: USB-Audio - Corsair ST100 Headset Output
                      Corsair Components Inc. Corsair ST100 Headset Output at usb-0000:15:00.0-5.1,
 4 [G              ]: USB-Audio - CORSAIR VIRTUOSO MAX WIRELESS G
                      Corsair CORSAIR VIRTUOSO MAX WIRELESS G at usb-0000:15:00.0-5.4, high speed
 5 [V2             ]: USB-Audio - SCUF Envision Pro Controller V2
                      Scuf Gaming SCUF Envision Pro Controller V2 at usb-0000:15:00.0-10.3, full spee
```
### Find your device - Wireless Dongle
```bash
$ cat /proc/asound/cards
 0 [NVidia         ]: HDA-Intel - HDA NVidia
                      HDA NVidia at 0xde080000 irq 163
 1 [Generic        ]: HDA-Intel - HD-Audio Generic
                      HD-Audio Generic at 0xdf188000 irq 165
 2 [Generic_1      ]: HDA-Intel - HD-Audio Generic
                      HD-Audio Generic at 0xdf180000 irq 166
 3 [Output         ]: USB-Audio - Corsair ST100 Headset Output
                      Corsair Components Inc. Corsair ST100 Headset Output at usb-0000:15:00.0-5.1,
 4 [G              ]: USB-Audio - CORSAIR VIRTUOSO MAX WIRELESS G
                      Corsair CORSAIR VIRTUOSO MAX WIRELESS G at usb-0000:15:00.0-5.4, high speed
 5 [USB            ]: USB-Audio - SCUF Envision Pro Wireless USB
                      Scuf Gaming SCUF Envision Pro Wireless USB at usb-0000:15:00.0-10.2, full speed
```

### Set proper amixer value for `V2` device. - Cable
```bash
$ sudo amixer -c 5 cset numid=8 32,32 && sudo alsactl store
```

### If you do not wish to use ID, you can use `hw:` flag. - Cable
```bash
$ sudo amixer -D hw:V2 cset numid=8 32,32 && sudo alsactl store
```
### If you do not wish to use ID, you can use `hw:` flag. - Wireless Dongle
```bash
$ sudo amixer -D hw:USB cset numid=8 32,32 && sudo alsactl store
```
This will give you proper audio output power. 

## Volume Control
In order to properly control volume on this value, you need to instruct the wire plumber not to use the ACP of the device. Instead, all volume control needs to be handled in software. 

### Find your device
```bash
$ pactl list sinks | sed -n '/SCUF Envision Pro/,/Properties:/p' | grep 'device.name'
```
This will give you output like this (your `device.name` will be different due to different serial number!)
```
device.name = "alsa_card.usb-Scuf_Gaming_SCUF_Envision_Pro_Wireless_USB_Receiver_V2_1c629ed800020217-00"
```
For V1 dongle use following
```bash
pactl list sinks | sed -n '/SCUF PC Controller/,/Properties:/p' | grep 'device.name'
```

### Setup Wire Plumber
```bash
$ mkdir -p ~/.config/wireplumber/wireplumber.conf.d
$ cat > ~/.config/wireplumber/wireplumber.conf.d/94-scuf-no-acp.conf <<'EOF'
monitor.alsa.rules = [
  {
    matches = [
      { device.name = "alsa_card.usb-Scuf_Gaming_SCUF_Envision_Pro_Wireless_USB_Receiver_V2_1c629ed800020217-00" }
    ]
    actions = {
      update-props = {
        api.alsa.use-acp = false
        api.acp.auto-profile = false
        api.acp.auto-port = false
        api.alsa.soft-mixer = true
        api.alsa.soft-vol = true
        api.alsa.disable-mixer = true
      }
    }
  }
]
EOF
```
You need to modify `device.name` to match your output! Repeat this config for both Cable and Wireless Dongle device and use different file names. 

- `94-scuf-no-acp.conf` - Wireless
- `95-scuf-no-acp.conf` - USB

### Reload Wire Plumber
```bash
systemctl --user restart wireplumber pipewire pipewire-pulse
```
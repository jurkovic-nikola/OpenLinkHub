## Memory Configuration

### Install i2c-tools
```bash
# Fedora
sudo dnf install i2c-tools

# Debian
sudo apt install i2c-tools
```

### Find your `smbus` controller
Your i2c-X device will have a different number. Usually, it's the first `smbus` device from the list. If you don't see your `smbus` device, you will need to use `acpi_enforce_resources=lax` boot parameter.
```bash
sudo i2cdetect -l
i2c-0   i2c             Synopsys DesignWare I2C adapter         I2C adapter
i2c-1   i2c             Synopsys DesignWare I2C adapter         I2C adapter
i2c-2   i2c             NVIDIA i2c adapter 1 at 1:00.0          I2C adapter
i2c-3   i2c             NVIDIA i2c adapter 2 at 1:00.0          I2C adapter
i2c-4   i2c             NVIDIA i2c adapter 3 at 1:00.0          I2C adapter
i2c-5   i2c             NVIDIA i2c adapter 4 at 1:00.0          I2C adapter
i2c-6   i2c             NVIDIA i2c adapter 5 at 1:00.0          I2C adapter
i2c-7   i2c             NVIDIA i2c adapter 6 at 1:00.0          I2C adapter
i2c-8   i2c             NVIDIA i2c adapter 7 at 1:00.0          I2C adapter
i2c-9   i2c             AMDGPU DM i2c hw bus 0                  I2C adapter
i2c-10  i2c             AMDGPU DM i2c hw bus 1                  I2C adapter
i2c-11  i2c             AMDGPU DM i2c hw bus 2                  I2C adapter
i2c-12  i2c             AMDGPU DM i2c hw bus 3                  I2C adapter
i2c-13  i2c             AMDGPU DM aux hw bus 1                  I2C adapter
i2c-14  i2c             AMDGPU DM aux hw bus 2                  I2C adapter
i2c-15  smbus           SMBus PIIX4 adapter port 0 at 0b00      SMBus adapter
i2c-16  smbus           SMBus PIIX4 adapter port 2 at 0b00      SMBus adapter
i2c-17  smbus           SMBus PIIX4 adapter port 1 at 0b20      SMBus adapter
```

### Find the correct SMBus device
Your physical memory starts at address 50 and increases. If you're seeing this, you're on the right SMBus controller. In this example, my SMBus is located on `i2c-15`
```bash
sudo i2cdetect -y 15
     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
00:                         -- -- -- -- -- -- -- -- 
10: -- -- -- -- -- -- -- -- -- 19 -- 1b -- -- -- -- 
20: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
30: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
40: -- -- -- -- -- -- -- -- -- 49 -- 4b -- -- -- -- 
50: -- UU -- UU -- -- -- -- -- -- -- -- -- -- -- -- 
60: -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- -- 
70: -- -- -- -- -- -- -- --
```

### Find your memory SKU
LumenForge attempts DDR5 SKU decoding automatically where the supported
`spd5118` EEPROM path is available. Find your memory SKU and take a note of it
as a manual fallback when automatic decoding produces no usable value.

```bash
sudo dmidecode -t memory | grep 'Part Number'
        Part Number: CMT64GX5M2B5600Z40
        Part Number: CMT64GX5M2B5600Z40
```

### Configure LumenForge `config.json`
Change `memorySmBus` and `memoryType` for your system. Set `memorySku` only when
the automatic decoder produces no usable SKU; it does not override a non-empty
decoded value.

```json
"memory": true,
"memorySmBus": "i2c-15",
"memoryType": 5,
"memorySku": "CMT64GX5M2B5600Z40",
"ramTempViaHwmon": true,
```

LumenForge does not implement a `decodeMemorySku` configuration field, alias,
command-line flag, or environment-variable override. The current
[OpenLinkHub README](https://github.com/jurkovic-nikola/OpenLinkHub#6-configuration)
may still mention that option, but its presence there should not be taken as
evidence that the setting is available in LumenForge. This is treated as an
upstream documentation/implementation mismatch, not as a confirmed deliberate
LumenForge divergence.

### Set permissions
Change `KERNEL=="i2c-15"` to your I2C SMBus device and use the rule that matches
your service mode.

For a user-service installation, grant the `lumenforge` group access:

```bash
echo 'KERNEL=="i2c-15", MODE="0660", GROUP="lumenforge"' | sudo tee /etc/udev/rules.d/98-corsair-memory.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

For a system-service installation, grant the dedicated `lumenforge` account
access:

```bash
echo 'KERNEL=="i2c-15", MODE="0600", OWNER="lumenforge"' | sudo tee /etc/udev/rules.d/98-corsair-memory.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

### Start or restart LumenForge

After the first user-service installation, reboot before testing RAM access if
the installer added the desktop user to the `lumenforge` group. A complete
logout and login can refresh supplementary groups, but reboot is the
recommended and tested procedure. Restarting only the service does not update
group membership in the existing login session. The enabled user service starts
automatically after the new session begins.

For later user-service configuration changes, when the current session already
has group membership:

```bash
systemctl --user restart LumenForge.service
```

For the system service:

```bash
sudo systemctl restart LumenForge.service
```

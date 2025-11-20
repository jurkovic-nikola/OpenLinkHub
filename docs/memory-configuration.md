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
Find your memory SKU and take a note of it.
```bash
sudo dmidecode -t memory | grep 'Part Number'
        Part Number: CMT64GX5M2B5600Z40
        Part Number: CMT64GX5M2B5600Z40
```

### Configure OpenLinkHub `config.json`
You will need to change your `memorySmBus`, `memoryType`, and `memorySku` depending on your system values.
```json
"memory": true,
"memorySmBus": "i2c-15",
"memoryType": 5,
"decodeMemorySku": false,
"memorySku": "CMT64GX5M2B5600Z40",
"ramTempViaHwmon": true,
```

### Set permissions
You will need to change `'KERNEL=="i2c-15"` to your i2c `smbus` device.
```bash
echo 'KERNEL=="i2c-15", MODE="0600", OWNER="openlinkhub"' | sudo tee /etc/udev/rules.d/98-corsair-memory.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

### Restart OpenLinkHub service
```bash
sudo systemctl restart OpenLinkHub.service
```
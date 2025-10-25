## Configuração de Memória

## Instalar i2c-tools
```bash
# Fedora
sudo dnf install i2c-tools

# Debian
sudo apt install i2c-tools
```

## Encontrar seu controlador `smbus`
Seu dispositivo i2c-X terá um número diferente deste exemplo. Normalmente é o primeiro dispositivo `smbus` da lista.  
Se você não visualizar seu dispositivo `smbus`, será necessário usar o parâmetro de boot `acpi_enforce_resources=lax`.

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

## Encontrar o dispositivo smbus correto
A memória física começa no endereço 50 e vai subindo.  
Se você visualizar isso, significa que está no controlador smbus correto.  
Neste exemplo, o meu smbus está localizado em `i2c-15`.

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

## Encontrar o SKU da sua memória
Encontre o SKU (código de peça) do seu módulo de memória e anote-o.  
```bash
sudo dmidecode -t memory | grep 'Part Number'
        Part Number: CMT64GX5M2B5600Z40
        Part Number: CMT64GX5M2B5600Z40
```

## Configurar o `config.json` do OpenLinkHub
Você precisará alterar `memorySmBus`, `memoryType` e `memorySku` de acordo com os valores do seu sistema.  
```json
"memory": true,
"memorySmBus": "i2c-15",
"memoryType": 5,
"decodeMemorySku": false,
"memorySku": "CMT64GX5M2B5600Z40",
"ramTempViaHwmon": true,
```

## Definir permissões
Você precisará alterar `'KERNEL=="i2c-15"'` para o número do seu dispositivo `smbus` identificado anteriormente.  
```bash
echo 'KERNEL=="i2c-15", MODE="0600", OWNER="openlinkhub"' | sudo tee /etc/udev/rules.d/98-corsair-memory.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## Reiniciar o serviço OpenLinkHub
```bash
sudo systemctl restart OpenLinkHub.service
```

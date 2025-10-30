# Interface OpenLinkHub para Linux
Uma interface Linux de código aberto para iCUE LINK Hub e outros AIOs, Hubs da Corsair.
Gerencie iluminação RGB, velocidades de ventiladores, métricas do sistema, bem como teclados, mouses, headsets via painel web.

![Build](https://github.com/jurkovic-nikola/OpenLinkHub/actions/workflows/go.yml/badge.svg)
[![Copr build status](https://copr.fedorainfracloud.org/coprs/jurkovic-nikola/OpenLinkHub/package/OpenLinkHub/status_image/last_build.png)](https://copr.fedorainfracloud.org/coprs/jurkovic-nikola/OpenLinkHub/package/OpenLinkHub/)
[![](https://dcbadge.limes.pink/api/server/https://discord.gg/mPHcasZRPy?style=flat)](https://discord.gg/mPHcasZRPy)

## Recursos

- Interface web acessível em `http://localhost:27003`
- Controle coolers AIO, ventiladores, hubs, bombas, LCDs e iluminação RGB
- Gerencie teclados, mouses e headsets
- Suporte para memória DDR4 e DDR5
- Perfis de ventiladores personalizados, sensores de temperatura e editor RGB
- Se precisar de menu na bandeja do sistema - https://github.com/jurkovic-nikola/openlinkhub-tray
- [Lista de dispositivos suportados](docs/supported-devices.md)

![Interface Web](https://github.com/jurkovic-nikola/OpenLinkHub/blob/main/static/img/ui.png?raw=true)

## Informações
- Este projeto foi criado por necessidade própria para controlar ventiladores e bombas em computadores após migrar tudo para Linux.
- Não assumo nenhuma responsabilidade por este código. Use por sua conta e risco.
- A maioria dos dispositivos foi testada em hardware real.
- Tenha cuidado e divirta-se!
- Este projeto não é um produto oficial da Corsair.

## Instalação (automática)
1. Baixe o pacote .deb ou .rpm da versão mais recente, dependendo da sua distribuição Linux
2. Abra o terminal
3. Navegue para a pasta onde o pacote foi baixado
```bash
# Baseado em Debian (deb)
$ sudo apt install ./OpenLinkHub_?.?.?_amd64.deb

# Baseado em RPM (rpm)
$ sudo dnf install ./OpenLinkHub-?.?.?-1.x86_64.rpm
```

## Instalação (PPA)
```bash
$ sudo add-apt-repository ppa:jurkovic-nikola/openlinkhub
$ sudo apt update
$ sudo apt-get install openlinkhub
```

## Instalação (Copr)
```bash
$ sudo dnf copr enable jurkovic-nikola/OpenLinkHub
$ sudo dnf install OpenLinkHub
```

## Instalação (manual)
### 1. Requisitos
- libudev-dev
- usbutils
- go 1.23.8 - https://go.dev/dl/
```bash
# Pacotes necessários (deb)
$ sudo apt-get install libudev-dev
$ sudo apt-get install usbutils

# Pacotes necessários (rpm)
$ sudo dnf install libudev-devel
$ sudo dnf install usbutils
```
Você também pode usar o devcontainer fornecido para o VScode. Isso é útil para distribuições imutáveis.
### 2. Compilar & instalar
```bash
$ git clone https://github.com/jurkovic-nikola/OpenLinkHub.git
$ cd OpenLinkHub/
$ go build .
$ chmod +x install.sh
$ sudo ./install.sh
```

### 3. Instalação a partir de build compilado
```bash
# Baixe o build mais recente de https://github.com/jurkovic-nikola/OpenLinkHub/releases/latest
$ wget "https://github.com/jurkovic-nikola/OpenLinkHub/releases/latest/download/OpenLinkHub_$(curl -s https://api.github.com/repos/jurkovic-nikola/OpenLinkHub/releases/latest | jq -r '.tag_name')_amd64.tar.gz"
$ tar xf OpenLinkHub_?.?.?_amd64.tar.gz
$ cd /home/$USER/OpenLinkHub/
$ chmod +x install.sh
$ sudo ./install.sh
```
### 4. Distribuições imutáveis (Bazzite OS, SteamOS, etc...)
```bash
# Não instale pacotes RPM ou DEB em distribuições imutáveis, eles não funcionarão.
# O mesmo procedimento pode ser seguido para atualizar uma instalação existente.
# Baixe o tar.gz mais recente da página de Release, ou use o seguinte comando para baixar a versão mais recente.
$ wget "https://github.com/jurkovic-nikola/OpenLinkHub/releases/latest/download/OpenLinkHub_$(curl -s https://api.github.com/repos/jurkovic-nikola/OpenLinkHub/releases/latest | jq -r '.tag_name')_amd64.tar.gz"

# Extraia o pacote para seu diretório home
$ tar xf OpenLinkHub_?.?.?_amd64.tar.gz -C /home/$USER/

# Vá para a pasta extraída
$ cd /home/$USER/OpenLinkHub

# Torne install-immutable.sh executável
$ chmod +x install-immutable.sh

# Execute install-immutable.sh. Digite sua senha para sudo quando solicitado para copiar o arquivo 99-openlinkhub.rules
$ ./install-immutable.sh

# Reinicie
$ systemctl reboot
```
### 5. Utilisation
```bash
sudo systemctl start OpenLinkHub.service
xdg-open http://127.0.0.1:27003
```

### 6. Configuração
`/opt/OpenLinkHub/config.json`
```json
{
  "debug": false,
  "listenPort": 27003,
  "listenAddress": "127.0.0.1",
  "cpuSensorChip": "k10temp",
  "manual": false,
  "frontend": true,
  "metrics": true,
  "resumeDelay": 15000,
  "memory": false,
  "memorySmBus": "i2c-0",
  "memoryType": 4,
  "exclude": [],
  "decodeMemorySku": true,
  "memorySku": "",
  "logFile": "",
  "logLevel": "info",
  "enhancementKits": [],
  "temperatureOffset": 0,
  "amdGpuIndex": 0,
  "amdsmiPath": "",
  "cpuTempFile": "",
  "graphProfiles": false,
  "ramTempViaHwmon": false,
  "nvidiaGpuIndex": [0],
  "defaultNvidiaGPU": 0
}
```
- listenPort: Porta do servidor HTTP.
- listenAddress: Endereço para o servidor HTTP escutar.
- cpuSensorChip: Chip sensor de CPU para temperatura. `k10temp` ou `zenpower` para AMD e `coretemp` para Intel
- manual: defina como true se quiser usar sua própria interface para controle de dispositivos. Definir como true desabilitará monitoramento de temperatura e ajustes automáticos de velocidade de dispositivos.
- frontend: defina como false se não precisar do console WebUI, e estiver fazendo seu próprio app de interface.
- metrics: habilitar ou desabilitar métricas Prometheus
- resumeDelay: quantidade de tempo em milissegundos para o programa reinicializar todos os dispositivos após suspensão / retomada
- memory: Habilitar visão geral / controle sobre a memória
- memorySmBus: id do sensor smbus i2c
- memoryType: 4 para DDR4. 5 para DDR5
- exclude: lista de ids de dispositivos em formato uint16 para excluir do controle do programa
- decodeMemorySku: defina como false para definir manualmente o valor `memorySku`.
- memorySku: Número da peça da memória, ex. (CMT64GX5M2B5600Z40)
- Você pode encontrar o número da peça da memória executando o comando: `sudo dmidecode -t memory | grep 'Part Number'`
- logFile: localização personalizada para logging. Padrão é vazio.
  - Definir `-` para logFile enviará todos os logs para saída padrão do console.
  - Se alterar a localização do logging, certifique-se de que o nome de usuário da aplicação tenha permissão para escrever nessa pasta.
- logLevel: nível de log para logar no console ou arquivo.
- enhancementKits: Endereços dos Kits de Aprimoramento de Luz DDR4/DDR5.
- Se seu kit estiver instalado no primeiro e terceiro slot, o valor seria: `"enhancementKits": [80,82],`. Este valor é o valor byte convertido da saída hexadecimal em `i2cdetect`
  - Quando kits são usados, você precisa definir `decodeMemorySku` como `false` e definir `memorySku`
- temperatureOffset: Deslocamento de temperatura para CPUs AMD Threadripper
- amdGpuIndex: Índice do dispositivo GPU. Você pode encontrar o índice da sua GPU via `amd-smi static --asic --json`
- amdsmiPath: Caminho manual para binário amd-smi (não recomendado). Melhor forma é definir o caminho `amd-smi` na variável `$PATH` se estiver faltando.
- cpuTempFile: arquivo de entrada de temperatura hwmon personalizado, ex. tempX_input. Use em combinação com `cpuSensorChip`.
- graphProfiles: Definir este valor como `true` habilitará perfis de temperatura baseados em gráfico no endpoint `/temperature` e habilitará interpolação de temperatura.
- ramTempViaHwmon: Mude para true se quiser monitorar a temperatura da RAM via sistema hwmon. Com esta opção, você não precisa descarregar módulos para obter temperatura. (Requer kernel 6.11+)
- nvidiaGpuIndex: Configuração multi GPU NVIDIA.
- defaultNvidiaGPU: índice padrão da GPU NVIDIA, padrão é 0.
  - Se usar vfio-pci/pass-through, você tem que definir como -1 para evitar conflitos com módulos nvidia.

### 7. Interface de Aplicativo Web Progressiva (PWA)
A interface web suporta instalação como aplicativo web progressivo (PWA). Com um navegador suportado, isso permite que a interface apareça como um aplicativo independente.
Navegadores baseados em Chromium suportam PWAs, Firefox atualmente não.
GNOME 'Web,' também conhecido como 'Epiphany' é uma boa opção para PWAs em sistemas GNOME.

### 8. Integração OpenRGB
[Veja detalhes](openrgb/README-pt_BR.md)

## Desinstalação
```bash
# Pare o serviço
sudo systemctl stop OpenLinkHub.service

# Remova o diretório da aplicação
sudo rm -rf /opt/OpenLinkHub/

# Remova o arquivo systemd (localização do arquivo pode ser encontrada executando sudo systemctl status OpenLinkHub.service)
sudo rm /etc/systemd/system/OpenLinkHub.service
# ou
sudo rm /usr/lib/systemd/system/OpenLinkHub.service

# Recarregue systemd
sudo systemctl daemon-reload

# Remova regras udev
sudo rm -f /etc/udev/rules.d/99-openlinkhub.rules
sudo rm -f /etc/udev/rules.d/98-corsair-memory.rules

# Recarregue udev
sudo udevadm control --reload-rules
sudo udevadm trigger
```
## Executando no Docker
Como alternativa, OpenLinkHub pode ser executado no Docker, usando o Dockerfile neste repositório para compilá-lo localmente. Um arquivo de configuração deve ser montado em /opt/OpenLinkHub/config.json
```bash
$ docker build . -t openlinkhub
$ # Para compilar uma versão específica você pode usar o argumento de build GIT_TAG
$ docker build --build-arg GIT_TAG=0.1.3-beta -t openlinkhub .

$ docker run --privileged openlinkhub

# Para acesso WebUI, rede é necessária
$ docker run --network host --privileged openlinkhub
```

## LCD
- Imagens / animações LCD estão localizadas em `/opt/OpenLinkHub/database/lcd/images/`
## Painel
- Painel de Dispositivo é acessível pelo navegador via link `http://127.0.0.1:27003/`
- Painel de Dispositivo permite controlar seus dispositivos.
## RGB
- Configuração RGB está localizada no arquivo `database/rgb/seu-dispositivo-serial.json`
- RGB pode ser configurado via Editor RGB no Painel

## API
- OpenLinkHub vem com servidor HTTP integrado para visão geral e controle de dispositivos.
- Documentação está disponível em `http://127.0.0.1:27003/docs`

## Memória - DDR4 / DDR5
[Ver detalhes](docs/memory-configuration_pt-BR.md)
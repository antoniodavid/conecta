# 📡 Conecta - Manual de Uso

**Conecta** es una herramienta unificada para gestionar conexión a internet ETECSA, hotspot WiFi y VPN en Cuba.

---

## 📋 Tabla de Contenidos

1. [Instalación](#instalación)
2. [Configuración](#configuración)
3. [Uso Básico](#uso-básico)
4. [Comandos CLI](#comandos-cli)
5. [Interfaces TUI](#interfaces-tui)
6. [Hotspot WiFi](#hotspot-wifi)
7. [VPN](#vpn)
8. [Solución de Problemas](#solución-de-problemas)
9. [Arquitectura](#arquitectura)

---

## Instalación

### Requisitos

- Go 1.21 o superior
- Linux con interfaz ethernet (para ETECSA)
- Interfaz WiFi (para hotspot)
- WireGuard (para VPN, opcional)
- Herramientas del sistema según la función usada:
  - `iw` — detección de WiFi y clientes del hotspot
  - `create_ap` + `systemctl` — servicio del hotspot
  - `iptables` + `sudo` — NAT para clientes del hotspot
  - `jq` — plugin de Omarchy (los adaptadores construyen JSON con `jq`)

### Compilar desde código fuente

```bash
cd ~/conecta
go build -o bin/conecta ./cmd/conecta/
go build -o bin/hotspot ./cmd/hotspot/
go build -o bin/conecta-cli ./cmd/cli/
```

### Instalar globalmente

```bash
# Opción 1: ~/.local/bin (sin sudo)
mkdir -p ~/.local/bin
cp ~/conecta/bin/* ~/.local/bin/

# Opción 2: /usr/local/bin (con sudo)
sudo cp ~/conecta/bin/* /usr/local/bin/
```

### Verificar instalación

```bash
conecta-cli help
conecta-cli status
```

### Privilegios y autorización no interactiva

Las acciones privilegiadas (hotspot, NAT, VPN) requieren autorización
previamente configurada: el CLI verifica `sudo -n true` **antes** de
cualquier paso destructivo y falla cerrado (código de salida `4`,
sin modificar el host) cuando la autorización falta o es denegada.
Un widget de barra no tiene forma de pedir contraseña, por lo que las
acciones de hotspot/NAT/VPN lanzadas desde la barra quedan deshabilitadas
hasta que exista una configuración autorizada (por ejemplo, una entrada
NOPASSWD de sudoers instalada por root para los comandos documentados).
Las operaciones de solo lectura (`status`, clientes) y el login al portal
no requieren privilegios.

Propiedad de la instalación:

```bash
# Binarios de usuario: pertenecen al usuario, sin setuid
mkdir -p ~/.local/bin
cp ~/conecta/bin/* ~/.local/bin/

# Alternativa global: root es dueño de /usr/local/bin
sudo cp ~/conecta/bin/* /usr/local/bin/
```

El archivo `~/.config/conecta/config.yaml` puede contener credenciales y
se guarda con permisos `0600` (solo el dueño lo lee).

---

## Configuración

### Archivo de configuración

**Ubicación:** `~/.config/conecta/config.yaml`

```yaml
# Configuración de red
network:
  gateway: 192.168.1.1        # Gateway ETECSA
  interface: enp3s0            # Interfaz ethernet
  portal_url: https://secure.etecsa.net:8443

# Configuración de hotspot
hotspot:
  ssid: MI_RED_WIFI             # Nombre de la red WiFi
  passphrase: CAMBIA-ESTA-CLAVE-SEGURA  # Contraseña del hotspot (mínimo 8 caracteres)
  channel: default             # Canal WiFi (default/auto)
  freq_band: "2.4"            # Banda de frecuencia (2.4 o 5)
  method: nat                  # Método de compartir (nat/bridge)
  subnet: 192.168.12.0/24     # Subred del hotspot
  gateway: 192.168.12.1        # Gateway del hotspot

# Configuración de VPN
vpn:
  enabled: false               # Habilitar VPN
  interface: wg0               # Interfaz WireGuard
  name: USA                    # Nombre de la conexión VPN

# Configuración de interfaz
ui:
  theme: default               # Tema (default/dark/light)
  refresh_sec: 2               # Intervalo de actualización

# Credenciales (opcional, se pueden pasar por CLI)
credentials:
  username: TU_USUARIO
  password: TU_CONTRASEÑA
```

### Editar configuración

```bash
# Abrir en editor
nano ~/.config/conecta/config.yaml

# O usar el CLI para ver config actual
conecta-cli status
```

---

## Uso Básico

### Conectarse a ETECSA

```bash
# 1. Verificar estado
conecta-cli status

# 2. Iniciar sesión (usa credenciales del config)
conecta-cli login

# 3. O con credenciales explícitas
conecta-cli login --user TU_USUARIO --pass TU_CONTRASEÑA

# 4. Verificar conexión
conecta-cli status
```

### Compartir conexión vía hotspot

```bash
# 1. Iniciar hotspot
conecta-cli hotspot start

# 2. Configurar NAT (para que los clientes tengan internet)
conecta-cli nat setup

# 3. Ver clientes conectados
conecta-cli hotspot clients

# 4. Detener hotspot
conecta-cli hotspot stop
```

### Usar VPN

```bash
# Ver estado VPN
conecta-cli vpn status

# Conectar VPN
conecta-cli vpn connect

# Desconectar VPN
conecta-cli vpn disconnect

# Alternar VPN
conecta-cli vpn toggle
```

---

## Comandos CLI

### Referencia rápida

| Comando | Descripción |
|---------|-------------|
| `conecta-cli status` | Estado de la conexión |
| `conecta-cli login` | Iniciar sesión ETECSA |
| `conecta-cli logout` | Cerrar sesión |
| `conecta-cli speed` | Test de velocidad |
| `conecta-cli hotspot start` | Iniciar hotspot |
| `conecta-cli hotspot stop` | Detener hotspot |
| `conecta-cli hotspot status` | Estado del hotspot |
| `conecta-cli hotspot clients` | Listar clientes |
| `conecta-cli nat setup` | Configurar NAT |
| `conecta-cli nat cleanup` | Limpiar NAT |
| `conecta-cli nat status` | Estado de NAT |
| `conecta-cli vpn status` | Estado VPN |
| `conecta-cli vpn connect` | Conectar VPN |
| `conecta-cli vpn disconnect` | Desconectar VPN |
| `conecta-cli vpn toggle` | Alternar VPN |
| `conecta-cli help` | Mostrar ayuda |

### Ejemplos detallados

```bash
# Ver todos los comandos disponibles
conecta-cli help

# Ver ayuda de un comando específico
conecta-cli hotspot --help

# Login silencioso (sin output)
conecta-cli login 2>/dev/null

# Ver estado con más detalle
conecta-cli status
```

---

## Interfaces TUI

### TUI Principal (`conecta`)

```bash
conecta
```

**Atajos de teclado:**

| Tecla | Acción |
|-------|--------|
| `1` | Dashboard (estado de red) |
| `2` | Hotspot (gestión y clientes) |
| `3` | VPN (estado y controles) |
| `4` | Log (historial de eventos) |
| `s` | Iniciar hotspot (en tab 2) |
| `S` | Detener hotspot (en tab 2) |
| `v` | Alternar VPN (en tab 3) |
| `n` | Configurar NAT (en tab 2) |
| `r` | Actualizar datos |
| `q` | Salir |

### TUI Hotspot (`hotspot`)

```bash
hotspot
```

**Atajos de teclado:**

| Tecla | Acción |
|-------|--------|
| `s` | Iniciar hotspot |
| `S` | Detener hotspot |
| `n` | Configurar NAT |
| `r` | Actualizar datos |
| `q` | Salir |

---

## Hotspot WiFi

### Configuración del hotspot

El hotspot utiliza `create_ap` para crear un punto de acceso WiFi.

**Configuración por defecto:**
- SSID: `MI_RED_WIFI`
- Contraseña: `CAMBIA-ESTA-CLAVE-SEGURA` (cámbiala en `~/.config/conecta/config.yaml`)
- Canal: automático
- Banda: 2.4 GHz
- Método: NAT

### Cambiar configuración del hotspot

Editar `~/.config/conecta/config.yaml`:

```yaml
hotspot:
  ssid: MI_RED_WIFI
  passphrase: mi_contraseña_segura
  channel: 6
  freq_band: "5"  # Para 5 GHz (mejor velocidad)
```

### Gestión del hotspot

```bash
# Iniciar
conecta-cli hotspot start

# Detener
conecta-cli hotspot stop

# Ver estado
conecta-cli hotspot status

# Ver clientes conectados
conecta-cli hotspot clients
```

### Compartir conexión ETECSA

Para que los dispositivos conectados al hotspot tengan acceso a internet:

```bash
# 1. Conectar a ETECSA
conecta-cli login

# 2. Iniciar hotspot
conecta-cli hotspot start

# 3. Configurar NAT
conecta-cli nat setup

# 4. Verificar que IP forwarding está activo
cat /proc/sys/net/ipv4/ip_forward
# Debe mostrar: 1

# 5. Los dispositivos ahora pueden conectarse a MI_RED_WIFI
```

---

## VPN

### Configuración de VPN

El soporte VPN utiliza WireGuard a través de NetworkManager.

**Configuración por defecto:**
- Interfaz: `wg0`
- Nombre de conexión: `USA`

### Cambiar configuración VPN

Editar `~/.config/conecta/config.yaml`:

```yaml
vpn:
  enabled: true
  interface: wg0
  name: Spain  # Nombre de la conexión en NetworkManager
```

### Uso de VPN

```bash
# Ver estado
conecta-cli vpn status

# Conectar
conecta-cli vpn connect

# Desconectar
conecta-cli vpn disconnect

# Alternar (conectar si está desconectada, desconectar si está conectada)
conecta-cli vpn toggle
```

### Compartir conexión VPN por hotspot

Para que los dispositivos del hotspot usen la VPN:

```bash
# 1. Conectar VPN
conecta-cli vpn connect

# 2. Iniciar hotspot
conecta-cli hotspot start

# 3. Configurar NAT (detectará automáticamente la VPN)
conecta-cli nat setup

# 4. Los clientes del hotspot usarán la VPN
```

---

## Solución de Problemas

### Problemas de conexión ETECSA

**Error: "Portal inalcanzable"**

```bash
# Verificar que el cable esté conectado
ip link show enp3s0

# Verificar gateway
ping -c 1 192.168.1.1

# Verificar ruta a ETECSA
ip route show 10.180.0.0/16

# Agregar ruta manualmente si falta
sudo ip route add 10.180.0.0/16 via 192.168.1.1 dev enp3s0
```

**Error: "Credenciales inválidas"**

```bash
# Verificar credenciales en config
cat ~/.config/conecta/config.yaml | grep -A2 credentials

# Probar login manual
conecta-cli login --user USUARIO --pass CONTRASEÑA
```

**Error: "Sesión perdida"**

```bash
# Re-login automático
conecta-cli login
```

### Problemas de hotspot

**Error: "No se pudo iniciar hotspot"**

```bash
# Verificar interfaz WiFi
iw dev

# Verificar si create_ap está instalado
which create_ap

# Verificar servicios
systemctl status create_ap
```

**Hotspot activo pero sin internet**

```bash
# Verificar NAT
conecta-cli nat status

# Configurar NAT
conecta-cli nat setup

# Verificar IP forwarding
cat /proc/sys/net/ipv4/ip_forward
# Si es 0, activar:
sudo sysctl -w net.ipv4.ip_forward=1
```

**No se detectan clientes**

```bash
# Verificar clientes con iw
iw dev ap0 station dump

# Verificar DHCP
cat /tmp/create_ap.wlan*.conf.*/dnsmasq.leases
```

### Problemas de VPN

**Error: "No se pudo conectar VPN"**

```bash
# Verificar interfaz WireGuard
ip link show wg0

# Verificar conexión en NetworkManager
nmcli con show --active | grep wireguard

# Conectar manualmente
nmcli con up USA
```

**VPN conectada pero sin internet**

```bash
# Verificar ruta
ip route show

# Verificar DNS
nslookup google.com

# Probar con/without VPN
curl --interface wg0 https://ifconfig.me  # Con VPN
curl https://ifconfig.me                    # Sin VPN
```

### Problemas generales

**El TUI no muestra datos**

```bash
# Verificar que los binarios funcionan
conecta-cli status

# Verificar config
cat ~/.config/conecta/config.yaml
```

**Errores de permisos**

```bash
# Para NAT necesita sudo
sudo sysctl -w net.ipv4.ip_forward=1

# Para gestionar servicios
sudo systemctl start create_ap
```

---

## Arquitectura

### Estructura del proyecto

```
conecta/
├── pkg/
│   ├── network/        # Lógica de red
│   │   ├── portal.go   # Portal ETECSA
│   │   ├── routing.go  # Rutas y NAT
│   │   ├── speed.go    # Test de velocidad
│   │   └── types.go    # Tipos compartidos
│   ├── hotspot/        # Lógica de hotspot
│   │   ├── create_ap.go # Gestión create_ap
│   │   ├── clients.go  # Detección de clientes
│   │   ├── nat.go      # NAT/iptables
│   │   └── types.go    # Tipos compartidos
│   ├── vpn/            # Lógica VPN
│   │   ├── manager.go  # Gestión WireGuard
│   │   └── types.go    # Tipos compartidos
│   └── config/         # Configuración
│       └── config.go   # Carga/guardado YAML
├── cmd/
│   ├── conecta/        # TUI principal
│   ├── hotspot/        # TUI de hotspot
│   └── cli/            # CLI no-interactivo
├── bin/                # Binarios compilados
├── config.yaml         # Config por defecto
├── README.md           # Documentación
└── MANUAL.md           # Este archivo
```

### Diseño

- **Separación de concerns**: Lógica en `pkg/`, UI en `cmd/`
- **Reutilización**: `pkg/` compartido entre TUIs y CLI
- **Configuración**: YAML en `~/.config/conecta/`
- **Modular**: Fácil agregar nuevos comandos o interfaces

### Testing

```bash
# Ejecutar todos los tests
go test ./...

# Tests con verbose
go test -v ./pkg/...

# Coverage
go test -cover ./pkg/...
```

---

## Licencia

MIT

---

## Soporte

- **Documentación**: `~/conecta/MANUAL.md`
- **Configuración**: `~/.config/conecta/config.yaml`
- **Código fuente**: `~/conecta/`

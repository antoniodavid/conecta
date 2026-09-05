# 📡 Conecta

Herramienta unificada para gestionar conexión a internet ETECSA, hotspot WiFi y VPN en Cuba.

## Características

- ✅ **Portal ETECSA**: Login, logout, auto-reconexión
- ✅ **Hotspot WiFi**: Compartir conexión vía create_ap
- ✅ **NAT/Enrutamiento**: Configurar forwarding para clientes
- ✅ **VPN**: Gestión de WireGuard (disconnect only deactivates — profiles persist and `vpn list` still shows them inactive)
- ✅ **Speed Test**: Medir velocidad de descarga
- ✅ **TUI**: Interfaces de terminal interactivas
- ✅ **CLI**: Comandos para scripts y automatización

## Instalación Rápida

```bash
# Clonar (si no existe)
cd ~
git clone <repo-url> conecta
cd conecta

# Compilar
go build -o bin/conecta ./cmd/conecta/
go build -o bin/hotspot ./cmd/hotspot/
go build -o bin/conecta-cli ./cmd/cli/

# Instalar
mkdir -p ~/.local/bin
cp bin/* ~/.local/bin/
```

## Instalación y privilegios

- Los binarios pertenecen al usuario (`~/.local/bin`) o a root
  (`/usr/local/bin`); nunca uses setuid.
- Las acciones privilegiadas (hotspot, NAT, VPN) exigen autorización no
  interactiva previamente configurada (`sudo -n`); sin ella fallan con
  salida `4` sin tocar el host. Las acciones de la barra quedan
  deshabilitadas hasta resolver esa autorización.
- `~/.config/conecta/config.yaml` puede contener credenciales y se guarda
  con `0600`.
- Herramientas del sistema: `iw`, `create_ap`, `iptables`, WireGuard
  (opcional), `jq` (plugin Omarchy).

Ver [MANUAL.md](MANUAL.md) (instalación, privilegios) y
[omarchy-plugin/README.md](omarchy-plugin/README.md) (plugin).

## Uso Rápido

```bash
# Ver ayuda
conecta-cli help

# Conectar a ETECSA
conecta-cli login

# Ver estado
conecta-cli status

# Iniciar hotspot
conecta-cli hotspot start

# Configurar NAT
conecta-cli nat setup

# Test de velocidad
conecta-cli speed
```

## Documentación

📖 **[Manual completo](MANUAL.md)** - Guía detallada con todos los comandos, configuración y solución de problemas.

## Configuración

El archivo de configuración está en `~/.config/conecta/config.yaml`.

Ver [MANUAL.md#configuración](MANUAL.md#configuración) para detalles.

## Estructura

```
conecta/
├── pkg/                # Lógica de negocio (reutilizable)
│   ├── network/        # Portal ETECSA, routing, speed
│   ├── hotspot/        # create_ap, clientes, NAT
│   ├── vpn/            # Gestión WireGuard
│   └── config/         # Configuración YAML
├── cmd/                # Frontends
│   ├── conecta/        # TUI principal
│   ├── hotspot/        # TUI de hotspot
│   └── cli/            # CLI
├── bin/                # Binarios compilados
├── MANUAL.md           # Documentación completa
└── config.yaml         # Config por defecto
```

## Desarrollo

```bash
# Ejecutar tests
go test ./...

# Compilar todo
go build ./...

# Ejecutar en desarrollo
go run ./cmd/conecta/
go run ./cmd/cli/ status
```

## Solución de Problemas

Ver [MANUAL.md#solución-de-problemas](MANUAL.md#solución-de-problemas).

## Omarchy Plugin

Conecta está disponible como plugin para Omarchy.

```bash
# Instalar plugin
cd ~/.local/share/omarchy/plugins/
git clone <repo-url> conecta

# Habilitar
omarchy plugin enable conecta.network
```

Ver [omarchy-plugin/README.md](omarchy-plugin/README.md) para detalles.

## Licencia

MIT

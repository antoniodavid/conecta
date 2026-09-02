# 📡 Conecta

Herramienta unificada para gestionar conexión a internet ETECSA, hotspot WiFi y VPN en Cuba.

## Características

- ✅ **Portal ETECSA**: Login, logout, auto-reconexión
- ✅ **Hotspot WiFi**: Compartir conexión vía create_ap
- ✅ **NAT/Enrutamiento**: Configurar forwarding para clientes
- ✅ **VPN**: Gestión de WireGuard
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

## Licencia

MIT

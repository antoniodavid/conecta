package main

const helpText = `
╔══════════════════════════════════════════════════════════════════╗
║                     📡 CONECTA - Ayuda                          ║
╚══════════════════════════════════════════════════════════════════╝

Uso: conecta-cli <comando> [opciones]

═══ COMANDOS PRINCIPALES ═════════════════════════════════════════

  status              Muestra el estado de la conexión
  login               Inicia sesión en el portal ETECSA
  logout              Cierra la sesión del portal
  speed               Ejecuta test de velocidad

═══ HOTSPOT ═════════════════════════════════════════════════════

  hotspot start       Inicia el hotspot WiFi
  hotspot stop        Detiene el hotspot WiFi
  hotspot status      Muestra estado del hotspot
  hotspot clients     Lista clientes conectados

═══ NAT (Enrutamiento) ══════════════════════════════════════════

  nat setup           Configura NAT para compartir conexión
  nat cleanup         Limpia reglas de NAT
  nat status          Muestra estado de NAT

═══ VPN ══════════════════════════════════════════════════════════

  vpn status          Muestra estado de la VPN
  vpn connect         Conecta la VPN
  vpn disconnect      Desconecta la VPN
  vpn toggle          Alterna estado de la VPN

═══ OPCIONES ════════════════════════════════════════════════════

  --user <usuario>    Usuario ETECSA (opcional si está en config)
  --pass <password>   Contraseña ETECSA (opcional si está en config)

═══ EJEMPLOS ════════════════════════════════════════════════════

  # Ver estado de conexión
  conecta-cli status

  # Login con credenciales del config
  conecta-cli login

  # Login con credenciales explícitas
  conecta-cli login --user 225823788119@nautaplus --pass xxx

  # Cerrar sesión
  conecta-cli logout

  # Test de velocidad
  conecta-cli speed

  # Iniciar hotspot
  conecta-cli hotspot start

  # Ver clientes conectados al hotspot
  configurar hotspot clients

  # Configurar NAT para compartir conexión
  conecta-cli nat setup

  # Conectar VPN
  conecta-cli vpn connect

  # Alternar VPN
  conecta-cli vpn toggle

═══ ARCHIVO DE CONFIGURACIÓN ════════════════════════════════════

  Ubicación: ~/.config/conecta/config.yaml

  Contiene:
  - Configuración de red (gateway, interfaz, portal)
  - Configuración de hotspot (SSID, contraseña, canal)
  - Configuración de VPN (interfaz, nombre)
  - Credenciales de acceso

═══ TUIs (Interfaces de Terminal) ═══════════════════════════════

  conecta             TUI principal (dashboard + hotspot + VPN)
  hotspot             TUI de hotspot (solo monitoreo)

═══ SOLUCIÓN DE PROBLEMAS ═══════════════════════════════════════

  Error: "Portal inalcanzable"
  → Verificar que el cable RJ45 esté conectado
  → Verificar que la interfaz esté activa: ip link show enp3s0

  Error: "Credenciales inválidas"
  → Verificar usuario y contraseña en ~/.config/conecta/config.yaml

  Error: "No se pudo iniciar hotspot"
  → Verificar que la interfaz WiFi esté disponible: iw dev
  → Verificar que create_ap esté instalado

  Hotspot activo pero sin internet
  → Configurar NAT: conecta-cli nat setup
  → Verificar IP forwarding: cat /proc/sys/net/ipv4/ip_forward

═══ ENLACES ═════════════════════════════════════════════════════

  Configuración:  ~/.config/conecta/config.yaml
  Documentación:  ~/conecta/README.md
  Código fuente:  ~/conecta/`

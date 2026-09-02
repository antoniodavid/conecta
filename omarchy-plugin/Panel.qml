import QtQuick 2.15
import QtQuick.Layouts 1.15
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui

Panel {
  id: root
  moduleName: "conecta.network"
  ipcTarget: "conecta.network"
  manageIpc: false

  property var anchorItem: null
  property bool cursorActive: false

  // Data
  property var status: null
  property bool isOn: false

  // Paths
  readonly property string pluginDir: Qt.resolvedUrl(".").toString().replace(/^file:\/\//, "")
  readonly property string statusScript: pluginDir + "/bin/omarchy-conecta-status"
  readonly property string hotspotScript: pluginDir + "/bin/omarchy-conecta-hotspot"
  readonly property string vpnScript: pluginDir + "/bin/omarchy-conecta-vpn"
  readonly property string loginScript: pluginDir + "/bin/omarchy-conecta-login"

  // Colors
  readonly property color foreground: bar ? bar.foreground : "#e0e0e0"
  readonly property color urgent: bar ? bar.urgent : "#ff4444"
  readonly property color dim: Qt.darker(foreground, 1.55)
  readonly property string fontFamily: bar ? bar.fontFamily : "monospace"

  // Settings
  readonly property bool showHotspot: (setting("showHotspot") ?? true) === true
  readonly property bool showVPN: (setting("showVPN") ?? true) === true
  readonly property bool showSpeedTest: (setting("showSpeedTest") ?? false) === true
  readonly property int refreshInterval: Number(setting("refreshIntervalSec") ?? 30)

  // ─── Status process ─────────────────────────────────────────────
  Process {
    id: statusProcess
    command: ["bash", root.statusScript]
    stdout: SplitParser {
      onRead: data => {
        try {
          root.status = JSON.parse(data)
          root.isOn = root.status.connection?.status === "connected" ||
                      root.status.connection?.status === "needs_auth"
        } catch (e) {}
      }
    }
  }

  // ─── Action processes ───────────────────────────────────────────
  Process { id: hotspotProcess
    stdout: SplitParser { onRead: data => root.refreshStatus() }
  }
  Process { id: vpnProcess
    stdout: SplitParser { onRead: data => root.refreshStatus() }
  }
  Process { id: loginProcess
    stdout: SplitParser { onRead: data => root.refreshStatus() }
  }

  // ─── Functions ──────────────────────────────────────────────────
  function getStatusIcon() {
    if (!isOn) return "🔴"
    if (status?.connection?.status === "needs_auth") return "🟠"
    if (status?.vpn?.connected) return "🔒"
    if (status?.hotspot?.active) return "📡"
    return "🟢"
  }

  function getStatusText() {
    if (!isOn) return "Disconnected"
    if (status?.connection?.status === "needs_auth") return "Auth needed"
    if (status?.vpn?.connected) return "VPN"
    if (status?.hotspot?.active) return "Hotspot"
    return "Connected"
  }

  function refreshStatus() {
    statusProcess.command = ["bash", root.statusScript]
    statusProcess.running = true
  }

  function runHotspot(action) {
    hotspotProcess.command = ["bash", root.hotspotScript, action]
    hotspotProcess.running = true
  }

  function runVPN(action) {
    vpnProcess.command = ["bash", root.vpnScript, action]
    vpnProcess.running = true
  }

  function runLogin(action) {
    loginProcess.command = ["bash", root.loginScript, action]
    loginProcess.running = true
  }

  function open() {
    root.controller.show()
    refreshStatus()
  }

  function close() {
    root.controller.hide()
  }

  function toggle() {
    if (root.opened) close(); else open()
  }

  // ─── Bar widget ─────────────────────────────────────────────────
  bar: RowLayout {
    spacing: 6

    Text {
      text: getStatusIcon()
      font.family: root.fontFamily
      font.pixelSize: 14
    }

    Text {
      text: getStatusText()
      font.family: root.fontFamily
      font.pixelSize: 12
      color: root.foreground
    }

    Text {
      text: {
        if (status?.hotspot?.active) return status.hotspot.ssid
        if (status?.vpn?.connected) return status.vpn.ip
        return ""
      }
      visible: text !== ""
      font.family: root.fontFamily
      font.pixelSize: 10
      color: root.dim
    }
  }

  // ─── Popup panel ────────────────────────────────────────────────
  popup: PanelPopup {
    implicitWidth: 280
    implicitHeight: content.height + 30

    ColumnLayout {
      id: content
      anchors.fill: parent
      anchors.margins: 15
      spacing: 10

      // Header
      Text {
        text: "📡 Conecta"
        font.family: root.fontFamily
        font.pixelSize: 16
        font.bold: true
        color: root.foreground
        Layout.fillWidth: true
      }

      Rectangle { Layout.fillWidth: true; height: 1; color: "#333" }

      // Connection
      Text {
        text: "Connection"
        font.family: root.fontFamily
        font.pixelSize: 12
        font.bold: true
        color: root.foreground
        Layout.fillWidth: true
      }

      RowLayout {
        Layout.fillWidth: true
        Text {
          text: "Status:"
          font.family: root.fontFamily
          font.pixelSize: 12
          color: root.dim
          Layout.fillWidth: true
        }
        Text {
          text: status?.connection?.status || "unknown"
          font.family: root.fontFamily
          font.pixelSize: 12
          font.bold: true
          color: status?.connection?.status === "connected" ? "#00ff88" :
                 status?.connection?.status === "needs_auth" ? "#ffcc00" : "#ff4444"
        }
      }

      // Login / Logout buttons
      RowLayout {
        Layout.fillWidth: true
        spacing: 8
        visible: status?.connection?.status === "needs_auth"

        PanelButton {
          text: "Login"
          Layout.fillWidth: true
          onClicked: runLogin("login")
        }
      }

      RowLayout {
        Layout.fillWidth: true
        spacing: 8
        visible: status?.connection?.status === "connected"

        PanelButton {
          text: "Logout"
          Layout.fillWidth: true
          onClicked: runLogin("logout")
        }
      }

      // Separator
      Rectangle { Layout.fillWidth: true; height: 1; color: "#333"; visible: root.showHotspot }

      // Hotspot
      ColumnLayout {
        spacing: 6
        visible: root.showHotspot

        Text {
          text: "Hotspot"
          font.family: root.fontFamily
          font.pixelSize: 12
          font.bold: true
          color: root.foreground
          Layout.fillWidth: true
        }

        RowLayout {
          Layout.fillWidth: true
          Text {
            text: "Status:"
            font.family: root.fontFamily
            font.pixelSize: 12
            color: root.dim
            Layout.fillWidth: true
          }
          Text {
            text: status?.hotspot?.active ? "Active" : "Inactive"
            font.family: root.fontFamily
            font.pixelSize: 12
            font.bold: true
            color: status?.hotspot?.active ? "#00ff88" : "#666"
          }
        }

        RowLayout {
          Layout.fillWidth: true
          visible: status?.hotspot?.active
          Text {
            text: "Clients:"
            font.family: root.fontFamily
            font.pixelSize: 12
            color: root.dim
            Layout.fillWidth: true
          }
          Text {
            text: (status?.hotspot?.clients ?? 0).toString()
            font.family: root.fontFamily
            font.pixelSize: 12
            color: root.foreground
          }
        }

        PanelButton {
          text: status?.hotspot?.active ? "Stop Hotspot" : "Start Hotspot"
          Layout.fillWidth: true
          onClicked: runHotspot(status?.hotspot?.active ? "stop" : "start")
        }
      }

      // Separator
      Rectangle { Layout.fillWidth: true; height: 1; color: "#333"; visible: root.showVPN }

      // VPN
      ColumnLayout {
        spacing: 6
        visible: root.showVPN

        Text {
          text: "VPN"
          font.family: root.fontFamily
          font.pixelSize: 12
          font.bold: true
          color: root.foreground
          Layout.fillWidth: true
        }

        RowLayout {
          Layout.fillWidth: true
          Text {
            text: "Status:"
            font.family: root.fontFamily
            font.pixelSize: 12
            color: root.dim
            Layout.fillWidth: true
          }
          Text {
            text: status?.vpn?.connected ? "Connected" : "Disconnected"
            font.family: root.fontFamily
            font.pixelSize: 12
            font.bold: true
            color: status?.vpn?.connected ? "#00ff88" : "#666"
          }
        }

        PanelButton {
          text: status?.vpn?.connected ? "Disconnect VPN" : "Connect VPN"
          Layout.fillWidth: true
          onClicked: runVPN("toggle")
        }
      }

      // Speed Test
      PanelButton {
        text: "Speed Test"
        Layout.fillWidth: true
        visible: root.showSpeedTest
        onClicked: runLogin("speed")
      }
    }
  }

  // ─── Init ───────────────────────────────────────────────────────
  Component.onCompleted: refreshStatus()

  Timer {
    interval: root.refreshInterval * 1000
    running: true
    repeat: true
    onTriggered: root.refreshStatus()
  }
}

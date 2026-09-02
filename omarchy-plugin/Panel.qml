import QtQuick 2.15
import QtQuick.Layouts 1.15
import "../../libs/omarchy/components" as Omarchy

Omarchy.PluggableWidget {
  id: root

  property var status: ({})
  property bool isOn: false

  function updateStatus(json) {
    try {
      status = JSON.parse(json)
      isOn = status.connection?.status === "connected" ||
             status.connection?.status === "needs_auth"
    } catch (e) {
      isOn = false
    }
  }

  function getStatusIcon() {
    if (!isOn) return "🔴"
    if (status.connection?.status === "needs_auth") return "🟠"
    if (status.vpn?.connected) return "🔒"
    if (status.hotspot?.active) return "📡"
    return "🟢"
  }

  function getStatusText() {
    if (!isOn) return "Disconnected"
    if (status.connection?.status === "needs_auth") return "Auth needed"
    if (status.vpn?.connected) return "VPN"
    if (status.hotspot?.active) return "Hotspot"
    return "Connected"
  }

  panel: Omarchy.SmartPanel {
    implicitWidth: 180

    contentItem: RowLayout {
      spacing: 6

      Omarchy.PanelIcon {
        icon: getStatusIcon()
      }

      ColumnLayout {
        spacing: 0

        Omarchy.PanelLabel {
          text: getStatusText()
          font.bold: true
        }

        Omarchy.PanelLabel {
          text: {
            if (status.hotspot?.active) return status.hotspot.ssid
            if (status.vpn?.connected) return status.vpn.ip
            return ""
          }
          visible: text !== ""
          font.pixelSize: 10
          opacity: 0.7
        }
      }
    }

    onClicked: popup.open()
  }

  popup: Omarchy.PluggablePopup {
    width: 280
    height: content.height + 40

    ColumnLayout {
      id: content
      anchors.fill: parent
      anchors.margins: 15
      spacing: 12

      // Header
      Omarchy.PanelLabel {
        text: "📡 Conecta"
        font.pixelSize: 16
        font.bold: true
        Layout.fillWidth: true
      }

      // Connection Status
      Rectangle {
        Layout.fillWidth: true
        height: 3
        color: "transparent"
      }

      Omarchy.PanelLabel {
        text: "Connection"
        font.bold: true
        Layout.fillWidth: true
      }

      RowLayout {
        Layout.fillWidth: true

        Omarchy.PanelLabel {
          text: "Status:"
          Layout.fillWidth: true
        }

        Omarchy.PanelLabel {
          text: status.connection?.status || "unknown"
          font.bold: true
          color: status.connection?.status === "connected" ? "#00ff88" :
                 status.connection?.status === "needs_auth" ? "#ffcc00" : "#ff4444"
        }
      }

      // Login/Logout buttons
      RowLayout {
        Layout.fillWidth: true
        spacing: 8

        Omarchy.PanelButton {
          text: "Login"
          Layout.fillWidth: true
          visible: status.connection?.status === "needs_auth"
          onClicked: {
            root.exec("omarchy-conecta-login", ["login"])
            root.refresh()
          }
        }

        Omarchy.PanelButton {
          text: "Logout"
          Layout.fillWidth: true
          visible: status.connection?.status === "connected"
          onClicked: {
            root.exec("omarchy-conecta-login", ["logout"])
            root.refresh()
          }
        }
      }

      // Separator
      Rectangle {
        Layout.fillWidth: true
        height: 1
        color: "#333"
      }

      // Hotspot
      Omarchy.PanelLabel {
        text: "Hotspot"
        font.bold: true
        Layout.fillWidth: true
        visible: root.config.showHotspot ?? true
      }

      RowLayout {
        Layout.fillWidth: true
        visible: root.config.showHotspot ?? true

        Omarchy.PanelLabel {
          text: "Status:"
          Layout.fillWidth: true
        }

        Omarchy.PanelLabel {
          text: status.hotspot?.active ? "Active" : "Inactive"
          font.bold: true
          color: status.hotspot?.active ? "#00ff88" : "#666"
        }
      }

      RowLayout {
        Layout.fillWidth: true
        visible: status.hotspot?.active && (root.config.showHotspot ?? true)

        Omarchy.PanelLabel {
          text: "Clients:"
          Layout.fillWidth: true
        }

        Omarchy.PanelLabel {
          text: status.hotspot?.clients?.toString() || "0"
        }
      }

      RowLayout {
        Layout.fillWidth: true
        spacing: 8
        visible: root.config.showHotspot ?? true

        Omarchy.PanelButton {
          text: status.hotspot?.active ? "Stop" : "Start"
          Layout.fillWidth: true
          onClicked: {
            root.exec("omarchy-conecta-hotspot", [status.hotspot?.active ? "stop" : "start"])
            root.refresh()
          }
        }
      }

      // Separator
      Rectangle {
        Layout.fillWidth: true
        height: 1
        color: "#333"
        visible: root.config.showVPN ?? true
      }

      // VPN
      Omarchy.PanelLabel {
        text: "VPN"
        font.bold: true
        Layout.fillWidth: true
        visible: root.config.showVPN ?? true
      }

      RowLayout {
        Layout.fillWidth: true
        visible: root.config.showVPN ?? true

        Omarchy.PanelLabel {
          text: "Status:"
          Layout.fillWidth: true
        }

        Omarchy.PanelLabel {
          text: status.vpn?.connected ? "Connected" : "Disconnected"
          font.bold: true
          color: status.vpn?.connected ? "#00ff88" : "#666"
        }
      }

      RowLayout {
        Layout.fillWidth: true
        spacing: 8
        visible: root.config.showVPN ?? true

        Omarchy.PanelButton {
          text: status.vpn?.connected ? "Disconnect" : "Connect"
          Layout.fillWidth: true
          onClicked: {
            root.exec("omarchy-conecta-vpn", ["toggle"])
            root.refresh()
          }
        }
      }

      // Speed Test
      RowLayout {
        Layout.fillWidth: true
        visible: root.config.showSpeedTest ?? false

        Omarchy.PanelButton {
          text: "Speed Test"
          Layout.fillWidth: true
          onClicked: {
            root.exec("omarchy-conecta-login", ["speed"])
          }
        }
      }
    }
  }

  Component.onCompleted: refresh()

  Timer {
    interval: (root.config.refreshIntervalSec ?? 30) * 1000
    running: true
    repeat: true
    onTriggered: root.refresh()
  }

  function refresh() {
    root.exec("omarchy-conecta-status", [], updateStatus)
  }
}

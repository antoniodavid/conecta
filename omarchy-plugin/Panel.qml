import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.Ui
import qs.Commons

Panel {
  id: root
  Component.onCompleted: {
    console.log("CONECTA DEBUG: Panel instantiated, moduleName=" + moduleName)
    console.log("CONECTA DEBUG: button implicitWidth=" + (typeof button !== "undefined" ? button.implicitWidth : "undefined"))
    refreshStatus()
  }
  moduleName: "conecta.network"
  ipcTarget: "conecta.network"
  manageIpc: false

  property var anchorItem: null
  property bool cursorActive: false

  // Data
  property var status: null
  property bool isOn: false
  // Buffered process output: SplitParser delivers line chunks, so chunks
  // accumulate until one complete JSON document parses.
  property string statusBuffer: ""
  property string actionBuffer: ""
  // Last action/speed result shown in the popup.
  property string actionText: ""

  // Paths
  readonly property string pluginDir: Qt.resolvedUrl(".").toString().replace(/^file:\/\//, "")
  readonly property string statusScript: pluginDir + "/bin/omarchy-conecta-status"
  readonly property string hotspotScript: pluginDir + "/bin/omarchy-conecta-hotspot"
  readonly property string vpnScript: pluginDir + "/bin/omarchy-conecta-vpn"
  readonly property string loginScript: pluginDir + "/bin/omarchy-conecta-login"

  // Colors
  readonly property color foreground: bar ? bar.barForeground : Color.foreground
  readonly property color urgent: bar ? bar.urgent : Color.urgent
  readonly property color dim: Qt.darker(foreground, 1.55)
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  // Settings
  readonly property bool showHotspot: (setting("showHotspot", true)) === true
  readonly property bool showVPN: (setting("showVPN", true)) === true
  readonly property bool showSpeedTest: (setting("showSpeedTest", false)) === true
  readonly property int refreshInterval: Number(setting("refreshIntervalSec", 30))

  // ─── Icon functions (nerd font like network plugin) ─────────────
  function getStatusIcon() {
    if (!isOn) return "󰤮"  // Disconnected
    if (status?.connection?.status === "needs_auth") return "󰤯"  // Needs auth
    if (status?.vpn?.connected) return "󰲝"  // VPN connected
    if (status?.hotspot?.active) return "󰤨"  // Hotspot active
    // WiFi connected - use signal strength
    return getWifiIcon(status?.connection?.signal ?? 0)
  }

  function getWifiIcon(signalStrength) {
    // Same as network plugin: wifiIconFor
    var icons = ["󰤯", "󰤟", "󰤢", "󰤥", "󰤨"]
    var index = Math.max(0, Math.min(4, Math.ceil(signalStrength / 20) - 1))
    return icons[index]
  }

  function getStatusText() {
    if (!isOn) return "Disconnected"
    if (status?.connection?.status === "needs_auth") return "Auth needed"
    if (status?.vpn?.connected) return "VPN"
    if (status?.hotspot?.active) return "Hotspot"
    return status?.connection?.ssid || "Connected"
  }

  // ─── Status process ─────────────────────────────────────────────
  Process {
    id: statusProcess
    command: ["bash", root.statusScript]
    stdout: SplitParser {
      onRead: data => {
        var fed = root.feedJson(root.statusBuffer, data);
        root.statusBuffer = fed.rest;
        if (fed.doc) root.applyStatus(fed.doc);
      }
    }
  }

  // ─── Action processes ───────────────────────────────────────────
  Process {
    id: hotspotProcess
    stdout: SplitParser { onRead: data => root.handleAction(data, false) }
  }
  Process {
    id: vpnProcess
    stdout: SplitParser { onRead: data => root.handleAction(data, false) }
  }
  Process {
    id: loginProcess
    stdout: SplitParser { onRead: data => root.handleAction(data, true) }
  }

  // ─── Functions ──────────────────────────────────────────────────
  // feedJson appends a SplitParser chunk and returns the parsed document
  // once the buffer holds one complete document; otherwise returns null
  // and keeps buffering (capped so garbage never grows unbounded).
  function feedJson(current, chunk) {
    var buf = current + chunk;
    try {
      return { doc: JSON.parse(buf), rest: "" };
    } catch (e) {
      if (buf.length > 65536) return { doc: null, rest: "" };
      return { doc: null, rest: buf };
    }
  }

  function applyStatus(doc) {
    if (doc && doc.ok === false) {
      root.actionText = (doc.error && doc.error.message) || "request failed";
      root.isOn = false;
      return;
    }
    root.status = doc;
    root.isOn = doc.connection?.status === "connected" ||
                doc.connection?.status === "needs_auth";
  }

  // handleAction stores a human-readable action/speed result, then refreshes.
  function handleAction(chunk, isSpeed) {
    var fed = root.feedJson(root.actionBuffer, chunk);
    root.actionBuffer = fed.rest;
    if (!fed.doc) return;
    var doc = fed.doc;
    if (doc.ok) {
      if (isSpeed && doc.data && doc.data.display)
        root.actionText = "Speed: " + doc.data.display;
      else
        root.actionText = "Done";
    } else {
      root.actionText = "Failed: " + ((doc.error && doc.error.message) || "unknown error");
    }
    root.refreshStatus();
  }

  function refreshStatus() {
    root.statusBuffer = "";
    statusProcess.command = ["bash", root.statusScript];
    statusProcess.running = true;
  }

  function runHotspot(action) {
    root.actionBuffer = "";
    hotspotProcess.command = ["bash", root.hotspotScript, action];
    hotspotProcess.running = true;
  }

  function runVPN(action) {
    root.actionBuffer = "";
    vpnProcess.command = ["bash", root.vpnScript, action];
    vpnProcess.running = true;
  }

  function runLogin(action) {
    root.actionBuffer = "";
    loginProcess.command = ["bash", root.loginScript, action];
    loginProcess.running = true;
  }

  // ─── Bar content ────────────────────────────────────────────────
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: root.getStatusIcon()
    active: root.isOn
    onPressed: function(buttonCode) {
      if (buttonCode === Qt.LeftButton) root.toggle()
    }
  }

  // ─── Popup content ──────────────────────────────────────────────
  KeyboardPanel {
    id: panel
    anchorItem: button
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(380))
    contentHeight: panel.fittedContentHeight(contentColumn.implicitHeight, Style.space(560))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent

      onCloseRequested: root.close()
    }

    Flickable {
      id: panelFlick
      anchors.fill: parent
      contentWidth: width
      contentHeight: contentColumn.implicitHeight
      clip: true
      boundsBehavior: Flickable.StopAtBounds
      flickableDirection: Flickable.VerticalFlick
      interactive: contentHeight > height
      ScrollBar.vertical: ScrollBar { policy: ScrollBar.AsNeeded }

      Column {
        id: contentColumn
        width: panelFlick.width
        spacing: Style.space(12)

        // Header
        PanelHero {
          width: parent.width
          title: root.getStatusIcon() + " Conecta"
          meta: root.getStatusText()
          foreground: root.foreground
          fontFamily: root.fontFamily
        }

        // Last action/speed result
        Text {
          width: parent.width
          visible: root.actionText !== ""
          text: root.actionText
          font.family: root.fontFamily
          font.pixelSize: 12
          color: root.actionText.startsWith("Failed") ? "#ff4444" : "#00d4ff"
          wrapMode: Text.Wrap
        }

        // Connection section
        Column {
          width: parent.width
          spacing: Style.space(8)

          PanelSectionHeader {
            width: parent.width
            text: "Connection"
          }

          // Status row
          Row {
            width: parent.width
            spacing: Style.space(8)

            Text {
              text: "Status:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
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

          // SSID row
          Row {
            width: parent.width
            spacing: Style.space(8)
            visible: status !== null && status.connection !== undefined && status.connection.ssid !== undefined && status.connection.ssid !== ""

            Text {
              text: "SSID:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: (status !== null && status.connection !== undefined) ? status.connection.ssid || "N/A" : "N/A"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.foreground
            }
          }

          // Signal row
          Row {
            width: parent.width
            spacing: Style.space(8)
            visible: status !== null && status.connection !== undefined && status.connection.signal !== undefined && status.connection.signal !== ""

            Text {
              text: "Signal:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: (status !== null && status.connection !== undefined && status.connection.signal !== undefined) ? status.connection.signal + "%" : "N/A"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.foreground
            }
          }

          // Login button (for captive portal)
          Rectangle {
            width: parent.width
            height: 32
            color: "#2a2a2a"
            radius: 4
            visible: status !== null && status.connection !== undefined && status.connection.status === "needs_auth"

            Row {
              anchors.centerIn: parent
              spacing: 8

              Text {
                text: "󰦕"  // Login icon
                font.family: root.fontFamily
                font.pixelSize: 14
                color: "#00ff88"
              }

              Text {
                text: "Login to Portal"
                font.family: root.fontFamily
                font.pixelSize: 12
                color: "#00ff88"
              }
            }

            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              onClicked: root.runLogin("login")
            }
          }

          // Logout button
          Rectangle {
            width: parent.width
            height: 32
            color: "#2a2a2a"
            radius: 4
            visible: status !== null && status.connection !== undefined && status.connection.status === "connected"

            Row {
              anchors.centerIn: parent
              spacing: 8

              Text {
                text: "󰍃"  // Logout icon
                font.family: root.fontFamily
                font.pixelSize: 14
                color: "#ff4444"
              }

              Text {
                text: "Logout"
                font.family: root.fontFamily
                font.pixelSize: 12
                color: "#ff4444"
              }
            }

            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              onClicked: root.runLogin("logout")
            }
          }
        }

        // Hotspot section
        Column {
          width: parent.width
          spacing: Style.space(8)
          visible: root.showHotspot

          PanelSectionHeader {
            width: parent.width
            text: "Hotspot"
          }

          // Hotspot status
          Row {
            width: parent.width
            spacing: Style.space(8)

            Text {
              text: "Status:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: status?.hotspot?.active ? "Active" : "Inactive"
              font.family: root.fontFamily
              font.pixelSize: 12
              font.bold: true
              color: status?.hotspot?.active ? "#00ff88" : "#666"
            }
          }

          // Hotspot SSID
          Row {
            width: parent.width
            spacing: Style.space(8)
            visible: status !== null && status.hotspot !== undefined && status.hotspot.active === true

            Text {
              text: "SSID:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: (status !== null && status.hotspot !== undefined) ? status.hotspot.ssid || "N/A" : "N/A"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.foreground
            }
          }

          // Hotspot clients
          Row {
            width: parent.width
            spacing: Style.space(8)
            visible: status !== null && status.hotspot !== undefined && status.hotspot.active === true

            Text {
              text: "Clients:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: (status?.hotspot?.clients ?? 0).toString()
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.foreground
            }
          }

          // Hotspot toggle button
          Rectangle {
            width: parent.width
            height: 32
            color: "#2a2a2a"
            radius: 4

            Row {
              anchors.centerIn: parent
              spacing: 8

              Text {
                text: status?.hotspot?.active ? "󰦞" : "󰝨"  // Stop/Start icon
                font.family: root.fontFamily
                font.pixelSize: 14
                color: status?.hotspot?.active ? "#ff4444" : "#00ff88"
              }

              Text {
                text: status?.hotspot?.active ? "Stop Hotspot" : "Start Hotspot"
                font.family: root.fontFamily
                font.pixelSize: 12
                color: status?.hotspot?.active ? "#ff4444" : "#00ff88"
              }
            }

            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              onClicked: root.runHotspot(status?.hotspot?.active ? "stop" : "start")
            }
          }
        }

        // VPN section
        Column {
          width: parent.width
          spacing: Style.space(8)
          visible: root.showVPN

          PanelSectionHeader {
            width: parent.width
            text: "VPN"
          }

          // VPN status
          Row {
            width: parent.width
            spacing: Style.space(8)

            Text {
              text: "Status:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: status?.vpn?.connected ? "Connected" : "Disconnected"
              font.family: root.fontFamily
              font.pixelSize: 12
              font.bold: true
              color: status?.vpn?.connected ? "#00ff88" : "#666"
            }
          }

          // VPN IP
          Row {
            width: parent.width
            spacing: Style.space(8)
            visible: status !== null && status.vpn !== undefined && status.vpn.connected === true

            Text {
              text: "IP:"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.dim
              width: 80
            }

            Text {
              text: (status !== null && status.vpn !== undefined) ? status.vpn.ip || "N/A" : "N/A"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: root.foreground
            }
          }

          // VPN toggle button
          Rectangle {
            width: parent.width
            height: 32
            color: "#2a2a2a"
            radius: 4

            Row {
              anchors.centerIn: parent
              spacing: 8

              Text {
                text: status?.vpn?.connected ? "󰦴" : "󰲜"  // Disconnect/Connect icon
                font.family: root.fontFamily
                font.pixelSize: 14
                color: status?.vpn?.connected ? "#ff4444" : "#00ff88"
              }

              Text {
                text: status?.vpn?.connected ? "Disconnect VPN" : "Connect VPN"
                font.family: root.fontFamily
                font.pixelSize: 12
                color: status?.vpn?.connected ? "#ff4444" : "#00ff88"
              }
            }

            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              onClicked: root.runVPN("toggle")
            }
          }
        }

        // Speed Test button
        Rectangle {
          width: parent.width
          height: 32
          color: "#2a2a2a"
          radius: 4
          visible: root.showSpeedTest

          Row {
            anchors.centerIn: parent
            spacing: 8

            Text {
              text: "󰄀"  // Speed icon
              font.family: root.fontFamily
              font.pixelSize: 14
              color: "#00d4ff"
            }

            Text {
              text: "Speed Test"
              font.family: root.fontFamily
              font.pixelSize: 12
              color: "#00d4ff"
            }
          }

          MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: root.runLogin("speed")
          }
        }
      }
    }
  }

  // ─── Init (moved to top with debug) ────────────────────────────

  Timer {
    interval: root.refreshInterval * 1000
    running: true
    repeat: true
    onTriggered: root.refreshStatus()
  }
}

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
  // Which login-family action is in flight (login, logout, speed), so busy
  // labels on the shared loginProcess stay specific.
  property string loginAction: ""

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

  // Signal-station palette: blue-tinted graphite, one mint accent for
  // healthy, amber for auth-needed, red reserved for failure.
  readonly property color stationCard: "#151c27"
  readonly property color stationEdge: "#263142"
  readonly property color stationFaint: "#8b98ad"
  readonly property color mint: "#4de3c2"
  readonly property color amber: "#f5b544"
  readonly property color failure: "#ff5d5d"
  readonly property color mintWash: "#14332b"
  readonly property color redWash: "#3a1d20"
  readonly property color amberWash: "#3a2c10"
  readonly property int cardRadius: 8
  readonly property int btnHeight: 34

  // Settings
  readonly property bool showHotspot: (setting("showHotspot", true)) === true
  readonly property bool showVPN: (setting("showVPN", true)) === true
  readonly property bool showSpeedTest: (setting("showSpeedTest", false)) === true
  readonly property int refreshInterval: Number(setting("refreshIntervalSec", 30))

  // ─── Icon functions (nerd font like network plugin) ─────────────
  function getStatusIcon() {
    if (!isOn) return "󰤮"  // Disconnected
    if ((status && status.connection && status.connection.status) === "needs_auth") return "󰤯"  // Needs auth
    if ((status && status.vpn && status.vpn.connected)) return "󰲝"  // VPN connected
    if ((status && status.hotspot && status.hotspot.active)) return "󰤨"  // Hotspot active
    // WiFi connected - use signal strength
    return getWifiIcon((status && status.connection && status.connection.signal !== undefined && status.connection.signal !== null ? status.connection.signal : 0))
  }

  function getWifiIcon(signalStrength) {
    // Same as network plugin: wifiIconFor
    var icons = ["󰤯", "󰤟", "󰤢", "󰤥", "󰤨"]
    var index = Math.max(0, Math.min(4, Math.ceil(signalStrength / 20) - 1))
    return icons[index]
  }

  function getStatusText() {
    if (!isOn) return "Disconnected"
    if ((status && status.connection && status.connection.status) === "needs_auth") return "Auth needed"
    if ((status && status.vpn && status.vpn.connected)) return "VPN"
    if ((status && status.hotspot && status.hotspot.active)) return "Hotspot"
    return (status && status.connection && status.connection.ssid) || "Connected"
  }

  // ─── Null-safe status helpers (no optional chaining: qmllint rejects it)
  function connKey() {
    if (root.status === null || root.status === undefined) return ""
    var c = root.status.connection
    if (c === undefined || c === null) return ""
    if (c.status === undefined || c.status === null) return ""
    return c.status
  }

  function hotspotActive() {
    if (root.status === null || root.status === undefined) return false
    var h = root.status.hotspot
    if (h === undefined || h === null) return false
    return h.active === true
  }

  function vpnConnected() {
    if (root.status === null || root.status === undefined) return false
    var v = root.status.vpn
    if (v === undefined || v === null) return false
    return v.connected === true
  }

  function sigValue() {
    if (root.status === null || root.status === undefined) return 0
    var c = root.status.connection
    if (c === undefined || c === null) return 0
    if (c.signal === undefined || c.signal === null || c.signal === "") return 0
    return Number(c.signal)
  }

  function sigBars() {
    if (!root.isOn) return 0
    var s = root.sigValue()
    if (s <= 0) return 0
    return Math.max(1, Math.min(4, Math.ceil(s / 25)))
  }

  function sigColor() {
    var k = root.connKey()
    if (k === "connected") return root.mint
    if (k === "needs_auth") return root.amber
    if (k === "") return root.stationFaint
    return root.failure
  }

  function connLabel() {
    var k = root.connKey()
    if (k === "") return "Waiting for first read"
    if (k === "needs_auth") return "Login required"
    if (k === "connected") {
      var s = root.teleSsid()
      return s === "—" ? "Connected" : s
    }
    return k
  }

  function connPill() {
    if (root.status === null || root.status === undefined)
      return statusProcess.running ? "Reading" : "No data"
    var k = root.connKey()
    if (k === "connected") return "Online"
    if (k === "needs_auth") return "Login needed"
    return "Offline"
  }

  function connPillFg() {
    var k = root.connKey()
    if (k === "connected") return root.mint
    if (k === "needs_auth") return root.amber
    if (k === "") return root.stationFaint
    return root.failure
  }

  function connPillBg() {
    var k = root.connKey()
    if (k === "connected") return root.mintWash
    if (k === "needs_auth") return root.amberWash
    if (k === "") return root.stationEdge
    return root.redWash
  }

  function heroSubline() {
    if (root.status === null || root.status === undefined)
      return statusProcess.running ? "Reading status from adapters…" : "Press Refresh status to read the link."
    var parts = root.teleSsid()
    if (root.isOn) parts = parts + "  ·  " + root.sigValue() + "%"
    var gw = root.teleGateway()
    if (gw !== "—") parts = parts + "  ·  gw " + gw
    if (statusProcess.running) parts = parts + "  ·  updating…"
    return parts
  }

  function teleSsid() {
    if (root.status === null || root.status === undefined) return "—"
    var c = root.status.connection
    if (c === undefined || c === null) return "—"
    if (c.ssid === undefined || c.ssid === null || c.ssid === "") return "—"
    return c.ssid
  }

  function teleSignal() {
    if (root.status === null || root.status === undefined) return "—"
    return root.sigValue() + "%"
  }

  function teleGateway() {
    if (root.status === null || root.status === undefined) return "—"
    var c = root.status.connection
    if (c === undefined || c === null) return "—"
    if (c.gateway === undefined || c.gateway === null || c.gateway === "") return "—"
    return c.gateway
  }

  function teleIface() {
    if (root.status === null || root.status === undefined) return "—"
    var c = root.status.connection
    if (c === undefined || c === null) return "—"
    if (c.interface === undefined || c.interface === null || c.interface === "") return "—"
    return c.interface
  }

  function teleVpnIp() {
    if (!root.vpnConnected()) return "—"
    var v = root.status.vpn
    if (v.ip === undefined || v.ip === null || v.ip === "") return "—"
    return v.ip
  }

  function teleClients() {
    if (!root.hotspotActive()) return "—"
    var h = root.status.hotspot
    if (h.clients === undefined || h.clients === null) return "—"
    return h.clients.toString()
  }

  function hotspotDetail() {
    if (!root.hotspotActive()) return "Share this link over Wi-Fi."
    var s = root.status.hotspot.ssid
    var name = (s === undefined || s === null || s === "") ? "hotspot" : s
    return name + "  ·  " + root.teleClients() + " client(s)"
  }

  function vpnDetail() {
    if (!root.vpnConnected()) return "Tunnel is down; traffic goes direct."
    var ip = root.teleVpnIp()
    return ip === "—" ? "Tunnel is up." : "Exit IP " + ip
  }

  function updatedText() {
    if (root.status === null || root.status === undefined) return ""
    var t = root.status.timestamp
    if (t === undefined || t === null || t === "") return ""
    return "Updated " + t
  }

  function isSpeedResult() {
    return root.actionText !== "" && root.actionText.indexOf("Speed:") === 0
  }

  function isFailResult() {
    return root.actionText !== "" && root.actionText.indexOf("Failed") === 0
  }

  function bannerHint() {
    if (root.isFailResult()) return "Check the message, then retry the same action."
    if (root.actionText === "Done") return "Finished — status was re-read automatically."
    return "Finished."
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
    root.isOn = (doc.connection && doc.connection.status) === "connected" ||
                (doc.connection && doc.connection.status) === "needs_auth";
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
    root.loginAction = action;
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

        // Signal-station hero: live meter next to the connection label.
        Rectangle {
          width: parent.width
          height: 76
          radius: root.cardRadius
          color: root.stationCard
          border.color: root.stationEdge
          border.width: 1

          Row {
            anchors.fill: parent
            anchors.leftMargin: 14
            anchors.rightMargin: 14
            spacing: 14

            // Signal meter: four ascending bars from plain rectangles,
            // lit count driven by the real signal value.
            Row {
              spacing: 3
              anchors.verticalCenter: parent.verticalCenter

              Repeater {
                model: 4
                Item {
                  width: 7
                  height: 28
                  Rectangle {
                    width: 7
                    height: 10 + index * 6
                    radius: 2
                    anchors.bottom: parent.bottom
                    color: index < root.sigBars() ? root.sigColor() : root.stationEdge
                  }
                }
              }
            }

            Column {
              width: parent.width - 120
              anchors.verticalCenter: parent.verticalCenter
              spacing: 3

              Text {
                width: parent.width
                text: root.connLabel()
                font.family: root.fontFamily
                font.pixelSize: 16
                font.bold: true
                color: root.foreground
                elide: Text.ElideRight
              }

              Text {
                width: parent.width
                text: root.heroSubline()
                font.family: root.fontFamily
                font.pixelSize: 11
                color: root.stationFaint
                elide: Text.ElideRight
              }
            }

            Rectangle {
              width: pillLabel.implicitWidth + 20
              height: 22
              radius: 11
              anchors.verticalCenter: parent.verticalCenter
              color: root.connPillBg()

              Text {
                id: pillLabel
                anchors.centerIn: parent
                text: root.connPill()
                font.family: root.fontFamily
                font.pixelSize: 11
                font.bold: true
                color: root.connPillFg()
              }
            }
          }
        }

        // Last action result (speed results live persistently in the
        // speed card below, so the banner only handles Done/Failed).
        Rectangle {
          width: parent.width
          radius: root.cardRadius
          height: bannerColumn.implicitHeight + 20
          visible: root.actionText !== "" && !root.isSpeedResult()
          opacity: visible ? 1 : 0
          color: root.stationCard
          border.color: root.isFailResult() ? root.failure : root.mint
          border.width: 1

          Behavior on opacity { NumberAnimation { duration: 120 } }

          Column {
            id: bannerColumn
            x: 12
            y: 10
            width: parent.width - 24
            spacing: 3

            Text {
              width: parent.width
              text: root.actionText
              font.family: root.fontFamily
              font.pixelSize: 12
              font.bold: true
              color: root.isFailResult() ? root.failure : root.mint
              wrapMode: Text.Wrap
            }

            Text {
              width: parent.width
              text: root.bannerHint()
              font.family: root.fontFamily
              font.pixelSize: 11
              color: root.stationFaint
              wrapMode: Text.Wrap
            }
          }
        }

        // Telemetry grid
        Column {
          width: parent.width
          spacing: Style.space(8)

          PanelSectionHeader {
            width: parent.width
            text: "Telemetry"
          }

          GridLayout {
            width: parent.width
            columns: 2
            columnSpacing: Style.space(8)
            rowSpacing: Style.space(8)

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "Network"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleSsid(); font.family: root.fontFamily; font.pixelSize: 13; font.bold: true; color: root.foreground; elide: Text.ElideRight; width: parent.width }
            }

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "Signal"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleSignal(); font.family: root.fontFamily; font.pixelSize: 13; font.bold: true; color: root.sigColor() }
            }

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "Gateway"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleGateway(); font.family: root.fontFamily; font.pixelSize: 13; color: root.foreground; elide: Text.ElideRight; width: parent.width }
            }

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "Interface"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleIface(); font.family: root.fontFamily; font.pixelSize: 13; color: root.foreground; elide: Text.ElideRight; width: parent.width }
            }

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "VPN exit IP"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleVpnIp(); font.family: root.fontFamily; font.pixelSize: 13; color: root.foreground; elide: Text.ElideRight; width: parent.width }
            }

            Column {
              Layout.fillWidth: true
              spacing: 1
              Text { text: "Hotspot clients"; font.family: root.fontFamily; font.pixelSize: 10; color: root.stationFaint }
              Text { text: root.teleClients(); font.family: root.fontFamily; font.pixelSize: 13; color: root.foreground }
            }
          }

          Text {
            width: parent.width
            visible: root.status === null || root.status === undefined
            text: "No readings yet — press Refresh status below."
            font.family: root.fontFamily
            font.pixelSize: 11
            color: root.stationFaint
            wrapMode: Text.Wrap
          }
        }

        // Connection card
        Rectangle {
          width: parent.width
          height: connCard.implicitHeight + 24
          radius: root.cardRadius
          color: root.stationCard
          border.color: root.stationEdge
          border.width: 1

          Column {
            id: connCard
            x: 12
            y: 12
            width: parent.width - 24
            spacing: Style.space(8)

            Row {
              width: parent.width
              spacing: 8

              Rectangle {
                width: 8
                height: 8
                radius: 4
                anchors.verticalCenter: parent.verticalCenter
                color: root.connPillFg()
              }

              Text {
                text: "Connection"
                font.family: root.fontFamily
                font.pixelSize: 13
                font.bold: true
                color: root.foreground
                anchors.verticalCenter: parent.verticalCenter
              }

              Text {
                text: root.connKey() === "" ? "waiting" : root.connKey()
                font.family: root.fontFamily
                font.pixelSize: 11
                font.bold: true
                color: root.connPillFg()
                anchors.verticalCenter: parent.verticalCenter
              }
            }

            Text {
              width: parent.width
              text: root.connKey() === "needs_auth" ? "Sign in through the portal to enable traffic." : (root.connKey() === "connected" ? "Portal session is live." : "Portal actions unlock once the link is up.")
              font.family: root.fontFamily
              font.pixelSize: 11
              color: root.stationFaint
              wrapMode: Text.Wrap
            }

            // Contextual portal action: label says exactly what happens.
            Rectangle {
              width: parent.width
              height: root.btnHeight
              radius: 6
              color: root.connKey() === "connected" ? root.redWash : root.mintWash
              opacity: loginProcess.running || (root.connKey() !== "" && root.connKey() !== "connected" && root.connKey() !== "needs_auth" && !statusProcess.running) ? 0.55 : 1

              Row {
                anchors.centerIn: parent
                spacing: 8

                Text {
                  text: root.connKey() === "connected" ? "󰍃" : "󰦕"
                  font.family: root.fontFamily
                  font.pixelSize: 14
                  color: root.connKey() === "connected" ? root.failure : root.mint
                }

                Text {
                  text: loginProcess.running ? (root.loginAction === "logout" ? "Logging out…" : "Logging in…") : (root.connKey() === "connected" ? "Log out of portal" : (root.connKey() === "needs_auth" ? "Log in to portal" : "Refresh status"))
                  font.family: root.fontFamily
                  font.pixelSize: 12
                  font.bold: true
                  color: root.connKey() === "connected" ? root.failure : root.mint
                }
              }

              MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                enabled: !loginProcess.running && !statusProcess.running
                onClicked: {
                  if (root.connKey() === "connected") root.runLogin("logout")
                  else if (root.connKey() === "needs_auth") root.runLogin("login")
                  else root.refreshStatus()
                }
              }
            }
          }
        }

        // Hotspot card
        Rectangle {
          width: parent.width
          height: hotspotCard.implicitHeight + 24
          radius: root.cardRadius
          color: root.stationCard
          border.color: root.stationEdge
          border.width: 1
          visible: root.showHotspot

          Column {
            id: hotspotCard
            x: 12
            y: 12
            width: parent.width - 24
            spacing: Style.space(8)

            Row {
              width: parent.width
              spacing: 8

              Rectangle {
                width: 8
                height: 8
                radius: 4
                anchors.verticalCenter: parent.verticalCenter
                color: root.hotspotActive() ? root.mint : root.stationFaint
              }

              Text {
                text: "Hotspot"
                font.family: root.fontFamily
                font.pixelSize: 13
                font.bold: true
                color: root.foreground
                anchors.verticalCenter: parent.verticalCenter
              }

              Text {
                text: root.hotspotActive() ? "Active" : "Inactive"
                font.family: root.fontFamily
                font.pixelSize: 11
                font.bold: true
                color: root.hotspotActive() ? root.mint : root.stationFaint
                anchors.verticalCenter: parent.verticalCenter
              }
            }

            Text {
              width: parent.width
              text: root.hotspotDetail()
              font.family: root.fontFamily
              font.pixelSize: 11
              color: root.stationFaint
              wrapMode: Text.Wrap
            }

            Rectangle {
              width: parent.width
              height: root.btnHeight
              radius: 6
              color: root.hotspotActive() ? root.redWash : root.mintWash
              opacity: hotspotProcess.running ? 0.55 : 1

              Row {
                anchors.centerIn: parent
                spacing: 8

                Text {
                  text: root.hotspotActive() ? "󰦞" : "󰝨"
                  font.family: root.fontFamily
                  font.pixelSize: 14
                  color: root.hotspotActive() ? root.failure : root.mint
                }

                Text {
                  text: hotspotProcess.running ? (root.hotspotActive() ? "Stopping hotspot…" : "Starting hotspot…") : (root.hotspotActive() ? "Stop hotspot" : "Start hotspot")
                  font.family: root.fontFamily
                  font.pixelSize: 12
                  font.bold: true
                  color: root.hotspotActive() ? root.failure : root.mint
                }
              }

              MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                enabled: !hotspotProcess.running
                onClicked: root.runHotspot(root.hotspotActive() ? "stop" : "start")
              }
            }
          }
        }

        // VPN card
        Rectangle {
          width: parent.width
          height: vpnCard.implicitHeight + 24
          radius: root.cardRadius
          color: root.stationCard
          border.color: root.stationEdge
          border.width: 1
          visible: root.showVPN

          Column {
            id: vpnCard
            x: 12
            y: 12
            width: parent.width - 24
            spacing: Style.space(8)

            Row {
              width: parent.width
              spacing: 8

              Rectangle {
                width: 8
                height: 8
                radius: 4
                anchors.verticalCenter: parent.verticalCenter
                color: root.vpnConnected() ? root.mint : root.stationFaint
              }

              Text {
                text: "VPN"
                font.family: root.fontFamily
                font.pixelSize: 13
                font.bold: true
                color: root.foreground
                anchors.verticalCenter: parent.verticalCenter
              }

              Text {
                text: root.vpnConnected() ? "Connected" : "Disconnected"
                font.family: root.fontFamily
                font.pixelSize: 11
                font.bold: true
                color: root.vpnConnected() ? root.mint : root.stationFaint
                anchors.verticalCenter: parent.verticalCenter
              }
            }

            Text {
              width: parent.width
              text: root.vpnDetail()
              font.family: root.fontFamily
              font.pixelSize: 11
              color: root.stationFaint
              wrapMode: Text.Wrap
            }

            Rectangle {
              width: parent.width
              height: root.btnHeight
              radius: 6
              color: root.vpnConnected() ? root.redWash : root.mintWash
              opacity: vpnProcess.running ? 0.55 : 1

              Row {
                anchors.centerIn: parent
                spacing: 8

                Text {
                  text: root.vpnConnected() ? "󰦴" : "󰲜"
                  font.family: root.fontFamily
                  font.pixelSize: 14
                  color: root.vpnConnected() ? root.failure : root.mint
                }

                Text {
                  text: vpnProcess.running ? "Updating VPN…" : (root.vpnConnected() ? "Disconnect VPN" : "Connect VPN")
                  font.family: root.fontFamily
                  font.pixelSize: 12
                  font.bold: true
                  color: root.vpnConnected() ? root.failure : root.mint
                }
              }

              MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                enabled: !vpnProcess.running
                onClicked: root.runVPN("toggle")
              }
            }
          }
        }

        // Speed test card with persistent result
        Rectangle {
          width: parent.width
          height: speedCard.implicitHeight + 24
          radius: root.cardRadius
          color: root.stationCard
          border.color: root.stationEdge
          border.width: 1
          visible: root.showSpeedTest

          Column {
            id: speedCard
            x: 12
            y: 12
            width: parent.width - 24
            spacing: Style.space(8)

            PanelSectionHeader {
              width: parent.width
              text: "Speed test"
            }

            Text {
              width: parent.width
              text: root.isSpeedResult() ? root.actionText : "No test yet — run one to measure this link."
              font.family: root.fontFamily
              font.pixelSize: 12
              font.bold: root.isSpeedResult()
              color: root.isSpeedResult() ? root.mint : root.stationFaint
              wrapMode: Text.Wrap
            }

            Text {
              width: parent.width
              visible: root.isSpeedResult()
              text: "Result kept until the next action runs."
              font.family: root.fontFamily
              font.pixelSize: 11
              color: root.stationFaint
            }

            Rectangle {
              width: parent.width
              height: root.btnHeight
              radius: 6
              color: root.stationEdge
              opacity: loginProcess.running ? 0.55 : 1

              Row {
                anchors.centerIn: parent
                spacing: 8

                Text {
                  text: "󰄀"
                  font.family: root.fontFamily
                  font.pixelSize: 14
                  color: root.mint
                }

                Text {
                  text: (loginProcess.running && root.loginAction === "speed") ? "Testing speed…" : "Run speed test"
                  font.family: root.fontFamily
                  font.pixelSize: 12
                  font.bold: true
                  color: root.mint
                }
              }

              MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                enabled: !loginProcess.running
                onClicked: root.runLogin("speed")
              }
            }
          }
        }

        // Refresh row
        Column {
          width: parent.width
          spacing: 4

          Rectangle {
            width: parent.width
            height: root.btnHeight
            radius: 6
            color: root.stationEdge
            opacity: statusProcess.running ? 0.55 : 1

            Row {
              anchors.centerIn: parent
              spacing: 8

              Text {
                text: "󰑓"
                font.family: root.fontFamily
                font.pixelSize: 14
                color: root.foreground
              }

              Text {
                text: statusProcess.running ? "Refreshing…" : "Refresh status"
                font.family: root.fontFamily
                font.pixelSize: 12
                font.bold: true
                color: root.foreground
              }
            }

            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              enabled: !statusProcess.running
              onClicked: root.refreshStatus()
            }
          }

          Text {
            width: parent.width
            visible: root.updatedText() !== ""
            text: root.updatedText()
            font.family: root.fontFamily
            font.pixelSize: 10
            color: root.stationFaint
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

import QtQuick
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui

// Bar slot host for Conecta: owns the status poll and the bar button,
// loads Panel.qml through a Loader for the popup (mirrors the stock
// clock / antoniodavid.calendar BarWidget+Loader shape).
BarWidget {
  id: root
  moduleName: "conecta.network"
  property string ipcTarget: "conecta.network"

  Component.onCompleted: {
    console.log("CONECTA DEBUG: BarWidget instantiated, moduleName=" + moduleName)
    console.log("CONECTA DEBUG: button implicitWidth=" + (typeof button !== "undefined" ? button.implicitWidth : "undefined"))
    refreshStatus()
  }

  // Data: polled here, pushed live into the popup via injectPanel bindings.
  property var status: null
  property bool isOn: false
  // Buffered process output: SplitParser delivers line chunks, so chunks
  // accumulate until one complete JSON document parses.
  property string statusBuffer: ""
  // Settle follow-ups after an action: NetworkManager converges a few
  // seconds after the first refresh, so requestRefresh schedules two more.
  property int settleLeft: 0

  // Paths
  readonly property string pluginDir: Qt.resolvedUrl(".").toString().replace(/^file:\/\//, "")
  readonly property string statusScript: pluginDir + "/bin/omarchy-conecta-status"

  // Settings
  readonly property int refreshInterval: Number(setting("refreshIntervalSec", 30))

  // ─── Icon functions (nerd font like network plugin) ─────────────
  function getStatusIcon() {
    if (!isOn) return "󰤮"  // Disconnected
    if ((status && status.connection && status.connection.status) === "needs auth") return "󰤯"  // Needs auth
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
      root.isOn = false;
      return;
    }
    root.status = doc;
    root.isOn = (doc.connection && doc.connection.status) === "connected" ||
                (doc.connection && doc.connection.status) === "needs auth";
  }

  function refreshStatus() {
    root.statusBuffer = "";
    statusProcess.command = ["bash", root.statusScript];
    statusProcess.running = true;
  }

  // Immediate refresh + two settle follow-ups (≈ covers NM convergence).
  // Rapid successive actions just reset the counter; the single
  // statusProcess keeps its current behavior when overlapping.
  function refreshSoon() {
    root.refreshStatus();
    root.settleLeft = 2;
  }

  // ─── Popup passthroughs. Shape contract for shell.summon/hide/toggle
  //      routing: Bar.findPanelWidget requires open/close/opened on the
  //      bar-widget root.
  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false

  function open() {
    root.refreshStatus();
    if (panelLoader.item) panelLoader.item.open()
  }

  function close() {
    if (panelLoader.item) panelLoader.item.close()
  }

  function togglePanel() {
    root.refreshStatus();
    if (panelLoader.item) panelLoader.item.toggle()
  }

  // Forwarded so this widget can stand in for the panel as the bar's popout
  // identity: Bar.requestPopout prefers closeForPopoutSwitch over close, and
  // KeyboardPanel reads popoutSwitchClosing back off its owner.
  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false

  function closeForPopoutSwitch() {
    if (panelLoader.item) panelLoader.item.closeForPopoutSwitch()
  }

  // Tracks which panel item already has requestRefresh hooked, so repeat
  // injectPanel calls (bar/settings changes) never double-connect.
  property var _panelHooked: null

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
    if ("status" in target) target.status = Qt.binding(function() { return root.status })
    if ("isOn" in target) target.isOn = Qt.binding(function() { return root.isOn })
    if (target.requestRefresh && root._panelHooked !== target) {
      target.requestRefresh.connect(root.refreshSoon)
      root._panelHooked = target
    }
  }

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  onBarChanged: injectPanel()
  onSettingsChanged: injectPanel()

  Timer {
    interval: root.refreshInterval * 1000
    running: true
    repeat: true
    onTriggered: root.refreshStatus()
  }

  Timer {
    id: settleTimer
    interval: 2500
    repeat: true
    running: root.settleLeft > 0
    onTriggered: {
      root.settleLeft = root.settleLeft - 1;
      root.refreshStatus();
    }
  }

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("Panel.qml")
    visible: false
    onLoaded: {
      root.injectPanel()
      Qt.callLater(root.injectPanel)
    }
  }

  IpcHandler {
    target: "conecta.network"

    function refresh() { root.broadcast("refreshStatus") }
    function open() { root.open() }
    function close() { root.close() }
    function show() { root.open() }
    function hide() { root.close() }
    function toggle() { root.togglePanel() }
  }

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: root.getStatusIcon()
    active: root.isOn
    onPressed: function(buttonCode) {
      if (buttonCode === Qt.LeftButton) root.togglePanel()
    }
  }
}

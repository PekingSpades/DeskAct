package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	deskact "github.com/PekingSpades/DeskAct"
)

const (
	stateIdle    = "idle"
	stateRunning = "running"
	stateDone    = "done"

	stepPending = "pending"
	stepRunning = "running"
	stepPass    = "pass"
	stepFail    = "fail"
)

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DeskAct Auto Tester</title>
  <style>
    :root {
      --bg-1: #f6f2e7;
      --bg-2: #e7f0f7;
      --ink: #1f2a33;
      --muted: #5d6b73;
      --panel: #ffffff;
      --border: #d8d2c2;
      --accent: #0b6b78;
      --accent-2: #e07a2f;
      --success: #0c7a39;
      --fail: #b3261e;
      --shadow: 0 18px 40px rgba(25, 30, 35, 0.14);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Trebuchet MS", "Lucida Sans Unicode", "Lucida Grande", sans-serif;
      color: var(--ink);
      min-height: 100vh;
      background: radial-gradient(120% 120% at 10% 10%, var(--bg-2), var(--bg-1));
    }
    .wrap {
      max-width: 1080px;
      margin: 0 auto;
      padding: 28px 24px 40px;
    }
    header {
      display: grid;
      gap: 8px;
      margin-bottom: 24px;
    }
    h1 {
      margin: 0;
      letter-spacing: 0.5px;
      font-size: 32px;
    }
    .sub {
      margin: 0;
      color: var(--muted);
      max-width: 720px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 18px 20px;
      box-shadow: var(--shadow);
    }
    .controls {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }
    button {
      border: none;
      background: var(--accent);
      color: #fff;
      padding: 12px 18px;
      border-radius: 999px;
      font-size: 15px;
      font-weight: 600;
      letter-spacing: 0.3px;
      cursor: pointer;
      transition: transform 120ms ease, box-shadow 120ms ease;
      box-shadow: 0 12px 25px rgba(11, 107, 120, 0.2);
    }
    button:hover { transform: translateY(-1px); }
    button:disabled {
      background: #a0b3b7;
      cursor: not-allowed;
      box-shadow: none;
    }
    .pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 12px;
      border-radius: 999px;
      background: #f0f0ea;
      color: var(--muted);
      font-size: 12px;
      letter-spacing: 0.4px;
      text-transform: uppercase;
    }
    .pill.running { background: #fdf0d8; color: #8a4b14; }
    .pill.success { background: #e3f3e9; color: var(--success); }
    .pill.fail { background: #fde8e5; color: var(--fail); }
    .note {
      margin-top: 12px;
      color: var(--muted);
      font-size: 14px;
    }
    .countdown {
      margin-top: 10px;
      font-weight: 700;
      color: var(--accent-2);
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
      margin: 18px 0;
    }
    ul {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 8px;
    }
    li {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 12px;
      border-radius: 12px;
      background: #f9f7f2;
      border: 1px solid #efe6d6;
      font-size: 14px;
    }
    li.pending { color: var(--muted); }
    li.running { background: #fff5df; border-color: #f1d9ad; }
    li.pass { background: #e9f6ee; border-color: #cfe9d9; }
    li.fail { background: #fde8e5; border-color: #f4b7b1; }
    .duration { color: var(--muted); font-size: 12px; }
    pre {
      margin: 0;
      padding: 12px;
      border-radius: 12px;
      background: #1f2a33;
      color: #dfe8ee;
      font-size: 12px;
      min-height: 160px;
      overflow: auto;
    }
    .input-row {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-top: 12px;
      flex-wrap: wrap;
    }
    input[type="text"] {
      flex: 1;
      min-width: 240px;
      padding: 10px 12px;
      border-radius: 10px;
      border: 1px solid var(--border);
      font-size: 14px;
    }
    #arena {
      position: relative;
      height: 180px;
      border-radius: 16px;
      background: linear-gradient(135deg, rgba(11, 107, 120, 0.08), rgba(224, 122, 47, 0.15));
      border: 1px dashed rgba(11, 107, 120, 0.4);
      display: grid;
      place-items: center;
      color: var(--muted);
      font-size: 13px;
      margin-top: 12px;
    }
    #crosshair {
      position: fixed;
      top: 50%;
      left: 50%;
      width: 22px;
      height: 22px;
      transform: translate(-50%, -50%);
      border: 2px solid rgba(11, 107, 120, 0.6);
      border-radius: 50%;
      pointer-events: none;
    }
    #report img {
      width: 100%;
      max-height: 380px;
      object-fit: contain;
      border-radius: 12px;
      border: 1px solid var(--border);
      margin-top: 12px;
    }
    #stage {
      position: relative;
      min-height: 60vh;
      overflow: hidden;
    }
    .stage-tabs {
      display: flex;
      gap: 12px;
      font-size: 12px;
      letter-spacing: 0.4px;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 10px;
    }
    .stage-tabs .tab {
      padding: 6px 10px;
      border-radius: 999px;
      background: #f2efe6;
      border: 1px solid transparent;
    }
    .stage-tabs .tab.active {
      border-color: var(--accent);
      color: var(--accent);
      background: #e7f4f5;
    }
    .stage-panel {
      position: absolute;
      inset: 44px 18px 18px;
      opacity: 0;
      pointer-events: none;
      transition: opacity 180ms ease;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .stage-panel.active {
      opacity: 1;
      pointer-events: auto;
    }
    .kbd-metrics {
      display: grid;
      gap: 8px;
      font-size: 14px;
    }
    .metric {
      padding: 10px 12px;
      border-radius: 12px;
      border: 1px solid #efe6d6;
      background: #f9f7f2;
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 8px;
    }
    .metric span.value {
      font-weight: 600;
    }
    #mouse-area {
      flex: 1;
      min-height: 320px;
      position: relative;
      border-radius: 18px;
      border: 2px dashed rgba(11, 107, 120, 0.45);
      background: linear-gradient(135deg, rgba(11, 107, 120, 0.08), rgba(224, 122, 47, 0.18));
      overflow: hidden;
      user-select: none;
      -webkit-user-select: none;
      touch-action: none;
    }
    #mouse-layer {
      position: absolute;
      inset: 0;
      pointer-events: none;
    }
    .mouse-target {
      position: absolute;
      width: 34px;
      height: 34px;
      border-radius: 50%;
      border: 2px solid var(--accent);
      background: #e7f4f5;
      display: grid;
      place-items: center;
      font-weight: 700;
      color: var(--accent);
      transform: translate(-50%, -50%);
    }
    .mouse-target.hit {
      background: #e3f3e9;
      border-color: var(--success);
      color: var(--success);
    }
    .drag-marker {
      position: absolute;
      width: 30px;
      height: 30px;
      border-radius: 8px;
      border: 2px solid var(--accent-2);
      background: #fde7d6;
      color: #8a4b14;
      font-weight: 700;
      display: grid;
      place-items: center;
      transform: translate(-50%, -50%);
    }
    .drag-marker.end {
      border-color: var(--accent);
      background: #dff2f4;
      color: var(--accent);
    }
    #mouse-results {
      display: grid;
      gap: 6px;
      font-size: 13px;
    }
    #mouse-results li {
      padding: 8px 10px;
    }
    #screen-grid {
      display: grid;
      grid-template-columns: repeat(6, minmax(60px, 1fr));
      gap: 10px;
    }
    .screen-cell {
      height: 64px;
      border-radius: 12px;
      display: grid;
      place-items: center;
      font-weight: 700;
      color: #fff;
      letter-spacing: 1px;
    }
    .screen-cell.a { background: #0b6b78; }
    .screen-cell.b { background: #e07a2f; }
    .screen-cell.c { background: #7c9c36; }
    .screen-cell.d { background: #6c4a8f; }
    .screen-cell.e { background: #c94f4f; }
    .screen-cell.f { background: #2d3e4f; }
    .fade-in { animation: fadeIn 420ms ease both; }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(6px); }
      to { opacity: 1; transform: translateY(0); }
    }
  </style>
</head>
<body>
  <main class="wrap">
    <header>
      <h1>DeskAct Auto Tester</h1>
      <p class="sub">Run a local browser-driven test that simulates keyboard input, mouse activity, and screen capture.</p>
    </header>

    <section class="card fade-in">
      <div class="controls">
        <button id="start">Start Test</button>
        <span id="state" class="pill">Idle</span>
      </div>
      <p class="note">Keep this tab focused. The run will auto-type, click, and drag inside the panels below.</p>
      <div id="countdown" class="countdown"></div>
    </section>

    <section class="card fade-in" id="stage">
      <div class="stage-tabs">
        <span id="tab-keyboard" class="tab active">Keyboard</span>
        <span id="tab-mouse" class="tab">Mouse</span>
        <span id="tab-screen" class="tab">Screenshot</span>
      </div>

      <div id="panel-keyboard" class="stage-panel active">
        <h2>Keyboard Test</h2>
        <div class="input-row">
          <label for="kb-input">Keyboard target</label>
          <input id="kb-input" type="text" autocomplete="off" placeholder="Keyboard test output appears here">
        </div>
        <div class="kbd-metrics">
          <div class="metric">
            <span>Expected text</span>
            <span class="value" id="kb-expected">-</span>
          </div>
          <div class="metric">
            <span>Text match</span>
            <span class="value" id="kb-text-status">pending</span>
          </div>
          <div class="metric">
            <span>Hold key</span>
            <span class="value" id="kb-hold">-</span>
          </div>
          <div class="metric">
            <span>Hold status</span>
            <span class="value" id="kb-hold-status">pending</span>
          </div>
          <div class="metric">
            <span>Numpad sequence</span>
            <span class="value" id="kb-numpad-expected">-</span>
          </div>
          <div class="metric">
            <span>Numpad status</span>
            <span class="value" id="kb-numpad-status">pending</span>
          </div>
        </div>
        <p class="note">The input stays focused while the test types, holds a modifier key, and sends numpad digits.</p>
      </div>

      <div id="panel-mouse" class="stage-panel">
        <h2>Mouse Test</h2>
        <p class="note">Random targets should receive single, double, and triple clicks. Drag from S to E.</p>
        <div id="mouse-area">
          <div id="mouse-layer"></div>
        </div>
        <ul id="mouse-results"></ul>
      </div>

      <div id="panel-screen" class="stage-panel">
        <h2>Screenshot Test</h2>
        <p class="note">The capture should include the colored blocks below.</p>
        <div id="screen-grid">
          <div class="screen-cell a">A</div>
          <div class="screen-cell b">B</div>
          <div class="screen-cell c">C</div>
          <div class="screen-cell d">D</div>
          <div class="screen-cell e">E</div>
          <div class="screen-cell f">F</div>
          <div class="screen-cell b">G</div>
          <div class="screen-cell c">H</div>
          <div class="screen-cell d">I</div>
          <div class="screen-cell e">J</div>
          <div class="screen-cell f">K</div>
          <div class="screen-cell a">L</div>
        </div>
      </div>
    </section>

    <section class="grid">
      <div class="card fade-in">
        <h2>Progress</h2>
        <ul id="steps"></ul>
      </div>
      <div class="card fade-in">
        <h2>Live Log</h2>
        <pre id="log"></pre>
      </div>
    </section>

    <section class="card fade-in" id="report" hidden>
      <h2>Report</h2>
      <div id="summary"></div>
      <div id="shotWrap" hidden>
        <img id="screenshot" alt="Screenshot result">
      </div>
    </section>
  </main>

  <script>
    var startBtn = document.getElementById('start');
    var statePill = document.getElementById('state');
    var stepsEl = document.getElementById('steps');
    var logEl = document.getElementById('log');
    var reportEl = document.getElementById('report');
    var summaryEl = document.getElementById('summary');
    var shotWrap = document.getElementById('shotWrap');
    var screenshotEl = document.getElementById('screenshot');
    var countdownEl = document.getElementById('countdown');

    var tabKeyboard = document.getElementById('tab-keyboard');
    var tabMouse = document.getElementById('tab-mouse');
    var tabScreen = document.getElementById('tab-screen');
    var panelKeyboard = document.getElementById('panel-keyboard');
    var panelMouse = document.getElementById('panel-mouse');
    var panelScreen = document.getElementById('panel-screen');

    var kbInput = document.getElementById('kb-input');
    var kbExpected = document.getElementById('kb-expected');
    var kbTextStatus = document.getElementById('kb-text-status');
    var kbHold = document.getElementById('kb-hold');
    var kbHoldStatus = document.getElementById('kb-hold-status');
    var kbNumpadExpected = document.getElementById('kb-numpad-expected');
    var kbNumpadStatus = document.getElementById('kb-numpad-status');

    var mouseArea = document.getElementById('mouse-area');
    var mouseLayer = document.getElementById('mouse-layer');
    var mouseResults = document.getElementById('mouse-results');

    var currentRun = '';
    var polling = false;
    var currentPlan = null;
    var keyboardState = {};
    var mouseState = {};
    var keyboardTimeout = null;
    var mouseTimeout = null;
    var lastStatusState = '';
    var lastStatusStep = '';

    function dbg(message, data) {
      var ts = new Date().toISOString().slice(11, 19);
      if (data !== undefined) {
        console.log('[deskact ' + ts + '] ' + message + ' ' + safeStringify(data));
      } else {
        console.log('[deskact ' + ts + '] ' + message);
      }
    }

    function safeStringify(data) {
      try {
        return JSON.stringify(data);
      } catch (err) {
        return '"[unserializable]"';
      }
    }

    function eventTargetLabel(evt) {
      if (!evt || !evt.target) { return 'unknown'; }
      var el = evt.target;
      if (el.id) { return '#' + el.id; }
      if (typeof el.className === 'string' && el.className) {
        return (el.tagName || 'el') + '.' + el.className;
      }
      return el.tagName || 'unknown';
    }

    function rectSummary(rect) {
      return {
        left: Math.round(rect.left),
        top: Math.round(rect.top),
        width: Math.round(rect.width),
        height: Math.round(rect.height)
      };
    }

    function logMouseEvent(evt, origin, rect, extra) {
      var inArea = rect ? pointInRect(evt.clientX, evt.clientY, rect) : false;
      var data = {
        origin: origin,
        type: evt.type,
        button: evt.button,
        buttons: evt.buttons,
        detail: evt.detail,
        clientX: Math.round(evt.clientX),
        clientY: Math.round(evt.clientY),
        inArea: inArea,
        target: eventTargetLabel(evt)
      };
      if (rect) {
        data.relX = Math.round(evt.clientX - rect.left);
        data.relY = Math.round(evt.clientY - rect.top);
        data.rect = rectSummary(rect);
      }
      if (extra) {
        for (var key in extra) {
          data[key] = extra[key];
        }
      }
      dbg('mouse event', data);
      return inArea;
    }

    function mouseRect() {
      return mouseArea.getBoundingClientRect();
    }

    function pointInRect(x, y, rect) {
      return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom;
    }

    function eventInMouseArea(evt) {
      var rect = mouseRect();
      return pointInRect(evt.clientX, evt.clientY, rect);
    }

    function sleep(ms) {
      return new Promise(function(resolve) { setTimeout(resolve, ms); });
    }

    function setState(text, kind) {
      statePill.textContent = text;
      statePill.className = 'pill';
      if (kind) {
        statePill.className += ' ' + kind;
      }
    }

    function formatDuration(ms) {
      if (!ms) { return ''; }
      if (ms < 1000) { return ms + ' ms'; }
      return (ms / 1000).toFixed(2) + ' s';
    }

    function activatePanel(name) {
      panelKeyboard.classList.toggle('active', name === 'keyboard');
      panelMouse.classList.toggle('active', name === 'mouse');
      panelScreen.classList.toggle('active', name === 'screen');
      tabKeyboard.classList.toggle('active', name === 'keyboard');
      tabMouse.classList.toggle('active', name === 'mouse');
      tabScreen.classList.toggle('active', name === 'screen');
    }

    function renderSteps(steps) {
      stepsEl.innerHTML = '';
      if (!steps || !steps.length) {
        return;
      }
      steps.forEach(function(step) {
        var li = document.createElement('li');
        li.className = step.status || '';
        var left = document.createElement('span');
        left.textContent = step.name + ' - ' + (step.status || 'pending');
        var right = document.createElement('span');
        right.className = 'duration';
        right.textContent = formatDuration(step.duration_ms);
        li.appendChild(left);
        li.appendChild(right);
        stepsEl.appendChild(li);
      });
    }

    function activeStepName(steps) {
      if (!steps) { return ''; }
      for (var i = 0; i < steps.length; i++) {
        if (steps[i].status === 'running') {
          return steps[i].name;
        }
      }
      return '';
    }

    function renderStatus(data) {
      if (!data || !data.state) {
        return;
      }
      var stateText = data.state.charAt(0).toUpperCase() + data.state.slice(1);
      var kind = '';
      if (data.state === 'running') {
        kind = 'running';
      } else if (data.state === 'done') {
        kind = data.success ? 'success' : 'fail';
      }
      setState(stateText, kind);
      renderSteps(data.steps || []);
      if (data.logs) {
        logEl.textContent = data.logs.join('\n');
      }

      var active = activeStepName(data.steps || []);
      if (data.state !== lastStatusState || active !== lastStatusStep) {
        dbg('status change', {
          state: data.state,
          active_step: active,
          success: data.success,
          steps: data.steps || []
        });
        lastStatusState = data.state;
        lastStatusStep = active;
      }
      if (active === 'Keyboard') {
        activatePanel('keyboard');
        startKeyboardTracking();
      } else if (active === 'Mouse') {
        activatePanel('mouse');
        startMouseTracking();
      } else if (active === 'Screenshot') {
        activatePanel('screen');
      }

      if (data.state === 'done') {
        reportEl.hidden = false;
        var parts = [];
        parts.push(data.success ? 'All steps passed.' : 'Completed with failures.');
        if (data.keyboard_report) {
          parts.push('Keyboard: ' + (data.keyboard_report.pass ? 'OK' : 'fail'));
        }
        if (data.mouse_report) {
          parts.push('Mouse: ' + (data.mouse_report.pass ? 'OK' : 'fail'));
        }
        if (data.screenshot_size_bytes) {
          parts.push('Screenshot size: ' + data.screenshot_size_bytes + ' bytes.');
        }
        summaryEl.textContent = parts.join(' ');
        if (data.screenshot_url) {
          shotWrap.hidden = false;
          screenshotEl.src = data.screenshot_url + '?t=' + Date.now();
        } else {
          shotWrap.hidden = true;
        }
        startBtn.disabled = false;
        startBtn.textContent = 'Run Again';
        if (document.fullscreenElement) {
          document.exitFullscreen().catch(function() {});
        }
      }
    }

    async function pollStatus() {
      if (!currentRun || polling) {
        return;
      }
      polling = true;
      while (currentRun) {
        try {
          var resp = await fetch('/api/status?id=' + encodeURIComponent(currentRun), { cache: 'no-store' });
          var data = await resp.json();
          if (!resp.ok) {
            throw new Error(data.error || 'status error');
          }
          renderStatus(data);
          if (data.state === 'done') {
            currentRun = '';
            polling = false;
            break;
          }
        } catch (err) {
          setState('Error', 'fail');
          logEl.textContent = String(err);
          currentRun = '';
          polling = false;
          startBtn.disabled = false;
          break;
        }
        await sleep(500);
      }
    }

    async function runCountdown(seconds) {
      for (var i = seconds; i > 0; i--) {
        countdownEl.textContent = 'Starting in ' + i + '...';
        await sleep(1000);
      }
      countdownEl.textContent = '';
    }

    function collectClientInfo() {
      var rect = mouseArea.getBoundingClientRect();
      var kbRect = kbInput.getBoundingClientRect();
      var screenX = (typeof window.screenX === 'number') ? window.screenX : (window.screenLeft || 0);
      var screenY = (typeof window.screenY === 'number') ? window.screenY : (window.screenTop || 0);
      var outerW = window.outerWidth || window.innerWidth || 0;
      var outerH = window.outerHeight || window.innerHeight || 0;
      var borderX = Math.max(0, (outerW - window.innerWidth) / 2);
      var borderY = Math.max(0, outerH - window.innerHeight);
      var viewportOffsetX = screenX + borderX;
      var viewportOffsetY = screenY + borderY;
      var info = {
        viewport_width: Math.round(window.innerWidth),
        viewport_height: Math.round(window.innerHeight),
        device_pixel_ratio: window.devicePixelRatio || 1,
        mouse_area: {
          left: rect.left,
          top: rect.top,
          width: rect.width,
          height: rect.height
        },
        keyboard_input: {
          left: kbRect.left,
          top: kbRect.top,
          width: kbRect.width,
          height: kbRect.height
        },
        viewport_offset_x: viewportOffsetX,
        viewport_offset_y: viewportOffsetY,
        user_agent: navigator.userAgent || ''
      };
      dbg('client info', {
        info: info,
        screenX: screenX,
        screenY: screenY,
        outerWidth: outerW,
        outerHeight: outerH,
        innerWidth: window.innerWidth,
        innerHeight: window.innerHeight,
        borderX: borderX,
        borderY: borderY,
        visualViewport: window.visualViewport ? {
          width: window.visualViewport.width,
          height: window.visualViewport.height,
          scale: window.visualViewport.scale,
          offsetLeft: window.visualViewport.offsetLeft,
          offsetTop: window.visualViewport.offsetTop
        } : null
      });
      return info;
    }

    async function loadPlan() {
      if (!currentRun) { return; }
      try {
        var resp = await fetch('/api/plan?id=' + encodeURIComponent(currentRun), { cache: 'no-store' });
        var data = await resp.json();
        if (!resp.ok) {
          throw new Error(data.error || 'plan error');
        }
        currentPlan = data;
        dbg('plan loaded', currentPlan);
        renderPlan();
      } catch (err) {
        logEl.textContent = String(err);
      }
    }

    function sendReady(step, client) {
      if (!currentRun) { return; }
      var payload = { run_id: currentRun, step: step };
      if (client) {
        payload.client = client;
      }
      dbg('ready sent', payload);
      fetch('/api/ready', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      }).catch(function() {});
    }

    function resetRunState() {
      dbg('reset run state');
      currentPlan = null;
      keyboardState = {};
      mouseState = {};
      if (keyboardTimeout) { clearTimeout(keyboardTimeout); }
      if (mouseTimeout) { clearTimeout(mouseTimeout); }
      keyboardTimeout = null;
      mouseTimeout = null;
      kbInput.readOnly = false;
      kbInput.value = '';
      kbExpected.textContent = '-';
      kbTextStatus.textContent = 'pending';
      kbHold.textContent = '-';
      kbHoldStatus.textContent = 'pending';
      kbNumpadExpected.textContent = '-';
      kbNumpadStatus.textContent = 'pending';
      mouseLayer.innerHTML = '';
      mouseResults.innerHTML = '';
    }

    function renderPlan() {
      if (!currentPlan) { return; }
      kbExpected.textContent = currentPlan.keyboard.text || '-';
      kbHold.textContent = currentPlan.keyboard.hold_key || '-';
      kbNumpadExpected.textContent = currentPlan.keyboard.numpad_sequence || '-';
      kbTextStatus.textContent = 'pending';
      kbHoldStatus.textContent = 'pending';
      kbNumpadStatus.textContent = 'pending';
      renderMousePlan(null);
    }

    function renderMousePlan(rectOverride) {
      if (!currentPlan) { return; }
      mouseLayer.innerHTML = '';
      mouseResults.innerHTML = '';

      var rect = rectOverride || mouseArea.getBoundingClientRect();
      var targets = currentPlan.mouse_targets || [];
      var targetLayout = [];
      targets.forEach(function(t) {
        var x = rect.width * t.x_norm;
        var y = rect.height * t.y_norm;
        targetLayout.push({
          id: t.id,
          clicks: t.clicks,
          x: Math.round(x),
          y: Math.round(y)
        });
        var marker = document.createElement('div');
        marker.className = 'mouse-target';
        marker.style.left = x + 'px';
        marker.style.top = y + 'px';
        marker.textContent = t.clicks + 'x';
        marker.dataset.targetId = t.id;
        mouseLayer.appendChild(marker);

        var li = document.createElement('li');
        li.id = 'mouse-result-' + t.id;
        li.textContent = 'Target ' + t.id + ' (' + t.clicks + 'x) pending';
        mouseResults.appendChild(li);
      });

      if (currentPlan.drag) {
        var sx = rect.width * currentPlan.drag.start_x_norm;
        var sy = rect.height * currentPlan.drag.start_y_norm;
        var ex = rect.width * currentPlan.drag.end_x_norm;
        var ey = rect.height * currentPlan.drag.end_y_norm;
        dbg('mouse layout', {
          rect: rectSummary(rect),
          targets: targetLayout,
          drag: {
            startX: Math.round(sx),
            startY: Math.round(sy),
            endX: Math.round(ex),
            endY: Math.round(ey)
          }
        });
        var sMarker = document.createElement('div');
        sMarker.className = 'drag-marker';
        sMarker.style.left = sx + 'px';
        sMarker.style.top = sy + 'px';
        sMarker.textContent = 'S';
        mouseLayer.appendChild(sMarker);
        var eMarker = document.createElement('div');
        eMarker.className = 'drag-marker end';
        eMarker.style.left = ex + 'px';
        eMarker.style.top = ey + 'px';
        eMarker.textContent = 'E';
        mouseLayer.appendChild(eMarker);

        var dragLi = document.createElement('li');
        dragLi.id = 'mouse-result-drag';
        dragLi.textContent = 'Drag S -> E pending';
        mouseResults.appendChild(dragLi);
      }
    }

    function startKeyboardTracking() {
      if (!currentPlan || keyboardState.active || keyboardState.sent) { return; }
      keyboardState = {
        active: true,
        expectedText: currentPlan.keyboard.text || '',
        holdKey: currentPlan.keyboard.hold_key || 'Shift',
        holdMinMs: currentPlan.keyboard.hold_min_ms || 400,
        numpadExpected: currentPlan.keyboard.numpad_sequence || '',
        holdDownAt: null,
        holdDuration: 0,
        numpadSequence: '',
        readySent: false,
        lastTextMatch: false,
        lastHoldPass: false,
        lastNumpadPass: false
      };
      dbg('keyboard tracking start', {
        expected: keyboardState.expectedText,
        hold_key: keyboardState.holdKey,
        hold_min_ms: keyboardState.holdMinMs,
        numpad: keyboardState.numpadExpected
      });
      kbInput.value = '';
      kbInput.readOnly = false;
      window.focus();
      kbInput.focus();
      dbg('keyboard focus', { activeElement: document.activeElement ? document.activeElement.id : '' });
      kbTextStatus.textContent = 'pending';
      kbHoldStatus.textContent = 'pending';
      kbNumpadStatus.textContent = 'pending';
      if (!keyboardState.readySent) {
        keyboardState.readySent = true;
        sendReady('keyboard');
      }
      if (keyboardTimeout) { clearTimeout(keyboardTimeout); }
      keyboardTimeout = setTimeout(function() {
        sendKeyboardReport('timeout');
      }, 7000);
    }

    function startMouseTracking() {
      if (!currentPlan || mouseState.active || mouseState.sent) { return; }
      var rect = mouseArea.getBoundingClientRect();
      renderMousePlan(rect);
      var targets = currentPlan.mouse_targets || [];
      mouseState = {
        active: true,
        targets: targets.map(function(t) {
          return {
            id: t.id,
            clicks: t.clicks,
            x: rect.width * t.x_norm,
            y: rect.height * t.y_norm,
            hit: false,
            actualClicks: 0,
            clickX: 0,
            clickY: 0
          };
        }),
        drag: {
          startX: rect.width * currentPlan.drag.start_x_norm,
          startY: rect.height * currentPlan.drag.start_y_norm,
          endX: rect.width * currentPlan.drag.end_x_norm,
          endY: rect.height * currentPlan.drag.end_y_norm,
          startHit: false,
          endHit: false,
          moved: false,
          distance: 0,
          dragging: false,
          lastX: 0,
          lastY: 0,
          lastMoveLogAt: 0
        },
        tolerance: Math.max(12, Math.min(30, Math.min(rect.width, rect.height) / 20)),
        sent: false,
        ignoreClicks: 1,
        readySent: false
      };
      dbg('mouse tracking start', {
        rect: rectSummary(rect),
        tolerance: mouseState.tolerance,
        ignoreClicks: mouseState.ignoreClicks,
        targets: mouseState.targets.map(function(t) {
          return { id: t.id, clicks: t.clicks, x: Math.round(t.x), y: Math.round(t.y) };
        }),
        drag: {
          startX: Math.round(mouseState.drag.startX),
          startY: Math.round(mouseState.drag.startY),
          endX: Math.round(mouseState.drag.endX),
          endY: Math.round(mouseState.drag.endY)
        }
      });
      if (!mouseState.readySent) {
        mouseState.readySent = true;
        sendReady('mouse', collectClientInfo());
      }
      if (mouseTimeout) { clearTimeout(mouseTimeout); }
      mouseTimeout = setTimeout(function() {
        sendMouseReport('timeout');
      }, 9000);
    }

    function isHoldKey(evt) {
      if (!keyboardState.holdKey) { return false; }
      var key = (evt.key || '').toLowerCase();
      var code = (evt.code || '').toLowerCase();
      var hold = keyboardState.holdKey.toLowerCase();
      if (key === hold || code === hold) { return true; }
      if (hold === 'shift' && (code === 'shiftleft' || code === 'shiftright')) { return true; }
      if (hold === 'control' && (code === 'controlleft' || code === 'controlright')) { return true; }
      if (hold === 'alt' && (code === 'altleft' || code === 'altright')) { return true; }
      return false;
    }

    function updateKeyboardStatus() {
      var textMatch = keyboardState.expectedText && kbInput.value === keyboardState.expectedText;
      if (textMatch) {
        kbInput.readOnly = true;
      }
      if (textMatch !== keyboardState.lastTextMatch) {
        dbg('keyboard text match', { match: textMatch, value: kbInput.value });
        keyboardState.lastTextMatch = textMatch;
      }
      kbTextStatus.textContent = textMatch ? 'OK' : 'pending';
      var holdPass = keyboardState.holdDuration >= keyboardState.holdMinMs;
      if (holdPass !== keyboardState.lastHoldPass) {
        dbg('keyboard hold status', { pass: holdPass, duration_ms: keyboardState.holdDuration });
        keyboardState.lastHoldPass = holdPass;
      }
      kbHoldStatus.textContent = holdPass ? ('OK ' + keyboardState.holdDuration + 'ms') : 'pending';
      var numpadPass = keyboardState.numpadExpected && keyboardState.numpadSequence === keyboardState.numpadExpected;
      if (numpadPass !== keyboardState.lastNumpadPass) {
        dbg('keyboard numpad status', { pass: numpadPass, sequence: keyboardState.numpadSequence });
        keyboardState.lastNumpadPass = numpadPass;
      }
      kbNumpadStatus.textContent = numpadPass ? ('OK ' + keyboardState.numpadSequence) : 'pending';
      if (textMatch && holdPass && numpadPass) {
        sendKeyboardReport('completed');
      }
    }

    function sendKeyboardReport(reason) {
      if (!keyboardState.active || keyboardState.sent || !currentRun) { return; }
      keyboardState.sent = true;
      keyboardState.active = false;
      if (keyboardTimeout) { clearTimeout(keyboardTimeout); }
      var report = {
        text: kbInput.value,
        text_match: kbInput.value === keyboardState.expectedText,
        hold_key: keyboardState.holdKey,
        hold_duration_ms: keyboardState.holdDuration,
        hold_pass: keyboardState.holdDuration >= keyboardState.holdMinMs,
        numpad_sequence: keyboardState.numpadSequence,
        numpad_pass: keyboardState.numpadSequence === keyboardState.numpadExpected,
        pass: false,
        errors: []
      };
      report.pass = report.text_match && report.hold_pass && report.numpad_pass;
      if (!report.text_match) { report.errors.push('text mismatch'); }
      if (!report.hold_pass) { report.errors.push('hold too short'); }
      if (!report.numpad_pass) { report.errors.push('numpad mismatch'); }
      dbg('keyboard report', { reason: reason, report: report });
      fetch('/api/report/keyboard', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ run_id: currentRun, report: report })
      }).catch(function() {});
    }

    function distanceSq(x1, y1, x2, y2) {
      var dx = x1 - x2;
      var dy = y1 - y2;
      return dx * dx + dy * dy;
    }

    function updateMouseResult(id, text, ok) {
      var li = document.getElementById('mouse-result-' + id);
      if (!li) { return; }
      li.textContent = text;
      if (ok) {
        li.style.background = '#e9f6ee';
        li.style.borderColor = '#cfe9d9';
      } else {
        li.style.background = '#fde8e5';
        li.style.borderColor = '#f4b7b1';
      }
    }

    function updateMouseProgress(id, text) {
      var li = document.getElementById('mouse-result-' + id);
      if (!li) { return; }
      li.textContent = text;
      li.style.background = '';
      li.style.borderColor = '';
    }

    function updateDragResult(text, ok) {
      var li = document.getElementById('mouse-result-drag');
      if (!li) { return; }
      li.textContent = text;
      if (ok) {
        li.style.background = '#e9f6ee';
        li.style.borderColor = '#cfe9d9';
      } else {
        li.style.background = '#fde8e5';
        li.style.borderColor = '#f4b7b1';
      }
    }

    function checkMouseCompletion() {
      if (!mouseState.active || mouseState.sent) { return; }
      var allTargets = mouseState.targets.every(function(t) { return t.hit; });
      var dragOk = mouseState.drag.startHit && mouseState.drag.endHit && mouseState.drag.moved;
      if (allTargets && dragOk) {
        sendMouseReport('completed');
      }
    }

    function sendMouseReport(reason) {
      if (!mouseState.active || mouseState.sent || !currentRun) { return; }
      mouseState.sent = true;
      mouseState.active = false;
      if (mouseTimeout) { clearTimeout(mouseTimeout); }
      var report = {
        targets: mouseState.targets.map(function(t) {
          return {
            id: t.id,
            expected_clicks: t.clicks,
            actual_clicks: t.actualClicks,
            hit: t.hit,
            click_x: t.clickX,
            click_y: t.clickY
          };
        }),
        drag: {
          start_hit: mouseState.drag.startHit,
          end_hit: mouseState.drag.endHit,
          moved: mouseState.drag.moved
        },
        pass: false,
        errors: []
      };
      report.pass = report.targets.every(function(t) { return t.hit && t.actual_clicks === t.expected_clicks; }) &&
        report.drag.start_hit && report.drag.end_hit && report.drag.moved;
      if (!report.pass) {
        report.errors.push(reason === 'timeout' ? 'timeout' : 'incomplete');
      }
      dbg('mouse report', { reason: reason, report: report });
      fetch('/api/report/mouse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ run_id: currentRun, report: report })
      }).catch(function() {});
    }

    kbInput.addEventListener('input', function() {
      if (!keyboardState.active) { return; }
      dbg('keyboard input', { value: kbInput.value, length: kbInput.value.length });
      updateKeyboardStatus();
    });

    kbInput.addEventListener('focus', function() {
      dbg('keyboard focus event', { activeElement: document.activeElement ? document.activeElement.id : '' });
    });

    kbInput.addEventListener('blur', function() {
      dbg('keyboard blur event', { activeElement: document.activeElement ? document.activeElement.id : '' });
    });

    function isNumpadDigit(evt) {
      if (evt.location === 3 && /^[0-9]$/.test(evt.key)) {
        return true;
      }
      if (evt.code && evt.code.indexOf('Numpad') === 0 && /^[0-9]$/.test(evt.key)) {
        return true;
      }
      return false;
    }

    document.addEventListener('keydown', function(evt) {
      if (!keyboardState.active) { return; }
      dbg('key down', { key: evt.key, code: evt.code, location: evt.location, hold: isHoldKey(evt), numpad: isNumpadDigit(evt) });
      if (isHoldKey(evt) && keyboardState.holdDownAt === null) {
        keyboardState.holdDownAt = performance.now();
      }
      if (isNumpadDigit(evt)) {
        evt.preventDefault();
        evt.stopPropagation();
        if (keyboardState.numpadSequence.length < keyboardState.numpadExpected.length) {
          keyboardState.numpadSequence += evt.key;
        }
      }
      updateKeyboardStatus();
    });

    document.addEventListener('keyup', function(evt) {
      if (!keyboardState.active) { return; }
      dbg('key up', { key: evt.key, code: evt.code, location: evt.location, hold: isHoldKey(evt) });
      if (isHoldKey(evt) && keyboardState.holdDownAt !== null) {
        keyboardState.holdDuration = Math.round(performance.now() - keyboardState.holdDownAt);
        keyboardState.holdDownAt = null;
      }
      updateKeyboardStatus();
    });

    function handleMouseClick(evt, origin) {
      if (!mouseState.active) { return; }
      var rect = mouseRect();
      var inArea = logMouseEvent(evt, origin, rect, { phase: 'click' });
      if (!inArea) { return; }
      if (evt.button !== 0) { return; }
      evt.preventDefault();
      if (mouseState.ignoreClicks > 0) {
        mouseState.ignoreClicks -= 1;
        dbg('mouse click ignored', { remaining: mouseState.ignoreClicks });
        return;
      }
      var x = evt.clientX - rect.left;
      var y = evt.clientY - rect.top;
      var best = null;
      var bestDist = Infinity;
      mouseState.targets.forEach(function(t) {
        if (t.hit) { return; }
        var dist = distanceSq(x, y, t.x, t.y);
        if (dist <= mouseState.tolerance * mouseState.tolerance && dist < bestDist) {
          best = t;
          bestDist = dist;
        }
      });
      if (best) {
        best.actualClicks += 1;
        best.clickX = x;
        best.clickY = y;
        if (best.actualClicks >= best.clicks) {
          best.hit = true;
          var marker = mouseLayer.querySelector('[data-target-id="' + best.id + '"]');
          if (marker) { marker.classList.add('hit'); }
          updateMouseResult(best.id, 'Target ' + best.id + ' (' + best.clicks + 'x) OK', true);
          dbg('mouse target hit', { id: best.id, actual: best.actualClicks, expected: best.clicks, x: Math.round(x), y: Math.round(y) });
          checkMouseCompletion();
        } else {
          updateMouseProgress(best.id, 'Target ' + best.id + ' (' + best.clicks + 'x) ' + best.actualClicks + '/' + best.clicks);
          dbg('mouse target progress', { id: best.id, actual: best.actualClicks, expected: best.clicks, x: Math.round(x), y: Math.round(y) });
        }
      } else {
        dbg('mouse click no target', { x: Math.round(x), y: Math.round(y) });
      }
    }

    function handleMouseDown(evt, origin) {
      if (!mouseState.active) { return; }
      var rect = mouseRect();
      var inArea = logMouseEvent(evt, origin, rect, { phase: 'down' });
      if (!inArea) { return; }
      if (evt.button !== 0) { return; }
      evt.preventDefault();
      var x = evt.clientX - rect.left;
      var y = evt.clientY - rect.top;
      var distSq = distanceSq(x, y, mouseState.drag.startX, mouseState.drag.startY);
      dbg('drag start check', { x: Math.round(x), y: Math.round(y), dist_sq: Math.round(distSq), tolerance: mouseState.tolerance });
      if (distSq <= mouseState.tolerance * mouseState.tolerance) {
        mouseState.drag.startHit = true;
        mouseState.drag.dragging = true;
        mouseState.drag.lastX = x;
        mouseState.drag.lastY = y;
        mouseState.drag.distance = 0;
        dbg('drag start hit', { x: Math.round(x), y: Math.round(y) });
      }
    }

    function handleMouseMove(evt, origin) {
      if (!mouseState.active || !mouseState.drag.dragging) { return; }
      var rect = mouseRect();
      var inArea = pointInRect(evt.clientX, evt.clientY, rect);
      var x = evt.clientX - rect.left;
      var y = evt.clientY - rect.top;
      mouseState.drag.distance += Math.sqrt(distanceSq(x, y, mouseState.drag.lastX, mouseState.drag.lastY));
      mouseState.drag.lastX = x;
      mouseState.drag.lastY = y;
      var now = Date.now();
      if (!mouseState.drag.lastMoveLogAt || now - mouseState.drag.lastMoveLogAt > 200) {
        mouseState.drag.lastMoveLogAt = now;
        logMouseEvent(evt, origin, rect, {
          phase: 'move',
          dragging: true,
          drag_distance: Math.round(mouseState.drag.distance),
          inArea: inArea
        });
      }
    }

    function handleMouseUp(evt, origin) {
      if (!mouseState.active || !mouseState.drag.dragging) { return; }
      var rect = mouseRect();
      var inArea = logMouseEvent(evt, origin, rect, { phase: 'up' });
      var x = evt.clientX - rect.left;
      var y = evt.clientY - rect.top;
      if (!inArea) {
        dbg('drag end outside area', { x: Math.round(x), y: Math.round(y) });
      }
      var distSq = distanceSq(x, y, mouseState.drag.endX, mouseState.drag.endY);
      dbg('drag end check', { x: Math.round(x), y: Math.round(y), dist_sq: Math.round(distSq), tolerance: mouseState.tolerance });
      if (distSq <= mouseState.tolerance * mouseState.tolerance) {
        mouseState.drag.endHit = true;
      }
      mouseState.drag.moved = mouseState.drag.distance > 40;
      mouseState.drag.dragging = false;
      updateDragResult(mouseState.drag.startHit && mouseState.drag.endHit && mouseState.drag.moved ? 'Drag OK' : 'Drag pending', mouseState.drag.startHit && mouseState.drag.endHit && mouseState.drag.moved);
      dbg('drag end result', {
        start_hit: mouseState.drag.startHit,
        end_hit: mouseState.drag.endHit,
        moved: mouseState.drag.moved,
        distance: Math.round(mouseState.drag.distance)
      });
      checkMouseCompletion();
    }

    function handleMouseContextMenu(evt, origin) {
      if (!mouseState.active) { return; }
      var rect = mouseRect();
      var inArea = logMouseEvent(evt, origin, rect, { phase: 'contextmenu' });
      if (!inArea) { return; }
      evt.preventDefault();
    }

    function handleMouseDblClick(evt, origin) {
      if (!mouseState.active) { return; }
      var rect = mouseRect();
      var inArea = logMouseEvent(evt, origin, rect, { phase: 'dblclick' });
      if (!inArea) { return; }
      evt.preventDefault();
    }

    document.addEventListener('contextmenu', function(evt) {
      handleMouseContextMenu(evt, 'document');
    }, true);

    document.addEventListener('dblclick', function(evt) {
      handleMouseDblClick(evt, 'document');
    }, true);

    document.addEventListener('click', function(evt) {
      handleMouseClick(evt, 'document');
    }, true);

    document.addEventListener('mousedown', function(evt) {
      handleMouseDown(evt, 'document');
    }, true);

    document.addEventListener('mousemove', function(evt) {
      handleMouseMove(evt, 'document');
    }, true);

    document.addEventListener('mouseup', function(evt) {
      handleMouseUp(evt, 'document');
    }, true);

    document.addEventListener('visibilitychange', function() {
      dbg('visibility change', { hidden: document.hidden });
    });

    window.addEventListener('blur', function() {
      dbg('window blur');
    });

    window.addEventListener('focus', function() {
      dbg('window focus');
    });

    startBtn.addEventListener('click', async function() {
      dbg('start clicked');
      startBtn.disabled = true;
      reportEl.hidden = true;
      shotWrap.hidden = true;
      setState('Preparing', 'running');
      resetRunState();
      try {
        if (document.documentElement.requestFullscreen) {
          dbg('request fullscreen');
          await document.documentElement.requestFullscreen();
        }
      } catch (err) {
        logEl.textContent = 'Fullscreen request failed: ' + err;
        dbg('fullscreen failed', { error: String(err) });
      }
      await runCountdown(3);
      dbg('countdown finished');
      kbInput.focus();
      try {
        var clientInfo = collectClientInfo();
        var resp = await fetch('/api/start', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ client: clientInfo })
        });
        var data = await resp.json();
        if (!resp.ok) {
          throw new Error(data.error || 'start error');
        }
        currentRun = data.run_id;
        dbg('run started', { run_id: currentRun });
        logEl.textContent = 'Run started: ' + currentRun;
        await loadPlan();
        pollStatus();
      } catch (err) {
        setState('Error', 'fail');
        logEl.textContent = String(err);
        dbg('start error', { error: String(err) });
        startBtn.disabled = false;
      }
    });
  </script>
</body>
</html>
`

type StepResult struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type RunStatus struct {
	RunID               string          `json:"run_id,omitempty"`
	State               string          `json:"state"`
	StartedAt           *time.Time      `json:"started_at,omitempty"`
	EndedAt             *time.Time      `json:"ended_at,omitempty"`
	Success             bool            `json:"success,omitempty"`
	Steps               []StepResult    `json:"steps,omitempty"`
	Logs                []string        `json:"logs,omitempty"`
	ScreenshotURL       string          `json:"screenshot_url,omitempty"`
	ScreenshotSizeBytes int64           `json:"screenshot_size_bytes,omitempty"`
	KeyboardReport      *KeyboardReport `json:"keyboard_report,omitempty"`
	MouseReport         *MouseReport    `json:"mouse_report,omitempty"`
	Error               string          `json:"error,omitempty"`
}

type MouseArea struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ClientInfo struct {
	ViewportWidth    int       `json:"viewport_width"`
	ViewportHeight   int       `json:"viewport_height"`
	DevicePixelRatio float64   `json:"device_pixel_ratio"`
	MouseArea        MouseArea `json:"mouse_area"`
	KeyboardInput    MouseArea `json:"keyboard_input"`
	ViewportOffsetX  float64   `json:"viewport_offset_x"`
	ViewportOffsetY  float64   `json:"viewport_offset_y"`
	UserAgent        string    `json:"user_agent,omitempty"`
}

type StartRequest struct {
	Client ClientInfo `json:"client"`
}

type KeyboardPlan struct {
	Text           string `json:"text"`
	HoldKey        string `json:"hold_key"`
	HoldMinMs      int    `json:"hold_min_ms"`
	NumpadSequence string `json:"numpad_sequence"`
}

type MouseTargetPlan struct {
	ID     string  `json:"id"`
	XNorm  float64 `json:"x_norm"`
	YNorm  float64 `json:"y_norm"`
	Clicks int     `json:"clicks"`
}

type DragPlan struct {
	StartXNorm float64 `json:"start_x_norm"`
	StartYNorm float64 `json:"start_y_norm"`
	EndXNorm   float64 `json:"end_x_norm"`
	EndYNorm   float64 `json:"end_y_norm"`
}

type RunPlan struct {
	Keyboard     KeyboardPlan      `json:"keyboard"`
	MouseTargets []MouseTargetPlan `json:"mouse_targets"`
	Drag         DragPlan          `json:"drag"`
}

type KeyboardReport struct {
	Text           string   `json:"text"`
	TextMatch      bool     `json:"text_match"`
	HoldKey        string   `json:"hold_key"`
	HoldDurationMs int64    `json:"hold_duration_ms"`
	HoldPass       bool     `json:"hold_pass"`
	NumpadSequence string   `json:"numpad_sequence"`
	NumpadPass     bool     `json:"numpad_pass"`
	Pass           bool     `json:"pass"`
	Errors         []string `json:"errors,omitempty"`
}

type MouseTargetReport struct {
	ID             string  `json:"id"`
	ExpectedClicks int     `json:"expected_clicks"`
	ActualClicks   int     `json:"actual_clicks"`
	Hit            bool    `json:"hit"`
	ClickX         float64 `json:"click_x"`
	ClickY         float64 `json:"click_y"`
}

type DragReport struct {
	StartHit bool `json:"start_hit"`
	EndHit   bool `json:"end_hit"`
	Moved    bool `json:"moved"`
}

type MouseReport struct {
	Targets []MouseTargetReport `json:"targets"`
	Drag    DragReport          `json:"drag"`
	Pass    bool                `json:"pass"`
	Errors  []string            `json:"errors,omitempty"`
}

type runState struct {
	status              RunStatus
	screenshotPath      string
	client              ClientInfo
	plan                RunPlan
	keyboardReport      *KeyboardReport
	mouseReport         *MouseReport
	keyboardReady       chan struct{}
	mouseReady          chan struct{}
	keyboardReadyClosed bool
	mouseReadyClosed    bool
}

type runManager struct {
	mu      sync.Mutex
	current *runState
}

var manager = &runManager{}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/start", handleStart)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/plan", handlePlan)
	mux.HandleFunc("/api/ready", handleReady)
	mux.HandleFunc("/api/report/keyboard", handleKeyboardReport)
	mux.HandleFunc("/api/report/mouse", handleMouseReport)
	mux.HandleFunc("/result/", handleResult)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	addr := listener.Addr().String()
	log.Printf("DeskAct tester running at http://%s", addr)

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req StartRequest
	if r.Body != nil {
		dec := json.NewDecoder(r.Body)
		_ = dec.Decode(&req)
	}
	status, err := manager.Start(req.Client)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"run_id": status.RunID})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	status, err := manager.Status(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	plan, err := manager.Plan(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RunID  string      `json:"run_id"`
		Step   string      `json:"step"`
		Client *ClientInfo `json:"client,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ready payload"})
		return
	}
	if req.Client != nil {
		if err := manager.UpdateClient(req.RunID, *req.Client); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
	}
	if err := manager.MarkReady(req.RunID, req.Step); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleKeyboardReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RunID  string         `json:"run_id"`
		Report KeyboardReport `json:"report"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid report"})
		return
	}
	if err := manager.StoreKeyboardReport(req.RunID, req.Report); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleMouseReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RunID  string      `json:"run_id"`
		Report MouseReport `json:"report"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid report"})
		return
	}
	if err := manager.StoreMouseReport(req.RunID, req.Report); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleResult(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/result/")
	path, ok := manager.ScreenshotPath(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (m *runManager) Start(client ClientInfo) (RunStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil && m.current.status.State == stateRunning {
		return RunStatus{}, errors.New("test already running")
	}

	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now()
	normalizeClientInfo(&client)
	plan := buildPlan()
	status := RunStatus{
		RunID:     runID,
		State:     stateRunning,
		StartedAt: &now,
		Steps: []StepResult{
			{Name: "Keyboard", Status: stepPending},
			{Name: "Mouse", Status: stepPending},
			{Name: "Screenshot", Status: stepPending},
		},
		Logs: []string{},
	}

	m.current = &runState{
		status:        status,
		client:        client,
		plan:          plan,
		keyboardReady: make(chan struct{}),
		mouseReady:    make(chan struct{}),
	}
	go m.run(runID)

	return cloneStatus(status), nil
}

func (m *runManager) Status(runID string) (RunStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		return RunStatus{State: stateIdle}, nil
	}
	if runID != "" && m.current.status.RunID != runID {
		return RunStatus{}, errors.New("run not found")
	}
	return cloneStatus(m.current.status), nil
}

func (m *runManager) Plan(runID string) (RunPlan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return RunPlan{}, errors.New("run not found")
	}
	if runID != "" && m.current.status.RunID != runID {
		return RunPlan{}, errors.New("run not found")
	}
	return m.current.plan, nil
}

func (m *runManager) StoreKeyboardReport(runID string, report KeyboardReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return errors.New("run not found")
	}
	report = evaluateKeyboardReport(m.current.plan.Keyboard, report)
	m.current.keyboardReport = &report
	m.current.status.KeyboardReport = &report
	return nil
}

func (m *runManager) UpdateClient(runID string, client ClientInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return errors.New("run not found")
	}
	normalizeClientInfo(&client)
	m.current.client = client
	return nil
}

func (m *runManager) StoreMouseReport(runID string, report MouseReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return errors.New("run not found")
	}
	report = evaluateMouseReport(m.current.plan, report)
	m.current.mouseReport = &report
	m.current.status.MouseReport = &report
	return nil
}

func (m *runManager) MarkReady(runID, step string) error {
	step = strings.ToLower(step)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return errors.New("run not found")
	}
	switch step {
	case "keyboard":
		if !m.current.keyboardReadyClosed {
			close(m.current.keyboardReady)
			m.current.keyboardReadyClosed = true
		}
	case "mouse":
		if !m.current.mouseReadyClosed {
			close(m.current.mouseReady)
			m.current.mouseReadyClosed = true
		}
	default:
		return errors.New("unknown step")
	}
	return nil
}

func (m *runManager) readyChannel(runID, step string) (chan struct{}, error) {
	step = strings.ToLower(step)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return nil, errors.New("run not found")
	}
	switch step {
	case "keyboard":
		return m.current.keyboardReady, nil
	case "mouse":
		return m.current.mouseReady, nil
	default:
		return nil, errors.New("unknown step")
	}
}

func (m *runManager) waitForReady(runID, step string, timeout time.Duration) error {
	ch, err := m.readyChannel(runID, step)
	if err != nil {
		return err
	}
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return errors.New("frontend not ready")
	}
}

func (m *runManager) ScreenshotPath(runID string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return "", false
	}
	if m.current.screenshotPath == "" {
		return "", false
	}
	return m.current.screenshotPath, true
}

func (m *runManager) runContext(runID string) (ClientInfo, RunPlan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return ClientInfo{}, RunPlan{}
	}
	return m.current.client, m.current.plan
}

func (m *runManager) run(runID string) {
	m.appendLog(runID, "Test started")
	time.Sleep(350 * time.Millisecond)

	keySettings := deskact.DefaultKeyboardSettings()
	mouseSettings := deskact.DefaultMouseSettings()
	display := detectActiveDisplay()
	if display != nil {
		m.appendLog(runID, fmt.Sprintf("Display %d: %dx%d", display.Index(), display.Width(), display.Height()))
	} else {
		m.appendLog(runID, "No display detected")
	}

	m.runStep(runID, 0, "Keyboard test", func() error {
		client, plan := m.runContext(runID)
		return runKeyboardTest(runID, plan.Keyboard, client, display, keySettings)
	})
	m.runStep(runID, 1, "Mouse test", func() error {
		client, plan := m.runContext(runID)
		return runMouseTest(runID, plan, client, display, mouseSettings)
	})
	m.runStep(runID, 2, "Screenshot test", func() error {
		return runScreenshotTest(runID, display)
	})

	m.finish(runID)
	m.appendLog(runID, "Test completed")
}

func (m *runManager) runStep(runID string, index int, label string, fn func() error) {
	m.setStepStatus(runID, index, stepRunning, "")
	m.appendLog(runID, label+" started")
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	if err != nil {
		m.setStepStatus(runID, index, stepFail, err.Error())
		m.appendLog(runID, label+" failed: "+err.Error())
	} else {
		m.setStepStatus(runID, index, stepPass, "")
		m.appendLog(runID, label+" passed")
	}
	m.setStepDuration(runID, index, duration)
}

func (m *runManager) setStepStatus(runID string, index int, status string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return
	}
	if index < 0 || index >= len(m.current.status.Steps) {
		return
	}
	m.current.status.Steps[index].Status = status
	m.current.status.Steps[index].Error = errMsg
}

func (m *runManager) setStepDuration(runID string, index int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return
	}
	if index < 0 || index >= len(m.current.status.Steps) {
		return
	}
	m.current.status.Steps[index].DurationMs = duration.Milliseconds()
}

func (m *runManager) appendLog(runID string, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return
	}
	entry := time.Now().Format("15:04:05") + " " + message
	m.current.status.Logs = append(m.current.status.Logs, entry)
}

func (m *runManager) finish(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.RunID != runID {
		return
	}
	now := time.Now()
	m.current.status.State = stateDone
	m.current.status.EndedAt = &now
	success := true
	for _, step := range m.current.status.Steps {
		if step.Status != stepPass {
			success = false
			break
		}
	}
	m.current.status.Success = success
}

func cloneStatus(status RunStatus) RunStatus {
	cloned := status
	if status.Steps != nil {
		cloned.Steps = make([]StepResult, len(status.Steps))
		copy(cloned.Steps, status.Steps)
	}
	if status.Logs != nil {
		cloned.Logs = make([]string, len(status.Logs))
		copy(cloned.Logs, status.Logs)
	}
	if status.KeyboardReport != nil {
		kr := *status.KeyboardReport
		cloned.KeyboardReport = &kr
	}
	if status.MouseReport != nil {
		mr := *status.MouseReport
		cloned.MouseReport = &mr
	}
	return cloned
}

func runKeyboardTest(runID string, plan KeyboardPlan, client ClientInfo, display *deskact.Display, settings deskact.KeyboardSettings) error {
	if err := manager.waitForReady(runID, "keyboard", 10*time.Second); err != nil {
		return err
	}
	updatedClient, updatedPlan := manager.runContext(runID)
	client = updatedClient
	plan = updatedPlan.Keyboard
	if display == nil {
		return errors.New("no display detected")
	}
	if plan.Text == "" {
		plan.Text = "DeskAct keyboard test OK"
	}
	if plan.HoldKey == "" {
		plan.HoldKey = "Shift"
	}
	if plan.HoldMinMs <= 0 {
		plan.HoldMinMs = 400
	}
	if plan.NumpadSequence == "" {
		plan.NumpadSequence = "123"
	}

	if err := focusKeyboardInput(client, display, deskact.DefaultMouseSettings()); err != nil {
		manager.appendLog(runID, "Keyboard focus warning: "+err.Error())
	}
	time.Sleep(150 * time.Millisecond)

	manager.appendLog(runID, "Typing expected text")
	deskact.Type(plan.Text, 0, settings)
	time.Sleep(150 * time.Millisecond)

	holdKey := strings.ToLower(plan.HoldKey)
	manager.appendLog(runID, fmt.Sprintf("Holding key %s", plan.HoldKey))
	if err := deskact.KeyToggle(holdKey, true, nil, settings); err != nil {
		return err
	}
	time.Sleep(time.Duration(plan.HoldMinMs+200) * time.Millisecond)
	if err := deskact.KeyToggle(holdKey, false, nil, settings); err != nil {
		return err
	}

	manager.appendLog(runID, "Typing numpad sequence "+plan.NumpadSequence)
	for _, ch := range plan.NumpadSequence {
		key := "num" + string(ch)
		if err := deskact.KeyTap(key, nil, settings); err != nil {
			return err
		}
		time.Sleep(80 * time.Millisecond)
	}

	report, err := manager.waitForKeyboardReport(runID, 9*time.Second)
	if err != nil {
		return err
	}
	if !report.Pass {
		return reportError("keyboard validation failed", report.Errors)
	}
	return nil
}

func runMouseTest(runID string, plan RunPlan, client ClientInfo, display *deskact.Display, settings deskact.MouseSettings) error {
	if display == nil {
		return errors.New("no display detected")
	}
	if err := manager.waitForReady(runID, "mouse", 12*time.Second); err != nil {
		return err
	}
	updatedClient, updatedPlan := manager.runContext(runID)
	client = updatedClient
	plan = updatedPlan
	if err := primeMouseArea(client, display, settings); err != nil {
		manager.appendLog(runID, "Mouse focus warning: "+err.Error())
	}
	time.Sleep(350 * time.Millisecond)
	width, height := display.Width(), display.Height()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid display size: %dx%d", width, height)
	}
	if len(plan.MouseTargets) == 0 {
		return errors.New("mouse plan not available")
	}
	tolerance := maxInt(6, minInt(26, maxInt(width, height)/140))
	normalizeClientInfo(&client)

	for _, target := range plan.MouseTargets {
		pt := targetToPhysicalPoint(target, client, display)
		manager.appendLog(runID, fmt.Sprintf("Mouse target %s (%dx) at %d,%d", target.ID, target.Clicks, pt.X, pt.Y))
		if err := verifyMouseMove(display, pt, settings, tolerance); err != nil {
			return fmt.Errorf("move to %s: %w", target.ID, err)
		}
		var err error
		if target.Clicks <= 1 {
			err = deskact.Click(deskact.MouseButtonLeft, false, settings)
		} else {
			err = deskact.MultiClick(deskact.MouseButtonLeft, target.Clicks, settings)
		}
		if err != nil {
			return err
		}
		time.Sleep(150 * time.Millisecond)
		if err := verifyMouseAt(display, pt, tolerance); err != nil {
			return fmt.Errorf("click verify %s: %w", target.ID, err)
		}
	}

	dragStart := targetPointFromNorm(plan.Drag.StartXNorm, plan.Drag.StartYNorm, client, display)
	dragEnd := targetPointFromNorm(plan.Drag.EndXNorm, plan.Drag.EndYNorm, client, display)
	manager.appendLog(runID, fmt.Sprintf("Drag from %d,%d to %d,%d", dragStart.X, dragStart.Y, dragEnd.X, dragEnd.Y))
	if err := verifyMouseMove(display, dragStart, settings, tolerance); err != nil {
		return fmt.Errorf("drag start: %w", err)
	}
	if err := deskact.Toggle(deskact.MouseButtonLeft, true, false, settings); err != nil {
		return err
	}
	err := display.MoveSmooth(dragEnd.X, dragEnd.Y, settings)
	time.Sleep(150 * time.Millisecond)
	if toggleErr := deskact.Toggle(deskact.MouseButtonLeft, false, false, settings); toggleErr != nil && err == nil {
		err = toggleErr
	}
	time.Sleep(120 * time.Millisecond)
	if err != nil {
		return fmt.Errorf("drag move failed: %w", err)
	}
	if err := verifyMouseAt(display, dragEnd, tolerance); err != nil {
		return fmt.Errorf("drag end: %w", err)
	}

	report, err := manager.waitForMouseReport(runID, 12*time.Second)
	if err != nil {
		return err
	}
	if !report.Pass {
		return reportError("mouse validation failed", report.Errors)
	}
	return nil
}

func runScreenshotTest(runID string, display *deskact.Display) error {
	if display == nil {
		return errors.New("no display detected")
	}
	width, height := display.Width(), display.Height()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid display size: %dx%d", width, height)
	}
	time.Sleep(300 * time.Millisecond)
	img, err := display.CaptureRect(0, 0, width, height, deskact.DefaultCaptureOptions())
	if err != nil {
		return err
	}
	outputDir := filepath.Join(".", ".deskact-tester")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(outputDir, "run-"+runID+".png")
	size, err := savePNG(img, outputPath)
	if err != nil {
		return err
	}
	manager.mu.Lock()
	if manager.current != nil && manager.current.status.RunID == runID {
		manager.current.screenshotPath = outputPath
		manager.current.status.ScreenshotURL = "/result/" + runID
		manager.current.status.ScreenshotSizeBytes = size
	}
	manager.mu.Unlock()
	return nil
}

func savePNG(img image.Image, path string) (int64, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func verifyMouseMove(display *deskact.Display, target deskact.Point, settings deskact.MouseSettings, tolerance int) error {
	if err := display.Move(target.X, target.Y, settings); err != nil {
		return err
	}
	time.Sleep(140 * time.Millisecond)
	return verifyMouseAt(display, target, tolerance)
}

func verifyMouseAt(display *deskact.Display, target deskact.Point, tolerance int) error {
	x, y, ok := display.MouseLocation()
	if !ok {
		return errors.New("mouse not on target display")
	}
	dx := absInt(x - target.X)
	dy := absInt(y - target.Y)
	if dx > tolerance || dy > tolerance {
		return fmt.Errorf("expected (%d,%d) got (%d,%d) tol=%d", target.X, target.Y, x, y, tolerance)
	}
	return nil
}

func (m *runManager) waitForKeyboardReport(runID string, timeout time.Duration) (KeyboardReport, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		if m.current != nil && m.current.status.RunID == runID && m.current.keyboardReport != nil {
			report := *m.current.keyboardReport
			m.mu.Unlock()
			return report, nil
		}
		m.mu.Unlock()
		time.Sleep(80 * time.Millisecond)
	}
	return KeyboardReport{}, errors.New("keyboard report timeout")
}

func (m *runManager) waitForMouseReport(runID string, timeout time.Duration) (MouseReport, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		if m.current != nil && m.current.status.RunID == runID && m.current.mouseReport != nil {
			report := *m.current.mouseReport
			m.mu.Unlock()
			return report, nil
		}
		m.mu.Unlock()
		time.Sleep(80 * time.Millisecond)
	}
	return MouseReport{}, errors.New("mouse report timeout")
}

func normalizeClientInfo(client *ClientInfo) {
	if client.DevicePixelRatio <= 0 {
		client.DevicePixelRatio = 1
	}
	if client.ViewportWidth < 0 {
		client.ViewportWidth = 0
	}
	if client.ViewportHeight < 0 {
		client.ViewportHeight = 0
	}
	if client.MouseArea.Width < 0 {
		client.MouseArea.Width = 0
	}
	if client.MouseArea.Height < 0 {
		client.MouseArea.Height = 0
	}
	if client.MouseArea.Width == 0 && client.ViewportWidth > 0 {
		client.MouseArea.Width = float64(client.ViewportWidth)
	}
	if client.MouseArea.Height == 0 && client.ViewportHeight > 0 {
		client.MouseArea.Height = float64(client.ViewportHeight)
	}
	if client.MouseArea.Left < 0 {
		client.MouseArea.Left = 0
	}
	if client.MouseArea.Top < 0 {
		client.MouseArea.Top = 0
	}
	if client.KeyboardInput.Width < 0 {
		client.KeyboardInput.Width = 0
	}
	if client.KeyboardInput.Height < 0 {
		client.KeyboardInput.Height = 0
	}
	if client.KeyboardInput.Left < 0 {
		client.KeyboardInput.Left = 0
	}
	if client.KeyboardInput.Top < 0 {
		client.KeyboardInput.Top = 0
	}
	if client.ViewportOffsetX < 0 {
		client.ViewportOffsetX = 0
	}
	if client.ViewportOffsetY < 0 {
		client.ViewportOffsetY = 0
	}
}

func buildPlan() RunPlan {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	targets := make([]MouseTargetPlan, 0, 3)
	clicks := []int{1, 2, 3}
	for i, count := range clicks {
		x, y := randomTarget(rng, targets)
		targets = append(targets, MouseTargetPlan{
			ID:     fmt.Sprintf("t%d", i+1),
			XNorm:  x,
			YNorm:  y,
			Clicks: count,
		})
	}
	return RunPlan{
		Keyboard: KeyboardPlan{
			Text:           "DeskAct keyboard test OK",
			HoldKey:        "Shift",
			HoldMinMs:      400,
			NumpadSequence: "123",
		},
		MouseTargets: targets,
		Drag: DragPlan{
			StartXNorm: 0.2,
			StartYNorm: 0.6,
			EndXNorm:   0.8,
			EndYNorm:   0.6,
		},
	}
}

func randomTarget(rng *rand.Rand, existing []MouseTargetPlan) (float64, float64) {
	const min = 0.15
	const max = 0.85
	for i := 0; i < 12; i++ {
		x := min + rng.Float64()*(max-min)
		y := min + rng.Float64()*(max-min)
		if farEnough(x, y, existing) {
			return x, y
		}
	}
	return min + rng.Float64()*(max-min), min + rng.Float64()*(max-min)
}

func farEnough(x, y float64, existing []MouseTargetPlan) bool {
	const minDistSq = 0.04
	for _, t := range existing {
		dx := x - t.XNorm
		dy := y - t.YNorm
		if dx*dx+dy*dy < minDistSq {
			return false
		}
	}
	return true
}

func targetToPhysicalPoint(target MouseTargetPlan, client ClientInfo, display *deskact.Display) deskact.Point {
	return targetPointFromNorm(target.XNorm, target.YNorm, client, display)
}

func targetPointFromNorm(xNorm, yNorm float64, client ClientInfo, display *deskact.Display) deskact.Point {
	dpr := client.DevicePixelRatio
	if dpr <= 0 {
		dpr = 1
	}
	area := client.MouseArea
	if area.Width <= 0 || area.Height <= 0 {
		area = MouseArea{
			Left:   0,
			Top:    0,
			Width:  float64(display.Width()),
			Height: float64(display.Height()),
		}
		dpr = 1
	}
	cssX := client.ViewportOffsetX + area.Left + area.Width*xNorm
	cssY := client.ViewportOffsetY + area.Top + area.Height*yNorm
	physX := int(cssX * dpr)
	physY := int(cssY * dpr)
	if physX < 0 {
		physX = 0
	}
	if physY < 0 {
		physY = 0
	}
	if physX >= display.Width() {
		physX = display.Width() - 1
	}
	if physY >= display.Height() {
		physY = display.Height() - 1
	}
	return deskact.Point{X: physX, Y: physY}
}

func rectCenterPoint(rect MouseArea, dpr float64, offsetX, offsetY float64, display *deskact.Display) deskact.Point {
	if dpr <= 0 {
		dpr = 1
	}
	cssX := offsetX + rect.Left + rect.Width*0.5
	cssY := offsetY + rect.Top + rect.Height*0.5
	physX := int(cssX * dpr)
	physY := int(cssY * dpr)
	if physX < 0 {
		physX = 0
	}
	if physY < 0 {
		physY = 0
	}
	if physX >= display.Width() {
		physX = display.Width() - 1
	}
	if physY >= display.Height() {
		physY = display.Height() - 1
	}
	return deskact.Point{X: physX, Y: physY}
}

func focusKeyboardInput(client ClientInfo, display *deskact.Display, settings deskact.MouseSettings) error {
	rect := client.KeyboardInput
	if rect.Width <= 0 || rect.Height <= 0 {
		return errors.New("keyboard input bounds missing")
	}
	pt := rectCenterPoint(rect, client.DevicePixelRatio, client.ViewportOffsetX, client.ViewportOffsetY, display)
	if err := display.Move(pt.X, pt.Y, settings); err != nil {
		return err
	}
	time.Sleep(120 * time.Millisecond)
	if err := deskact.Click(deskact.MouseButtonLeft, false, settings); err != nil {
		return err
	}
	time.Sleep(120 * time.Millisecond)
	return deskact.Click(deskact.MouseButtonLeft, false, settings)
}

func primeMouseArea(client ClientInfo, display *deskact.Display, settings deskact.MouseSettings) error {
	area := client.MouseArea
	if area.Width <= 0 || area.Height <= 0 {
		return errors.New("mouse area bounds missing")
	}
	pt := targetPointFromNorm(0.5, 0.5, client, display)
	if err := display.Move(pt.X, pt.Y, settings); err != nil {
		return err
	}
	time.Sleep(120 * time.Millisecond)
	return deskact.Click(deskact.MouseButtonLeft, false, settings)
}

func evaluateKeyboardReport(plan KeyboardPlan, report KeyboardReport) KeyboardReport {
	if plan.Text == "" {
		plan.Text = "DeskAct keyboard test OK"
	}
	if plan.HoldKey == "" {
		plan.HoldKey = "Shift"
	}
	if plan.HoldMinMs <= 0 {
		plan.HoldMinMs = 400
	}
	if plan.NumpadSequence == "" {
		plan.NumpadSequence = "123"
	}
	report.HoldKey = plan.HoldKey
	report.TextMatch = report.Text == plan.Text
	report.HoldPass = report.HoldDurationMs >= int64(plan.HoldMinMs)
	report.NumpadPass = report.NumpadSequence == plan.NumpadSequence
	report.Pass = report.TextMatch && report.HoldPass && report.NumpadPass
	if !report.TextMatch {
		report.Errors = append(report.Errors, "typed text mismatch")
	}
	if !report.HoldPass {
		report.Errors = append(report.Errors, "hold duration too short")
	}
	if !report.NumpadPass {
		report.Errors = append(report.Errors, "numpad sequence mismatch")
	}
	return report
}

func evaluateMouseReport(plan RunPlan, report MouseReport) MouseReport {
	report.Pass = true
	if len(plan.MouseTargets) == 0 {
		report.Pass = false
		report.Errors = append(report.Errors, "mouse plan missing")
		return report
	}
	reportTargets := make(map[string]MouseTargetReport)
	for _, t := range report.Targets {
		reportTargets[t.ID] = t
	}
	for _, target := range plan.MouseTargets {
		rt, ok := reportTargets[target.ID]
		if !ok {
			report.Pass = false
			report.Errors = append(report.Errors, "missing target "+target.ID)
			continue
		}
		if !rt.Hit || rt.ActualClicks != target.Clicks {
			report.Pass = false
			report.Errors = append(report.Errors, "target "+target.ID+" mismatch")
		}
	}
	if !(report.Drag.StartHit && report.Drag.EndHit && report.Drag.Moved) {
		report.Pass = false
		report.Errors = append(report.Errors, "drag mismatch")
	}
	return report
}

func reportError(prefix string, errs []string) error {
	if len(errs) == 0 {
		return errors.New(prefix)
	}
	return errors.New(prefix + ": " + strings.Join(errs, "; "))
}

func detectActiveDisplay() *deskact.Display {
	options := deskact.DefaultDisplayOptions()
	displays := deskact.AllDisplays(options)
	absX, absY := deskact.Location()
	for _, display := range displays {
		if display.Contains(absX, absY) {
			return display
		}
	}
	return deskact.MainDisplay(options)
}

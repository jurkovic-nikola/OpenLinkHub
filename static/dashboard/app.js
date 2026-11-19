/* Cooling — a dashboard for OpenLinkHub hubs and AIOs, driven entirely by OpenLinkHub's REST API (same origin). */
"use strict";

const POLL_MS = 2000;
const SLOW_POLL_MS = 15000;
const HISTORY_LEN = 300; // 300 × 2 s = 10 min
const BUILTIN = new Set(["Normal", "Quiet", "Performance"]);
const SENSORS = { 0: "CPU", 1: "GPU", 2: "Coolant", 3: "Drive", 4: "Probe", 5: "CPU + GPU", 6: "hwmon", 7: "Script", 8: "GPU #", 9: "Probe", 10: "PSU", 11: "GPUs" };
const NEW_SENSORS = [[0, "CPU"], [1, "GPU"], [2, "Coolant"], [5, "Hotter of CPU and GPU"], [3, "Drive"], [4, "Probe on this hub"], [9, "Probe on any device"], [8, "Specific GPU"], [11, "Hottest of several GPUs"], [6, "Any hwmon sensor"], [7, "Executable"]];
/* Page-only warning thresholds (°C). OpenLinkHub has no coolant failsafe for most hubs. */
const ALERTS = { liquid: 50, cpu: 95, gpu: 87 };
/* Effect field rules (mirror the stock RGB editor; see static/js/rgb.js). */
const FX_HIDDEN = new Set(["off", "led"]);
const FX_TEMP = new Set(["cpu-temperature", "gpu-temperature", "liquid-temperature"]);
const FX_SPEED_ONLY = new Set(["rainbow", "pastelrainbow", "spiralrainbow", "pastelspiralrainbow"]);
const FX_NO_SPEED = new Set(["static", "storm", "off", "probe-temperature", ...FX_TEMP]);
const FX_SLOW = new Set(["spiralrainbow", "rain"]); // speed 1–3
/* Effects a driver lists but cannot render (cc.go has no "led" case). */
const UNRENDERED = { "iCUE COMMANDER Core": new Set(["led"]) };

const $ = (s, root = document) => root.querySelector(s);
const $$ = (s, root = document) => [...root.querySelectorAll(s)];
const el = (tag, attrs = {}, ...kids) => {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") n.className = v;
    else if (k === "text") n.textContent = v;
    else if (k.startsWith("on")) n.addEventListener(k.slice(2), v);
    else if (v !== undefined && v !== null && v !== false) n.setAttribute(k, v === true ? "" : v);
  }
  for (const kid of kids) if (kid != null) n.append(kid);
  return n;
};
const SVGNS = "http://www.w3.org/2000/svg";
const sv = (tag, attrs = {}, text) => {
  const n = document.createElementNS(SVGNS, tag);
  for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, v);
  if (text != null) n.textContent = text;
  return n;
};
const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
const fmt = (v, d = 0) => (v == null || Number.isNaN(v) ? "—" : Number(v).toFixed(d));
const cssVar = (name) => getComputedStyle(document.documentElement).getPropertyValue(name).trim();
const hex = (c) => "#" + [c?.red ?? 0, c?.green ?? 0, c?.blue ?? 0].map((v) => Math.round(v).toString(16).padStart(2, "0")).join("");
const rgbOf = (h) => ({ red: parseInt(h.slice(1, 3), 16), green: parseInt(h.slice(3, 5), 16), blue: parseInt(h.slice(5, 7), 16) });
/* Display units follow the shared dashboard setting; everything sent to the API stays °C. */
const useF = () => S.dash && S.dash.celsius === false;
const tU = (c) => (c == null ? null : useF() ? c * 9 / 5 + 32 : c);
const uT = () => (useF() ? "°F" : "°C");
const fmtT = (c, d = 0) => `${fmt(tU(c), d)} ${uT()}`;

/* ---------- API ---------- */
class AuthError extends Error {}
async function api(path, { method = "GET", body, form } = {}) {
  const res = await fetch(path, {
    method,
    headers: body ? { "Content-Type": "application/json" } : form ? { "Content-Type": "application/x-www-form-urlencoded" } : undefined,
    body: body ? JSON.stringify(body) : form ? new URLSearchParams(form) : undefined,
    cache: "no-store",
  });
  if (res.status === 401 || (res.redirected && new URL(res.url).pathname === "/login")) throw new AuthError("Sign-in required");
  if (!res.ok) throw new Error(`${method} ${path}: HTTP ${res.status}`);
  return res.json();
}
async function act(path, method, body, okMsg, opts = {}) {
  try {
    const r = await api(path, { method, body: opts.form ? undefined : body, form: opts.form });
    if (r.status === 1) { if (okMsg !== false) toast(okMsg || r.message || "Saved"); return true; }
    toast(r.message || "No change", "warn");
    return false;
  } catch (e) {
    toast(e.message, "err");
    return false;
  }
}
function toast(msg, kind = "ok") {
  const t = el("div", { class: `toast ${kind}`, role: kind === "err" ? "alert" : "status", text: msg });
  $("#toasts").append(t);
  setTimeout(() => t.remove(), kind === "ok" ? 2600 : 5500);
}
/* Rebuild part of the page without dropping keyboard focus: focusable rebuilt elements carry data-fk. */
function keepFocus(rebuild) {
  const k = document.activeElement?.dataset?.fk;
  rebuild();
  if (k) document.querySelector(`[data-fk="${CSS.escape(k)}"]`)?.focus();
}
/* Two-click confirm for destructive buttons. */
function armed(btn, label) {
  if (btn.dataset.armed === "1") { btn.dataset.armed = ""; btn.textContent = btn.dataset.label; return true; }
  btn.dataset.label = btn.textContent;
  btn.dataset.armed = "1";
  btn.textContent = label;
  setTimeout(() => { if (btn.dataset.armed === "1") { btn.dataset.armed = ""; btn.textContent = btn.dataset.label; } }, 3500);
  return false;
}

/* ---------- State ---------- */
const S = {
  serial: null,
  devices: [],           // devices with fan/pump channels
  all: [],               // every visible device
  dev: null,             // GetDevice of the selected device
  profiles: {},
  rgb: null,             // /api/color/<serial> data
  dash: null,            // /api/dashboard
  sources: null,         // sensor choices scraped from /temperature
  sensors: { cpu: null, gpu: null, cpuLoad: null, gpuLoad: null, drive: null, driveName: "" },
  hist: [],
  rpmMax: {},
  curve: { name: null, type: 1, points: null, saved: null, dirty: false, drag: -1, bands: null },
  fx: { name: null, draft: null },
  graphMode: true,       // config.json graphProfiles (detected from the stock /temperature page)
  sel: new Set(),        // selected channel ids
  cluster: null,         // /api/devices/cluster when this hub has joined the RGB cluster
  hiddenSeries: new Set(),
  built: { channels: "", profiles: "", leds: "", device: "" },
};

const channels = () => Object.values(S.dev?.devices || {}).sort((a, b) => a.channelId - b.channelId);
const speedChannels = () => channels().filter((c) => c.HasSpeed);
const isPump = (c) => !!(c.ContainsPump || c.AIO || /^(AIO|Pump\/Res)$/i.test(c.description || ""));
const pumpChannel = () => channels().find(isPump);
const fanChannels = () => speedChannels().filter((c) => !isPump(c));
const liquidTemp = () => { const p = pumpChannel(); return p && p.temperature > 0 ? p.temperature : null; };
const realLabel = (l) => (l && l !== "Set Label" ? l : null);
const displayName = (c) => realLabel(c.label) || (isPump(c) ? "Pump" : c.HasSpeed ? `Fan ${c.channelId}` : c.name);
/* When this hub has joined the RGB cluster, lighting is driven through the virtual "cluster" device. */
const inCluster = () => !!S.dev?.DeviceProfile?.RGBCluster && !!S.cluster;
const lightId = () => (inCluster() ? "cluster" : S.serial);
const rgbModes = () => {
  if (inCluster()) return S.cluster.RGBModes || [];
  const skip = UNRENDERED[S.dev?.product] || new Set();
  return (S.dev?.RGBModes || Object.keys(S.rgb?.profiles || {})).filter((m) => !skip.has(m));
};
const prof = () => S.dev?.DeviceProfile || {};

function sensorTemp(p) {
  if (!p) return null;
  const { cpu, gpu } = S.sensors;
  switch (Number(p.sensor)) {
    case 0: return cpu;
    case 1: return gpu;
    case 8: return S.sensors.gpus?.[p.gpuIndex] ?? (p.gpuIndex ? null : gpu);
    case 11: { const v = Object.values(S.sensors.gpus || {}); return v.length ? Math.max(...v) : gpu; }
    case 2: return liquidTemp();
    case 5: return cpu == null && gpu == null ? null : Math.max(cpu ?? 0, gpu ?? 0);
    case 3: return S.sensors.drive;
    case 4: case 9: {
      if (p.device && p.device !== S.serial) return null; // probe on another device: not polled here
      const ch = S.dev?.devices?.[p.channelId];
      return ch?.temperature > 0 ? ch.temperature : null;
    }
    default: return null;
  }
}
const axisMax = (p) => (Number(p?.sensor) === 2 ? 60 : 100); // coolant curves live in 20–60 °C

function interp(points, x) {
  if (!points?.length) return null;
  const p = [...points].sort((a, b) => a.x - b.x);
  if (x <= p[0].x) return p[0].y;
  for (let i = 1; i < p.length; i++) {
    if (x <= p[i].x) { const a = p[i - 1], b = p[i]; return a.y + ((b.y - a.y) * (x - a.x)) / (b.x - a.x || 1); }
  }
  return p[p.length - 1].y;
}
/* Mirrors cc.go clamps: fans ≥20 unless zero-rpm; pump <50 → 70 in graph mode, → 50 in table mode. */
function effective(p, type, raw) {
  if (raw == null) return null;
  let v = Math.round(raw);
  if (type === 1 && v < 20 && !p.zeroRpm) v = 20;
  if (type === 0 && v < 50) v = S.graphMode ? 70 : 50;
  return Math.min(v, 100);
}
/* Speed a profile asks for at temperature t (graph = interpolated points, table = min/max bands). */
function profileSpeed(p, type, t) {
  if (!p || t == null) return null;
  if (S.graphMode) return interp(p.points?.[type], t);
  const band = (p.profiles || []).find((b) => t >= b.min && t <= b.max) || (p.profiles || []).at(-1);
  return band ? (type === 0 ? band.pump : band.fans) : null;
}
const isFlat = (p) => [0, 1].every((t) => { const ys = (p?.points?.[t] || []).map((q) => q.y); return ys.length > 0 && ys.every((y) => y === ys[0]); });

/* ---------- Polling ---------- */
async function loadDevices() {
  const r = await api("/api/devices/");
  S.all = Object.values(r.devices || {}).filter((d) => !d.Hidden && d.Serial !== "cluster").sort((a, b) => a.Product.localeCompare(b.Product));
  const PSU = new Set([501, 502]); // common.ProductTypePSUHid / ProductTypePSUDongle
  S.devices = S.all.filter((d) => !PSU.has(d.ProductType) && Object.values(d.GetDevice?.devices || {}).some((c) => c.HasSpeed));
  const want = new URLSearchParams(location.search).get("device");
  S.serial = (S.devices.find((d) => d.Serial === want) || S.devices[0])?.Serial || null;
  const pick = $("#devicePick");
  pick.hidden = S.devices.length < 2;
  pick.textContent = "";
  for (const d of S.devices) pick.append(el("option", { value: d.Serial, text: d.Product, selected: d.Serial === S.serial }));
}

async function pollFast(record) {
  const clean = (r) => (r && r.status === 1 && typeof r.data === "number" && r.data > 0 ? r.data : null);
  let failed = 0;
  const soft = (path) => api(path).catch((e) => { if (e instanceof AuthError) throw e; failed++; return null; });
  const [dev, cpu, gpu, cpuLoad, gpuLoad, storage] = await Promise.all([
    S.serial ? api(`/api/devices/${S.serial}`) : null,
    soft("/api/cpuTemp/clean"),
    soft("/api/gpuTemp/clean"),
    soft("/api/cpuLoad"),
    soft("/api/gpuLoad"),
    soft("/api/storageTemp"),
  ]);
  if (failed === 5) throw new Error("OpenLinkHub is not responding");
  if ((S.sources?.gpus?.length || 0) > 1) {
    const g = await soft("/api/gpuTemps");
    S.sensors.gpus = g?.status === 1 ? Object.fromEntries(Object.entries(g.data || {}).map(([k, v]) => [k, parseFloat(v)]).filter(([, v]) => v > 0)) : {};
  }
  S.dev = dev ? dev.device?.GetDevice || dev.device || null : null;
  S.sensors.cpu = clean(cpu);
  S.sensors.gpu = clean(gpu);
  S.sensors.cpuLoad = cpuLoad?.status === 1 ? cpuLoad.data : null;
  S.sensors.gpuLoad = gpuLoad?.status === 1 ? gpuLoad.data : null;
  const drives = Array.isArray(storage?.data) ? storage.data.filter((s) => s.Temperature > 0) : [];
  const hot = drives.sort((a, b) => b.Temperature - a.Temperature)[0];
  S.sensors.drive = hot?.Temperature ?? null;
  S.sensors.driveName = hot ? `${hot.Model}${drives.length > 1 ? ` (hottest of ${drives.length})` : ""}` : "";

  if (!record) return;
  const fans = fanChannels().filter((c) => c.rpm > 0);
  S.hist.push({
    cpu: S.sensors.cpu, gpu: S.sensors.gpu, liquid: liquidTemp(), drive: S.sensors.drive,
    pump: pumpChannel()?.rpm || null,
    fans: fans.length ? fans.reduce((s, c) => s + c.rpm, 0) / fans.length : null,
  });
  if (S.hist.length > HISTORY_LEN) S.hist.shift();
}
async function pollProfiles() {
  const r = await api("/api/temperatures/");
  S.profiles = Object.fromEntries(Object.entries(r.data || {}).filter(([, v]) => !v.Hidden));
}
async function pollRgb() {
  S.cluster = null;
  if (S.dev?.DeviceProfile?.RGBCluster) { try { const c = await api("/api/devices/cluster"); S.cluster = c.device?.serial === "cluster" ? c.device : null; } catch { /* no cluster */ } }
  const r = await api(`/api/color/${lightId()}`);
  S.rgb = r.data || r;
}
async function pollDash() { const r = await api("/api/dashboard"); S.dash = r.dashboard || null; }

/* Sensors a new profile can follow, and whether profiles are curves or tables (GET /api/temperatureSources). */
async function loadSources() {
  try {
    const d = (await api("/api/temperatureSources")).data || {};
    const probes = (d.probes || []).flat().filter(Boolean).map((p) => ({ value: `${p.Serial};${p.ChannelId}`, text: `${p.Product} - ${p.Name}${p.Label && p.Label !== "Set Label" ? ` (${p.Label})` : ""}`, serial: p.Serial, ch: p.ChannelId }));
    S.sources = {
      probes: probes.filter((o) => o.serial === S.serial && o.ch >= 1), // sensor 4: this hub, channel ≥1 (temperatures.go)
      anyProbes: probes,                                               // sensor 9: any device
      hwmon: (d.hwmon || []).map((h) => ({ value: `${h.HwmonName};${h.InputName};${h.SensorName}`, text: `${h.SensorName} - ${h.Label} - ${fmt(h.TempC, 1)} °C` })),
      drives: (d.storage || []).map((x) => ({ value: x.Key, text: x.Model })),
      gpus: Object.values(d.gpus || {}).map((g) => ({ value: g.Index, text: `${g.Model} - ${g.TemperatureString}` })),
    };
    S.graphMode = d.graphProfiles !== false;
  } catch { S.sources = { probes: [], anyProbes: [], hwmon: [], drives: [], gpus: [] }; }
}
async function loadSchedule() {
  try {
    const d = (await api("/api/scheduler")).data || {};
    return { on: !!d.rgbControl, off: d.rgbOff || "", at: d.rgbOn || "", lcd: !!d.lcdControl };
  } catch { return null; }
}

function setLive(ok, text) {
  const l = $("#live");
  l.classList.toggle("ok", ok);
  l.classList.toggle("bad", !ok);
  $("#liveText").textContent = text;
}

/* ---------- Tiles ---------- */
const TILES = [
  { key: "cpu", label: "CPU", c: "--cpu", temp: true, d: 0, warm: 75, hot: 90, sub: () => S.sensors.cpuLoad != null ? `${fmt(S.sensors.cpuLoad)}% load` : "" },
  { key: "gpu", label: "GPU", c: "--gpu", temp: true, d: 0, warm: 75, hot: 85, sub: () => S.sensors.gpuLoad != null ? `${fmt(S.sensors.gpuLoad)}% load` : "" },
  { key: "liquid", label: "Coolant", c: "--liquid", temp: true, d: 1, warm: 42, hot: 50, when: () => !!pumpChannel(), sub: () => pumpChannel()?.name || "" },
  { key: "pump", label: "Pump", c: "--pump", unit: "rpm", d: 0, when: () => !!pumpChannel(), sub: () => pumpChannel()?.profile ? `${pumpChannel().profile} profile` : "" },
  { key: "fans", label: "Fans", c: "--fan", unit: "rpm", d: 0, when: () => fanChannels().length > 0, sub: () => {
      const r = fanChannels().filter((c) => c.rpm > 0).map((c) => c.rpm);
      return r.length ? `${r.length} spinning · ${Math.min(...r)}–${Math.max(...r)}` : "none spinning";
    } },
  { key: "drive", label: "Drive", c: "--accent", temp: true, d: 0, warm: 60, hot: 70, sub: () => S.sensors.driveName },
];
let tilesKey = "";
function renderTiles() {
  const live = TILES.filter((t) => !t.when || t.when());
  const key = live.map((t) => t.key).join();
  const root = $("#tiles");
  if (key !== tilesKey) {
    tilesKey = key;
    root.textContent = "";
    for (const t of live) {
      root.append(el("div", { class: "tile", style: `--c: var(${t.c})`, id: `tile-${t.key}` },
        el("div", { class: "tile-label" }, el("span", { text: t.label }), el("span", { class: "badge", hidden: true })),
        el("div", { class: "tile-value" }, el("span", { class: "v", text: "—" }), el("span", { class: "unit", text: t.temp ? uT() : t.unit })),
        el("div", { class: "tile-sub" }),
        sv("svg", { class: "spark", viewBox: "0 0 100 34", preserveAspectRatio: "none", "aria-hidden": "true" })));
    }
  }
  const last = S.hist[S.hist.length - 1] || {};
  for (const t of live) {
    const tile = $(`#tile-${t.key}`);
    const v = last[t.key];
    $(".v", tile).textContent = fmt(t.temp ? tU(v) : v, t.d);
    $(".unit", tile).textContent = t.temp ? uT() : t.unit;
    $(".tile-sub", tile).textContent = t.sub() || "";
    const b = $(".badge", tile);
    const level = t.hot && v != null ? (v >= t.hot ? "hot" : v >= t.warm ? "warm" : "") : "";
    b.hidden = !level;
    b.className = `badge ${level}`;
    b.textContent = level;
    drawSpark($(".spark", tile), S.hist.map((h) => h[t.key]), cssVar(t.c));
  }
  // Text only changes when the set of hot readings changes, so screen readers aren't re-notified every poll.
  const hot = Object.entries(ALERTS).filter(([k, lim]) => last[k] != null && last[k] >= lim).map(([k]) => TILES.find((t) => t.key === k).label);
  const a = $("#alert");
  const msg = hot.length ? `Running hot: ${hot.join(", ")} (see the readings below). ${hot.includes("Coolant") ? "Check the pump is spinning and the fans are ramping. " : ""}This warning only shows while this page is open.` : "";
  if (!failures && a.dataset.msg !== msg) { a.dataset.msg = msg; a.hidden = !msg; a.textContent = msg; }
}
function drawSpark(svg, vals, color) {
  svg.textContent = "";
  const pts = vals.map((v, i) => [i, v]).filter(([, v]) => v != null);
  if (pts.length < 2) return;
  const ys = pts.map(([, v]) => v);
  let lo = Math.min(...ys), hi = Math.max(...ys);
  if (hi - lo < 2) { const m = (hi + lo) / 2; lo = m - 1; hi = m + 1; }
  const n = Math.max(vals.length - 1, 1);
  const X = (i) => (i / n) * 100;
  const Y = (v) => 31 - ((v - lo) / (hi - lo)) * 28;
  const d = pts.map(([i, v], k) => `${k ? "L" : "M"}${X(i).toFixed(2)},${Y(v).toFixed(2)}`).join("");
  svg.append(sv("path", { d: `${d}L${X(pts[pts.length - 1][0]).toFixed(2)},34L${X(pts[0][0]).toFixed(2)},34Z`, fill: color, opacity: ".12" }));
  svg.append(sv("path", { d, fill: "none", stroke: color, "stroke-width": "1.6", "vector-effect": "non-scaling-stroke", "stroke-linejoin": "round" }));
}

/* ---------- History chart ---------- */
const SERIES = [{ key: "cpu", label: "CPU", c: "--cpu" }, { key: "gpu", label: "GPU", c: "--gpu" }, { key: "liquid", label: "Coolant", c: "--liquid" }];
function renderLegend() {
  const lg = $("#historyLegend");
  lg.textContent = "";
  for (const s of SERIES) {
    if (s.key === "liquid" && !pumpChannel()) continue;
    const b = el("button", { class: `legend-btn${S.hiddenSeries.has(s.key) ? " off" : ""}`, "aria-pressed": !S.hiddenSeries.has(s.key), title: `Show or hide ${s.label}`, "data-fk": `legend-${s.key}` }, el("i", { style: `background: var(${s.c})` }), s.label);
    b.addEventListener("click", () => { S.hiddenSeries.has(s.key) ? S.hiddenSeries.delete(s.key) : S.hiddenSeries.add(s.key); keepFocus(renderLegend); renderHistory(); });
    lg.append(b);
  }
}
function renderHistory() {
  const svg = $("#historyChart");
  const W = svg.clientWidth || 800, H = svg.clientHeight || 200;
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  svg.textContent = "";
  const padL = 34, padR = 8, padT = 8, padB = 20;
  const shown = SERIES.filter((s) => !S.hiddenSeries.has(s.key));
  const vals = S.hist.flatMap((h) => shown.map((s) => tU(h[s.key]))).filter((v) => v != null);
  if (!vals.length) return;
  let lo = Math.floor((Math.min(...vals) - 3) / 10) * 10, hi = Math.ceil((Math.max(...vals) + 3) / 10) * 10;
  if (hi - lo < 20) hi = lo + 20;
  const X = (i) => padL + ((i + HISTORY_LEN - S.hist.length) / (HISTORY_LEN - 1)) * (W - padL - padR);
  const Y = (v) => padT + (1 - (tU(v) - lo) / (hi - lo)) * (H - padT - padB);
  const YL = (v) => padT + (1 - (v - lo) / (hi - lo)) * (H - padT - padB);
  const g = sv("g", { class: "grid" });
  const step = hi - lo > 50 ? 20 : 10;
  for (let v = lo; v <= hi; v += step) {
    g.append(sv("line", { x1: padL, x2: W - padR, y1: YL(v), y2: YL(v) }));
    g.append(sv("text", { x: padL - 6, y: YL(v) + 4, "text-anchor": "end" }, `${v}°`));
  }
  [["−10m", 0], ["−5m", 0.5], ["now", 1]].forEach(([lab, f]) =>
    g.append(sv("text", { x: padL + f * (W - padL - padR), y: H - 4, "text-anchor": f === 0 ? "start" : f === 1 ? "end" : "middle" }, lab)));
  svg.append(g);
  for (const s of shown) {
    const color = cssVar(s.c);
    const pts = S.hist.map((h, i) => [i, h[s.key]]).filter(([, v]) => v != null);
    if (pts.length < 2) continue;
    svg.append(sv("path", { class: "series", d: pts.map(([i, v], k) => `${k ? "L" : "M"}${X(i).toFixed(1)},${Y(v).toFixed(1)}`).join(""), stroke: color }));
    const [li, lv] = pts[pts.length - 1];
    svg.append(sv("circle", { cx: X(li), cy: Y(lv), r: 3, fill: color }));
  }
}

/* ---------- Channels ---------- */
function fanIcon() {
  const s = sv("svg", { class: "fan-icon", viewBox: "0 0 24 24", "aria-hidden": "true" });
  s.append(sv("circle", { cx: 12, cy: 12, r: 11, fill: "none", stroke: "currentColor", "stroke-width": "1.2", opacity: ".35" }));
  const g = sv("g", { fill: "currentColor" });
  g.append(sv("path", { d: "M12 12c0-4.5 2-7 5.5-7-1 3.5-2.4 5.8-5.5 7zm0 0c4.5 0 7 2 7 5.5-3.5-1-5.8-2.4-7-5.5zm0 0c0 4.5-2 7-5.5 7 1-3.5 2.4-5.8 5.5-7zm0 0c-4.5 0-7-2-7-5.5 3.5 1 5.8 2.4 7 5.5z" }));
  g.append(sv("circle", { cx: 12, cy: 12, r: 1.8 }));
  s.append(g);
  return s;
}
function profileOptions(select, current) {
  select.textContent = "";
  for (const [name, p] of Object.entries(S.profiles)) {
    if (Number(p.sensor) === 2 && !pumpChannel()) continue;
    select.append(el("option", { value: name, text: `${name} · ${isFlat(p) ? "fixed" : SENSORS[p.sensor] ?? "?"}`, selected: name === current }));
  }
  if (current && !S.profiles[current]) select.append(el("option", { value: current, text: current, selected: true }));
}
const channelsKey = () => channels().map((c) => `${c.channelId}:${c.label}:${c.HasSpeed}`).join("|");

function buildChannels() {
  const root = $("#channels");
  root.textContent = "";
  for (const c of channels()) {
    const pump = isPump(c);
    const card = el("div", { class: `ch${pump ? " pump" : ""}${c.HasSpeed ? "" : " probe"}`, id: `ch-${c.channelId}`, style: `--c: var(${pump ? "--pump" : c.HasSpeed ? "--fan" : "--liquid"})` });
    const name = el("div", { class: "ch-name", tabindex: "0", role: "button", title: "Rename", text: displayName(c) });
    name.addEventListener("click", () => startRename(name, c));
    name.addEventListener("keydown", (e) => { if (e.key === "Enter" && !name.isContentEditable) { e.preventDefault(); startRename(name, c); } });
    const pick = c.HasSpeed ? el("input", { type: "checkbox", class: "pick", checked: S.sel.has(c.channelId), "aria-label": `Select ${displayName(c)}`, title: "Select for “Apply”" }) : null;
    pick?.addEventListener("change", () => { pick.checked ? S.sel.add(c.channelId) : S.sel.delete(c.channelId); card.classList.toggle("picked", pick.checked); renderBulkTargets(); });
    if (S.sel.has(c.channelId)) card.classList.add("picked");
    card.append(el("div", { class: "ch-top" }, c.HasSpeed ? fanIcon() : el("span", { class: "probe-icon", text: "🌡" }),
      el("div", { class: "grow" }, name, el("div", { class: "ch-kind", text: pump ? c.name : c.HasSpeed ? `Port ${c.channelId}` : "Temperature probe" })), pick));
    if (c.HasSpeed) {
      const sel = el("select", { "aria-label": `${displayName(c)} speed profile` });
      sel.addEventListener("change", async () => {
        if (await act("/api/speed", "POST", { deviceId: S.serial, channelId: c.channelId, profile: sel.value }, `${displayName(c)} → ${sel.value}`)) tick();
      });
      card.append(
        el("div", { class: "ch-row spread" },
          el("div", { class: "ch-rpm" }, el("span", { class: "rpm", text: "—" }), el("span", { class: "unit", text: "rpm" })),
          el("div", { class: "ch-extra" })),
        el("div", { class: "bar" }, el("span")),
        sel);
    } else {
      card.append(el("div", { class: "ch-rpm" }, el("span", { class: "rpm", text: "—" }), el("span", { class: "unit tunit", text: uT() })));
    }
    root.append(card);
  }
  S.built.channels = channelsKey();
  S.built.profiles = "";
}
function renderChannels() {
  if (S.built.channels !== channelsKey()) buildChannels();
  const pk = Object.keys(S.profiles).join("|");
  const rebuild = S.built.profiles !== pk;
  for (const c of channels()) {
    const card = $(`#ch-${c.channelId}`);
    if (!card) continue;
    if (!c.HasSpeed) { $(".rpm", card).textContent = fmt(tU(c.temperature), 1); $(".tunit", card).textContent = uT(); continue; }
    const pump = isPump(c);
    $(".rpm", card).textContent = c.rpm > 0 ? c.rpm : "—";
    S.rpmMax[c.channelId] = Math.max(S.rpmMax[c.channelId] || (pump ? 3000 : 2000), c.rpm || 0);
    $(".bar > span", card).style.width = `${clamp((c.rpm / S.rpmMax[c.channelId]) * 100, 0, 100)}%`;
    const p = S.profiles[c.profile];
    const t = sensorTemp(p);
    const target = effective(p, pump ? 0 : 1, profileSpeed(p, pump ? 0 : 1, t));
    $(".ch-extra", card).textContent = [pump && c.temperature > 0 ? fmtT(c.temperature, 1) : null, target != null ? `target ${target}%` : null].filter(Boolean).join(" · ");
    const icon = $(".fan-icon g", card);
    const dur = c.rpm > 0 ? clamp(2400 / c.rpm, 0.3, 4) : 0;
    if (Math.abs((parseFloat(icon.dataset.dur) || 0) - dur) > 0.08) { icon.dataset.dur = dur; icon.style.setProperty("--dur", `${dur}s`); }
    const sel = $("select", card);
    if ((rebuild || sel.value !== c.profile) && document.activeElement !== sel) profileOptions(sel, c.profile);
  }
  if (rebuild) {
    const fans = fanChannels();
    profileOptions($("#bulkProfile"), fans.length && fans.every((c) => c.profile === fans[0].profile) ? fans[0].profile : "Normal");
    S.built.profiles = pk;
    renderBulkTargets();
  }
  $("#channelsNote").textContent = `Speeds follow each channel's profile. ${pumpChannel() && !/LINK/i.test(S.dev?.product || "") ? "OpenLinkHub has no automatic coolant failsafe for this hub, so give the pump and fans a profile that reacts to load. " : ""}Click a name to rename it; tick cards to apply a profile to just those.`;
}
/* Targets shared by the speed and lighting bulk controls. */
function renderBulkTargets() {
  const n = S.sel.size;
  const fill = (sel, opts) => {
    const cur = sel.value;
    sel.textContent = "";
    for (const [v, t, dis] of opts) sel.append(el("option", { value: v, text: t, disabled: !!dis }));
    const valid = (v) => [...sel.options].some((o) => o.value === v && !o.disabled);
    sel.value = n && !renderBulkTargets.prev ? "sel" : valid(cur) ? cur : sel.options[0].value;
  };
  fill($("#bulkTarget"), [["fans", "All fans"], ...(pumpChannel() ? [["all", "Pump and fans"]] : []), ["sel", `Selected (${n})`, !n]]);
  const nl = [...S.sel].filter((id) => S.dev?.rgbDevices?.[id]).length;
  fill($("#rgbTarget"), inCluster() ? [["cluster", "Whole cluster"]] : [["device", "Every LED here"], ...(Object.keys(S.dev?.rgbDevices || {}).length ? [["sel", `Selected (${nl})`, !nl]] : []), ["global", "Every device"]]);
  renderBulkTargets.prev = n;
}

function startRename(node, c, deviceType = 0) {
  if (node.isContentEditable) return;
  const before = node.textContent;
  node.contentEditable = "true";
  node.focus();
  document.getSelection().selectAllChildren(node);
  const finish = async (save) => {
    node.contentEditable = "false";
    node.removeEventListener("keydown", onKey);
    node.removeEventListener("blur", onBlur);
    const label = node.textContent.trim();
    if (!save || !label || label === before) { node.textContent = before; return; }
    if (!/^[a-zA-Z0-9#.:_ -]+$/.test(label)) { toast("Names can use letters, numbers, spaces and # . : _ -", "warn"); node.textContent = before; return; }
    const ok = await act("/api/label", "POST", { deviceId: S.serial, channelId: c.channelId, deviceType, label }, `Renamed to “${label}”`);
    const led = S.dev?.rgbDevices?.[c.channelId];
    if (ok && deviceType === 0 && led && (!realLabel(led.label) || led.label === realLabel(c.label))) await act("/api/label", "POST", { deviceId: S.serial, channelId: c.channelId, deviceType: 1, label }, false);
    if (!ok) node.textContent = before;
    tick();
  };
  const onKey = (e) => { if (e.key === "Enter") { e.preventDefault(); finish(true); } if (e.key === "Escape") finish(false); };
  const onBlur = () => finish(true);
  node.addEventListener("keydown", onKey);
  node.addEventListener("blur", onBlur);
}

$("#bulkApply").addEventListener("click", async () => {
  const profile = $("#bulkProfile").value;
  const target = $("#bulkTarget").value;
  const ids = target === "sel" ? [...S.sel] : target === "all" ? speedChannels().map((c) => c.channelId) : fanChannels().map((c) => c.channelId);
  if (!ids.length) return;
  const what = target === "sel" ? `${ids.length} selected` : target === "all" ? "Pump and fans" : `${ids.length} fans`;
  if (await act("/api/speed", "POST", { deviceId: S.serial, channelId: -1, channelIds: ids, profile }, `${what} → ${profile}`)) tick();
});

/* ---------- Curve editor ---------- */
const CV = { W: 640, H: 320, L: 44, R: 16, T: 16, B: 34 };
const xmax = () => axisMax(S.profiles[S.curve.name]);
const cx = (t) => CV.L + (clamp(t, 0, xmax()) / xmax()) * (CV.W - CV.L - CV.R);
const cy = (p) => CV.T + (1 - clamp(p, 0, 100) / 100) * (CV.H - CV.T - CV.B);
const toT = (x) => ((x - CV.L) / (CV.W - CV.L - CV.R)) * xmax();
const toP = (y) => (1 - (y - CV.T) / (CV.H - CV.T - CV.B)) * 100;
const minSpeed = () => (S.curve.type === 0 ? 50 : (S.profiles[S.curve.name]?.zeroRpm ? 0 : 20));
const readOnly = () => BUILTIN.has(S.curve.name);

function selectProfile(name) {
  if (S.curve.dirty && name !== S.curve.name) toast(`Unsaved changes to ${S.curve.name} discarded`, "warn");
  S.curve.name = name;
  loadCurve();
}
function loadCurve() {
  const p = S.profiles[S.curve.name];
  if (!S.graphMode) { loadBands(p); return; }
  const pts = p?.points?.[S.curve.type] || [];
  S.curve.saved = pts.map((q) => ({ x: q.x, y: q.y })).sort((a, b) => a.x - b.x);
  S.curve.points = S.curve.saved.map((q) => ({ ...q }));
  S.curve.dirty = false;
  renderProfiles();
  renderCurve();
}
function renderProfiles() {
  const ul = $("#profileList");
  const used = {};
  for (const c of speedChannels()) used[c.profile] = (used[c.profile] || 0) + 1;
  const key = JSON.stringify([Object.keys(S.profiles), S.curve.name, Object.values(S.profiles).map(isFlat)]);
  if (ul.dataset.key !== key) {
    ul.dataset.key = key;
    ul.textContent = "";
    for (const [name, p] of Object.entries(S.profiles)) {
      ul.append(el("li", {}, el("button", { class: name === S.curve.name ? "on" : "", "data-name": name, "data-fk": `prof-${name}`, onclick: () => keepFocus(() => selectProfile(name)), "aria-pressed": name === S.curve.name },
        el("span", { class: "pname" }, el("span", { text: name }), el("span", { class: "badge", text: isFlat(p) ? "fixed" : SENSORS[p.sensor] ?? "?" })),
        el("span", { class: "pmeta" }))));
    }
  }
  for (const btn of ul.querySelectorAll("button[data-name]")) {
    const name = btn.dataset.name, p = S.profiles[name], n = used[name] || 0, t = sensorTemp(p);
    btn.querySelector(".pmeta").textContent = [n ? `${n} channel${n > 1 ? "s" : ""}` : "unused", t != null ? `${fmtT(t)} now` : null, BUILTIN.has(name) ? "built-in" : null, p.zeroRpm ? "zero-rpm" : null].filter(Boolean).join(" · ");
  }
  const p = S.profiles[S.curve.name];
  const ro = readOnly();
  const what = S.curve.type === 0 ? "Pump" : "Fan";
  $("#editorTitle").textContent = S.curve.name || "—";
  $("#editorMeta").textContent = p ? `${what} speed vs ${SENSORS[p.sensor] ?? "sensor"} temperature${ro ? " · built-in, read-only" : ""}` : "";
  $("#curveDelete").hidden = !p || ro;
  $("#curveDuplicate").hidden = !p;
  $("#curveDuplicate").textContent = ro ? "Duplicate to edit" : "Duplicate";
  $("#curveSave").hidden = ro;
  $("#curveRevert").hidden = ro;
  $("#curveSave").disabled = !S.curve.dirty;
  $("#curveRevert").disabled = !S.curve.dirty;
  const flat = p && isFlat(p) && !ro && S.graphMode;
  $("#fixedBox").hidden = !flat;
  if (flat) {
    const v = S.curve.points?.[0]?.y ?? 50;
    const r = $("#fixedSpeed");
    r.min = minSpeed();
    r.value = v;
    $("#fixedLabel").textContent = `${what} speed`;
    $("#fixedVal").textContent = `${Math.round(v)}%`;
  }
  $("#curveHint").textContent = !S.graphMode
    ? `Table mode (graphProfiles is off in config.json): each row sets the speed for a temperature range.${ro ? " Built-in profiles are rebuilt at every start, so duplicate one to change it." : ""}`
    : ro
    ? "Built-in profiles are rebuilt each time OpenLinkHub starts, so edits to them don't last. Duplicate one to make it your own."
    : flat
      ? "Fixed-speed profile: the same speed at every temperature."
      : `Drag points to shape the curve. Double-click empty space to add a point, double-click a point to remove it.${S.curve.type === 0 ? " The pump never runs below 50%." : p?.zeroRpm ? "" : " Fans never run below 20%."}`;
}
function renderCurve() {
  if (!S.graphMode) return;
  const svg = $("#curve");
  const focused = document.activeElement?.closest?.("#curve .pt")?.dataset.i;
  svg.textContent = "";
  svg.classList.toggle("ro", readOnly());
  svg.style.setProperty("--c", cssVar(S.curve.type === 0 ? "--pump" : "--fan"));
  const X = xmax(), step = X > 60 ? 10 : 5;
  for (let t = 0; t <= X; t += step) {
    svg.append(sv("line", { class: "gl", x1: cx(t), x2: cx(t), y1: CV.T, y2: CV.H - CV.B }));
    svg.append(sv("text", { x: cx(t), y: CV.H - CV.B + 18, "text-anchor": "middle" }, `${fmt(tU(t))}°`));
  }
  for (let p = 0; p <= 100; p += 20) {
    svg.append(sv("line", { class: "gl", x1: CV.L, x2: CV.W - CV.R, y1: cy(p), y2: cy(p) }));
    svg.append(sv("text", { x: CV.L - 8, y: cy(p) + 4, "text-anchor": "end" }, `${p}%`));
  }
  const floor = minSpeed();
  if (floor > 0) svg.append(sv("rect", { class: "floor", x: CV.L, y: cy(floor), width: CV.W - CV.L - CV.R, height: cy(0) - cy(floor) }));
  const pts = S.curve.points || [];
  if (!pts.length) { svg.append(sv("text", { x: CV.W / 2, y: CV.H / 2, "text-anchor": "middle" }, "This profile has no curve")); return; }
  const path = (arr) => {
    if (!arr?.length) return "";
    const vis = arr.filter((q) => q.x < X).map((q) => [cx(q.x), cy(q.y)]);
    return [[CV.L, cy(arr[0].y)], ...vis, [CV.W - CV.R, cy(interp(arr, X))]].map(([x, y], i) => `${i ? "L" : "M"}${x.toFixed(1)},${y.toFixed(1)}`).join("");
  };
  if (S.curve.dirty && S.curve.saved.length) svg.append(sv("path", { class: "ghost", d: path(S.curve.saved) }));
  const d = path(pts);
  svg.append(sv("path", { class: "fill", d: `${d}L${CV.W - CV.R},${CV.H - CV.B}L${CV.L},${CV.H - CV.B}Z` }));
  svg.append(sv("path", { class: "line", d }));
  const p = S.profiles[S.curve.name];
  svg.append(sv("g", { class: "now" }));
  updateNow();
  if (readOnly() || isFlat(p)) return;
  pts.forEach((q, i) => { if (q.x <= X) svg.append(sv("circle", { class: `pt${S.curve.drag === i ? " drag" : ""}`, cx: cx(q.x), cy: cy(q.y), r: 7, "data-i": i, tabindex: "0", role: "slider", "aria-valuenow": q.y, "aria-valuetext": `${fmt(tU(q.x))} degrees, ${q.y} percent`, "aria-label": `Curve point ${i + 1}` })); });
  if (focused != null) $(`#curve .pt[data-i="${focused}"]`)?.focus();
  if (S.curve.drag >= 0) {
    const q = pts[S.curve.drag];
    const bx = clamp(cx(q.x) - 44, CV.L, CV.W - CV.R - 88), by = clamp(cy(q.y) - 36, CV.T, CV.H - CV.B - 24);
    const g = sv("g", { class: "tip" });
    g.append(sv("rect", { x: bx, y: by, width: 88, height: 24, rx: 6 }));
    g.append(sv("text", { x: bx + 44, y: by + 16, "text-anchor": "middle" }, `${fmt(tU(q.x))}° → ${q.y}%`));
    svg.append(g);
  }
}
/* Live marker on the curve: where the sensor is now and the speed the driver will use. */
function updateNow() {
  if (!S.graphMode) return;
  const g = $("#curve .now");
  if (!g) return;
  g.textContent = "";
  const p = S.profiles[S.curve.name], pts = S.curve.points || [];
  const t = sensorTemp(p);
  if (t == null || !pts.length) return;
  const raw = interp(pts, t);
  const eff = effective(p, S.curve.type, raw);
  g.append(sv("line", { x1: cx(t), x2: cx(t), y1: CV.T, y2: CV.H - CV.B }));
  g.append(sv("circle", { cx: cx(t), cy: cy(raw), r: 4 }));
  g.append(sv("text", { x: clamp(cx(t) + 8, CV.L + 4, CV.W - CV.R - 150), y: CV.T + 14 }, `now ${fmt(tU(t))}° → ${eff}%${eff !== Math.round(raw) ? " (floor)" : ""}`));
}
/* Table mode (config graphProfiles=false): min/max bands with a fan % and pump % each. */
function loadBands(p) {
  S.curve.bands = (p?.profiles || []).map((b) => ({ ...b }));
  S.curve.saved = JSON.stringify(S.curve.bands);
  S.curve.dirty = false;
  $("#curve").hidden = true;
  $(".seg").hidden = true;
  renderProfiles();
  const ro = readOnly();
  const box = $("#bandTable");
  box.hidden = false;
  box.textContent = "";
  const t = sensorTemp(p);
  const table = el("table", {}, el("thead", {}, el("tr", {}, el("th", { text: `Temperature (${uT()})` }), el("th", { text: "Fans %" }), ...(pumpChannel() ? [el("th", { text: "Pump %" })] : []))));
  const tb = el("tbody");
  S.curve.bands.forEach((b) => {
    const num = (key, min) => {
      const i = el("input", { type: "number", min, max: 100, step: 1, value: b[key], disabled: ro, "aria-label": `${key} at ${b.min}–${b.max}` });
      i.addEventListener("input", () => { b[key] = clamp(Math.round(Number(i.value) || 0), min, 100); S.curve.dirty = JSON.stringify(S.curve.bands) !== S.curve.saved; $("#curveSave").disabled = $("#curveRevert").disabled = !S.curve.dirty; });
      return el("td", {}, i);
    };
    const now = t != null && t >= b.min && t <= b.max;
    tb.append(el("tr", { class: now ? "now" : "" }, el("td", { text: `${fmt(tU(b.min))}–${fmt(tU(b.max))}${now ? "  ← now" : ""}` }), num("fans", p?.zeroRpm ? 0 : 20), ...(pumpChannel() ? [num("pump", 50)] : [])));
  });
  table.append(tb);
  box.append(table);
}
function svgPoint(evt) {
  const r = $("#curve").getBoundingClientRect();
  return [((evt.clientX - r.left) / r.width) * CV.W, ((evt.clientY - r.top) / r.height) * CV.H];
}
function markDirty() {
  S.curve.dirty = JSON.stringify(S.curve.points) !== JSON.stringify(S.curve.saved);
  $("#curveSave").disabled = !S.curve.dirty;
  $("#curveRevert").disabled = !S.curve.dirty;
}
function movePoint(i, t, p) {
  const pts = S.curve.points;
  const lo = i > 0 ? pts[i - 1].x + 1 : 0;
  const hi = i < pts.length - 1 ? Math.min(pts[i + 1].x - 1, xmax()) : xmax();
  pts[i] = { x: Math.round(clamp(t, lo, hi)), y: Math.round(clamp(p, minSpeed(), 100)) };
  markDirty();
  renderCurve();
}
(function wireCurve() {
  const svg = $("#curve");
  svg.addEventListener("pointerdown", (e) => {
    const t = e.target.closest(".pt");
    if (!t) return;
    S.curve.drag = Number(t.dataset.i);
    S.curve.moved = false;
    t.classList.add("drag");
  });
  svg.addEventListener("pointermove", (e) => {
    if (S.curve.drag < 0) return;
    if (!S.curve.moved) { S.curve.moved = true; try { svg.setPointerCapture(e.pointerId); } catch { /* synthetic events */ } }
    const [x, y] = svgPoint(e);
    movePoint(S.curve.drag, toT(x), toP(y));
  });
  const end = () => { if (S.curve.drag >= 0) { const moved = S.curve.moved; S.curve.drag = -1; if (moved) renderCurve(); else $$("#curve .pt.drag").forEach((n) => n.classList.remove("drag")); } };
  svg.addEventListener("pointerup", end);
  svg.addEventListener("pointercancel", end);
  svg.addEventListener("keydown", (e) => {
    const t = e.target.closest(".pt");
    if (!t) return;
    const i = Number(t.dataset.i), q = S.curve.points[i];
    const k = { ArrowUp: [0, 1], ArrowDown: [0, -1], ArrowLeft: [-1, 0], ArrowRight: [1, 0] }[e.key];
    if (!k) return;
    e.preventDefault();
    movePoint(i, q.x + k[0], q.y + k[1]);
    $(`#curve .pt[data-i="${i}"]`)?.focus();
  });
  svg.addEventListener("dblclick", (e) => {
    const pts = S.curve.points;
    if (!pts || readOnly() || isFlat(S.profiles[S.curve.name])) return;
    const [x, y] = svgPoint(e);
    let hit = -1, best = 12; // svg units
    pts.forEach((q, i) => { const dd = Math.hypot(cx(q.x) - x, cy(q.y) - y); if (q.x <= xmax() && dd < best) { best = dd; hit = i; } });
    if (hit >= 0) {
      if (pts.filter((q) => q.x <= xmax()).length <= 2) { toast("A curve needs at least two points", "warn"); return; }
      pts.splice(hit, 1);
    } else {
      const t = Math.round(clamp(toT(x), 0, xmax()));
      if (pts.some((q) => q.x === t)) return;
      pts.push({ x: t, y: Math.round(clamp(toP(y), minSpeed(), 100)) });
      pts.sort((a, b) => a.x - b.x);
    }
    markDirty();
    renderCurve();
  });
})();
$("#fixedSpeed").addEventListener("input", () => {
  const v = Number($("#fixedSpeed").value);
  $("#fixedVal").textContent = `${v}%`;
  S.curve.points = S.curve.points.map((q) => ({ x: q.x, y: v }));
  markDirty();
  renderCurve();
});
$$(".seg-btn").forEach((b) => b.addEventListener("click", () => {
  if (S.curve.dirty) toast(`Unsaved ${S.curve.type === 0 ? "pump" : "fan"} changes discarded`, "warn");
  $$(".seg-btn").forEach((x) => { x.classList.toggle("on", x === b); x.setAttribute("aria-selected", x === b); });
  S.curve.type = Number(b.dataset.curve);
  loadCurve();
}));
$("#curveRevert").addEventListener("click", loadCurve);
$("#curveSave").addEventListener("click", async () => {
  if (!S.graphMode) {
    const data = JSON.stringify(Object.fromEntries(S.curve.bands.map((b) => [b.id, { fans: b.fans, pump: b.pump }])));
    if (await act("/api/temperatures/update", "PUT", null, `${S.curve.name} saved`, { form: { profile: S.curve.name, data } })) { await pollProfiles(); loadCurve(); }
    return;
  }
  const body = { profile: S.curve.name, updateType: S.curve.type, points: S.curve.points.map((q) => ({ x: q.x, y: q.y })) };
  if (await act("/api/temperatures/updateGraph", "PUT", body, `${S.curve.name}: ${S.curve.type === 0 ? "pump" : "fan"} curve saved`)) { await pollProfiles(); loadCurve(); }
});
$("#curveDelete").addEventListener("click", async () => {
  const btn = $("#curveDelete");
  if (!armed(btn, "Click again to delete")) return;
  const name = S.curve.name;
  if (await act("/api/temperatures/delete", "DELETE", { profile: name }, `Deleted ${name}; channels using it went back to Normal`)) {
    await pollProfiles();
    S.curve.name = "Normal";
    loadCurve();
    tick();
  }
});
let copyFrom = null;
$("#curveDuplicate").addEventListener("click", () => {
  const p = S.profiles[S.curve.name];
  copyFrom = S.curve.name;
  const box = $("#newProfileBox");
  box.open = true;
  let name = `My${S.curve.name}`.replace(/[^A-Za-z0-9]/g, "");
  for (let i = 2; S.profiles[name]; i++) name = `My${S.curve.name}${i}`.replace(/[^A-Za-z0-9]/g, "");
  $("#newName").value = name;
  $("#newSensor").value = String(p.sensor);
  if ($("#newSensor").value !== String(p.sensor)) {
    box.open = false;
    copyFrom = null;
    toast(`${SENSORS[p.sensor] ?? "This"} profiles can't be created here, so ${S.curve.name} can't be duplicated`, "warn");
    return;
  }
  syncNewForm();
  const pick = (sel, v) => { if ([...sel.options].some((o) => o.value === v)) sel.value = v; };
  if (p.sensor === 3) pick($("#newStorage"), p.device);
  if (p.sensor === 4 || p.sensor === 9) pick($("#newProbe"), `${p.device};${p.channelId}`);
  if (p.sensor === 8) pick($("#newGpu"), String(p.gpuIndex));
  if (p.sensor === 7) $("#newExec").value = p.device || "";
  if (p.sensor === 6 && $("#newHwmonPick")) { const m = /\/(hwmon\d+)\/([^/]+)$/.exec(p.device || ""); if (m) pick($("#newHwmonPick"), `${m[1]};${m[2]};${p.hwmonDevice}`); }
  $("#newZero").checked = !!p.zeroRpm;
  $("#newStatic").checked = false;
  syncNewForm();
  $("#newName").focus();
  $("#newName").select();
  toast(`Pick a name — the copy starts with ${copyFrom}'s curves`);
});

/* New profile form */
function buildNewForm() {
  const sel = $("#newSensor");
  const cur = sel.value;
  sel.textContent = "";
  const src = S.sources || {};
  for (const [id, label] of NEW_SENSORS) {
    if (id === 2 && !pumpChannel()) continue;
    if (id === 3 && !src.drives?.length) continue;
    if (id === 4 && !src.probes?.length) continue;
    if (id === 9 && !src.anyProbes?.length) continue;
    if ((id === 8 || id === 11) && (src.gpus?.length || 0) < 2) continue;
    if (id === 6 && !src.hwmon?.length) continue;
    sel.append(el("option", { value: id, text: label }));
  }
  if (cur) sel.value = cur;
  const fill = (s, list) => { s.textContent = ""; for (const o of list || []) s.append(el("option", { value: o.value, text: o.text })); };
  fill($("#newStorage"), src.drives);
  fill($("#newGpu"), src.gpus);
  const hw = $("#newHwmonWrap");
  if (src.hwmon?.length) {
    const s = el("select", { id: "newHwmonPick" });
    fill(s, src.hwmon);
    hw.replaceChildren(el("span", { text: "Sensor" }), s);
  }
  syncNewForm();
}
function syncNewForm() {
  const s = Number($("#newSensor").value);
  $("#newStorageWrap").hidden = s !== 3;
  $("#newProbeWrap").hidden = s !== 4 && s !== 9;
  if (s === 4 || s === 9) {
    const list = (s === 4 ? S.sources?.probes : S.sources?.anyProbes) || [];
    const ps = $("#newProbe");
    if (ps.dataset.for !== String(s)) { ps.dataset.for = s; ps.textContent = ""; for (const o of list) ps.append(el("option", { value: o.value, text: o.text })); }
  }
  $("#newGpuWrap").hidden = s !== 8;
  $("#newExecWrap").hidden = s !== 7;
  $("#newHwmonWrap").hidden = s !== 6;
}
$("#newSensor").addEventListener("change", syncNewForm);
$("#newProfileBox").addEventListener("toggle", () => { if (!$("#newProfileBox").open) copyFrom = null; });
$("#newProfile").addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = $("#newName").value.trim();
  const sensor = Number($("#newSensor").value);
  const body = { profile: name, sensor, static: $("#newStatic").checked, zeroRpm: $("#newZero").checked, linear: false };
  if (sensor === 3) body.hwmonDeviceId = $("#newStorage").value;
  if (sensor === 4 || sensor === 9) { const [dev, ch] = $("#newProbe").value.split(";"); body.deviceId = dev; body.channelId = Number(ch); }
  if (sensor === 8) body.gpuIndex = Number($("#newGpu").value);
  if (sensor === 7) body.externalExecutable = $("#newExec").value.trim();
  if (sensor === 6) {
    const v = $("#newHwmonPick")?.value || `${$("#newHwmon").value};${$("#newHwmonInput").value};${$("#newHwmon").value}`;
    const [hid, input, label] = v.split(";");
    Object.assign(body, { hwmonDeviceId: hid, temperatureInputId: input, hwmonDevice: label });
  }
  if (!(await act("/api/temperatures/new", "POST", body, `Created ${name}`))) return;
  if (copyFrom && S.profiles[copyFrom] && !body.static && !S.graphMode) {
    await pollProfiles();
    const src = S.profiles[copyFrom].profiles || [], dst = S.profiles[name]?.profiles || [];
    if (src.length === dst.length) {
      const data = JSON.stringify(Object.fromEntries(dst.map((b, i) => [b.id, { fans: src[i].fans, pump: src[i].pump }])));
      await act("/api/temperatures/update", "PUT", null, false, { form: { profile: name, data } });
    } else toast("Created with default bands — the copy's temperature ranges differ", "warn");
  } else if (copyFrom && S.profiles[copyFrom] && !body.static) {
    const src = S.profiles[copyFrom];
    for (const type of [0, 1]) {
      const points = (src.points?.[type] || []).map((q) => ({ x: q.x, y: q.y }));
      if (points.length) await act("/api/temperatures/updateGraph", "PUT", { profile: name, updateType: type, points }, false);
    }
  }
  copyFrom = null;
  $("#newProfile").reset();
  $("#newProfileBox").open = false;
  await pollProfiles();
  S.curve.name = name;
  loadCurve();
  syncNewForm();
});

/* ---------- Lighting ---------- */
const ledKey = () => JSON.stringify([Object.values(S.dev?.rgbDevices || {}).map((d) => [d.channelId, d.rgb, d.label]), channelsKey(), prof().OpenRGBIntegration, prof().RGBCluster]);
function fillModes(sel, cur, withNone) {
  sel.textContent = "";
  if (withNone) sel.append(el("option", { value: "", text: "Mixed", selected: !cur, disabled: true }));
  for (const m of rgbModes()) sel.append(el("option", { value: m, text: m, selected: m === cur }));
}
function buildLighting() {
  const rgbDevs = Object.values(S.dev?.rgbDevices || {}).sort((a, b) => a.channelId - b.channelId);
  const common = inCluster() ? S.cluster.DeviceProfile?.RGBProfile : rgbDevs.length && rgbDevs.every((d) => d.rgb === rgbDevs[0].rgb) ? rgbDevs[0].rgb : "";
  fillModes($("#rgbAll"), common, !common);
  const n = S.cluster?.Controllers ? Object.keys(S.cluster.Controllers).length : 0;
  const locked = prof().OpenRGBIntegration ? "OpenRGB is controlling these LEDs (turn it off under Device)."
    : prof().RGBCluster ? `These LEDs are part of the RGB cluster${n ? ` with ${n} device${n > 1 ? "s" : ""}` : ""}. The effect, brightness and colours above and below apply to the whole cluster.` : "";

  const row = $("#ledRow");
  row.textContent = "";
  if (locked) row.append(el("p", { class: "note", text: locked }));
  for (const d of rgbDevs) {
    const ch = S.dev.devices?.[d.channelId];
    const name = realLabel(d.label) || (ch ? displayName(ch) : d.name);
    const sel = el("select", { "aria-label": `${name} effect`, disabled: !!locked, "data-fk": `led-sel-${d.channelId}` });
    fillModes(sel, d.rgb);
    sel.addEventListener("change", async () => {
      if (await act("/api/color", "POST", { deviceId: S.serial, channelId: d.channelId, profile: sel.value }, `${name} → ${sel.value}`)) refreshLighting();
    });
    const ovBtn = el("button", { class: "btn ghost tiny", text: "Override", title: "Give this channel its own colours", disabled: !!locked, "data-fk": `led-ov-${d.channelId}` });
    const ov = el("div", { class: "override", hidden: true });
    ovBtn.addEventListener("click", () => toggleOverride(d.channelId, ov, ovBtn, d.rgb));
    const lname = el("span", { class: "lname ch-name", text: name, title: `${d.name} — click to rename this LED channel`, tabindex: "0", role: "button" });
    lname.addEventListener("click", () => startRename(lname, { channelId: d.channelId }, 1));
    lname.addEventListener("keydown", (e) => { if (e.key === "Enter" && !lname.isContentEditable) { e.preventDefault(); startRename(lname, { channelId: d.channelId }, 1); } });
    row.append(el("div", { class: "led", id: `led-${d.channelId}` },
      el("span", { class: "sw" }),
      lname,
      el("div", { class: "ch-row" }, sel, ovBtn),
      ov));
  }
  $("#rgbAll").disabled = $("#rgbAllApply").disabled = $("#rgbTarget").disabled = !!prof().OpenRGBIntegration;
  renderBulkTargets();

  const fx = $("#fxProfile");
  const keep = S.fx.name || "static";
  fx.textContent = "";
  for (const m of rgbModes().filter((m) => !FX_HIDDEN.has(m) && S.rgb?.profiles?.[m]).sort()) fx.append(el("option", { value: m, text: m, selected: m === keep }));
  S.fx.name = fx.value;
  loadFx();

  $("#brightness").value = (inCluster() ? S.cluster.DeviceProfile?.BrightnessSlider : prof().BrightnessSlider) ?? 100;
  $("#brightnessVal").textContent = `${$("#brightness").value}%`;
  $("#rgbOffAll").checked = !!S.dash?.rgbOff;
  S.built.leds = ledKey();
  paintSwatches();
}
function paintSwatches() {
  for (const d of Object.values(S.dev?.rgbDevices || {})) {
    const sw = $(`#led-${d.channelId} .sw`);
    if (!sw) continue;
    const p = S.rgb?.profiles?.[d.rgb];
    sw.className = "sw";
    sw.style.removeProperty("--sw");
    if (d.rgb === "off" || S.dash?.rgbOff) sw.classList.add("off");
    else if (d.rgb === "static" && p) sw.style.setProperty("--sw", hex(p.start));
    else if (p && !FX_SPEED_ONLY.has(d.rgb) && d.rgb !== "gradient") { sw.classList.add("duo"); sw.style.setProperty("--a", hex(p.start)); sw.style.setProperty("--b", hex(p.end)); }
    else sw.classList.add("anim");
  }
}
async function toggleOverride(channelId, box, btn, effect) {
  if (!box.hidden) { box.hidden = true; btn.classList.remove("on"); return; }
  let o;
  try {
    const r = await api("/api/color/getOverride", { method: "POST", body: { deviceId: S.serial, channelId, subDeviceId: 0 } });
    if (r.status !== 1 || typeof r.data !== "object" || !r.data) { toast(r.message || "This channel has no override", "warn"); return; }
    o = r.data;
  } catch (e) { toast(e.message, "err"); return; }
  const en = el("input", { type: "checkbox", checked: o.Enabled });
  const temps = FX_TEMP.has(effect);
  const pickers = [["RGBStartColor", temps ? "Cool" : "Colour 1"], ["RGBMiddleColor", temps ? "Warm" : "Middle"], ["RGBEndColor", temps ? "Hot" : "Colour 2"]].map(([key, label]) => {
    const c = el("input", { type: "color", value: hex(o[key]), "aria-label": label });
    const t = el("input", { type: "number", class: "num", min: 0, max: 105, step: 1, value: o[key]?.temperature ?? 0, "aria-label": `${label} °C` });
    return { key, c, t, node: el("label", { class: "field" }, el("span", { text: label }), c, temps ? el("span", { class: "with-unit" }, t, "°C") : null) };
  });
  // The API accepts 0–2 here (requests.go), although the driver would take up to 10.
  const sp = el("input", { type: "range", min: "0.1", max: "2", step: "0.1", value: clamp(o.RgbModeSpeed || 1, 0.1, 2), "aria-label": "Speed" });
  const save = el("button", { class: "btn primary tiny", text: "Save" });
  const over = (o.RgbModeSpeed || 0) > 2 ? el("p", { class: "muted small", text: `Stored speed is ${o.RgbModeSpeed}; OpenLinkHub's API only accepts up to 2 here, so saving sets it to 2 at most.` }) : null;
  save.addEventListener("click", async () => {
    const col = (k) => { const x = pickers.find((q) => q.key === k); return { ...rgbOf(x.c.value), temperature: clamp(Number(x.t.value) || 0, 0, 105) }; };
    const body = {
      deviceId: S.serial, channelId, subDeviceId: 0, enabled: en.checked,
      startColor: col("RGBStartColor"), middleColor: col("RGBMiddleColor"), endColor: col("RGBEndColor"),
      speed: Number(sp.value),
    };
    if (await act("/api/color/setOverride", "POST", body, en.checked ? "Override on" : "Override off")) { box.hidden = true; btn.classList.toggle("on", en.checked); }
  });
  box.replaceChildren(
    el("label", { class: "check" }, en, " Use these colours instead of the effect's"),
    el("div", { class: "ch-row wrap" }, ...pickers.map((q) => q.node)),
    el("div", { class: "ch-row" }, el("label", { class: "field grow" }, el("span", { text: "Speed" }), sp), save), over);
  box.hidden = false;
  btn.classList.add("on");
}

/* Effect editor */
function loadFx() {
  const name = $("#fxProfile").value;
  S.fx.name = name;
  const p = S.rgb?.profiles?.[name];
  const body = $("#fxBody");
  body.textContent = "";
  const users = Object.values(S.dev?.rgbDevices || {}).filter((d) => d.rgb === name).length;
  $("#fxUsed").textContent = inCluster() ? (S.cluster.DeviceProfile?.RGBProfile === name ? "the cluster's current effect" : "cluster effect") : users ? `used by ${users} channel${users > 1 ? "s" : ""}` : "not in use";
  if (!p) return;
  S.fx.draft = JSON.parse(JSON.stringify(p));
  const D = S.fx.draft;
  const color = (label, key, withTemp) => {
    const c = el("input", { type: "color", value: hex(D[key]) });
    c.addEventListener("input", () => Object.assign(D[key], rgbOf(c.value)));
    const f = el("label", { class: "field" }, el("span", { text: label }), c);
    if (withTemp) {
      const t = el("input", { type: "number", min: 0, max: 105, step: 1, value: D[key].temperature ?? 0, class: "num", "aria-label": `${label} temperature` });
      t.addEventListener("input", () => { D[key].temperature = clamp(Number(t.value) || 0, 0, 105); });
      f.append(el("span", { class: "with-unit" }, t, "°C"));
    }
    return f;
  };
  if (name === "probe-temperature") {
    const num = (label, key) => {
      const i = el("input", { type: "number", class: "num", min: 0, max: 100, step: 1, value: D[key] ?? 0 });
      i.addEventListener("input", () => { D[key] = clamp(Number(i.value) || 0, 0, 100); });
      return el("label", { class: "field" }, el("span", { text: label }), el("span", { class: "with-unit" }, i, "°C"));
    };
    body.append(color("Cool", "start"), color("Hot", "end"), num("From", "minTemp"), num("To", "maxTemp"));
  } else if (FX_TEMP.has(name)) {
    body.append(color("Cool", "start", true), color("Warm", "middle", true), color("Hot", "end", true),
      el("p", { class: "note grow", text: "Colour blends between these points as the temperature rises." }));
  } else if (name === "gradient" || D.gradients) {
    body.append(gradientEditor(D));
  } else if (name === "static") {
    body.append(color("Colour", "start"));
  } else if (!FX_SPEED_ONLY.has(name)) {
    body.append(color("Colour 1", "start"), color("Colour 2", "end"));
  }
  if (!FX_NO_SPEED.has(name)) {
    const max = FX_SLOW.has(name) ? 3 : 10;
    const r = el("input", { type: "range", min: 1, max, step: FX_SLOW.has(name) ? 1 : 0.1, value: clamp(D.speed || 4, 1, max) });
    const out = el("span", { class: "muted", text: fmt(r.value, 1) });
    r.addEventListener("input", () => { D.speed = Number(r.value); out.textContent = fmt(r.value, 1); });
    body.append(el("label", { class: "field grow" }, el("span", {}, "Speed ", out), r));
  }
}
function gradientEditor(D) {
  const wrap = el("div", { class: "grad grow" });
  const stops = () => Object.keys(D.gradients || {}).sort((a, b) => a - b).map((k) => D.gradients[k]);
  const paintBar = () => {
    const bar = wrap.querySelector(".grad-bar");
    if (bar) bar.style.background = `linear-gradient(90deg, ${stops().map((s) => `${hex(s)} ${Math.round((s.position ?? 0) * 100)}%`).join(", ")})`;
  };
  const draw = () => {
    wrap.textContent = "";
    const list = stops();
    const bar = el("div", { class: "grad-bar" });
    wrap.append(bar);
    paintBar();
    list.forEach((s, i) => {
      const c = el("input", { type: "color", value: hex(s), "aria-label": `Stop ${i + 1} colour` });
      const pos = el("input", { type: "range", min: 0, max: 1, step: 0.01, value: s.position ?? 0, "aria-label": `Stop ${i + 1} position` });
      const rm = el("button", { class: "btn ghost tiny", text: "Remove", disabled: list.length <= 2 });
      c.addEventListener("input", () => { Object.assign(s, rgbOf(c.value)); paintBar(); });
      pos.addEventListener("change", () => { s.position = Number(pos.value); draw(); });
      rm.addEventListener("click", () => { const arr = stops().filter((x) => x !== s); D.gradients = Object.fromEntries(arr.map((x, j) => [j, x])); draw(); });
      const br = el("input", { type: "range", class: "bright", min: 0, max: 1, step: 0.05, value: s.brightness ?? 1, "aria-label": `Stop ${i + 1} brightness`, title: "Brightness" });
      br.addEventListener("change", () => { s.brightness = Number(br.value); });
      wrap.append(el("div", { class: "ch-row stop" }, c, el("label", { class: "field grow" }, el("span", { text: "Position" }), pos), el("label", { class: "field" }, el("span", { text: "Brightness" }), br), rm));
    });
    const add = el("button", { class: "btn ghost tiny", text: "＋ Add stop" });
    add.addEventListener("click", () => {
      const arr = stops();
      const last = arr[arr.length - 1] || { red: 255, green: 255, blue: 255 };
      arr.push({ red: last.red, green: last.green, blue: last.blue, brightness: 1, position: 1 });
      D.gradients = Object.fromEntries(arr.map((x, j) => [j, x]));
      draw();
    });
    wrap.append(add);
  };
  if (!D.gradients) D.gradients = { 0: { ...D.start, position: 0 }, 1: { ...D.end, position: 1 } };
  draw();
  return wrap;
}
$("#fxProfile").addEventListener("change", loadFx);
$("#fxRevert").addEventListener("click", loadFx);
$("#fxSave").addEventListener("click", async () => {
  const name = S.fx.name, D = S.fx.draft;
  if (!D) return;
  const c = (x) => ({ red: x?.red ?? 0, green: x?.green ?? 0, blue: x?.blue ?? 0, temperature: x?.temperature ?? 0 });
  const start = c(D.start);
  const body = {
    deviceId: lightId(), profile: name, startColor: start,
    middleColor: c(D.middle), endColor: name === "static" ? { ...start, temperature: D.end?.temperature ?? 0 } : c(D.end),
    speed: clamp(Number(D.speed) || 4, 1, 10), alternateColors: !!D.alternateColors, rgbDirection: D.rgbDirection || 0,
    rgbMinTemp: D.minTemp ?? 0, rgbMaxTemp: D.maxTemp ?? 0,
  };
  if (body.rgbMaxTemp < body.rgbMinTemp) { toast("“To” must be at least “From”", "warn"); return; }
  if (D.gradients) body.colorZones = Object.fromEntries(Object.entries(D.gradients).map(([k, s]) => [k, { red: s.red, green: s.green, blue: s.blue, brightness: s.brightness ?? 1, position: s.position ?? 0 }]));
  if (await act("/api/color/change", "PUT", body, `${name} saved`)) refreshLighting();
});
async function refreshLighting() {
  await Promise.all([pollFast(), pollDash()]);
  await pollRgb();
  keepFocus(buildLighting);
}
$("#rgbAllApply").addEventListener("click", async () => {
  const profile = $("#rgbAll").value, target = $("#rgbTarget").value;
  if (!profile) return;
  if (inCluster()) {
    if (await act("/api/color", "POST", { deviceId: "cluster", channelId: 0, profile }, `Cluster → ${profile}`)) refreshLighting();
  } else if (target === "global") {
    if (await act("/api/color/global", "POST", { profile }, `Every device → ${profile}`)) refreshLighting();
  } else if (target === "sel") {
    // Per channel: cc has no UpdateRgbProfileBulk, so a channelIds request would fail there.
    const ids = [...S.sel].filter((id) => S.dev?.rgbDevices?.[id]);
    let ok = 0;
    for (const id of ids) { try { if ((await api("/api/color", { method: "POST", body: { deviceId: S.serial, channelId: id, profile } })).status === 1) ok++; } catch { /* counted */ } }
    toast(`${ok}/${ids.length} LEDs → ${profile}`, ok === ids.length ? "ok" : "warn");
    refreshLighting();
  } else if (await act("/api/color", "POST", { deviceId: S.serial, channelId: -1, profile }, `All LEDs → ${profile}`)) refreshLighting();
});
$("#oneColourApply").addEventListener("click", async () => {
  const btn = $("#oneColourApply");
  if (!armed(btn, "Overwrites every static colour — click again")) return;
  if (await act("/api/color/all", "POST", { color: { ...rgbOf($("#oneColour").value), brightness: 1 } }, "Every LED set to one colour")) refreshLighting();
});
let brightTimer;
$("#brightness").addEventListener("input", () => {
  $("#brightnessVal").textContent = `${$("#brightness").value}%`;
  clearTimeout(brightTimer);
  brightTimer = setTimeout(() => act("/api/brightness/gradual", "POST", { deviceId: lightId(), brightness: Number($("#brightness").value) }, `Brightness ${$("#brightness").value}%`), 300);
});
async function saveDashboard(patch, msg) {
  await pollDash();
  const d = S.dash;
  if (!d) { toast("Couldn't read dashboard settings", "err"); return false; }
  const body = {
    showCpu: d.showCpu, showGpu: d.showGpu, showDisk: d.showDisk, showDevices: d.showDevices, showLabels: d.showLabels,
    celsius: d.celsius, showBattery: d.showBattery, temperatureBar: d.temperatureBar, addDeviceToDashboard: d.addDeviceToDashboard,
    rgbOff: d.rgbOff, languageCode: d.languageCode, theme: d.theme, keyboardLayout: d.keyboardLayout, ...patch,
  };
  return act("/api/dashboard/update", "POST", body, msg);
}
$("#rgbOffAll").addEventListener("change", async (e) => {
  const on = e.target.checked;
  if (!(await saveDashboard({ rgbOff: on }, on ? "All LEDs off" : "LEDs back on"))) e.target.checked = !on;
  refreshLighting();
});

/* ---------- Device ---------- */
function buildDevice() {
  const d = S.dev, p = prof();
  // Saved setups
  const ul = $("#setups");
  ul.textContent = "";
  for (const [name, up] of Object.entries(d.userProfiles || {})) {
    const active = !!up.Active;
    const use = el("button", { class: "btn tiny", text: active ? "Active" : "Switch", disabled: active });
    use.addEventListener("click", async () => {
      if (await act("/api/userProfile/change", "POST", { deviceId: S.serial, userProfileName: name }, `Switched to ${name}`)) { await Promise.all([pollFast(), pollRgb()]); S.built.device = ""; buildLighting(); tick(); }
    });
    const del = el("button", { class: "btn ghost danger tiny", text: "Delete", hidden: active || name === "default" });
    del.addEventListener("click", async () => {
      if (!armed(del, "Sure?")) return;
      if (await act("/api/userProfile/delete", "DELETE", { deviceId: S.serial, userProfileName: name }, `Deleted ${name}`)) { await pollFast(); buildDevice(); }
    });
    ul.append(el("li", { class: active ? "on" : "" }, el("span", { class: "grow", text: name }), use, del));
  }
  $("#openrgb").checked = !!p.OpenRGBIntegration;
  $("#cluster").checked = !!p.RGBCluster;
  // Custom ARGB ports (only shown when the hub has empty LED ports)
  const free = Object.entries(d.FreeLedPorts || {});
  $("#argbBox").hidden = !free.length;
  const ports = $("#argbPorts");
  ports.textContent = "";
  for (const [port, label] of free) {
    const sel = el("select", { "aria-label": label });
    for (const x of d.ExternalLedDevice || []) sel.append(el("option", { value: x.Index, text: x.Total ? `${x.Name} (${x.Total} LEDs)` : x.Name, selected: Number(p.CustomLEDs?.[port] || 0) === x.Index }));
    sel.addEventListener("change", () => act("/api/argb", "POST", { deviceId: S.serial, portId: Number(port), deviceType: Number(sel.value) }, `${label} updated`));
    ports.append(el("label", { class: "field" }, el("span", { text: label }), sel));
  }
  const facts = $("#facts");
  facts.textContent = "";
  for (const [k, v] of [["Device", d.product], ["Firmware", d.firmware], ["Serial", d.serial], ["Cooler", pumpChannel()?.name]]) if (v) facts.append(el("dt", { text: k }), el("dd", { text: v }));
  S.built.device = JSON.stringify([Object.keys(d.userProfiles || {}), Object.values(d.userProfiles || {}).map((u) => u.Active), p.OpenRGBIntegration, p.RGBCluster, free.length]);
}
/* Every connected device: cooling hubs are handled on this page, everything else opens its own page. */
function renderDeviceList() {
  const root = $("#deviceList");
  root.textContent = "";
  if (!S.all.length) { root.append(el("p", { class: "note", text: "OpenLinkHub hasn't found any devices. Check that they're plugged in and enabled under Settings." })); return; }
  for (const d of S.all) {
    const here = S.devices.some((x) => x.Serial === d.Serial);
    const icon = el("img", { src: `/static/img/icons/${encodeURIComponent(d.Image || "icon-device.svg")}`, alt: "", class: "dev-icon" });
    const action = here
      ? (d.Serial === S.serial ? el("span", { class: "badge", text: "shown above" }) : el("a", { class: "btn tiny", href: `/?device=${encodeURIComponent(d.Serial)}`, text: "Show here" }))
      : el("a", { class: "btn tiny", href: `/device/${encodeURIComponent(d.Serial)}`, text: "Open" });
    root.append(el("div", { class: "dev" }, icon, el("div", { class: "grow" }, el("div", { class: "lname", text: d.Product }), el("div", { class: "muted small", text: d.Serial })), action));
  }
}

/* Display units, supported devices, sign-in, metrics */
async function buildServer() {
  $("#fahrenheit").checked = useF();
  try {
    const txt = await (await fetch("/api/metrics", { cache: "no-store" })).text();
    const on = txt.startsWith("# HELP") || txt.includes("# TYPE");
    $("#serverFacts").replaceChildren(el("dt", { text: "Metrics" }), el("dd", { text: on ? "Prometheus at /api/metrics" : "off (metrics in config.json)" }));
  } catch { /* optional */ }
  try {
    const r = await api("/api/getSupportedDevices");
    const list = (r.data || []).slice().sort((a, b) => a.Name.localeCompare(b.Name));
    const present = new Set(S.all.map((d) => d.Product.toUpperCase()));
    const row = (d) => {
      const sw = el("input", { type: "checkbox", checked: d.Enabled });
      sw.addEventListener("change", async () => {
        const ok = await act("/api/setSupportedDevices", "POST", { supportedDevices: { [d.ProductId]: sw.checked } }, `${d.Name} ${sw.checked ? "enabled" : "excluded"} — restart OpenLinkHub to apply`);
        if (!ok) sw.checked = !sw.checked; else d.Enabled = sw.checked;
      });
      sw.setAttribute("aria-label", `OpenLinkHub controls ${d.Name}`);
      sw.dataset.fk = `sup-${d.ProductId}`;
      return el("li", {}, el("span", { class: "grow", text: d.Name }), el("span", { class: "muted small", text: `0x${d.ProductId.toString(16).padStart(4, "0")}` }), el("label", { class: "switch" }, sw, el("span", { class: "track" })));
    };
    const mine = list.filter((d) => present.has(d.Name.toUpperCase()) || !d.Enabled);
    const dupes = mine.filter((d, i) => mine.findIndex((x) => x.Name === d.Name) !== i).length;
    if (dupes) $("#supported").after(el("p", { class: "muted small", text: "Some products ship under more than one USB ID with the same name; lsusb shows which one you have." }));
    $("#supported").replaceChildren(...(mine.length ? mine.map(row) : [el("li", { class: "muted small", text: "No connected devices matched by name" })]));
    const all = $("#supportedAll");
    const draw = () => { const q = $("#supportedFilter").value.trim().toLowerCase(); all.replaceChildren(...list.filter((d) => !q || d.Name.toLowerCase().includes(q)).slice(0, 200).map(row)); };
    $("#supportedFilter").addEventListener("input", draw);
    draw();
  } catch { $("#supported").replaceChildren(el("li", { class: "muted small", text: "Couldn't load the device list" })); }
  const d = S.dash || {};
  $("#authEnabled").checked = !!d.authEnabled;
  $("#authUser").value = d.authUsername || "";
  $("#authOrigin").value = d.authOrigin || `https://${location.hostname}:${location.port || 27003}`;
  $("#authLife").value = d.sessionLifetime || 60;
}
$("#fahrenheit").addEventListener("change", async (e) => {
  if (!(await saveDashboard({ celsius: !e.target.checked }, e.target.checked ? "Showing °F" : "Showing °C"))) { e.target.checked = !e.target.checked; return; }
  await pollDash();
  tilesKey = "";
  S.built.channels = "";
  renderTiles(); renderHistory(); renderChannels(); loadCurve();
});
$("#authForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  const enabled = $("#authEnabled").checked, pass = $("#authPass").value;
  if (enabled) {
    if (pass.length < 12 || !/[A-Za-z]/.test(pass) || !/\d/.test(pass) || !/[^A-Za-z0-9\s]/.test(pass)) { toast("Password: 12+ characters with a letter, a digit and a symbol", "warn"); return; }
    if (!/^https:\/\//.test($("#authOrigin").value)) { toast("Sign-in needs an https:// address", "warn"); return; }
  }
  const btn = e.submitter || $("#authForm button");
  if (enabled && !armed(btn, "After restart you'll need this password — click again")) return;
  const ok = await act("/api/auth/update", "POST", { authEnabled: enabled, authUsername: $("#authUser").value.trim(), authPassword: pass, authOrigin: $("#authOrigin").value.trim(), sessionLifetime: Number($("#authLife").value) }, enabled ? "Sign-in saved — restart OpenLinkHub to turn it on" : "Sign-in turned off — restart OpenLinkHub to apply");
  if (ok) $("#authPass").value = "";
});
async function buildSchedule() {
  const s = await loadSchedule();
  if (!s) { $("#schedSave").disabled = true; return; }
  $("#schedOn").checked = s.on;
  $("#schedOff").value = s.off;
  $("#schedOnAt").value = s.at;
  $("#schedSave").dataset.lcd = s.lcd ? "1" : "";
}
$("#schedSave").addEventListener("click", () => {
  const on = $("#schedOn").checked, off = $("#schedOff").value, at = $("#schedOnAt").value;
  if (!off || !at) { toast("Set both times (the API needs them even when the schedule is off)", "warn"); return; }
  act("/api/scheduler/rgb", "POST", { rgbControl: on, rgbOff: off, rgbOn: at, lcdControl: $("#schedSave").dataset.lcd === "1" }, on ? `LEDs off ${off} – on ${at}` : "Schedule off");
});
$("#saveSetup").addEventListener("submit", async (e) => {
  e.preventDefault();
  const name = $("#setupName").value.trim();
  if (await act("/api/userProfile", "PUT", { deviceId: S.serial, userProfileName: name }, `Saved setup ${name}`)) { $("#setupName").value = ""; await pollFast(); buildDevice(); }
});
for (const [id, path, label] of [["openrgb", "/api/color/setOpenRgbIntegration", "OpenRGB"], ["cluster", "/api/color/setCluster", "RGB cluster"]]) {
  $(`#${id}`).addEventListener("change", async (e) => {
    const on = e.target.checked;
    if (!(await act(path, "POST", { deviceId: S.serial, mode: on ? 1 : 0 }, `${label} ${on ? "on" : "off"}`))) e.target.checked = !on;
    await refreshLighting();
  });
}
$("#restoreFile").addEventListener("change", async (e) => {
  const f = e.target.files?.[0];
  e.target.value = "";
  if (!f) return;
  const lbl = e.target.closest("label");
  if (!lbl.dataset.ready) {
    lbl.dataset.ready = "1";
    lbl.firstChild.textContent = `Restore ${f.name}? Pick it again to confirm`;
    setTimeout(() => { lbl.dataset.ready = ""; lbl.firstChild.textContent = "Restore…"; }, 8000);
    return;
  }
  lbl.dataset.ready = "";
  lbl.firstChild.textContent = "Restore…";
  const fd = new FormData();
  fd.append("backupFile", f);
  try {
    const r = await (await fetch("/api/restore", { method: "POST", body: fd })).json();
    toast(r.message || (r.status === 1 ? "Restored" : "Restore failed"), r.status === 1 ? "ok" : "err");
  } catch (err) { toast(err.message, "err"); }
});
$("#devicePick").addEventListener("change", (e) => { location.search = `?device=${encodeURIComponent(e.target.value)}`; });

/* ---------- Main loop ---------- */
let failures = 0;
function showDown(e) {
  failures++;
  const a = $("#alert");
  if (e instanceof AuthError) {
    setLive(false, "signed out");
    a.hidden = false;
    a.replaceChildren("OpenLinkHub needs you to sign in. ", el("a", { href: "/login", text: "Sign in" }), ", then come back to this page.");
    return;
  }
  setLive(false, failures > 2 ? "offline" : "retrying…");
  if (failures > 2) { a.hidden = false; a.textContent = "Can't reach OpenLinkHub. This page reconnects by itself when it's back."; }
}
let ticking = false, tickAgain = false;
async function tick(record = false) {
  if (ticking) { tickAgain = true; return; }
  ticking = true;
  try {
    await pollFast(record);
  } catch (e) {
    showDown(e);
    return;
  } finally {
    ticking = false;
    if (tickAgain) { tickAgain = false; setTimeout(tick, 0); }
  }
  if (failures) { failures = 0; $("#alert").hidden = true; $("#alert").dataset.msg = ""; }
  setLive(true, "live");
  try {
    render();
  } catch (e) {
    console.error(e);
    if (!tick.reported) { tick.reported = true; toast(`Display error: ${e.message}`, "err"); }
  }
}
function render() {
  const d = S.dev;
  $("#deviceLine").textContent = d ? [d.product, d.firmware && `fw ${d.firmware}`, pumpChannel()?.name].filter(Boolean).join(" · ")
    : `${S.all.length} device${S.all.length === 1 ? "" : "s"} · no fan or pump controller`;
  if (document.hidden) return;
  renderTiles();
  renderHistory();
  if (!d) return;
  renderChannels();
  const busy = document.activeElement?.closest?.("#sec-light, #sec-device");
  if (!busy && S.built.leds !== ledKey()) keepFocus(buildLighting);
  else if (!busy) paintSwatches();
  if (S.curve.drag < 0) { updateNow(); if (!S.curve.dirty) renderProfiles(); }
}
let booted = false, booting = false;
async function boot() {
  if (booting) return;
  booting = true;
  try {
    await loadDevices();
    const cooling = !!S.serial;
    $$(".cooling-only").forEach((n) => { n.hidden = !cooling; });
    renderDeviceList();
    await pollDash();
    if (cooling) await Promise.all([pollProfiles(), loadSources()]);
    await tick(true);
    renderLegend();
    if (cooling) {
      await pollRgb();
      S.curve.name = fanChannels()[0]?.profile || Object.keys(S.profiles)[0];
      loadCurve();
      buildNewForm();
      buildLighting();
      buildDevice();
    }
    buildSchedule();
    buildServer();
    booted = true;
  } catch (e) {
    showDown(e);
  } finally {
    booting = false;
  }
}
async function init() {
  renderLegend();
  await boot();
  setInterval(() => (booted ? tick(true) : boot()), POLL_MS);
  setInterval(async () => {
    if (!booted) return;
    try {
      await pollDash();
      if (!S.serial) return;
      await pollProfiles();
      if (!S.curve.dirty && S.curve.drag < 0) loadCurve();
      if (!document.activeElement?.closest?.("#sec-device")) buildDevice();
    } catch { /* next round */ }
  }, SLOW_POLL_MS);
  window.addEventListener("resize", renderHistory);
  document.addEventListener("visibilitychange", () => { if (!document.hidden && booted) tick(); });
  matchMedia("(prefers-color-scheme: light)").addEventListener("change", () => { renderTiles(); renderHistory(); renderCurve(); });
}
init();

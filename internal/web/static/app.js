// Vanilla JS frontend. Designed so adding routes/views later is just adding
// view handlers and (optionally) re-rendering. No build step.

async function jsonOrError(res) {
  if (res.ok) {
    if (res.status === 204) return null;
    return res.json();
  }
  let msg = `HTTP ${res.status}`;
  try {
    const body = await res.json();
    if (body.error) msg = body.error;
  } catch (_) { /* ignore */ }
  const err = new Error(msg);
  err.status = res.status;
  throw err;
}

const api = {
  async parse(text) {
    return fetch("/api/parse", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text }) }).then(jsonOrError);
  },
  async listProjects() { return fetch("/api/projects?include_archived=true").then(jsonOrError); },
  async createProject(body) {
    return fetch("/api/projects", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async patchProject(id, body) {
    return fetch(`/api/projects/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async deleteProject(id) {
    return fetch(`/api/projects/${id}`, { method: "DELETE" }).then(jsonOrError);
  },
  async listTasks(params = {}) {
    // Always pass the client's local "today" so the server's visibility filter
    // (visible_from <= today) uses the user's local date rather than UTC.
    const q = new URLSearchParams({ today: todayISO(), ...params });
    return fetch("/api/tasks?" + q.toString()).then(jsonOrError);
  },
  async getTask(id) {
    return fetch(`/api/tasks/${id}`).then(jsonOrError);
  },
  async createTask(body) {
    return fetch("/api/tasks", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async patchTask(id, body) {
    return fetch(`/api/tasks/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async completeTask(id) {
    return fetch(`/api/tasks/${id}/complete`, { method: "POST" }).then(jsonOrError);
  },
  async reopenTask(id) {
    return fetch(`/api/tasks/${id}/reopen`, { method: "POST" }).then(jsonOrError);
  },
  async deleteTask(id) {
    return fetch(`/api/tasks/${id}`, { method: "DELETE" }).then(jsonOrError);
  },
  async listAudit(params = {}) {
    const q = new URLSearchParams(params);
    return fetch("/api/audit?" + q.toString()).then(jsonOrError);
  },
  async listComments(taskId) {
    return fetch(`/api/tasks/${taskId}/comments`).then(jsonOrError);
  },
  async addComment(taskId, body) {
    return fetch(`/api/tasks/${taskId}/comments`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ body }) }).then(jsonOrError);
  },
  async listAbsences() { return fetch("/api/absences?today=" + todayISO()).then(jsonOrError); },
  async createAbsence(body) {
    return fetch("/api/absences", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async patchAbsence(id, body) {
    return fetch(`/api/absences/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async deleteAbsence(id) {
    return fetch(`/api/absences/${id}`, { method: "DELETE" }).then(jsonOrError);
  },
  // Epics are loaded with include_archived=true, like projects: a task whose
  // epic was tidied away still has to be able to name it.
  async listEpics() { return fetch("/api/epics?include_archived=true").then(jsonOrError); },
  async createEpic(body) {
    return fetch("/api/epics", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async patchEpic(id, body) {
    return fetch(`/api/epics/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(jsonOrError);
  },
  async completeEpic(id) {
    return fetch(`/api/epics/${id}/complete`, { method: "POST" }).then(jsonOrError);
  },
  async reopenEpic(id) {
    return fetch(`/api/epics/${id}/reopen`, { method: "POST" }).then(jsonOrError);
  },
  async deleteEpic(id) {
    return fetch(`/api/epics/${id}`, { method: "DELETE" }).then(jsonOrError);
  },
  async nextWorkday() { return fetch("/api/next-workday").then(jsonOrError); },
  async shareLinks() { return fetch("/api/share-links").then(jsonOrError); },
};

// Recurrence presets — value is the canonical RRULE string posted to the API.
// "" = no recurrence, "custom" = show a free-text RRULE input.
const RECURRENCE_PRESETS = [
  { value: "",                                       label: "Keine Wiederholung" },
  { value: "FREQ=DAILY",                             label: "Täglich" },
  { value: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",       label: "Werktäglich" },
  { value: "FREQ=WEEKLY",                            label: "Wöchentlich" },
  { value: "FREQ=WEEKLY;INTERVAL=2",                 label: "Alle 2 Wochen" },
  { value: "FREQ=MONTHLY",                           label: "Monatlich" },
  { value: "FREQ=MONTHLY;INTERVAL=3",                label: "Alle 3 Monate" },
  { value: "FREQ=YEARLY",                            label: "Jährlich" },
  { value: "custom",                                 label: "Eigene RRULE…" },
];

function fillRecurrenceSelect(id) {
  const el = document.getElementById(id);
  if (!el || el.dataset.filled === "1") return;
  for (const p of RECURRENCE_PRESETS) {
    const opt = document.createElement("option");
    opt.value = p.value;
    opt.textContent = p.label;
    el.appendChild(opt);
  }
  el.dataset.filled = "1";
}

// Returns the rule the user has currently selected/typed in the given pair of
// controls, or "" if none. Selecting "custom" reads from the custom input.
function readRecurrence(selectId, customId) {
  const sel = document.getElementById(selectId);
  if (!sel) return "";
  if (sel.value === "custom") {
    return document.getElementById(customId).value.trim();
  }
  return sel.value;
}

// Snap selector + custom field to a given RRULE string. If the rule matches a
// preset exactly, the preset is picked; otherwise "custom" is selected and the
// rule appears in the custom input.
function setRecurrenceControls(selectId, customWrapId, customId, rule) {
  const sel = document.getElementById(selectId);
  const wrap = document.getElementById(customWrapId);
  const cust = document.getElementById(customId);
  if (!sel) return;
  const matched = RECURRENCE_PRESETS.some(p => p.value === rule && p.value !== "custom");
  if (matched) {
    sel.value = rule;
    if (wrap) wrap.classList.add("hidden");
    if (cust) cust.value = "";
  } else if (rule) {
    sel.value = "custom";
    if (wrap) wrap.classList.remove("hidden");
    if (cust) cust.value = rule;
  } else {
    sel.value = "";
    if (wrap) wrap.classList.add("hidden");
    if (cust) cust.value = "";
  }
}

// --- Recurrence mode (Lastenheft 4.6) ---------------------------------------
// "fixed" keeps a task on its planned date (a birthday stays on its day even if
// ticked off late); "after_completion" restarts the interval on the day it is
// actually done (clean the coffee machine every 4 weeks).
const RECUR_FIXED = "fixed";
const RECUR_AFTER_DONE = "after_completion";

// Mirrors recurrence.Rule.HasCalendarAnchor on the server: rules pinned to a
// weekday or day of month have no coherent "ab Erledigung" reading, so the
// backend rejects that combination and we grey the option out instead.
function ruleHasCalendarAnchor(rule) {
  return /(^|;)\s*BY(DAY|MONTHDAY|MONTH)\s*=/i.test(rule || "");
}

function currentRecurrenceMode() {
  const el = document.getElementById("td-recur-mode");
  return (el && el.dataset.mode) || RECUR_FIXED;
}

// The series date lives in its own date field, deliberately separate from the
// task's start date: postponing one occurrence must not move the series.
function currentRecurrenceAnchor() {
  const el = document.getElementById("td-recur-anchor");
  return (el && el.value) || "";
}

function setRecurrenceAnchor(iso) {
  const el = document.getElementById("td-recur-anchor");
  if (el) el.value = iso || "";
}

function setRecurrenceMode(mode) {
  const group = document.getElementById("td-recur-mode");
  if (!group) return;
  group.dataset.mode = mode === RECUR_AFTER_DONE ? RECUR_AFTER_DONE : RECUR_FIXED;
  for (const btn of group.querySelectorAll(".td-seg")) {
    const on = btn.dataset.mode === group.dataset.mode;
    btn.classList.toggle("is-active", on);
    btn.setAttribute("aria-pressed", on ? "true" : "false");
  }
}

// Shows the mode switch and the series-date field only for recurring tasks, and
// disables "ab Erledigung" for calendar-anchored rules.
function syncRecurrenceModeUI() {
  const wrap = document.getElementById("td-recur-mode-wrap");
  const group = document.getElementById("td-recur-mode");
  const anchorField = document.getElementById("td-recur-anchor-field");
  if (!wrap || !group || !anchorField) return;

  const rule = readRecurrence("td-recurrence", "td-recur-custom");
  wrap.classList.toggle("hidden", !rule);
  if (!rule) {
    anchorField.classList.add("hidden");
    return;
  }

  const anchored = ruleHasCalendarAnchor(rule);
  const afterBtn = group.querySelector('[data-mode="after_completion"]');
  afterBtn.disabled = anchored;
  afterBtn.title = anchored
    ? "Nur bei einfachen Intervallen möglich — nicht bei festem Wochentag oder Monatstag."
    : "";
  // A rule with a weekday/monthday anchor can only be fixed; snap back so the
  // UI never shows a state the server would reject.
  if (anchored && currentRecurrenceMode() === RECUR_AFTER_DONE) setRecurrenceMode(RECUR_FIXED);

  // Only "Fester Termin" has a series date — "Ab Erledigung" re-anchors on every
  // completion, so there is nothing stable to edit.
  const fixed = currentRecurrenceMode() === RECUR_FIXED;
  anchorField.classList.toggle("hidden", !fixed);
  // A task that has just become recurring inherits its start date as the series date.
  if (fixed && !currentRecurrenceAnchor()) {
    setRecurrenceAnchor(document.getElementById("td-visible-from").value || "");
  }
}

// Projektfarben als Druckfarben: dieselbe Helligkeit und Sättigung wie der
// Rest der Oberfläche, damit ein Projektpunkt neben einem Prioritätshaken
// nicht wie ein Signal aussieht. Bestehende Projekte behalten ihren
// gespeicherten Hex — die Liste gilt für neue.
const PROJECT_COLOR_PRESETS = [
  { name: "Zinnober",       hex: "#b5342a" },
  { name: "Ocker",          hex: "#a2620f" },
  { name: "Safran",         hex: "#b08b12" },
  { name: "Reseda",         hex: "#6b8f3a" },
  { name: "Tannengrün",     hex: "#2c7351" },
  { name: "Grünspan",       hex: "#2b8d84" },
  { name: "Prozesscyan",    hex: "#0b6e88" },
  { name: "Stahlblau",      hex: "#2f6f9e" },
  { name: "Preußischblau",  hex: "#3a4fa0" },
  { name: "Indigo",         hex: "#6a4ca8" },
  { name: "Aubergine",      hex: "#8e3d86" },
  { name: "Krapp",          hex: "#a83c62" },
  { name: "Graphit",        hex: "#666c76" },
];
const DEFAULT_PROJECT_COLOR = "#666c76";

function projectColor(p) {
  if (!p) return DEFAULT_PROJECT_COLOR;
  if (p.color && p.color !== "") return p.color;
  return p.id === "inbox" ? "#3a4fa0" : DEFAULT_PROJECT_COLOR;
}
function projectById(id) { return state.projects.find(p => p.id === id); }
function epicById(id) { return state.epics.find(e => e.id === id); }

// Epics that may still take work: not archived, not finished. Optionally
// narrowed to one project (mirrors domain.ActiveEpics on the server).
function activeEpics(projectId) {
  return state.epics
    .filter(e => !e.archived && e.status === "active")
    .filter(e => !projectId || e.project_id === projectId)
    .sort((a, b) => a.sort_order - b.sort_order || a.title.localeCompare(b.title));
}

const state = {
  projects: [],
  epics: [], // all of them incl. archived, so old labels still resolve
  absences: [], // loaded on demand when the absences view is open
  shareLinks: null, // { share_url?, ical_url? } from /api/share-links, fetched once
  view: { kind: "today" }, // or { kind: "project", id }, { kind: "epic", id }, { kind: "upcoming" }, { kind: "activity" }, { kind: "absences" }
  openCounts: {}, // project_id -> number of open tasks
  epicCounts: {}, // epic_id -> number of open tasks
  activity: { from: null, to: null }, // ISO date range for the activity view
};

// ---------- URL router ----------
// URL scheme:
//   /                          → today (canonicalised to /today via replaceState)
//   /today, /upcoming, /inbox  → standard views
//   /projects/<slug>           → project view (slug resolved against state.projects)
//   /epics/<slug>              → epic view (slug resolved against state.epics)
//   /tasks/<id>                → opens task modal on top of the task's natural background view
// Unknown paths fall back to /today.

function projectBySlug(slug) { return state.projects.find(p => p.slug === slug); }
function epicBySlug(slug) { return state.epics.find(e => e.slug === slug); }

function parsePath(pathname) {
  const trimmed = pathname.replace(/\/+$/, "") || "/";
  if (trimmed === "/" || trimmed === "/today") return { view: { kind: "today" } };
  if (trimmed === "/upcoming") return { view: { kind: "upcoming" } };
  if (trimmed === "/inbox") return { view: { kind: "inbox" } };
  if (trimmed === "/activity") return { view: { kind: "activity" } };
  if (trimmed === "/absences") return { view: { kind: "absences" } };
  const proj = trimmed.match(/^\/projects\/([^\/]+)$/);
  if (proj) return { view: { kind: "project", slug: decodeURIComponent(proj[1]) } };
  const epic = trimmed.match(/^\/epics\/([^\/]+)$/);
  if (epic) return { view: { kind: "epic", slug: decodeURIComponent(epic[1]) } };
  const task = trimmed.match(/^\/tasks\/([^\/]+)$/);
  if (task) return { view: { kind: "today" }, openTaskId: decodeURIComponent(task[1]) };
  return { view: { kind: "today" } };
}

function viewToPath(view) {
  switch (view.kind) {
    case "today":    return "/today";
    case "upcoming": return "/upcoming";
    case "inbox":    return "/inbox";
    case "activity": return "/activity";
    case "absences": return "/absences";
    case "project": {
      const p = projectById(view.id);
      const slug = p && p.slug ? p.slug : view.id;
      return `/projects/${encodeURIComponent(slug)}`;
    }
    case "epic": {
      const e = epicById(view.id);
      const slug = e && e.slug ? e.slug : view.id;
      return `/epics/${encodeURIComponent(slug)}`;
    }
  }
  return "/today";
}

// pushView updates state.view, syncs the URL, and re-renders. Use replace=true
// when canonicalising the URL on first load so the back button still works.
function pushView(view, { replace = false } = {}) {
  state.view = view;
  // Mark the clicked entry before refresh() awaits the network. Otherwise the
  // accent only lands once the fetch resolves, and the item sits in its
  // hover/tap state until then.
  markActiveNav();
  const path = viewToPath(view);
  if (location.pathname !== path) {
    if (replace) history.replaceState({}, "", path);
    else history.pushState({}, "", path);
  }
  refresh();
}

function pushTaskUrl(taskId) {
  const path = `/tasks/${encodeURIComponent(taskId)}`;
  if (location.pathname !== path) history.pushState({}, "", path);
}

function popTaskUrl() {
  if (/^\/tasks\//.test(location.pathname)) history.back();
}

// Guard so popstate handling doesn't re-push history entries when it opens/closes modals programmatically.
let routerSuppressPush = false;

async function handlePopState() {
  const parsed = parsePath(location.pathname);
  const dlg = document.getElementById("task-dialog");
  const modalOpen = dlg && dlg.open;
  routerSuppressPush = true;
  try {
    if (parsed.openTaskId) {
      // Direct task URL: ensure background view is the task's project, then open modal.
      try {
        const t = await api.getTask(parsed.openTaskId);
        applyBackgroundViewForTask(t);
        await refresh();
        openTaskDialogInternal(t);
      } catch (err) {
        // Task gone or fetch failed → redirect to today.
        pushView({ kind: "today" }, { replace: true });
      }
    } else {
      if (modalOpen) dlg.close();
      const view = resolveViewFromPath(parsed.view);
      state.view = view;
      await refresh();
    }
  } finally {
    routerSuppressPush = false;
  }
}

// Resolves a parsed view to a concrete state.view. Project slugs are looked up
// against the cached project list; unknown slugs degrade to today.
function resolveViewFromPath(parsedView) {
  if (parsedView.kind === "project") {
    const p = projectBySlug(parsedView.slug);
    if (!p) return { kind: "today" };
    return { kind: "project", id: p.id };
  }
  if (parsedView.kind === "epic") {
    const e = epicBySlug(parsedView.slug);
    if (!e) return { kind: "today" };
    return { kind: "epic", id: e.id };
  }
  return parsedView;
}

// Picks the natural background view for a task: its project, or inbox for inbox tasks.
function applyBackgroundViewForTask(t) {
  if (t.epic_id && epicById(t.epic_id)) {
    state.view = { kind: "epic", id: t.epic_id };
    return;
  }
  if (t.project_id === "inbox") {
    state.view = { kind: "inbox" };
  } else {
    state.view = { kind: "project", id: t.project_id };
  }
}

// Convert ISO YYYY-MM-DD to German DD.MM.YYYY for display in NL date inputs.
// The backend parser accepts both, so we can present the friendlier form.
function isoToDE(iso) {
  if (!iso) return "";
  const [y, m, d] = iso.split("-");
  return `${d}.${m}.${y}`;
}

// Hooks up an input element for natural-language date entry. On blur (or
// Enter), the text is sent to /api/parse; the parsed date appears as a small
// hint next to the field. Empty input clears the hint.
function wireNLDateInput(inputId) {
  const input = document.getElementById(inputId);
  if (!input) return;
  const hint = document.querySelector(`.date-hint[data-for="${inputId}"]`);
  let lastQuery = "";
  async function update() {
    const text = input.value.trim();
    if (text === lastQuery) return;
    lastQuery = text;
    if (!hint) return;
    if (!text) { hint.textContent = ""; hint.classList.remove("bad"); return; }
    try {
      const res = await api.parse(text);
      if (res.visible_from) {
        const warn = absenceWarningText(res.visible_from);
        hint.innerHTML = "→ " + escapeHTML(formatDate(res.visible_from)) +
          (warn ? ` <span class="absence-warn">${escapeHTML(warn)}</span>` : "");
        hint.classList.remove("bad");
      } else {
        hint.textContent = "?";
        hint.classList.add("bad");
      }
    } catch (err) {
      hint.textContent = "?";
      hint.classList.add("bad");
    }
  }
  input.addEventListener("blur", update);
  input.addEventListener("change", update);
}

function clearDateHint(inputId) {
  const hint = document.querySelector(`.date-hint[data-for="${inputId}"]`);
  if (hint) { hint.textContent = ""; hint.classList.remove("bad"); }
}

function todayISO() {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

function projectName(id) {
  const p = state.projects.find(p => p.id === id);
  return p ? p.name : "—";
}

function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

function priorityClass(p) { return `p${p}`; }

function formatDate(iso) {
  if (!iso) return "";
  const [y, m, d] = iso.split("-").map(Number);
  return `${String(d).padStart(2, "0")}.${String(m).padStart(2, "0")}.${y}`;
}

const WEEKDAY_SHORT = ["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"];

// Compact absence date: "Mo 13.7." — day and month without leading zeros, the
// year only appended (2-digit) when it differs from the current one, e.g.
// "Mo 4.1.27". Fixed weekday table instead of toLocaleDateString so the
// abbreviation is stable across engines.
function formatAbsenceDate(iso) {
  if (!iso) return "";
  const [y, m, d] = iso.split("-").map(Number);
  const wd = WEEKDAY_SHORT[new Date(y, m - 1, d).getDay()];
  const base = `${wd} ${d}.${m}.`;
  return y === new Date().getFullYear() ? base : `${base}${String(y).slice(-2)}`;
}

// Relative date label: "Heute", "Morgen", weekday (within next 6 days),
// then "DD.MM." (current year) or "DD.MM.YYYY" further out.
function smartDateLabel(iso) {
  if (!iso) return "";
  const today = new Date(); today.setHours(0,0,0,0);
  const [y, m, d] = iso.split("-").map(Number);
  const target = new Date(y, m - 1, d);
  const diffDays = Math.round((target - today) / 86400000);
  if (diffDays === 0) return "Heute";
  if (diffDays === 1) return "Morgen";
  if (diffDays === -1) return "Gestern";
  if (diffDays > 1 && diffDays <= 6) {
    return target.toLocaleDateString("de-DE", { weekday: "long" });
  }
  if (diffDays >= -6 && diffDays < -1) {
    return target.toLocaleDateString("de-DE", { weekday: "long" });
  }
  const sameYear = target.getFullYear() === today.getFullYear();
  return sameYear
    ? `${String(d).padStart(2,"0")}.${String(m).padStart(2,"0")}.`
    : `${String(d).padStart(2,"0")}.${String(m).padStart(2,"0")}.${y}`;
}

function dateColorClass(iso) {
  if (!iso) return "";
  const today = new Date(); today.setHours(0,0,0,0);
  const [y, m, d] = iso.split("-").map(Number);
  const target = new Date(y, m - 1, d);
  const diffDays = Math.round((target - today) / 86400000);
  if (diffDays < 0) return "overdue";
  if (diffDays === 0) return "due-today";
  if (diffDays === 1) return "due-tomorrow";
  if (diffDays <= 7) return "due-future";
  return "";
}

// ---------- Inline SVG icons ----------
const ICON = {
  calendar: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18"/><path d="M8 3v4"/><path d="M16 3v4"/></svg>',
  flag:     '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 21V4"/><path d="M5 4h12l-2 4 2 4H5"/></svg>',
  repeat:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/></svg>',
  edit:     '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>',
  trash:    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>',
  hashtag:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/></svg>',
  epic:     '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="m3 12 9 5 9-5"/><path d="m3 17 9 5 9-5"/></svg>',
  comment:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg>',
};

function todayIconSVG() {
  const day = new Date().getDate();
  return `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <rect x="3" y="5" width="18" height="16" rx="2"/>
      <path d="M3 10h18"/>
    </svg>
    <span class="today-icon-day">${day}</span>
  `;
}

function activeProjects() {
  return state.projects.filter(p => !p.archived);
}

// Toggles the .active highlight on the sidebar entries to match state.view.
// Cheap and synchronous on purpose — pushView calls it before the refresh
// round-trip so the accent appears on the same frame as the click.
function markActiveNav() {
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.classList.toggle("active", el.dataset.view === state.view.kind);
  });
  document.querySelectorAll("#project-list .project-item").forEach(el => {
    el.classList.toggle(
      "active",
      state.view.kind === "project" && el.dataset.projectId === state.view.id,
    );
  });
  document.querySelectorAll("#epic-list .epic-item").forEach(el => {
    el.classList.toggle(
      "active",
      state.view.kind === "epic" && el.dataset.epicId === state.view.id,
    );
  });
}

function renderSidebar() {
  const list = document.getElementById("project-list");
  list.innerHTML = "";
  // Inbox has its own top-level nav item, so exclude it from the project list.
  const projects = activeProjects()
    .filter(p => p.id !== "inbox")
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  for (const p of projects) {
    const el = document.createElement("a");
    el.className = "project-item";
    el.href = viewToPath({ kind: "project", id: p.id });
    el.dataset.projectId = p.id;
    if (state.view.kind === "project" && state.view.id === p.id) el.classList.add("active");
    const count = state.openCounts[p.id] || 0;
    const canEdit = p.id !== "inbox";
    el.title = canEdit ? "Klick zum Öffnen, Doppelklick zum Bearbeiten" : "";
    el.innerHTML = `
      <span class="dot" style="background:${escapeHTML(projectColor(p))}"></span>
      <span class="project-name">${escapeHTML(p.name)}</span>
      <span class="project-count">${count > 0 ? count : ""}</span>
    `;
    el.addEventListener("click", (e) => {
      // Cmd/Ctrl-click should open in a new tab — let the browser handle it.
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) return;
      e.preventDefault();
      pushView({ kind: "project", id: p.id });
    });
    if (canEdit) {
      el.addEventListener("dblclick", (e) => {
        e.preventDefault();
        openProjectDialog(p);
      });
    }
    list.appendChild(el);
  }
  renderEpicSidebar();
  markActiveNav();

  // Task edit: include archived ones so existing tasks still resolve their current project.
  const editProjects = [...state.projects].sort((a, b) => Number(a.archived) - Number(b.archived) || a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  fillProjectSelect("td-project", editProjects, true);
}

// Only running epics appear here. A finished one has nothing left to do, and a
// list that only grows would drown the ones that matter — it stays reachable
// through its URL and the API.
function renderEpicSidebar() {
  const list = document.getElementById("epic-list");
  const empty = document.getElementById("epic-empty");
  if (!list) return;
  list.innerHTML = "";
  const epics = activeEpics();
  if (empty) empty.classList.toggle("hidden", epics.length > 0);
  for (const e of epics) {
    const el = document.createElement("a");
    el.className = "epic-item";
    el.href = viewToPath({ kind: "epic", id: e.id });
    el.dataset.epicId = e.id;
    if (state.view.kind === "epic" && state.view.id === e.id) el.classList.add("active");
    const count = state.epicCounts[e.id] || 0;
    el.title = `${projectName(e.project_id)} — Klick zum Öffnen, Doppelklick zum Bearbeiten`;
    // The dot carries the project color: an epic is visibly part of its project.
    el.innerHTML = `
      <span class="dot" style="background:${escapeHTML(projectColor(projectById(e.project_id)))}"></span>
      <span class="project-name">${escapeHTML(e.title)}</span>
      <span class="project-count">${count > 0 ? count : ""}</span>
    `;
    el.addEventListener("click", (ev) => {
      if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.button === 1) return;
      ev.preventDefault();
      pushView({ kind: "epic", id: e.id });
    });
    el.addEventListener("dblclick", (ev) => {
      ev.preventDefault();
      openEpicDialog(e);
    });
    list.appendChild(el);
  }
}

function fillProjectSelect(id, projects, markArchived) {
  const el = document.getElementById(id);
  if (!el) return;
  const prev = el.value;
  el.innerHTML = "";
  for (const p of projects) {
    const opt = document.createElement("option");
    opt.value = p.id;
    opt.textContent = markArchived && p.archived ? `${p.name} (archiviert)` : p.name;
    el.appendChild(opt);
  }
  if (prev) el.value = prev;
}

// fillEpicSelect populates the task modal's epic dropdown with the running
// epics of one project, plus an explicit "no epic" entry.
//
// Two rules pull in opposite directions here, and the project decides between
// them. A finished or archived epic is not in the running list, but a task that
// already hangs on one must not lose the link just by opening the dialog — so
// it is carried over. An epic of a *different* project is not carried over:
// picking another project means leaving the epic (the backend does the same),
// and keeping it selected would send the old epic_id back on save, which wins
// over project_id and quietly undoes the move.
function fillEpicSelect(projectId, selectedId) {
  const el = document.getElementById("td-epic");
  if (!el) return;
  const options = activeEpics(projectId);
  const current = selectedId ? epicById(selectedId) : null;
  if (current && current.project_id === projectId && !options.some(e => e.id === current.id)) {
    options.unshift(current);
  }
  el.innerHTML = "";
  const none = document.createElement("option");
  none.value = "";
  none.textContent = "— kein Epic —";
  el.appendChild(none);
  for (const e of options) {
    const opt = document.createElement("option");
    opt.value = e.id;
    opt.textContent = e.status === "completed" ? `${e.title} (beendet)`
      : e.archived ? `${e.title} (archiviert)`
      : e.title;
    el.appendChild(opt);
  }
  el.value = options.some(e => e.id === selectedId) ? selectedId : "";
}

function viewTitle() {
  switch (state.view.kind) {
    case "today": return "Heute";
    case "upcoming": return "Demnächst";
    case "inbox": return "Inbox";
    case "activity": return "Aktivität";
    case "absences": return "Abwesenheiten";
    case "project": {
      const p = state.projects.find(p => p.id === state.view.id);
      return p ? p.name : "Projekt";
    }
    case "epic": {
      const e = epicById(state.view.id);
      return e ? e.title : "Epic";
    }
  }
  return "";
}

function buildTaskQuery() {
  // The "scope=all" parameter disables the visible_from <= today filter on the API side.
  switch (state.view.kind) {
    case "today":
      return { status: "open", sort: "priority" };
    case "upcoming":
      return { status: "open", scope: "all", sort: "visible_from" };
    case "inbox":
      return { status: "open", scope: "all", project_id: "inbox", sort: "priority" };
    case "project":
      return { status: "open", scope: "all", project_id: state.view.id, sort: "priority" };
    case "epic":
      return { status: "open", scope: "all", epic_id: state.view.id, sort: "priority" };
  }
  return {};
}

// Project ID that quick-add should target by default for the current view.
function defaultProjectForView() {
  if (state.view.kind === "project") return state.view.id;
  if (state.view.kind === "epic") {
    const e = epicById(state.view.id);
    if (e) return e.project_id;
  }
  return "inbox";
}

// Epic a new task should join by default: only the epic view implies one.
function defaultEpicForView() {
  return state.view.kind === "epic" ? state.view.id : "";
}

// Everything above the task list is drawn from cached state *before* the first
// await: title, epic head, which of the three main sections is on screen. The
// data behind it is at most one navigation old and gets corrected further down
// once the fetches land, but the box it sits in has its final height right
// away. Drawing it afterwards moved the list twice per click — once when the
// old head vanished, once when the new one arrived half a second later.
function renderViewChrome() {
  document.getElementById("view-title").textContent = viewTitle();
  document.getElementById("view-subtitle").textContent = new Date().toLocaleDateString("de-DE", { weekday: "long", day: "2-digit", month: "long" });
  // Registermarke am Kopf: die Ansicht "Heute" ist selbst der laufende
  // Bereich und trägt sie deshalb dort, wo sie in den gruppierten Ansichten
  // an der laufenden Woche steht (Styleguide 7.1). Genau eine pro Bildschirm.
  document.querySelector(".main-header").classList.toggle("is-now", state.view.kind === "today");

  switch (state.view.kind) {
    case "absences": setMainSection("absences"); break;
    case "activity": setMainSection("activity"); break;
    default:         setMainSection("tasks"); break;
  }

  // state.epics carries every epic incl. archived ones, so the head of the
  // epic you just clicked is already in hand — no reason to wait for the
  // round trip before reserving its space.
  const cached = state.view.kind === "epic" ? epicById(state.view.id) : null;
  if (cached) renderEpicHead(cached);
  setEpicHeadVisible(!!cached);
}

// Exactly one of the three main sections is on screen at a time: the task list
// (with its quick-add), the activity log, or the absences table. One place
// switches them, so the swap is atomic instead of hiding one section now and
// showing the other after the fetch.
function setMainSection(section) {
  const tasks = section === "tasks";
  const qa = document.querySelector(".quick-add");
  const list = document.getElementById("task-list");
  const empty = document.getElementById("empty-state");
  const activity = document.getElementById("activity-view");
  const absences = document.getElementById("absences-view");
  if (qa) qa.classList.toggle("hidden", !tasks);
  if (list) list.classList.toggle("hidden", !tasks);
  // The empty state belongs to the task list; renderTasks decides whether it
  // comes back.
  if (empty && !tasks) empty.classList.add("hidden");
  if (activity) activity.classList.toggle("hidden", section !== "activity");
  if (absences) absences.classList.toggle("hidden", section !== "absences");
}

// The view the chrome was last drawn for, so refresh() can tell a navigation
// (scroll back to the top) from a re-render after a mutation (stay put).
let lastRenderedView = null;

async function refresh() {
  const viewKey = state.view.kind + ":" + (state.view.id || "");
  const navigated = viewKey !== lastRenderedView;
  lastRenderedView = viewKey;
  renderViewChrome();
  if (navigated) {
    const main = document.querySelector(".main");
    if (main) main.scrollTop = 0;
  }

  if (state.view.kind === "absences") {
    const allOpen = await api.listTasks({ status: "open", scope: "all" });
    recomputeSidebarCounts(allOpen);
    renderSidebar();
    state.absences = await api.listAbsences();
    renderAbsences();
    await renderShareLinks();
    return;
  }

  if (state.view.kind === "activity") {
    // The sidebar counts still come from a quick open-task fetch so the
    // sidebar doesn't go blank while you browse history.
    const allOpen = await api.listTasks({ status: "open", scope: "all" });
    recomputeSidebarCounts(allOpen);
    renderSidebar();
    await renderActivity();
    return;
  }

  const isToday = state.view.kind === "today";
  const isUpcoming = state.view.kind === "upcoming";
  const isEpic = state.view.kind === "epic";
  const isProject = state.view.kind === "project";
  const fetches = [
    api.listTasks(buildTaskQuery()),
    api.listTasks({ status: "open", scope: "all" }),
  ];
  if (isToday) {
    // For the Heute view we additionally show tasks completed today, grayed out.
    fetches.push(api.listTasks({ status: "completed", scope: "all" }));
  }
  if (isUpcoming) {
    // The upcoming view annotates week headers with absences overlapping them.
    fetches.push(api.listAbsences());
  }
  if (isEpic) {
    // An initiative is read by what is behind it as much as by what is left,
    // so the epic view carries its finished tasks along.
    fetches.push(api.listTasks({ status: "completed", scope: "all", epic_id: state.view.id }));
  }
  // Both views that show epics also show their tallies, and those come from the
  // server — reload them rather than guessing from a partial task list.
  if (isEpic || isProject) fetches.push(api.listEpics());
  const results = await Promise.all(fetches);
  const tasks = results[0];
  const allOpen = results[1];
  const completedToday = isToday
    ? (results[2] || []).filter(t => t.completed_at && t.completed_at.slice(0, 10) === todayISO())
    : [];
  if (isUpcoming) state.absences = results[2] || [];
  const completedInEpic = isEpic ? (results[2] || []) : [];
  if (isEpic || isProject) state.epics = results[results.length - 1] || state.epics;

  recomputeSidebarCounts(allOpen);
  renderSidebar();
  if (isEpic) {
    const epic = epicById(state.view.id);
    if (!epic) {
      // Epic vanished (archived elsewhere, stale deep link) — don't strand the user.
      pushView({ kind: "today" }, { replace: true });
      return;
    }
    // Second pass with the numbers straight from the server. Same shape as
    // the cached paint in renderViewChrome(), so nothing moves.
    renderEpicHead(epic);
    setEpicHeadVisible(true);
  }
  renderTasks(tasks, isEpic ? completedInEpic : completedToday);
}

// ---------- Epic head (shown above the quick-add in the epic view) ----------

function setEpicHeadVisible(visible) {
  const el = document.getElementById("epic-head");
  if (el) el.classList.toggle("hidden", !visible);
}

// epicRangeLabel renders the two optional dates as one readable span. An epic
// without any dates gets an empty string, not a placeholder: "ohne Zeitraum"
// takes up a slot to say nothing.
function epicRangeLabel(e) {
  if (e.start_date && e.end_date) return `${formatDate(e.start_date)} – ${formatDate(e.end_date)}`;
  if (e.start_date) return `seit ${formatDate(e.start_date)}`;
  if (e.end_date) return `bis ${formatDate(e.end_date)}`;
  return "";
}

function renderEpicHead(e) {
  const project = projectById(e.project_id);
  const link = document.getElementById("eh-project");
  link.href = viewToPath({ kind: "project", id: e.project_id });
  link.onclick = (ev) => {
    if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.button === 1) return;
    ev.preventDefault();
    pushView({ kind: "project", id: e.project_id });
  };
  document.getElementById("eh-project-dot").style.background = projectColor(project);
  document.getElementById("eh-project-name").textContent = projectName(e.project_id);

  const status = document.getElementById("eh-status");
  const finished = e.status === "completed";
  status.textContent = finished ? "Beendet" : "Laufend";
  status.classList.toggle("done", finished);

  const range = document.getElementById("eh-range");
  range.textContent = epicRangeLabel(e);
  range.classList.toggle("hidden", range.textContent === "");

  const total = (e.open_tasks || 0) + (e.done_tasks || 0);
  document.getElementById("eh-progress").textContent =
    total > 0 ? `${e.done_tasks} / ${total} erledigt` : "noch keine Aufgaben";

  const desc = document.getElementById("eh-desc");
  if (e.description && e.description.trim()) {
    desc.innerHTML = renderMarkdown(e.description);
    desc.classList.remove("hidden");
  } else {
    desc.innerHTML = "";
    desc.classList.add("hidden");
  }

  const toggle = document.getElementById("eh-toggle");
  toggle.textContent = finished ? "Wieder öffnen" : "Beenden";
  // Wired here rather than in boot(), because the handler closes over the epic
  // currently on screen.
  toggle.onclick = async () => {
    hideEpicHeadError();
    try {
      if (finished) await api.reopenEpic(e.id);
      else await api.completeEpic(e.id);
      state.epics = await api.listEpics();
      await refresh();
    } catch (err) {
      showEpicHeadError(err.message);
    }
  };
  document.getElementById("eh-edit").onclick = () => openEpicDialog(e);
  hideEpicHeadError();
}

function showEpicHeadError(msg) {
  const el = document.getElementById("eh-error");
  el.textContent = msg;
  el.classList.remove("hidden");
}

function hideEpicHeadError() {
  document.getElementById("eh-error").classList.add("hidden");
}

function recomputeSidebarCounts(allOpen) {
  const today = todayISO();
  state.openCounts = {};
  state.epicCounts = {};
  let todayCount = 0;
  for (const t of allOpen) {
    state.openCounts[t.project_id] = (state.openCounts[t.project_id] || 0) + 1;
    if (t.epic_id) state.epicCounts[t.epic_id] = (state.epicCounts[t.epic_id] || 0) + 1;
    if (!t.visible_from || t.visible_from <= today) todayCount++;
  }
  const todayCountEl = document.getElementById("today-count");
  if (todayCountEl) todayCountEl.textContent = todayCount > 0 ? todayCount : "";
  const inboxCount = state.openCounts["inbox"] || 0;
  const inboxCountEl = document.getElementById("inbox-count");
  if (inboxCountEl) inboxCountEl.textContent = inboxCount > 0 ? inboxCount : "";
}

// ---------- Absences view ----------

const ABSENCE_TYPE_LABEL = {
  vacation:      "Urlaub",
  holiday:       "Feiertag",
  business_trip: "Dienstreise",
  sick:          "Krankheit",
  other:         "Sonstiges",
};

// Absences (any type) from state that cover a given ISO date, i.e.
// start_date <= iso <= end_date. Same string-compare overlap as the week
// grouping in groupTasksByWeek. Empty/null iso -> no matches.
function absencesCoveringDate(iso) {
  if (!iso) return [];
  return (state.absences || []).filter(a => a.start_date <= iso && iso <= a.end_date);
}

// Warning text for a task date landing inside an absence, or "" if it doesn't.
// Picks the first overlapping absence, e.g. "Fällt in Urlaub (Kreta, bis 19.6.)".
function absenceWarningText(iso) {
  const hits = absencesCoveringDate(iso);
  if (!hits.length) return "";
  const a = hits[0];
  const type = ABSENCE_TYPE_LABEL[a.type] || a.type;
  const label = a.label ? ` (${a.label}, bis ${smartDateLabel(a.end_date)})` : ` (bis ${smartDateLabel(a.end_date)})`;
  return `Fällt in ${type}${label}`;
}

// ---------- Absence date arithmetic ----------
// All of these work on ISO strings ("YYYY-MM-DD") and go through local
// midnight Dates. Distances use Math.round so a DST-shifted 23/25-hour day
// still counts as one.

function isoToDate(iso) {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
}

function dateToISO(dt) {
  const mm = String(dt.getMonth() + 1).padStart(2, "0");
  const dd = String(dt.getDate()).padStart(2, "0");
  return `${dt.getFullYear()}-${mm}-${dd}`;
}

function addDaysISO(iso, n) {
  const dt = isoToDate(iso);
  dt.setDate(dt.getDate() + n);
  return dateToISO(dt);
}

// Days from `from` to `to`, negative when `to` lies before `from`.
function daysBetweenISO(from, to) {
  return Math.round((isoToDate(to) - isoToDate(from)) / 86400000);
}

// Distance as the phrase that saves counting on a calendar. Only ever a rough
// hint — the exact date always sits next to it. Non-positive => "" (the caller
// has its own wording for today).
function relativeDaysDE(days) {
  if (days <= 0) return "";
  if (days === 1) return "morgen";
  if (days < 7) return `in ${days} Tagen`;
  if (days < 14) return "in einer Woche";
  return `in ${Math.floor(days / 7)} Wochen`;
}

// Inclusive length of an absence, as "1 Tag" / "12 Tage".
function absenceDurationDE(a) {
  const days = daysBetweenISO(a.start_date, a.end_date) + 1;
  return days === 1 ? "1 Tag" : `${days} Tage`;
}

// Where the absence stands relative to today: "in 3 Wochen",
// "noch 4 Tage", "endet heute" — or "" for past ones, where the date says it all.
function absenceTimingDE(a, today) {
  if (a.end_date < today) return "";
  if (a.start_date > today) {
    const rel = relativeDaysDE(daysBetweenISO(today, a.start_date));
    return rel === "morgen" ? "morgen" : `${rel}`;
  }
  const left = daysBetweenISO(today, a.end_date);
  if (left === 0) return "endet heute";
  if (left === 1) return "endet morgen";
  return `noch ${left} Tage`;
}

const MONTH_SHORT = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

// ---------- Timeline ----------

// Twelve weeks as a slim horizontal rail: one colored span per absence, thin
// ticks dividing it into calendar weeks, a "today" dot, month labels below.
// It answers "where are the gaps?" at a glance, which the list can't. The
// window starts on the Monday of the current week so it doesn't crawl
// mid-week; spans are clipped to the window, absences fully outside it are
// omitted (they stay in the list). Hovering a list row highlights its span.
//
// !! This rail has a twin: absenceTimelineHTML in internal/httpapi/share.go
// renders the same thing server-side for the public page, down to the class
// names and the magic numbers (window start, pos(), the half-cell offset of
// the today dot, the 9%/94% month-label rules). Go and JS share no code here —
// the share page is a self-contained document that never loads app.js — so a
// change to the geometry has to be made in both places or the two views drift
// apart. What differs on purpose: colors there are the two anonymized states,
// not the four absence types, and the rail carries no tooltips.
const TIMELINE_WEEKS = 12; // == domain.ShareWindowWeeks (internal/domain/absence.go)

function renderAbsenceTimeline(absences, today) {
  const box = document.getElementById("absence-timeline");
  if (!box) return;
  const totalDays = TIMELINE_WEEKS * 7;
  // getDay(): 0 = Sunday, so shift into a Monday-first week.
  const start = addDaysISO(today, -((isoToDate(today).getDay() + 6) % 7));
  const end = addDaysISO(start, totalDays - 1);
  const pos = iso => daysBetweenISO(start, iso) * 100 / totalDays;

  const visible = absences.filter(a => a.start_date <= end && a.end_date >= start);
  if (!visible.length) { box.innerHTML = ""; return; }

  let spans = "";
  for (const a of visible) {
    const s = a.start_date < start ? start : a.start_date;
    const e = a.end_date > end ? end : a.end_date;
    const width = (daysBetweenISO(s, e) + 1) * 100 / totalDays;
    const type = ABSENCE_TYPE_LABEL[a.type] || a.type;
    const range = a.start_date === a.end_date
      ? formatAbsenceDate(a.start_date)
      : `${formatAbsenceDate(a.start_date)} – ${formatAbsenceDate(a.end_date)}`;
    const title = `${type}${a.label ? " · " + a.label : ""} · ${range} · ${absenceDurationDE(a)}`;
    spans += `<div class="tl-span type-${escapeHTML(a.type)}" data-id="${escapeHTML(a.id)}"` +
      ` style="left:${pos(s).toFixed(3)}%;width:${width.toFixed(3)}%"` +
      ` title="${escapeHTML(title)}"></div>`;
  }
  // The marker sits mid-cell on today's column.
  spans += `<div class="tl-today" style="left:${(pos(today) + 100 / (2 * totalDays)).toFixed(3)}%"></div>`;

  // Month labels: the window's own month, then every first of a month. Labels
  // closer than ~9% to the previous one (or hugging the right edge) are dropped
  // rather than overlapped.
  let months = "", last = -100;
  for (let d = start; d <= end; d = addDaysISO(d, 1)) {
    if (d !== start && !d.endsWith("-01")) continue;
    const p = pos(d);
    if (p - last < 9 || p > 94) continue;
    months += `<span style="left:${p.toFixed(3)}%">${MONTH_SHORT[isoToDate(d).getMonth()]}</span>`;
    last = p;
  }

  // ISO week numbers above the rail, centered over their cell — the same weeks
  // the "Demnächst" headers name (isoWeek), but bare: over a calendar strip two
  // digits can't mean anything else, and twelve "KW" would be twelve times the
  // same word. The current week is always the first cell (the window starts on
  // its Monday) and is emphasized to anchor the reading.
  let weeks = "";
  for (let i = 0; i < TIMELINE_WEEKS; i++) {
    weeks += `<span${i === 0 ? ` class="cur"` : ""} style="left:${((i + 0.5) * 100 / TIMELINE_WEEKS).toFixed(3)}%">` +
      `${isoWeek(addDaysISO(start, i * 7)).week}</span>`;
  }

  box.innerHTML = `<section class="timeline">` +
    `<span class="tl-cap">Nächste ${TIMELINE_WEEKS} Wochen</span>` +
    `<div class="tl-weeks">${weeks}</div>` +
    `<div class="tl-track">${spans}</div>` +
    `<div class="tl-months">${months}</div></section>`;
}

// "Teilen" (Share-Seite) und "iCal" (Feed) in der Abwesenheits-Toolbar. Die
// Tokens leben ausschließlich auf dem Server; /api/share-links liefert die
// fertigen URLs und lässt ein Feld weg, wenn der zugehörige Token nicht gesetzt
// ist — dann ist die Route ohnehin 404 und der Button bleibt versteckt.
//
// "Teilen" öffnet die Seite (<a>), "iCal" kopiert nur die Abo-URL (<button>):
// Ein Kalender-Client will den Link, nicht die heruntergeladene .ics-Datei.
async function renderShareLinks() {
  const share = document.getElementById("share-link");
  const ical = document.getElementById("ical-copy-btn");
  if (!share || !ical) return;
  if (!state.shareLinks) {
    // Fehlschlag (alter Server, Netzwerk) => keine Links, aber die Ansicht bleibt nutzbar.
    state.shareLinks = await api.shareLinks().catch(() => ({}));
  }
  const shareURL = state.shareLinks.share_url;
  if (shareURL) share.href = shareURL;
  else share.removeAttribute("href");
  share.classList.toggle("hidden", !shareURL);
  ical.classList.toggle("hidden", !state.shareLinks.ical_url);
}

// Absolute URL des iCal-Feeds — Kalender-Clients brauchen sie mit Host, der
// Pfad aus /api/share-links reicht dort nicht.
function icalFeedURL() {
  const rel = state.shareLinks && state.shareLinks.ical_url;
  return rel ? new URL(rel, location.href).href : "";
}

// In die Zwischenablage schreiben. navigator.clipboard gibt es nur im Secure
// Context (https oder localhost) — im LAN über http fällt es auf das alte
// execCommand("copy") über ein temporäres Textarea zurück.
async function copyToClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  const ok = document.execCommand("copy");
  document.body.removeChild(ta);
  if (!ok) throw new Error("copy failed");
}

// Klick-Feedback direkt am Button: Beschriftung kurz austauschen, danach zurück.
let icalCopyTimer = null;
function flashICalButton(btn, text) {
  clearTimeout(icalCopyTimer);
  btn.textContent = text;
  icalCopyTimer = setTimeout(() => { btn.textContent = "iCal"; }, 1600);
}

function renderAbsences() {
  const list = document.getElementById("absence-list");
  if (!list) return;
  const today = todayISO();
  renderAbsenceTimeline(state.absences, today);
  list.innerHTML = "";
  if (!state.absences.length) {
    list.innerHTML = `<p class="absence-empty">Noch keine Abwesenheiten hinterlegt.</p>`;
    return;
  }
  // Upcoming/current (end today or later) sorted soonest-first; past below,
  // most-recent-first — mirroring the grouped layout of the "Demnächst" view.
  // This is the maintenance view: it shows everything, including absences past
  // the timeline's twelve-week window (the rail above simply omits those).
  const upcoming = state.absences.filter(a => a.end_date >= today).sort((a, b) => a.start_date.localeCompare(b.start_date));
  const past = state.absences.filter(a => a.end_date < today).sort((a, b) => b.start_date.localeCompare(a.start_date));

  // Past absences are read-only (no edit button); the API already limits them
  // to the current year.
  if (upcoming.length) appendAbsenceGroup(list, "Bevorstehend", upcoming, "", today, true);
  if (past.length) appendAbsenceGroup(list, "Vergangen", past, "completed-header", today, false);
}

function appendAbsenceGroup(list, label, items, headerClass, today, editable) {
  const header = document.createElement("div");
  header.className = "group-header" + (headerClass ? " " + headerClass : "");
  header.innerHTML = `<span>${escapeHTML(label)}</span><span class="group-count">${items.length}</span>`;
  list.appendChild(header);
  for (const a of items) list.appendChild(createAbsenceRow(a, today, editable));
}

function createAbsenceRow(a, today, editable) {
  const row = document.createElement("div");
  const active = a.start_date <= today && a.end_date >= today;
  row.className = "absence-row" + (active ? " active" : "");
  row.dataset.id = a.id;
  const range = a.start_date === a.end_date
    ? formatAbsenceDate(a.start_date)
    : `${formatAbsenceDate(a.start_date)} – ${formatAbsenceDate(a.end_date)}`;
  // Keep the grid column for edit even when read-only, so delete stays aligned.
  const editCell = editable
    ? `<button type="button" class="icon-btn absence-edit" title="Bearbeiten">${ICON.edit}</button>`
    : `<span class="absence-edit-placeholder"></span>`;
  // Duration and distance sit under the range: how long it lasts and how far
  // off it is are exactly what the bare dates make you compute yourself.
  const timing = absenceTimingDE(a, today);
  const meta = [absenceDurationDE(a), timing].filter(Boolean).join(" · ");
  // The comment (booking numbers and the like) sits under the label, the way
  // comments sit under a task — free text, so it keeps its line breaks.
  const comment = (a.comment || "").trim();
  row.innerHTML = `
    <span class="absence-type type-${escapeHTML(a.type)}">${escapeHTML(ABSENCE_TYPE_LABEL[a.type] || a.type)}</span>
    <span class="absence-main">
      <span class="absence-label">${escapeHTML(a.label || "")}</span>
      ${comment ? `<span class="absence-comment">${escapeHTML(comment)}</span>` : ""}
    </span>
    <span class="absence-when">
      <span class="absence-range">${escapeHTML(range)}</span>
      <span class="absence-meta">${escapeHTML(meta)}</span>
    </span>
    ${editCell}
    <button type="button" class="icon-btn absence-delete" title="Löschen">${ICON.trash}</button>
  `;
  // Hovering a row lights up its span in the rail above, so the two views read
  // as one thing rather than two lists of dates.
  row.addEventListener("mouseenter", () => highlightTimelineSpan(a.id, true));
  row.addEventListener("mouseleave", () => highlightTimelineSpan(a.id, false));
  if (editable) {
    row.querySelector(".absence-edit").addEventListener("click", () => openAbsenceDialog(a));
  }
  row.querySelector(".absence-delete").addEventListener("click", async () => {
    if (!confirm("Abwesenheit wirklich löschen?")) return;
    await api.deleteAbsence(a.id);
    state.absences = await api.listAbsences();
    renderAbsences();
  });
  return row;
}

function highlightTimelineSpan(id, on) {
  const span = document.querySelector(`#absence-timeline .tl-span[data-id="${CSS.escape(id)}"]`);
  if (span) span.classList.toggle("hl", on);
}

// ---------- Absence dialog (create + edit) ----------

let absenceDialogState = { mode: "create", id: null };

function openAbsenceDialog(a) {
  absenceDialogState = { mode: a ? "edit" : "create", id: a ? a.id : null };
  document.getElementById("ad-title").textContent = a ? "Abwesenheit bearbeiten" : "Neue Abwesenheit";
  document.getElementById("ad-type").value = a ? a.type : "vacation";
  document.getElementById("ad-label").value = a ? (a.label || "") : "";
  document.getElementById("ad-comment").value = a ? (a.comment || "") : "";
  document.getElementById("ad-start").value = a ? a.start_date : todayISO();
  document.getElementById("ad-end").value = a ? a.end_date : todayISO();
  document.getElementById("ad-error").classList.add("hidden");
  document.getElementById("ad-delete").classList.toggle("hidden", !a);
  document.getElementById("absence-dialog").showModal();
  setTimeout(() => document.getElementById("ad-type").focus(), 0);
}

function showAbsenceDialogError(msg) {
  const el = document.getElementById("ad-error");
  el.textContent = msg;
  el.classList.remove("hidden");
}

function wireAbsenceDialog() {
  const dlg = document.getElementById("absence-dialog");
  if (!dlg) return;
  document.getElementById("add-absence-btn").addEventListener("click", () => openAbsenceDialog(null));

  const icalBtn = document.getElementById("ical-copy-btn");
  if (icalBtn) {
    icalBtn.addEventListener("click", async () => {
      const url = icalFeedURL();
      if (!url) return;
      try {
        await copyToClipboard(url);
        flashICalButton(icalBtn, "Kopiert!");
      } catch {
        flashICalButton(icalBtn, "Fehlgeschlagen");
      }
    });
  }

  document.getElementById("ad-cancel").addEventListener("click", () => dlg.close("cancel"));

  document.getElementById("ad-save").addEventListener("click", async (e) => {
    e.preventDefault();
    const type = document.getElementById("ad-type").value;
    const label = document.getElementById("ad-label").value.trim();
    const comment = document.getElementById("ad-comment").value.trim();
    const start = document.getElementById("ad-start").value;
    const end = document.getElementById("ad-end").value;
    if (!start || !end) { showAbsenceDialogError("Start- und Enddatum sind erforderlich."); return; }
    if (end < start) { showAbsenceDialogError("Das Enddatum darf nicht vor dem Startdatum liegen."); return; }
    const body = { type, label, comment, start_date: start, end_date: end };
    try {
      if (absenceDialogState.mode === "edit") {
        await api.patchAbsence(absenceDialogState.id, body);
      } else {
        await api.createAbsence(body);
      }
      dlg.close();
      state.absences = await api.listAbsences();
      renderAbsences();
    } catch (err) {
      showAbsenceDialogError(err.message);
    }
  });

  document.getElementById("ad-delete").addEventListener("click", async (e) => {
    e.preventDefault();
    if (!absenceDialogState.id) return;
    if (!confirm("Abwesenheit wirklich löschen?")) return;
    try {
      await api.deleteAbsence(absenceDialogState.id);
      dlg.close();
      state.absences = await api.listAbsences();
      renderAbsences();
    } catch (err) {
      showAbsenceDialogError(err.message);
    }
  });
}

// ---------- Activity view ----------

const ACTIVITY_LABEL = {
  created:               "erstellt",
  completed:             "abgehakt",
  reopened:              "wieder geöffnet",
  deleted:               "gelöscht",
  visible_from_set:      "Startdatum gesetzt",
  visible_from_cleared:  "Startdatum entfernt",
  visible_from_changed:  "Startdatum geändert",
  deadline_set:          "Deadline gesetzt",
  deadline_cleared:      "Deadline entfernt",
  deadline_changed:      "Deadline geändert",
  project_changed:       "Projekt geändert",
  priority_changed:      "Priorität geändert",
  epic_changed:          "Epic gewechselt",
};

// Das Audit-Log kennt für Epics nur eine Aktion; ob zugeordnet, entfernt oder
// gewechselt wurde, steht in den beiden Werten. Die Beschriftung liest sie aus,
// statt für alle drei Fälle "Epic gewechselt" zu behaupten.
function activityLabelFor(e) {
  if (e.action === "epic_changed") {
    if (!e.old_value) return "Epic zugeordnet";
    if (!e.new_value) return "aus Epic entfernt";
    return "Epic gewechselt";
  }
  return ACTIVITY_LABEL[e.action] || e.action;
}

const ACTIVITY_ICON = {
  created:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  completed: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 12 9 17 20 6"/></svg>',
  reopened:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>',
  deleted:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/></svg>',
  date:      '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18"/></svg>',
  deadline:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 21V4"/><path d="M5 4h12l-2 4 2 4H5"/></svg>',
  project:   '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="9" x2="20" y2="9"/><line x1="4" y1="15" x2="20" y2="15"/><line x1="10" y1="3" x2="8" y2="21"/><line x1="16" y1="3" x2="14" y2="21"/></svg>',
  priority:  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" y1="22" x2="4" y2="15"/></svg>',
  epic:      ICON.epic,
};

function activityIconFor(action) {
  if (action === "created" || action === "completed" || action === "reopened" || action === "deleted") {
    return { svg: ACTIVITY_ICON[action], cls: action };
  }
  if (action.startsWith("visible_from")) return { svg: ACTIVITY_ICON.date, cls: "date" };
  if (action.startsWith("deadline"))     return { svg: ACTIVITY_ICON.deadline, cls: "deadline" };
  if (action === "project_changed")      return { svg: ACTIVITY_ICON.project, cls: "project" };
  if (action === "priority_changed")     return { svg: ACTIVITY_ICON.priority, cls: "priority" };
  if (action === "epic_changed")         return { svg: ACTIVITY_ICON.epic, cls: "project" };
  return { svg: ACTIVITY_ICON.date, cls: "" };
}

function activityDiffHTML(e) {
  // Dates: format human-friendly.
  const isDate = e.action.startsWith("visible_from") || e.action.startsWith("deadline");
  const fmt = (v) => isDate ? (v ? formatDate(v) : "—") : (v || "—");
  if (e.action === "epic_changed") {
    if (!e.old_value) return `<span class="activity-diff"><span class="activity-diff-new">${escapeHTML(e.new_value)}</span></span>`;
    if (!e.new_value) return `<span class="activity-diff"><span class="activity-diff-old">${escapeHTML(e.old_value)}</span></span>`;
  }
  if (e.action === "project_changed") {
    return `<span class="activity-diff"><span class="activity-diff-old">${escapeHTML(e.old_value || "—")}</span><span class="activity-diff-sep">→</span><span class="activity-diff-new">${escapeHTML(e.new_value || "—")}</span></span>`;
  }
  if (e.action.endsWith("_set")) {
    return `<span class="activity-diff"><span class="activity-diff-new">${escapeHTML(fmt(e.new_value))}</span></span>`;
  }
  if (e.action.endsWith("_cleared")) {
    return `<span class="activity-diff"><span class="activity-diff-old">${escapeHTML(fmt(e.old_value))}</span></span>`;
  }
  if (e.action.endsWith("_changed")) {
    return `<span class="activity-diff"><span class="activity-diff-old">${escapeHTML(fmt(e.old_value))}</span><span class="activity-diff-sep">→</span><span class="activity-diff-new">${escapeHTML(fmt(e.new_value))}</span></span>`;
  }
  return "";
}

function ensureActivityRange() {
  if (!state.activity.from || !state.activity.to) {
    const t = todayISO();
    state.activity.from = t;
    state.activity.to = t;
  }
}

function syncActivityInputs() {
  ensureActivityRange();
  document.getElementById("activity-from").value = state.activity.from;
  document.getElementById("activity-to").value = state.activity.to;
}

function shiftActivityDay(deltaDays) {
  ensureActivityRange();
  const shift = (iso) => {
    const [y, m, d] = iso.split("-").map(Number);
    const dt = new Date(y, m - 1, d);
    dt.setDate(dt.getDate() + deltaDays);
    const yyyy = dt.getFullYear();
    const mm = String(dt.getMonth() + 1).padStart(2, "0");
    const dd = String(dt.getDate()).padStart(2, "0");
    return `${yyyy}-${mm}-${dd}`;
  };
  state.activity.from = shift(state.activity.from);
  state.activity.to = shift(state.activity.to);
  syncActivityInputs();
  renderActivity();
}

function formatActivityDayHeader(iso) {
  const [y, m, d] = iso.split("-").map(Number);
  const dt = new Date(y, m - 1, d);
  const label = dt.toLocaleDateString("de-DE", { weekday: "long", day: "2-digit", month: "long", year: "numeric" });
  if (iso === todayISO()) return `Heute · ${label}`;
  return label;
}

async function renderActivity() {
  ensureActivityRange();
  syncActivityInputs();
  const list = document.getElementById("activity-list");
  const empty = document.getElementById("activity-empty");
  list.innerHTML = "";
  let entries;
  try {
    if (state.activity.from === state.activity.to) {
      entries = await api.listAudit({ date: state.activity.from });
    } else {
      entries = await api.listAudit({ from: state.activity.from, to: state.activity.to });
    }
  } catch (err) {
    empty.classList.remove("hidden");
    empty.textContent = "Fehler beim Laden: " + (err.message || err);
    return;
  }
  if (!entries || entries.length === 0) {
    empty.classList.remove("hidden");
    empty.textContent = "Keine Aktivität in diesem Zeitraum.";
    return;
  }
  empty.classList.add("hidden");

  // Group by local calendar day.
  const groups = new Map(); // iso -> entries[]
  for (const e of entries) {
    const dt = new Date(e.at);
    const yyyy = dt.getFullYear();
    const mm = String(dt.getMonth() + 1).padStart(2, "0");
    const dd = String(dt.getDate()).padStart(2, "0");
    const iso = `${yyyy}-${mm}-${dd}`;
    if (!groups.has(iso)) groups.set(iso, []);
    groups.get(iso).push(e);
  }
  // Newest day first.
  const days = [...groups.keys()].sort((a, b) => b.localeCompare(a));
  for (const iso of days) {
    const header = document.createElement("li");
    header.className = "activity-day";
    header.innerHTML = `<span>${escapeHTML(formatActivityDayHeader(iso))}</span><span class="activity-day-count">${groups.get(iso).length}</span>`;
    list.appendChild(header);
    for (const e of groups.get(iso)) {
      list.appendChild(createActivityLI(e));
    }
  }
}

function createActivityLI(e) {
  const li = document.createElement("li");
  li.className = "activity-entry";
  const dt = new Date(e.at);
  const time = dt.toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" });
  const icon = activityIconFor(e.action);
  const label = activityLabelFor(e);
  const diff = activityDiffHTML(e);
  li.innerHTML = `
    <span class="activity-time">${escapeHTML(time)}</span>
    <span class="activity-icon ${icon.cls}">${icon.svg}</span>
    <span class="activity-body">
      <a class="activity-task-link" href="/tasks/${encodeURIComponent(e.task_id)}">${escapeHTML(e.task_title || "(ohne Titel)")}</a>
      <span class="activity-action"> · ${escapeHTML(label)}</span>
      ${diff ? " " + diff : ""}
    </span>
  `;
  const link = li.querySelector(".activity-task-link");
  link.addEventListener("click", async (ev) => {
    if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.button === 1) return;
    ev.preventDefault();
    try {
      const t = await api.getTask(e.task_id);
      openTaskDialog(t);
    } catch (err) {
      alert("Aufgabe nicht mehr verfügbar.");
    }
  });
  return li;
}

function wireActivityControls() {
  const fromI = document.getElementById("activity-from");
  const toI = document.getElementById("activity-to");
  if (!fromI || !toI) return;
  fromI.addEventListener("change", () => {
    if (!fromI.value) return;
    state.activity.from = fromI.value;
    if (state.activity.to < state.activity.from) state.activity.to = state.activity.from;
    syncActivityInputs();
    renderActivity();
  });
  toI.addEventListener("change", () => {
    if (!toI.value) return;
    state.activity.to = toI.value;
    if (state.activity.to < state.activity.from) state.activity.from = state.activity.to;
    syncActivityInputs();
    renderActivity();
  });
  document.getElementById("activity-prev").addEventListener("click", () => shiftActivityDay(-1));
  document.getElementById("activity-next").addEventListener("click", () => shiftActivityDay(1));
  document.getElementById("activity-today").addEventListener("click", () => {
    const t = todayISO();
    state.activity.from = t;
    state.activity.to = t;
    syncActivityInputs();
    renderActivity();
  });
}

function isoWeek(isoDate) {
  // ISO 8601: week containing the year's first Thursday is KW 1.
  const [y, m, d] = isoDate.split("-").map(Number);
  const date = new Date(Date.UTC(y, m - 1, d));
  const dayNum = date.getUTCDay() || 7;
  date.setUTCDate(date.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1));
  const week = Math.ceil(((date - yearStart) / 86400000 + 1) / 7);
  return { year: date.getUTCFullYear(), week };
}

function compareTasks(a, b) {
  // "Demnächst" sorts by date first so the next days are easy to read:
  // visible_from asc (none last), then priority, then deadline, then created_at.
  const av = a.visible_from || "9999-99-99";
  const bv = b.visible_from || "9999-99-99";
  if (av !== bv) return av < bv ? -1 : 1;
  if (a.priority !== b.priority) return a.priority - b.priority;
  const ad = a.deadline || "9999-99-99";
  const bd = b.deadline || "9999-99-99";
  if (ad !== bd) return ad < bd ? -1 : 1;
  return a.created_at < b.created_at ? -1 : a.created_at > b.created_at ? 1 : 0;
}

function isoWeekRange(year, week) {
  // Monday of ISO week N: take Jan 4 (always in week 1), back up to its Monday,
  // then advance N-1 weeks. Sunday = Monday + 6.
  const jan4 = new Date(Date.UTC(year, 0, 4));
  const jan4Day = jan4.getUTCDay() || 7;
  const monday = new Date(jan4);
  monday.setUTCDate(jan4.getUTCDate() - (jan4Day - 1) + (week - 1) * 7);
  const sunday = new Date(monday);
  sunday.setUTCDate(monday.getUTCDate() + 6);
  return { monday, sunday };
}

function formatDayMonth(d) {
  return `${String(d.getUTCDate()).padStart(2, "0")}.${String(d.getUTCMonth() + 1).padStart(2, "0")}.`;
}

function groupTasksByWeek(tasks) {
  const undated = [];
  const buckets = new Map(); // key = `${year}-${week}` -> { year, week, tasks }
  for (const t of tasks) {
    if (!t.visible_from) {
      undated.push(t);
      continue;
    }
    const { year, week } = isoWeek(t.visible_from);
    const key = `${year}-${String(week).padStart(2, "0")}`;
    if (!buckets.has(key)) buckets.set(key, { key, year, week, tasks: [] });
    buckets.get(key).tasks.push(t);
  }
  const weeks = [...buckets.values()].sort((a, b) => a.key.localeCompare(b.key));
  for (const g of weeks) g.tasks.sort(compareTasks);
  undated.sort(compareTasks);
  const groups = [];
  if (undated.length) groups.push({ key: "none", label: "Ohne Datum", range: "", past: false, current: false, tasks: undated });
  const todayStr = todayISO();
  const thisYear = new Date().getFullYear();
  for (const g of weeks) {
    const { monday, sunday } = isoWeekRange(g.year, g.week);
    const range = `${formatDayMonth(monday)}–${formatDayMonth(sunday)}`;
    const label = g.year === thisYear ? `KW ${g.week}` : `KW ${g.week} (${g.year})`;
    const mondayStr = `${monday.getUTCFullYear()}-${String(monday.getUTCMonth()+1).padStart(2,"0")}-${String(monday.getUTCDate()).padStart(2,"0")}`;
    const sundayStr = `${sunday.getUTCFullYear()}-${String(sunday.getUTCMonth()+1).padStart(2,"0")}-${String(sunday.getUTCDate()).padStart(2,"0")}`;
    const past = sundayStr < todayStr;
    const current = !past && mondayStr <= todayStr;
    // Absences (any type) overlapping this calendar week, for the header hint.
    // Chronological: what comes first in the week sits leftmost. The repo hands
    // them over newest-first, so sort explicitly — earlier start wins, an
    // already-running absence therefore leads; ties break on the shorter one.
    const absences = (state.absences || [])
      .filter(a => a.start_date <= sundayStr && a.end_date >= mondayStr)
      .sort((a, b) => a.start_date.localeCompare(b.start_date) || a.end_date.localeCompare(b.end_date));
    groups.push({ key: g.key, label, range, past, current, tasks: g.tasks, absences });
  }
  return groups;
}

function viewIsGrouped() {
  return state.view.kind === "upcoming";
}

// A project view groups its tasks under the project's running epics — but only
// when there are any. A project that does not work with epics keeps the plain
// flat list it always had; no feature should hand out chrome to people who
// aren't using it.
function viewIsGroupedByEpic() {
  return state.view.kind === "project" && activeEpics(state.view.id).length > 0;
}

function groupTasksByEpic(tasks) {
  const epics = activeEpics(state.view.id);
  const buckets = new Map(epics.map(e => [e.id, []]));
  const loose = [];
  for (const t of (tasks || [])) {
    if (t.epic_id && buckets.has(t.epic_id)) buckets.get(t.epic_id).push(t);
    else loose.push(t);
  }
  const groups = [];
  for (const e of epics) {
    const items = buckets.get(e.id);
    items.sort(compareTasks);
    // Empty running epics stay visible: "nothing to do here right now" is an
    // answer, and hiding the epic would make it look like it doesn't exist.
    groups.push({ key: e.id, epic: e, tasks: items });
  }
  loose.sort(compareTasks);
  if (loose.length) groups.push({ key: "none", epic: null, tasks: loose });
  return groups;
}

// createGroupHeader builds the Rubrikzeile (Styleguide 7.1) for one epic group.
// No register mark is ever set here: a project view can hold several epics, and
// the mark only means something while there is exactly one per screen.
function createEpicGroupHeader(group) {
  const header = document.createElement("li");
  if (!group.epic) {
    header.className = "group-header";
    header.innerHTML = `
      <span>Ohne Epic</span>
      <span class="group-count">${group.tasks.length}</span>
    `;
    return header;
  }
  const e = group.epic;
  const total = (e.open_tasks || 0) + (e.done_tasks || 0);
  const range = epicRangeLabel(e);
  header.className = "group-header epic";
  header.innerHTML = `
    <span class="group-epic-title">${escapeHTML(e.title)}</span>
    ${range ? `<span class="group-range">${escapeHTML(range)}</span>` : ""}
    <span class="group-count" title="${e.done_tasks || 0} von ${total} erledigt">${e.done_tasks || 0}/${total}</span>
  `;
  header.title = "Doppelklick zum Bearbeiten";
  header.addEventListener("click", (ev) => {
    if (ev.metaKey || ev.ctrlKey || ev.shiftKey) return;
    pushView({ kind: "epic", id: e.id });
  });
  header.addEventListener("dblclick", (ev) => {
    ev.preventDefault();
    openEpicDialog(e);
  });
  return header;
}

function createTaskLI(t, today) {
  const li = document.createElement("li");
  li.className = "task";
  const dateParts = [];
  if (t.visible_from) {
    const cls = dateColorClass(t.visible_from);
    dateParts.push(`<span class="date-item ${cls}">${ICON.calendar}<span>${escapeHTML(smartDateLabel(t.visible_from))}</span></span>`);
  }
  if (t.deadline) {
    const overdue = t.deadline < today;
    dateParts.push(`<span class="date-item ${overdue ? "overdue" : "deadline"}">${ICON.flag}<span>${escapeHTML(smartDateLabel(t.deadline))}</span></span>`);
  }
  if (t.recurrence_label) {
    dateParts.push(`<span class="date-item recur-label">${ICON.repeat}<span>${escapeHTML(t.recurrence_label)}</span></span>`);
  }
  if (t.comment_count) {
    const label = t.comment_count === 1 ? "1 Kommentar" : `${t.comment_count} Kommentare`;
    dateParts.push(`<span class="date-item comment-count" title="${escapeHTML(label)}">${ICON.comment}<span>${t.comment_count}</span></span>`);
  }
  // In the project and epic views the epic is already the heading above the
  // task, so naming it again on every row would only be noise.
  if (t.epic_title && state.view.kind !== "epic" && state.view.kind !== "project") {
    dateParts.push(`<span class="date-item epic-label" title="Epic: ${escapeHTML(t.epic_title)}">${ICON.epic}<span>${escapeHTML(t.epic_title)}</span></span>`);
  }
  const isCompleted = t.status === "completed";
  if (isCompleted) li.classList.add("completed");
  li.innerHTML = `
    <div class="checkbox ${priorityClass(t.priority)}${isCompleted ? " checked" : ""}" title="${isCompleted ? "Wieder öffnen" : "Erledigen"}"></div>
    <div class="body">
      <div class="title">${escapeHTML(t.title)}</div>
      ${t.description && t.description.trim() ? `<div class="desc">${renderDescPreview(t.description)}</div>` : ""}
      ${dateParts.length ? `<div class="dates">${dateParts.join('<span class="date-sep">·</span>')}</div>` : ""}
    </div>
    ${state.view.kind === "project" || state.view.kind === "epic" ? "" : `
    <div class="task-project" title="${escapeHTML(projectName(t.project_id))}">
      <span>${escapeHTML(projectName(t.project_id))}</span>
      <span class="task-project-dot" style="background:${escapeHTML(projectColor(projectById(t.project_id)))}"></span>
    </div>`}
  `;
  if (isCompleted) {
    li.addEventListener("click", async () => {
      await api.reopenTask(t.id);
      refresh();
    });
  } else {
    li.querySelector(".checkbox").addEventListener("click", async (e) => {
      e.stopPropagation();
      await api.completeTask(t.id);
      refresh();
    });
    li.addEventListener("click", () => openTaskDialog(t));
  }
  return li;
}

function renderTasks(tasks, completedToday = []) {
  const list = document.getElementById("task-list");
  const empty = document.getElementById("empty-state");
  list.innerHTML = "";
  const noOpen = !tasks || tasks.length === 0;
  // A project that runs epics still has its headings to show, even with no
  // open task left under them.
  if (noOpen && completedToday.length === 0 && !viewIsGroupedByEpic()) {
    empty.classList.remove("hidden");
    return;
  }
  empty.classList.add("hidden");
  const today = todayISO();

  if (viewIsGroupedByEpic()) {
    for (const group of groupTasksByEpic(tasks)) {
      list.appendChild(createEpicGroupHeader(group));
      for (const t of group.tasks) list.appendChild(createTaskLI(t, today));
    }
  } else if (viewIsGrouped()) {
    for (const group of groupTasksByWeek(tasks)) {
      const header = document.createElement("li");
      header.className = "group-header" + (group.past ? " past" : "") + (group.current ? " current" : "");
      const absBadges = (group.absences || []).map(a => {
        const typeLabel = ABSENCE_TYPE_LABEL[a.type] || a.type;
        // The badge shows the label only ("Kreta"); the type is carried by the
        // badge color, and spelled out in the tooltip.
        const text = a.label || typeLabel;
        const title = a.label ? `${typeLabel} (${a.label})` : typeLabel;
        return `<span class="group-absence type-${escapeHTML(a.type)}" title="${escapeHTML(title)}">${escapeHTML(text)}</span>`;
      }).join("");
      header.innerHTML = `
        <span>${escapeHTML(group.label)}</span>
        ${group.range ? `<span class="group-range">${escapeHTML(group.range)}</span>` : ""}
        ${absBadges ? `<span class="group-absences">${absBadges}</span>` : ""}
        <span class="group-count">${group.tasks.length}</span>
      `;
      list.appendChild(header);
      for (const t of group.tasks) list.appendChild(createTaskLI(t, today));
    }
  } else {
    for (const t of (tasks || [])) list.appendChild(createTaskLI(t, today));
  }

  if (completedToday.length > 0) {
    // Newest completion first.
    const sorted = [...completedToday].sort((a, b) => (b.completed_at || "").localeCompare(a.completed_at || ""));
    const header = document.createElement("li");
    header.className = "group-header completed-header";
    header.innerHTML = `
      <span>Erledigt</span>
      <span class="group-count">${sorted.length}</span>
    `;
    list.appendChild(header);
    for (const t of sorted) list.appendChild(createTaskLI(t, today));
  }
}

// ---------- Quick add ----------

// Inline tokens in the quick-add title input: @date, #project, !!priority.
// The input keeps its plain string value; a same-styled overlay div behind it
// renders colored marks for matched ranges.
const qaInline = {
  date: null,            // {start, end, iso, text}
  dateUnresolved: null,  // {start, end}  -- @ present but parser returned nothing
  project: null,         // {start, end, text, id, name}  -- id set only on exact match
  priority: null,        // {start, end, value}
  parseCache: new Map(), // phrase (lowercased) -> iso|null
  parseGen: 0,           // request token so late /api/parse responses don't clobber newer state
  menu: { visible: false, items: [], activeIdx: 0 },
};

function findAtPhrase(text) {
  const i = text.indexOf("@");
  if (i < 0) return null;
  let end = text.length;
  for (let j = i + 1; j < text.length; j++) {
    if (text[j] === "@" || text[j] === "#") { end = j; break; }
    if (text[j] === "!" && text[j + 1] === "!") { end = j; break; }
  }
  return { atIndex: i, end, phrase: text.slice(i + 1, end) };
}

// Try the longest prefix first, then drop the trailing word until something
// parses. Returns {iso, matchedLen} where matchedLen counts characters in the
// original phrase (so the highlight ends right after the matched word).
async function parseShrinking(phrase, generation) {
  let attempt = phrase.replace(/\s+$/, "");
  while (attempt.length > 0) {
    const key = attempt.toLowerCase();
    let iso;
    if (qaInline.parseCache.has(key)) {
      iso = qaInline.parseCache.get(key);
    } else {
      try {
        const res = await api.parse(attempt);
        iso = res.visible_from || null;
      } catch (e) {
        iso = null;
      }
      if (generation !== qaInline.parseGen) return null;
      qaInline.parseCache.set(key, iso);
    }
    if (iso) return { iso, matchedLen: attempt.length };
    const lastSpace = attempt.lastIndexOf(" ");
    if (lastSpace < 0) break;
    attempt = attempt.slice(0, lastSpace).replace(/\s+$/, "");
  }
  return null;
}

async function processQuickAddInput() {
  const input = document.getElementById("quick-add-input");
  const text = input.value;
  const generation = ++qaInline.parseGen;

  // Priority: !!1..!!4
  const prioMatch = /!!([1-4])(?!\d)/.exec(text);
  qaInline.priority = prioMatch
    ? { start: prioMatch.index, end: prioMatch.index + prioMatch[0].length, value: parseInt(prioMatch[1], 10) }
    : null;

  // Project: #<word>, non-breaking-space allowed for multi-word names that we
  // injected ourselves after a menu selection.
  const projMatch = /#([^\s@#!]+(?: [^\s@#!]+)*)/u.exec(text);
  if (projMatch) {
    const raw = projMatch[1];
    const needle = raw.replace(/ /g, " ").toLowerCase();
    const exact = activeProjects().find(p => p.name.toLowerCase() === needle);
    qaInline.project = {
      start: projMatch.index,
      end: projMatch.index + projMatch[0].length,
      text: raw,
      id: exact ? exact.id : null,
      name: exact ? exact.name : null,
    };
  } else {
    qaInline.project = null;
  }

  // Date: @<phrase> with longest-match shrinking
  qaInline.date = null;
  qaInline.dateUnresolved = null;
  const at = findAtPhrase(text);
  if (at && at.phrase.trim()) {
    const phraseTrim = at.phrase.replace(/\s+$/, "");
    const matched = await parseShrinking(phraseTrim, generation);
    if (generation !== qaInline.parseGen) return; // a newer keystroke is in flight
    if (matched) {
      qaInline.date = {
        start: at.atIndex,
        end: at.atIndex + 1 + matched.matchedLen,
        iso: matched.iso,
        text: phraseTrim.slice(0, matched.matchedLen),
      };
    } else {
      qaInline.dateUnresolved = { start: at.atIndex, end: at.atIndex + 1 + phraseTrim.length };
    }
  }

  renderQuickAddOverlay();
  renderQuickAddAbsenceHint();
  updateProjectMenu();
}

// Show a non-blocking warning under the quick-add when the resolved @date lands
// inside an absence. Cleared when there is no date or no overlap.
function renderQuickAddAbsenceHint() {
  const hint = document.getElementById("qa-absence-hint");
  if (!hint) return;
  const text = qaInline.date ? absenceWarningText(qaInline.date.iso) : "";
  hint.textContent = text;
  hint.classList.toggle("hidden", !text);
}

function renderQuickAddOverlay() {
  const input = document.getElementById("quick-add-input");
  const overlay = document.getElementById("qa-overlay");
  const text = input.value;
  const ranges = [];
  if (qaInline.date) ranges.push({ start: qaInline.date.start, end: qaInline.date.end, cls: "qa-mark-date" });
  if (qaInline.dateUnresolved) ranges.push({ start: qaInline.dateUnresolved.start, end: qaInline.dateUnresolved.end, cls: "qa-mark-unresolved" });
  if (qaInline.project && qaInline.project.id) ranges.push({ start: qaInline.project.start, end: qaInline.project.end, cls: "qa-mark-project" });
  if (qaInline.priority) ranges.push({ start: qaInline.priority.start, end: qaInline.priority.end, cls: `qa-mark-prio${qaInline.priority.value}` });
  ranges.sort((a, b) => a.start - b.start);

  let html = "";
  let cursor = 0;
  for (const r of ranges) {
    if (r.start < cursor) continue; // shouldn't happen, but stay defensive
    html += escapeHTML(text.slice(cursor, r.start));
    html += `<span class="qa-mark ${r.cls}">${escapeHTML(text.slice(r.start, r.end))}</span>`;
    cursor = r.end;
  }
  html += escapeHTML(text.slice(cursor));
  overlay.innerHTML = html || "&#8203;";
  syncQuickAddScroll();
}

// Sobald der Text breiter ist als das Feld, scrollt das <input> mit dem Cursor
// mit -- das Overlay tut das nicht von allein und wuerde am Zeilenanfang
// haengenbleiben. Ohne diesen Abgleich steht der Cursor irgendwo im Text.
function syncQuickAddScroll() {
  const input = document.getElementById("quick-add-input");
  const overlay = document.getElementById("qa-overlay");
  if (!input || !overlay) return;
  overlay.scrollLeft = input.scrollLeft;
}

function updateProjectMenu() {
  const menu = document.getElementById("qa-project-menu");
  // Only show when the user is mid-typing an unresolved #token.
  if (!qaInline.project || qaInline.project.id) {
    closeProjectMenu();
    return;
  }
  const needle = qaInline.project.text.replace(/ /g, " ").toLowerCase();
  const items = activeProjects()
    .filter(p => p.name.toLowerCase().includes(needle))
    .sort((a, b) => {
      // Prefer prefix matches, then alphabetical.
      const ap = a.name.toLowerCase().startsWith(needle) ? 0 : 1;
      const bp = b.name.toLowerCase().startsWith(needle) ? 0 : 1;
      if (ap !== bp) return ap - bp;
      return a.name.localeCompare(b.name);
    });
  qaInline.menu.items = items;
  qaInline.menu.activeIdx = 0;
  if (items.length === 0) {
    menu.innerHTML = `<div class="qa-project-empty">Keine Projekte für „${escapeHTML(needle)}"</div>`;
    menu.classList.remove("hidden");
    qaInline.menu.visible = true;
    return;
  }
  menu.innerHTML = items.map((p, i) => `
    <div class="qa-project-item ${i === 0 ? "active" : ""}" data-idx="${i}">
      <span class="dot" style="background:${escapeHTML(projectColor(p))}"></span>
      <span>${escapeHTML(p.name)}</span>
    </div>
  `).join("");
  menu.classList.remove("hidden");
  qaInline.menu.visible = true;
  menu.querySelectorAll(".qa-project-item").forEach(el => {
    el.addEventListener("mousedown", (e) => {
      e.preventDefault();
      selectProjectFromMenu(parseInt(el.dataset.idx, 10));
    });
  });
}

function closeProjectMenu() {
  const menu = document.getElementById("qa-project-menu");
  menu.classList.add("hidden");
  qaInline.menu.visible = false;
  qaInline.menu.items = [];
}

function highlightActiveMenuItem() {
  const menu = document.getElementById("qa-project-menu");
  menu.querySelectorAll(".qa-project-item").forEach((el, i) => {
    el.classList.toggle("active", i === qaInline.menu.activeIdx);
    if (i === qaInline.menu.activeIdx) el.scrollIntoView({ block: "nearest" });
  });
}

function selectProjectFromMenu(idx) {
  const p = qaInline.menu.items[idx];
  if (!p || !qaInline.project) return;
  const input = document.getElementById("quick-add-input");
  const text = input.value;
  // Use non-breaking spaces so multi-word names stay one token under our regex.
  const nbName = p.name.replace(/ /g, " ");
  const before = text.slice(0, qaInline.project.start);
  const after = text.slice(qaInline.project.end);
  const replacement = "#" + nbName;
  input.value = before + replacement + after;
  const newCursor = before.length + replacement.length;
  input.setSelectionRange(newCursor, newCursor);
  closeProjectMenu();
  input.focus();
  processQuickAddInput();
}

function wireQuickAdd() {
  const form = document.getElementById("quick-add-form");
  const input = document.getElementById("quick-add-input");

  input.addEventListener("input", processQuickAddInput);
  // Cursorbewegungen ohne Texteingabe (Pfeiltasten, Klick, Auswahl) scrollen
  // das Feld ebenfalls; das Overlay muss jedes Mal nachziehen.
  input.addEventListener("scroll", syncQuickAddScroll);
  input.addEventListener("keyup", syncQuickAddScroll);
  input.addEventListener("click", syncQuickAddScroll);
  input.addEventListener("keydown", (e) => {
    if (qaInline.menu.visible && qaInline.menu.items.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        qaInline.menu.activeIdx = (qaInline.menu.activeIdx + 1) % qaInline.menu.items.length;
        highlightActiveMenuItem();
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        qaInline.menu.activeIdx = (qaInline.menu.activeIdx - 1 + qaInline.menu.items.length) % qaInline.menu.items.length;
        highlightActiveMenuItem();
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        selectProjectFromMenu(qaInline.menu.activeIdx);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        closeProjectMenu();
        return;
      }
    }
  });
  // Closing the menu on blur is wrapped in a small timeout so a mousedown on a
  // menu item still registers before the input loses focus.
  input.addEventListener("blur", () => setTimeout(closeProjectMenu, 120));

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    let title = input.value;
    let projectId = null, priority = 0, visibleFrom = null;

    const ranges = [];
    if (qaInline.priority) {
      priority = qaInline.priority.value;
      ranges.push({ start: qaInline.priority.start, end: qaInline.priority.end });
    }
    if (qaInline.project && qaInline.project.id) {
      projectId = qaInline.project.id;
      ranges.push({ start: qaInline.project.start, end: qaInline.project.end });
    }
    if (qaInline.date) {
      visibleFrom = qaInline.date.iso;
      ranges.push({ start: qaInline.date.start, end: qaInline.date.end });
    }
    ranges.sort((a, b) => b.start - a.start);
    for (const r of ranges) title = title.slice(0, r.start) + title.slice(r.end);
    // Collapse whitespace left behind by stripped tokens.
    title = title.replace(/\s+/g, " ").trim();
    if (!title) return;

    // An explicit #project token beats the view's epic: the user just said
    // where the task should go.
    const epicId = projectId ? "" : defaultEpicForView();
    const body = {
      title,
      epic_id: epicId,
      project_id: projectId || defaultProjectForView(),
      priority: priority || 4,
      visible_from: visibleFrom || "",
    };
    try {
      await api.createTask(body);
    } catch (err) {
      alert(err.message || "Fehler beim Speichern");
      return;
    }
    form.reset();
    qaInline.date = qaInline.dateUnresolved = qaInline.project = qaInline.priority = null;
    closeProjectMenu();
    renderQuickAddOverlay();
    renderQuickAddAbsenceHint();
    refresh();
  });
}

// ---------- Edit dialog ----------

// Render the breadcrumb (project pill) at the top of the modal.
function renderTaskDialogBreadcrumb(projectId) {
  const p = projectById(projectId);
  const dot = document.getElementById("td-breadcrumb-dot");
  const name = document.getElementById("td-breadcrumb-name");
  if (dot) dot.style.background = projectColor(p);
  if (name) name.textContent = p ? p.name : "—";
}

// Apply the current priority's color to the modal checkbox.
function updateTaskDialogCheckbox(priority) {
  const cb = document.getElementById("td-checkbox");
  if (!cb) return;
  cb.classList.remove("p1", "p2", "p3", "p4");
  cb.classList.add(priorityClass(priority || 4));
}

// Sync the priority button group with the current value.
function updateTaskDialogPriorityButtons(priority) {
  const p = String(priority || 4);
  document.querySelectorAll("#td-priority-buttons .td-prio").forEach(btn => {
    btn.classList.toggle("selected", btn.dataset.priority === p);
  });
}

// Render user-supplied Markdown to *sanitized* HTML. marked converts Markdown to
// HTML but does NOT strip embedded raw HTML — the `sanitize` option was removed
// in marked v8+, so `<img src=x onerror=…>` in a description/comment would pass
// straight through. Every rendered value MUST therefore go through DOMPurify
// before it reaches innerHTML. Fail-safe: on any error (incl. DOMPurify not
// loaded) we return escaped plain text rather than unsanitized HTML.
function renderMarkdown(md, { inline = false } = {}) {
  const text = (md || "").trim();
  if (!text) return "";
  try {
    const raw = inline
      ? window.marked.parseInline(text, { gfm: true })
      : window.marked.parse(text, { breaks: true, gfm: true });
    return window.DOMPurify.sanitize(raw);
  } catch (e) {
    return escapeHTML(text);
  }
}

function renderDescPreview(md) {
  return renderMarkdown((md || "").replace(/\s+/g, " "), { inline: true });
}

// Render the description as sanitized HTML, or show a placeholder if empty.
function renderTaskDescription(md) {
  const el = document.getElementById("td-description-rendered");
  if (!el) return;
  const text = (md || "").trim();
  if (!text) {
    el.classList.add("empty");
    el.textContent = "Beschreibung hinzufügen…";
    return;
  }
  el.classList.remove("empty");
  el.innerHTML = renderMarkdown(text);
}

// Show the textarea editor for the description (hides the rendered preview).
function showDescriptionEditor() {
  const rendered = document.getElementById("td-description-rendered");
  const input = document.getElementById("td-description");
  rendered.classList.add("hidden");
  input.classList.remove("hidden");
  // Grow to roughly fit content.
  input.style.height = "auto";
  input.style.height = Math.max(80, input.scrollHeight) + "px";
  input.focus();
}

// Hide the textarea, re-render the markdown.
function hideDescriptionEditor() {
  const rendered = document.getElementById("td-description-rendered");
  const input = document.getElementById("td-description");
  input.classList.add("hidden");
  rendered.classList.remove("hidden");
  renderTaskDescription(input.value);
}

// Auto-grow the title textarea to fit its content.
function autoGrow(el) {
  el.style.height = "auto";
  el.style.height = el.scrollHeight + "px";
}

// ---------- Task comments ----------

// Human-readable timestamp for a comment, in the user's locale.
function formatCommentTime(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  return d.toLocaleString("de-DE", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

// Render a comment's Markdown body to sanitized HTML (see renderMarkdown).
function renderCommentBody(md) {
  return renderMarkdown(md);
}

// Render the comment list (server returns newest first).
function renderComments(comments) {
  const list = document.getElementById("td-comment-list");
  list.innerHTML = "";
  for (const c of (comments || [])) {
    const li = document.createElement("li");
    li.className = "td-comment";
    li.innerHTML = `
      <div class="td-comment-meta">${escapeHTML(formatCommentTime(c.created_at))}</div>
      <div class="td-comment-body">${renderCommentBody(c.body)}</div>
    `;
    list.appendChild(li);
  }
}

// Fetch + render a task's comments. Best-effort: failures just leave it empty.
async function loadComments(taskId) {
  const list = document.getElementById("td-comment-list");
  list.innerHTML = "";
  try {
    renderComments(await api.listComments(taskId));
  } catch (e) {
    /* ignore */
  }
}

// openTaskDialogInternal opens the modal without touching browser history.
// Used by popstate/boot when the URL itself caused the modal to open.
function openTaskDialogInternal(t) {
  const dlg = document.getElementById("task-dialog");
  dlg.dataset.id = t.id;
  const titleEl = document.getElementById("td-title");
  titleEl.value = t.title;
  document.getElementById("td-description").value = t.description || "";
  document.getElementById("td-project").value = t.project_id;
  fillEpicSelect(t.project_id, t.epic_id || "");
  document.getElementById("td-priority").value = String(t.priority);
  document.getElementById("td-visible-from").value = t.visible_from || "";
  syncVisibleFromUI();
  document.getElementById("td-deadline").value = isoToDE(t.deadline);
  clearDateHint("td-deadline");
  setRecurrenceControls("td-recurrence", "td-recur-custom-wrap", "td-recur-custom", t.recurrence_rule || "");
  setRecurrenceMode(t.recurrence_mode || RECUR_FIXED);
  setRecurrenceAnchor(t.recurrence_anchor || t.visible_from || "");
  syncRecurrenceModeUI();
  document.getElementById("td-error").classList.add("hidden");
  renderTaskDialogBreadcrumb(t.project_id);
  updateTaskDialogCheckbox(t.priority);
  updateTaskDialogPriorityButtons(t.priority);
  renderTaskDescription(t.description || "");
  // Make sure the editor is hidden whenever we (re)open the modal.
  document.getElementById("td-description").classList.add("hidden");
  document.getElementById("td-description-rendered").classList.remove("hidden");
  // Comments: reset the input and (re)load the list for this task.
  document.getElementById("td-comment-input").value = "";
  document.getElementById("td-comment-error").classList.add("hidden");
  loadComments(t.id);
  // Reset the mobile "more details" collapse so every opening starts compact.
  const layout = dlg.querySelector(".td-layout");
  if (layout) layout.classList.remove("show-extra");
  const moreBtn = document.getElementById("td-more-toggle");
  if (moreBtn) {
    moreBtn.textContent = "Weitere Details";
    moreBtn.setAttribute("aria-expanded", "false");
  }
  dlg.showModal();
  // Title textarea height needs the dialog to be visible to measure correctly.
  requestAnimationFrame(() => autoGrow(titleEl));
}

// openTaskDialog is the user-facing entry point. It pushes a history entry so
// the URL reflects the open modal (and back/share works).
function openTaskDialog(t) {
  if (!routerSuppressPush) pushTaskUrl(t.id);
  openTaskDialogInternal(t);
}

// Returns the ISO date for the quick-reschedule presets, based on today.
function quickRescheduleISO(kind) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  if (kind === "today") {
    // no-op: d is already today at midnight
  } else if (kind === "tomorrow") {
    d.setDate(d.getDate() + 1);
  } else if (kind === "weekend") {
    // Next Saturday (or today if today is Saturday and you want today? — we go
    // strictly to the upcoming Saturday).
    const dow = d.getDay(); // 0=Sun..6=Sat
    const add = ((6 - dow) + 7) % 7 || 7;
    d.setDate(d.getDate() + add);
  } else {
    return "";
  }
  return toISODate(d);
}

// Returns the ISO date of the next upcoming occurrence of a weekday, where
// targetDow follows Date.getDay() (0=Sun..6=Sat). "Upcoming" is strict: if
// today already is that weekday, we jump a full week ahead (mirroring the
// behaviour expected for the Mo–Fr quick buttons).
function nextWeekdayISO(targetDow) {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  const add = ((targetDow - d.getDay()) + 7) % 7 || 7;
  d.setDate(d.getDate() + add);
  return toISODate(d);
}

// Formats a Date as YYYY-MM-DD in local time.
function toISODate(d) {
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

// Reflect the dialog's visible_from (an ISO date input) into everything that
// hangs off it: the warning span that fires when the date lands inside an
// absence, and the Mo–Fr buttons, of which the one matching the date is
// highlighted (their click behaviour is unaffected — they still jump to the
// *next* occurrence of that weekday).
function syncVisibleFromUI() {
  const value = document.getElementById("td-visible-from").value || "";

  const el = document.getElementById("td-visible-from-absence");
  if (el) {
    const text = absenceWarningText(value);
    el.textContent = text;
    el.classList.toggle("hidden", !text);
  }

  // Parsed with an explicit time so it stays local — new Date("YYYY-MM-DD")
  // is UTC and would shift the weekday west of Greenwich. getDay() is
  // 0=Sun..6=Sat, the same numbering as data-weekday.
  const dow = /^\d{4}-\d{2}-\d{2}$/.test(value) ? new Date(value + "T00:00:00").getDay() : -1;
  document.querySelectorAll(".td-weekday").forEach(btn => {
    btn.classList.toggle("is-active", parseInt(btn.dataset.weekday, 10) === dow);
  });
}

function wireTaskDialog() {
  const dlg = document.getElementById("task-dialog");
  dlg.addEventListener("close", () => {
    // Sync URL back to background view whenever the modal closes (cancel, save,
    // delete, ESC). Skip when popstate already drove the close.
    if (!routerSuppressPush) popTaskUrl();
  });
  document.getElementById("td-cancel").addEventListener("click", () => dlg.close("cancel"));
  document.getElementById("td-close").addEventListener("click", () => dlg.close("cancel"));

  const moreBtn = document.getElementById("td-more-toggle");
  if (moreBtn) {
    moreBtn.addEventListener("click", () => {
      const layout = dlg.querySelector(".td-layout");
      const expanded = layout.classList.toggle("show-extra");
      moreBtn.textContent = expanded ? "Weniger anzeigen" : "Weitere Details";
      moreBtn.setAttribute("aria-expanded", expanded ? "true" : "false");
    });
  }

  document.getElementById("td-save").addEventListener("click", async (e) => {
    e.preventDefault();
    const id = dlg.dataset.id;
    const body = {
      title: document.getElementById("td-title").value.trim(),
      description: document.getElementById("td-description").value,
      project_id: document.getElementById("td-project").value,
      epic_id: document.getElementById("td-epic").value,
      priority: parseInt(document.getElementById("td-priority").value, 10),
      visible_from: document.getElementById("td-visible-from").value || "",
      deadline: document.getElementById("td-deadline").value || "",
      recurrence_rule: readRecurrence("td-recurrence", "td-recur-custom"),
      recurrence_mode: currentRecurrenceMode(),
      recurrence_anchor: currentRecurrenceAnchor(),
    };
    try {
      await api.patchTask(id, body);
    } catch (err) {
      const el = document.getElementById("td-error");
      el.textContent = err.message || "Fehler beim Speichern";
      el.classList.remove("hidden");
      return;
    }
    dlg.close();
    refresh();
  });

  // Complete the task right from the modal.
  document.getElementById("td-checkbox").addEventListener("click", async () => {
    const id = dlg.dataset.id;
    if (!id) return;
    try {
      await api.completeTask(id);
    } catch (err) {
      const el = document.getElementById("td-error");
      el.textContent = err.message || "Fehler beim Erledigen";
      el.classList.remove("hidden");
      return;
    }
    dlg.close();
    refresh();
  });

  // Keep the breadcrumb / checkbox color in sync when the user changes
  // project or priority inside the modal.
  document.getElementById("td-project").addEventListener("change", (e) => {
    renderTaskDialogBreadcrumb(e.target.value);
    // Epics belong to one project, so switching project switches the choices —
    // and drops an epic that no longer applies.
    fillEpicSelect(e.target.value, document.getElementById("td-epic").value);
  });

  // Safety net. The dropdown only ever offers epics of the selected project, so
  // this normally does nothing — but if an entry from another project ever made
  // it into the list, saving would send an epic_id that overrides project_id
  // and undoes the move. Better to move the project field than to lie about it.
  document.getElementById("td-epic").addEventListener("change", (e) => {
    const chosen = epicById(e.target.value);
    if (!chosen) return;
    const projectSelect = document.getElementById("td-project");
    if (projectSelect.value !== chosen.project_id) {
      projectSelect.value = chosen.project_id;
      renderTaskDialogBreadcrumb(chosen.project_id);
      fillEpicSelect(chosen.project_id, chosen.id);
    }
  });

  // Warn when the chosen visible_from falls inside an absence.
  document.getElementById("td-visible-from").addEventListener("change", syncVisibleFromUI);
  document.getElementById("td-visible-from").addEventListener("input", syncVisibleFromUI);
  document.querySelectorAll("#td-priority-buttons .td-prio").forEach(btn => {
    btn.addEventListener("click", () => {
      const p = parseInt(btn.dataset.priority, 10);
      document.getElementById("td-priority").value = String(p);
      updateTaskDialogPriorityButtons(p);
      updateTaskDialogCheckbox(p);
    });
  });

  // Title textarea: auto-grow + Enter submits.
  const titleEl = document.getElementById("td-title");
  titleEl.addEventListener("input", () => autoGrow(titleEl));
  titleEl.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      document.getElementById("td-save").click();
    }
  });

  // Description: click to edit, blur to render.
  const descRendered = document.getElementById("td-description-rendered");
  const descInput = document.getElementById("td-description");
  descRendered.addEventListener("click", showDescriptionEditor);
  descInput.addEventListener("blur", hideDescriptionEditor);
  descInput.addEventListener("input", () => {
    descInput.style.height = "auto";
    descInput.style.height = Math.max(80, descInput.scrollHeight) + "px";
  });

  // Quick-reschedule buttons set visible_from to a preset date. "workday" is
  // special: the next-workday date (skipping weekends + absences, but not
  // business trips) is computed server-side, so it needs an async round-trip.
  document.querySelectorAll(".td-quick").forEach(btn => {
    btn.addEventListener("click", async () => {
      if (btn.dataset.quick === "workday") {
        try {
          const res = await api.nextWorkday();
          if (res && res.date) document.getElementById("td-visible-from").value = res.date;
        } catch (_) { /* ignore — leave the field unchanged */ }
        syncVisibleFromUI();
        return;
      }
      const iso = quickRescheduleISO(btn.dataset.quick);
      if (!iso) return;
      document.getElementById("td-visible-from").value = iso;
      syncVisibleFromUI();
    });
  });

  // Weekday buttons (Mo–Fr) jump to the next upcoming occurrence of that day.
  document.querySelectorAll(".td-weekday").forEach(btn => {
    btn.addEventListener("click", () => {
      const iso = nextWeekdayISO(parseInt(btn.dataset.weekday, 10));
      document.getElementById("td-visible-from").value = iso;
      syncVisibleFromUI();
    });
  });

  // Toggle visibility of the custom-RRULE inputs when the user picks "custom".
  document.getElementById("td-recurrence").addEventListener("change", (e) => {
    document.getElementById("td-recur-custom-wrap").classList.toggle("hidden", e.target.value !== "custom");
    syncRecurrenceModeUI();
  });
  // A hand-typed rule can introduce (or drop) a BYDAY anchor, which decides
  // whether "ab Erledigung" is available at all.
  document.getElementById("td-recur-custom").addEventListener("input", syncRecurrenceModeUI);
  document.getElementById("td-recur-mode").addEventListener("click", (e) => {
    const btn = e.target.closest(".td-seg");
    if (!btn || btn.disabled) return;
    setRecurrenceMode(btn.dataset.mode);
    syncRecurrenceModeUI();
  });
  // Comments: submit on form submit (button) or Ctrl/Cmd+Enter in the textarea.
  const commentForm = document.getElementById("td-comment-form");
  const commentInput = document.getElementById("td-comment-input");
  commentForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const id = dlg.dataset.id;
    if (!id) return;
    const body = commentInput.value.trim();
    const errEl = document.getElementById("td-comment-error");
    errEl.classList.add("hidden");
    if (!body) return;
    try {
      await api.addComment(id, body);
    } catch (err) {
      errEl.textContent = err.message || "Fehler beim Speichern";
      errEl.classList.remove("hidden");
      return;
    }
    commentInput.value = "";
    commentInput.style.height = "auto";
    await loadComments(id);
    // Keep the list badge in sync behind the open modal.
    refresh();
  });
  commentInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      commentForm.requestSubmit();
    }
  });
  commentInput.addEventListener("input", () => {
    commentInput.style.height = "auto";
    commentInput.style.height = commentInput.scrollHeight + "px";
  });

  document.getElementById("td-delete").addEventListener("click", async (e) => {
    e.preventDefault();
    const id = dlg.dataset.id;
    if (!confirm("Aufgabe wirklich löschen?")) return;
    try {
      await api.deleteTask(id);
    } catch (err) {
      const el = document.getElementById("td-error");
      el.textContent = err.message || "Fehler beim Löschen";
      el.classList.remove("hidden");
      return;
    }
    dlg.close();
    refresh();
  });
}

// ---------- Project dialog ----------

// ---------- Project dialog (create + edit) ----------

let projectDialogState = { mode: "create", id: null, color: "" };

function setProjectDialogColor(hex) {
  projectDialogState.color = hex || "";
  document.getElementById("pd-hex").value = projectDialogState.color;
  document.querySelectorAll("#pd-swatches .swatch").forEach(el => {
    el.classList.toggle("selected", el.dataset.hex.toLowerCase() === projectDialogState.color.toLowerCase());
  });
}

function renderColorSwatches() {
  const wrap = document.getElementById("pd-swatches");
  wrap.innerHTML = "";
  for (const c of PROJECT_COLOR_PRESETS) {
    const el = document.createElement("button");
    el.type = "button";
    el.className = "swatch";
    el.dataset.hex = c.hex;
    el.title = `${c.name} (${c.hex})`;
    el.style.background = c.hex;
    el.addEventListener("click", (e) => {
      e.preventDefault();
      setProjectDialogColor(c.hex);
    });
    wrap.appendChild(el);
  }
}

function openProjectDialog(p) {
  projectDialogState = { mode: p ? "edit" : "create", id: p ? p.id : null, color: "" };
  document.getElementById("pd-title").textContent = p ? "Projekt bearbeiten" : "Neues Projekt";
  document.getElementById("pd-name").value = p ? p.name : "";
  document.getElementById("pd-error").classList.add("hidden");
  document.getElementById("pd-delete").classList.toggle("hidden", !p);
  setProjectDialogColor(p ? (p.color || "") : "");
  document.getElementById("project-dialog").showModal();
  setTimeout(() => document.getElementById("pd-name").focus(), 0);
}

function showProjectDialogError(msg) {
  const el = document.getElementById("pd-error");
  el.textContent = msg;
  el.classList.remove("hidden");
}

function wireProjectDialog() {
  renderColorSwatches();
  const dlg = document.getElementById("project-dialog");
  document.getElementById("add-project-btn").addEventListener("click", () => openProjectDialog(null));

  document.getElementById("pd-cancel").addEventListener("click", () => dlg.close("cancel"));

  document.getElementById("pd-hex").addEventListener("input", (e) => {
    let v = e.target.value.trim();
    if (v && !v.startsWith("#")) v = "#" + v;
    projectDialogState.color = v;
    document.querySelectorAll("#pd-swatches .swatch").forEach(el => {
      el.classList.toggle("selected", el.dataset.hex.toLowerCase() === v.toLowerCase());
    });
  });

  document.getElementById("pd-save").addEventListener("click", async (e) => {
    e.preventDefault();
    const name = document.getElementById("pd-name").value.trim();
    if (!name) { showProjectDialogError("Name darf nicht leer sein."); return; }
    const color = projectDialogState.color.trim();
    if (color && !/^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(color)) {
      showProjectDialogError("Hex-Farbe muss wie #1a2b3c aussehen.");
      return;
    }
    try {
      if (projectDialogState.mode === "edit") {
        await api.patchProject(projectDialogState.id, { name, color });
      } else {
        await api.createProject({ name, color });
      }
      dlg.close();
      state.projects = await api.listProjects();
      await refresh();
    } catch (err) {
      showProjectDialogError(err.message);
    }
  });

  document.getElementById("pd-delete").addEventListener("click", async (e) => {
    e.preventDefault();
    if (!projectDialogState.id) return;
    const p = projectById(projectDialogState.id);
    if (!confirm(`Projekt "${p ? p.name : ""}" wirklich löschen?`)) return;
    try {
      await api.deleteProject(projectDialogState.id);
      const fallbackToToday = state.view.kind === "project" && state.view.id === projectDialogState.id;
      dlg.close();
      state.projects = await api.listProjects();
      if (fallbackToToday) {
        pushView({ kind: "today" });
      } else {
        await refresh();
      }
    } catch (err) {
      showProjectDialogError(err.message);
    }
  });
}

// ---------- Epic dialog (create + edit) ----------

let epicDialogState = { mode: "create", id: null };

// Only real, unarchived projects can hold an epic — the Inbox is the unsorted
// pile, not an undertaking, and the backend refuses it too.
function fillEpicProjectSelect(selectedId) {
  const el = document.getElementById("ed-project");
  const projects = activeProjects()
    .filter(p => p.id !== "inbox")
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  el.innerHTML = "";
  for (const p of projects) {
    const opt = document.createElement("option");
    opt.value = p.id;
    opt.textContent = p.name;
    el.appendChild(opt);
  }
  if (selectedId) el.value = selectedId;
}

function openEpicDialog(e) {
  epicDialogState = { mode: e ? "edit" : "create", id: e ? e.id : null };
  document.getElementById("ed-title").textContent = e ? "Epic bearbeiten" : "Neues Epic";
  document.getElementById("ed-name").value = e ? e.title : "";
  document.getElementById("ed-description").value = e ? (e.description || "") : "";
  document.getElementById("ed-start").value = e ? (e.start_date || "") : "";
  document.getElementById("ed-end").value = e ? (e.end_date || "") : "";
  fillEpicProjectSelect(e ? e.project_id : defaultProjectForEpicDialog());
  document.getElementById("ed-error").classList.add("hidden");
  document.getElementById("ed-delete").classList.toggle("hidden", !e);
  document.getElementById("epic-dialog").showModal();
  setTimeout(() => document.getElementById("ed-name").focus(), 0);
}

// A new epic starts in the project you are looking at, if that is a project at
// all; otherwise in the first real one.
function defaultProjectForEpicDialog() {
  if (state.view.kind === "project" && state.view.id !== "inbox") return state.view.id;
  if (state.view.kind === "epic") {
    const e = epicById(state.view.id);
    if (e) return e.project_id;
  }
  const first = activeProjects().find(p => p.id !== "inbox");
  return first ? first.id : "";
}

function showEpicDialogError(msg) {
  const el = document.getElementById("ed-error");
  el.textContent = msg;
  el.classList.remove("hidden");
}

function wireEpicDialog() {
  const dlg = document.getElementById("epic-dialog");
  document.getElementById("add-epic-btn").addEventListener("click", () => openEpicDialog(null));
  document.getElementById("ed-cancel").addEventListener("click", () => dlg.close("cancel"));

  document.getElementById("ed-save").addEventListener("click", async (ev) => {
    ev.preventDefault();
    const title = document.getElementById("ed-name").value.trim();
    if (!title) { showEpicDialogError("Titel darf nicht leer sein."); return; }
    const projectId = document.getElementById("ed-project").value;
    if (!projectId) { showEpicDialogError("Es gibt noch kein Projekt, zu dem ein Epic gehören könnte."); return; }
    const start = document.getElementById("ed-start").value || "";
    const end = document.getElementById("ed-end").value || "";
    if (start && end && end < start) {
      showEpicDialogError("Das Ende darf nicht vor dem Anfang liegen.");
      return;
    }
    const body = { title, project_id: projectId, description: document.getElementById("ed-description").value, start_date: start, end_date: end };
    try {
      if (epicDialogState.mode === "edit") {
        await api.patchEpic(epicDialogState.id, body);
      } else {
        await api.createEpic(body);
      }
      dlg.close();
      state.epics = await api.listEpics();
      await refresh();
    } catch (err) {
      showEpicDialogError(err.message);
    }
  });

  document.getElementById("ed-delete").addEventListener("click", async (ev) => {
    ev.preventDefault();
    if (!epicDialogState.id) return;
    const e = epicById(epicDialogState.id);
    if (!confirm(`Epic "${e ? e.title : ""}" wirklich löschen? Die Aufgaben bleiben erhalten.`)) return;
    try {
      await api.deleteEpic(epicDialogState.id);
      const wasOpen = state.view.kind === "epic" && state.view.id === epicDialogState.id;
      dlg.close();
      state.epics = await api.listEpics();
      if (wasOpen) {
        pushView({ kind: "today" });
      } else {
        await refresh();
      }
    } catch (err) {
      showEpicDialogError(err.message);
    }
  });
}

// ---------- Navigation ----------

function wireSidebarNav() {
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.addEventListener("click", (e) => {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) return;
      e.preventDefault();
      pushView({ kind: el.dataset.view });
    });
  });
}

// wireMobileNav handles the hamburger drawer for narrow viewports.
function wireMobileNav() {
  const btn = document.getElementById("menu-toggle");
  const sidebar = document.getElementById("sidebar");
  const backdrop = document.getElementById("sidebar-backdrop");
  if (!btn || !sidebar || !backdrop) return;
  const open = () => {
    backdrop.hidden = false;
    requestAnimationFrame(() => {
      sidebar.classList.add("open");
      backdrop.classList.add("visible");
    });
    btn.setAttribute("aria-expanded", "true");
  };
  const close = () => {
    sidebar.classList.remove("open");
    backdrop.classList.remove("visible");
    btn.setAttribute("aria-expanded", "false");
  };
  btn.addEventListener("click", () => {
    if (sidebar.classList.contains("open")) close();
    else open();
  });
  backdrop.addEventListener("click", close);
  sidebar.addEventListener("click", (e) => {
    if (e.target.closest(".nav-item, .project-item")) close();
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && sidebar.classList.contains("open")) close();
  });
}

// ---------- Keyboard shortcuts (Lastenheft 6.2) ----------

function wireShortcuts() {
  document.addEventListener("keydown", (e) => {
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA" || e.target.tagName === "SELECT") return;
    if (e.key === "n") {
      e.preventDefault();
      document.getElementById("quick-add-input").focus();
    }
  });
}

// ---------- Boot ----------

async function boot() {
  const todayIconEl = document.getElementById("today-icon");
  if (todayIconEl) todayIconEl.innerHTML = todayIconSVG();
  fillRecurrenceSelect("td-recurrence");
  wireNLDateInput("td-deadline");
  renderQuickAddOverlay();
  wireSidebarNav();
  wireMobileNav();
  wireQuickAdd();
  wireTaskDialog();
  wireProjectDialog();
  wireEpicDialog();
  wireAbsenceDialog();
  wireActivityControls();
  wireShortcuts();
  window.addEventListener("popstate", () => { handlePopState(); });
  state.projects = await api.listProjects();
  // Epics up front too: the sidebar shows them in every view, and the router
  // resolves /epics/<slug> against this list before the first render.
  try { state.epics = await api.listEpics(); } catch (_) { /* best-effort */ }
  // Load absences up front so the "falls on an absence" warning works in every
  // view (quick-add + task dialog), not just the absences/upcoming views.
  try { state.absences = await api.listAbsences(); } catch (_) { /* best-effort */ }

  const parsed = parsePath(location.pathname);
  if (parsed.openTaskId) {
    // Deep link to a task: set background view first, then open modal.
    let task;
    try {
      task = await api.getTask(parsed.openTaskId);
    } catch (err) {
      pushView({ kind: "today" }, { replace: true });
      return;
    }
    applyBackgroundViewForTask(task);
    // Replace history so the background view is recoverable when the modal closes.
    history.replaceState({}, "", viewToPath(state.view));
    history.pushState({}, "", `/tasks/${encodeURIComponent(task.id)}`);
    await refresh();
    openTaskDialogInternal(task);
    return;
  }
  const view = resolveViewFromPath(parsed.view);
  pushView(view, { replace: true });
}

boot();

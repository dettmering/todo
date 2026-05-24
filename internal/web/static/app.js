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

const PROJECT_COLOR_PRESETS = [
  { name: "Tomate",    hex: "#db4c3f" },
  { name: "Orange",    hex: "#eb8909" },
  { name: "Gelb",      hex: "#d8af1a" },
  { name: "Olive",     hex: "#7ecc49" },
  { name: "Grün",      hex: "#299438" },
  { name: "Mint",      hex: "#6accbc" },
  { name: "Türkis",    hex: "#158fad" },
  { name: "Hellblau",  hex: "#14aaf5" },
  { name: "Blau",      hex: "#246fe0" },
  { name: "Lila",      hex: "#884dff" },
  { name: "Magenta",   hex: "#af38eb" },
  { name: "Rosa",      hex: "#e05194" },
  { name: "Grau",      hex: "#808080" },
];
const DEFAULT_PROJECT_COLOR = "#808080";

function projectColor(p) {
  if (!p) return DEFAULT_PROJECT_COLOR;
  if (p.color && p.color !== "") return p.color;
  return p.id === "inbox" ? "#246fe0" : DEFAULT_PROJECT_COLOR;
}
function projectById(id) { return state.projects.find(p => p.id === id); }

const state = {
  projects: [],
  view: { kind: "today" }, // or { kind: "project", id }, { kind: "upcoming" }, { kind: "all-open" }
  openCounts: {}, // project_id -> number of open tasks
};

// ---------- URL router ----------
// URL scheme:
//   /                          → today (canonicalised to /today via replaceState)
//   /today, /upcoming, /inbox  → standard views
//   /projects/<slug>           → project view (slug resolved against state.projects)
//   /tasks/<id>                → opens task modal on top of the task's natural background view
// Unknown paths fall back to /today.

function projectBySlug(slug) { return state.projects.find(p => p.slug === slug); }

function parsePath(pathname) {
  const trimmed = pathname.replace(/\/+$/, "") || "/";
  if (trimmed === "/" || trimmed === "/today") return { view: { kind: "today" } };
  if (trimmed === "/upcoming") return { view: { kind: "upcoming" } };
  if (trimmed === "/inbox") return { view: { kind: "inbox" } };
  const proj = trimmed.match(/^\/projects\/([^\/]+)$/);
  if (proj) return { view: { kind: "project", slug: decodeURIComponent(proj[1]) } };
  const task = trimmed.match(/^\/tasks\/([^\/]+)$/);
  if (task) return { view: { kind: "today" }, openTaskId: decodeURIComponent(task[1]) };
  return { view: { kind: "today" } };
}

function viewToPath(view) {
  switch (view.kind) {
    case "today":    return "/today";
    case "upcoming": return "/upcoming";
    case "inbox":    return "/inbox";
    case "project": {
      const p = projectById(view.id);
      const slug = p && p.slug ? p.slug : view.id;
      return `/projects/${encodeURIComponent(slug)}`;
    }
  }
  return "/today";
}

// pushView updates state.view, syncs the URL, and re-renders. Use replace=true
// when canonicalising the URL on first load so the back button still works.
function pushView(view, { replace = false } = {}) {
  state.view = view;
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
  return parsedView;
}

// Picks the natural background view for a task: its project, or inbox for inbox tasks.
function applyBackgroundViewForTask(t) {
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
        hint.textContent = "→ " + formatDate(res.visible_from);
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

// Todoist-style relative date label: "Heute", "Morgen", weekday (within next 6 days),
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
  // sidebar nav active state
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.classList.toggle("active", el.dataset.view === state.view.kind);
  });

  // Task edit: include archived ones so existing tasks still resolve their current project.
  const editProjects = [...state.projects].sort((a, b) => Number(a.archived) - Number(b.archived) || a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  fillProjectSelect("td-project", editProjects, true);
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

function viewTitle() {
  switch (state.view.kind) {
    case "today": return "Heute";
    case "upcoming": return "Demnächst";
    case "inbox": return "Inbox";
    case "project": {
      const p = state.projects.find(p => p.id === state.view.id);
      return p ? p.name : "Projekt";
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
  }
  return {};
}

// Project ID that quick-add should target by default for the current view.
function defaultProjectForView() {
  if (state.view.kind === "project") return state.view.id;
  return "inbox";
}

async function refresh() {
  document.getElementById("view-title").textContent = viewTitle();
  document.getElementById("view-subtitle").textContent = new Date().toLocaleDateString("de-DE", { weekday: "long", day: "2-digit", month: "long" });
  const isToday = state.view.kind === "today";
  const fetches = [
    api.listTasks(buildTaskQuery()),
    api.listTasks({ status: "open", scope: "all" }),
  ];
  if (isToday) {
    // For the Heute view we additionally show tasks completed today, grayed out.
    fetches.push(api.listTasks({ status: "completed", scope: "all" }));
  }
  const results = await Promise.all(fetches);
  const tasks = results[0];
  const allOpen = results[1];
  const completedToday = isToday
    ? (results[2] || []).filter(t => t.completed_at && t.completed_at.slice(0, 10) === todayISO())
    : [];

  const today = todayISO();
  state.openCounts = {};
  let todayCount = 0;
  for (const t of allOpen) {
    state.openCounts[t.project_id] = (state.openCounts[t.project_id] || 0) + 1;
    if (!t.visible_from || t.visible_from <= today) todayCount++;
  }
  const todayCountEl = document.getElementById("today-count");
  if (todayCountEl) todayCountEl.textContent = todayCount > 0 ? todayCount : "";
  const inboxCount = state.openCounts["inbox"] || 0;
  const inboxCountEl = document.getElementById("inbox-count");
  if (inboxCountEl) inboxCountEl.textContent = inboxCount > 0 ? inboxCount : "";
  renderSidebar();
  renderTasks(tasks, completedToday);
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
  // Priority asc, then deadline (none last), then visible_from, then created_at.
  if (a.priority !== b.priority) return a.priority - b.priority;
  const ad = a.deadline || "9999-99-99";
  const bd = b.deadline || "9999-99-99";
  if (ad !== bd) return ad < bd ? -1 : 1;
  const av = a.visible_from || "9999-99-99";
  const bv = b.visible_from || "9999-99-99";
  if (av !== bv) return av < bv ? -1 : 1;
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
  const today = new Date(); today.setHours(0,0,0,0);
  const thisYear = today.getFullYear();
  for (const g of weeks) {
    const { monday, sunday } = isoWeekRange(g.year, g.week);
    const range = `${formatDayMonth(monday)}–${formatDayMonth(sunday)}`;
    const label = g.year === thisYear ? `KW ${g.week}` : `KW ${g.week} (${g.year})`;
    // A week is "past" once its Sunday lies strictly before today;
    // "current" if today falls within Monday..Sunday inclusive.
    const past = sunday.getTime() < today.getTime();
    const current = !past && monday.getTime() <= today.getTime();
    groups.push({ key: g.key, label, range, past, current, tasks: g.tasks });
  }
  return groups;
}

function viewIsGrouped() {
  return state.view.kind === "upcoming";
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
  const isCompleted = t.status === "completed";
  if (isCompleted) li.classList.add("completed");
  li.innerHTML = `
    <div class="checkbox ${priorityClass(t.priority)}${isCompleted ? " checked" : ""}" title="${isCompleted ? "Wieder öffnen" : "Erledigen"}"></div>
    <div class="body">
      <div class="title">${escapeHTML(t.title)}</div>
      ${t.description && t.description.trim() ? `<div class="desc">${renderDescPreview(t.description)}</div>` : ""}
      ${(dateParts.length || t.recurrence_label) ? `<div class="dates">${dateParts.join('<span class="date-sep">·</span>')}${t.recurrence_label ? `${dateParts.length ? '<span class="date-sep">·</span>' : ""}<span class="date-item recur-label">${ICON.repeat}<span>${escapeHTML(t.recurrence_label)}</span></span>` : ""}</div>` : ""}
    </div>
    ${state.view.kind === "project" ? "" : `
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
  if (noOpen && completedToday.length === 0) {
    empty.classList.remove("hidden");
    return;
  }
  empty.classList.add("hidden");
  const today = todayISO();

  if (viewIsGrouped()) {
    for (const group of groupTasksByWeek(tasks)) {
      const header = document.createElement("li");
      header.className = "group-header" + (group.past ? " past" : "") + (group.current ? " current" : "");
      header.innerHTML = `
        <span>${escapeHTML(group.label)}</span>
        ${group.range ? `<span class="group-range">${escapeHTML(group.range)}</span>` : ""}
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
// renders colored marks for matched ranges (Todoist-style).
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
  updateProjectMenu();
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

    const body = {
      title,
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

function renderDescPreview(md) {
  const text = (md || "").replace(/\s+/g, " ").trim();
  if (!text) return "";
  try {
    return window.marked.parseInline(text, { gfm: true });
  } catch (e) {
    return escapeHTML(text);
  }
}

// Render the description as HTML via marked, or show a placeholder if empty.
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
  // marked is loaded as a global (UMD). Safe defaults: no raw HTML passthrough.
  try {
    el.innerHTML = window.marked.parse(text, { breaks: true, gfm: true });
  } catch (e) {
    el.textContent = text;
  }
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

// openTaskDialogInternal opens the modal without touching browser history.
// Used by popstate/boot when the URL itself caused the modal to open.
function openTaskDialogInternal(t) {
  const dlg = document.getElementById("task-dialog");
  dlg.dataset.id = t.id;
  const titleEl = document.getElementById("td-title");
  titleEl.value = t.title;
  document.getElementById("td-description").value = t.description || "";
  document.getElementById("td-project").value = t.project_id;
  document.getElementById("td-priority").value = String(t.priority);
  document.getElementById("td-visible-from").value = isoToDE(t.visible_from);
  document.getElementById("td-deadline").value = isoToDE(t.deadline);
  clearDateHint("td-visible-from");
  clearDateHint("td-deadline");
  setRecurrenceControls("td-recurrence", "td-recur-custom-wrap", "td-recur-custom", t.recurrence_rule || "");
  document.getElementById("td-error").classList.add("hidden");
  renderTaskDialogBreadcrumb(t.project_id);
  updateTaskDialogCheckbox(t.priority);
  updateTaskDialogPriorityButtons(t.priority);
  renderTaskDescription(t.description || "");
  // Make sure the editor is hidden whenever we (re)open the modal.
  document.getElementById("td-description").classList.add("hidden");
  document.getElementById("td-description-rendered").classList.remove("hidden");
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
    // strictly to the upcoming Saturday, mirroring Todoist).
    const dow = d.getDay(); // 0=Sun..6=Sat
    const add = ((6 - dow) + 7) % 7 || 7;
    d.setDate(d.getDate() + add);
  } else if (kind === "monday") {
    const dow = d.getDay();
    const add = ((1 - dow) + 7) % 7 || 7;
    d.setDate(d.getDate() + add);
  } else {
    return "";
  }
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
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
      priority: parseInt(document.getElementById("td-priority").value, 10),
      visible_from: document.getElementById("td-visible-from").value || "",
      deadline: document.getElementById("td-deadline").value || "",
      recurrence_rule: readRecurrence("td-recurrence", "td-recur-custom"),
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
  });
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

  // Quick-reschedule buttons in the sidebar set visible_from to a preset date.
  document.querySelectorAll(".td-quick").forEach(btn => {
    btn.addEventListener("click", () => {
      const iso = quickRescheduleISO(btn.dataset.quick);
      if (!iso) return;
      const input = document.getElementById("td-visible-from");
      input.value = isoToDE(iso);
      // Trigger NL parser hint refresh.
      input.dispatchEvent(new Event("change"));
    });
  });

  // Toggle visibility of the custom-RRULE inputs when the user picks "custom".
  document.getElementById("td-recurrence").addEventListener("change", (e) => {
    document.getElementById("td-recur-custom-wrap").classList.toggle("hidden", e.target.value !== "custom");
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
  wireNLDateInput("td-visible-from");
  wireNLDateInput("td-deadline");
  renderQuickAddOverlay();
  wireSidebarNav();
  wireMobileNav();
  wireQuickAdd();
  wireTaskDialog();
  wireProjectDialog();
  wireShortcuts();
  window.addEventListener("popstate", () => { handlePopState(); });
  state.projects = await api.listProjects();

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

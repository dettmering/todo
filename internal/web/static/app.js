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
    const q = new URLSearchParams(params);
    return fetch("/api/tasks?" + q.toString()).then(jsonOrError);
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
  async deleteTask(id) {
    return fetch(`/api/tasks/${id}`, { method: "DELETE" }).then(jsonOrError);
  },
};

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
    el.href = "#";
    el.dataset.projectId = p.id;
    if (state.view.kind === "project" && state.view.id === p.id) el.classList.add("active");
    const count = state.openCounts[p.id] || 0;
    const canEdit = p.id !== "inbox";
    el.innerHTML = `
      <span class="dot" style="background:${escapeHTML(projectColor(p))}"></span>
      <span class="project-name">${escapeHTML(p.name)}</span>
      ${canEdit ? `<button class="project-edit" title="Bearbeiten" aria-label="Bearbeiten">✎</button>` : ""}
      <span class="project-count">${count > 0 ? count : ""}</span>
    `;
    el.addEventListener("click", (e) => {
      if (e.target.closest(".project-edit")) return; // edit button handled separately
      e.preventDefault();
      state.view = { kind: "project", id: p.id };
      refresh();
    });
    const editBtn = el.querySelector(".project-edit");
    if (editBtn) {
      editBtn.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        openProjectDialog(p);
      });
    }
    list.appendChild(el);
  }
  // sidebar nav active state
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.classList.toggle("active", el.dataset.view === state.view.kind);
  });

  // Quick-add: all active projects (including Inbox, so users can still target it).
  const qaProjects = activeProjects().sort((a, b) => {
    if (a.id === "inbox") return -1;
    if (b.id === "inbox") return 1;
    return a.sort_order - b.sort_order || a.name.localeCompare(b.name);
  });
  fillProjectSelect("qa-project", qaProjects);
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
  const [tasks, allOpen] = await Promise.all([
    api.listTasks(buildTaskQuery()),
    api.listTasks({ status: "open", scope: "all" }),
  ]);
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
  setQuickAddDefaults();
  renderTasks(tasks);
}

// Sync the quick-add controls with the current view so defaults always match
// the user's context: project = current project (or Inbox), visible_from = today.
function setQuickAddDefaults() {
  const projectSel = document.getElementById("qa-project");
  if (projectSel) {
    const target = defaultProjectForView();
    if ([...projectSel.options].some(o => o.value === target)) {
      projectSel.value = target;
    }
  }
  const visibleFromEl = document.getElementById("qa-visible-from");
  if (visibleFromEl && !visibleFromEl.value) visibleFromEl.value = todayISO();
  const prioSel = document.getElementById("qa-priority");
  if (prioSel && !prioSel.value) prioSel.value = "4";
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
  if (undated.length) groups.push({ key: "none", label: "Ohne Datum", range: "", past: false, tasks: undated });
  const today = new Date(); today.setHours(0,0,0,0);
  const thisYear = today.getFullYear();
  for (const g of weeks) {
    const { monday, sunday } = isoWeekRange(g.year, g.week);
    const range = `${formatDayMonth(monday)}–${formatDayMonth(sunday)}`;
    const label = g.year === thisYear ? `KW ${g.week}` : `KW ${g.week} (${g.year})`;
    // A week is "past" once its Sunday lies strictly before today.
    const past = sunday.getTime() < today.getTime();
    groups.push({ key: g.key, label, range, past, tasks: g.tasks });
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
  li.innerHTML = `
    <div class="checkbox ${priorityClass(t.priority)}" title="Erledigen"></div>
    <div class="body">
      <div class="title">${escapeHTML(t.title)}${t.recurrence_rule ? ` <span class="recur" title="Wiederkehrend">${ICON.repeat}</span>` : ""}</div>
      ${dateParts.length ? `<div class="dates">${dateParts.join('<span class="date-sep">·</span>')}</div>` : ""}
    </div>
    ${state.view.kind === "project" ? "" : `
    <div class="task-project" title="${escapeHTML(projectName(t.project_id))}">
      <span>${escapeHTML(projectName(t.project_id))}</span>
      <span class="task-project-dot" style="background:${escapeHTML(projectColor(projectById(t.project_id)))}"></span>
    </div>`}
    <div class="actions">
      <button data-action="edit" title="Bearbeiten" aria-label="Bearbeiten">${ICON.edit}</button>
      <button data-action="delete" title="Löschen" aria-label="Löschen">${ICON.trash}</button>
    </div>
  `;
  li.querySelector(".checkbox").addEventListener("click", async (e) => {
    e.stopPropagation();
    await api.completeTask(t.id);
    refresh();
  });
  li.querySelector('[data-action="edit"]').addEventListener("click", (e) => {
    e.stopPropagation();
    openTaskDialog(t);
  });
  li.querySelector('[data-action="delete"]').addEventListener("click", async (e) => {
    e.stopPropagation();
    await api.deleteTask(t.id);
    refresh();
  });
  li.addEventListener("click", () => openTaskDialog(t));
  return li;
}

function renderTasks(tasks) {
  const list = document.getElementById("task-list");
  const empty = document.getElementById("empty-state");
  list.innerHTML = "";
  if (!tasks || tasks.length === 0) {
    empty.classList.remove("hidden");
    return;
  }
  empty.classList.add("hidden");
  const today = todayISO();

  if (!viewIsGrouped()) {
    for (const t of tasks) list.appendChild(createTaskLI(t, today));
    return;
  }

  for (const group of groupTasksByWeek(tasks)) {
    const header = document.createElement("li");
    header.className = "group-header" + (group.past ? " past" : "");
    header.innerHTML = `
      <span>${escapeHTML(group.label)}</span>
      ${group.range ? `<span class="group-range">${escapeHTML(group.range)}</span>` : ""}
      <span class="group-count">${group.tasks.length}</span>
    `;
    list.appendChild(header);
    for (const t of group.tasks) list.appendChild(createTaskLI(t, today));
  }
}

// ---------- Quick add ----------

function wireQuickAdd() {
  const form = document.getElementById("quick-add-form");
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const title = document.getElementById("quick-add-input").value.trim();
    if (!title) return;
    const projectSel = document.getElementById("qa-project").value;
    const body = {
      title,
      project_id: projectSel || defaultProjectForView(),
      priority: parseInt(document.getElementById("qa-priority").value, 10),
      visible_from: document.getElementById("qa-visible-from").value || "",
      deadline: document.getElementById("qa-deadline").value || "",
    };
    await api.createTask(body);
    form.reset();
    document.getElementById("qa-priority").value = "4";
    refresh(); // refresh calls setQuickAddDefaults() which re-fills visible_from + project
  });
}

// ---------- Edit dialog ----------

function openTaskDialog(t) {
  const dlg = document.getElementById("task-dialog");
  dlg.dataset.id = t.id;
  document.getElementById("td-title").value = t.title;
  document.getElementById("td-description").value = t.description || "";
  document.getElementById("td-project").value = t.project_id;
  document.getElementById("td-priority").value = String(t.priority);
  document.getElementById("td-visible-from").value = t.visible_from || "";
  document.getElementById("td-deadline").value = t.deadline || "";
  dlg.showModal();
}

function wireTaskDialog() {
  const dlg = document.getElementById("task-dialog");
  document.getElementById("td-cancel").addEventListener("click", () => dlg.close("cancel"));
  document.getElementById("td-save").addEventListener("click", async (e) => {
    e.preventDefault();
    const id = dlg.dataset.id;
    const body = {
      title: document.getElementById("td-title").value,
      description: document.getElementById("td-description").value,
      project_id: document.getElementById("td-project").value,
      priority: parseInt(document.getElementById("td-priority").value, 10),
      visible_from: document.getElementById("td-visible-from").value || "",
      deadline: document.getElementById("td-deadline").value || "",
    };
    await api.patchTask(id, body);
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
      // If we were viewing this project, fall back to "today".
      if (state.view.kind === "project" && state.view.id === projectDialogState.id) {
        state.view = { kind: "today" };
      }
      dlg.close();
      state.projects = await api.listProjects();
      await refresh();
    } catch (err) {
      showProjectDialogError(err.message);
    }
  });
}

// ---------- Navigation ----------

function wireSidebarNav() {
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.addEventListener("click", (e) => {
      e.preventDefault();
      state.view = { kind: el.dataset.view };
      refresh();
    });
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
  wireSidebarNav();
  wireQuickAdd();
  wireTaskDialog();
  wireProjectDialog();
  wireShortcuts();
  state.projects = await api.listProjects();
  await refresh();
}

boot();

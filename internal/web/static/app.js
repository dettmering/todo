// Vanilla JS frontend. Designed so adding routes/views later is just adding
// view handlers and (optionally) re-rendering. No build step.

const api = {
  async listProjects() { return fetch("/api/projects").then(r => r.json()); },
  async createProject(body) {
    return fetch("/api/projects", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(r => r.json());
  },
  async listTasks(params = {}) {
    const q = new URLSearchParams(params);
    return fetch("/api/tasks?" + q.toString()).then(r => r.json());
  },
  async createTask(body) {
    return fetch("/api/tasks", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(r => r.json());
  },
  async patchTask(id, body) {
    return fetch(`/api/tasks/${id}`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then(r => r.json());
  },
  async completeTask(id) {
    return fetch(`/api/tasks/${id}/complete`, { method: "POST" }).then(r => r.json());
  },
  async deleteTask(id) {
    return fetch(`/api/tasks/${id}`, { method: "DELETE" });
  },
};

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

function renderSidebar() {
  const list = document.getElementById("project-list");
  list.innerHTML = "";
  const projects = [...state.projects].sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
  for (const p of projects) {
    const el = document.createElement("a");
    el.className = "project-item";
    el.href = "#";
    el.dataset.projectId = p.id;
    if (state.view.kind === "project" && state.view.id === p.id) el.classList.add("active");
    const count = state.openCounts[p.id] || 0;
    el.innerHTML = `
      <span class="dot" style="background:${p.id === "inbox" ? "#246fe0" : "#888"}"></span>
      <span class="project-name">${escapeHTML(p.name)}</span>
      <span class="project-count">${count > 0 ? count : ""}</span>
    `;
    el.addEventListener("click", (e) => { e.preventDefault(); state.view = { kind: "project", id: p.id }; refresh(); });
    list.appendChild(el);
  }
  // sidebar nav active state
  document.querySelectorAll(".sidebar-nav .nav-item").forEach(el => {
    el.classList.toggle("active", el.dataset.view === state.view.kind);
  });

  // project select fields
  for (const sel of ["qa-project", "td-project"]) {
    const el = document.getElementById(sel);
    if (!el) continue;
    const prev = el.value;
    el.innerHTML = "";
    for (const p of projects) {
      const opt = document.createElement("option");
      opt.value = p.id;
      opt.textContent = p.name;
      el.appendChild(opt);
    }
    if (prev) el.value = prev;
  }
}

function viewTitle() {
  switch (state.view.kind) {
    case "today": return "Heute";
    case "upcoming": return "Demnächst";
    case "all-open": return "Alle offenen Aufgaben";
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
    case "all-open":
      return { status: "open", scope: "all", sort: "priority" };
    case "project":
      return { status: "open", scope: "all", project_id: state.view.id, sort: "priority" };
  }
  return {};
}

async function refresh() {
  document.getElementById("view-title").textContent = viewTitle();
  document.getElementById("view-subtitle").textContent = new Date().toLocaleDateString("de-DE", { weekday: "long", day: "2-digit", month: "long" });
  const [tasks, allOpen] = await Promise.all([
    api.listTasks(buildTaskQuery()),
    api.listTasks({ status: "open", scope: "all" }),
  ]);
  state.openCounts = {};
  for (const t of allOpen) state.openCounts[t.project_id] = (state.openCounts[t.project_id] || 0) + 1;
  renderSidebar();
  renderTasks(tasks);
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
  for (const t of tasks) {
    const li = document.createElement("li");
    li.className = "task";
    const overdue = t.deadline && t.deadline < today;
    const dateParts = [];
    if (t.visible_from) {
      dateParts.push(`<span class="date-item"><span class="date-label">Fällig:</span> ${formatDate(t.visible_from)}</span>`);
    }
    if (t.deadline) {
      dateParts.push(`<span class="date-item ${overdue ? "overdue" : "deadline"}"><span class="date-label">Deadline:</span> ${formatDate(t.deadline)}</span>`);
    }
    li.innerHTML = `
      <div class="checkbox ${priorityClass(t.priority)}" title="Erledigen"></div>
      <div class="body">
        <div class="title">${escapeHTML(t.title)}${t.recurrence_rule ? ` <span class="recur">🔁</span>` : ""}</div>
        ${dateParts.length ? `<div class="dates">${dateParts.join('<span class="date-sep">·</span>')}</div>` : ""}
      </div>
      <div class="task-project">${escapeHTML(projectName(t.project_id))}</div>
      <div class="actions">
        <button data-action="edit" title="Bearbeiten">✎</button>
        <button data-action="delete" title="Löschen">🗑</button>
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
    list.appendChild(li);
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
      project_id: state.view.kind === "project" ? state.view.id : (projectSel || "inbox"),
      priority: parseInt(document.getElementById("qa-priority").value, 10),
      visible_from: document.getElementById("qa-visible-from").value || "",
      deadline: document.getElementById("qa-deadline").value || "",
    };
    await api.createTask(body);
    form.reset();
    document.getElementById("qa-priority").value = "3";
    refresh();
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

function wireProjectDialog() {
  const dlg = document.getElementById("project-dialog");
  document.getElementById("add-project-btn").addEventListener("click", () => {
    document.getElementById("project-name-input").value = "";
    dlg.showModal();
  });
  document.getElementById("project-create-btn").addEventListener("click", async (e) => {
    e.preventDefault();
    const name = document.getElementById("project-name-input").value.trim();
    if (!name) return;
    await api.createProject({ name });
    dlg.close();
    state.projects = await api.listProjects();
    renderSidebar();
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
  wireSidebarNav();
  wireQuickAdd();
  wireTaskDialog();
  wireProjectDialog();
  wireShortcuts();
  state.projects = await api.listProjects();
  await refresh();
}

boot();

// Package mcp exposes the todo application as a Model Context Protocol server
// over Streamable HTTP. It talks to the same *service.Service the REST API
// uses, so it inherits validation, ID generation, timestamps, and event
// emission — the MCP layer is transport only, never business logic.
//
// The handler is guarded by a bearer token (see New). It is mounted behind
// Traefik on a dedicated router that bypasses BasicAuth, mirroring the iCal
// feed: MCP clients cannot perform an interactive login, so a static token is
// the whole access control.
package mcp

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/nldate"
	"github.com/till/todo/internal/recurrence"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage"
)

// New returns an http.Handler serving the MCP endpoint over Streamable HTTP,
// wrapped in bearer-token authentication. Every request must carry
// "Authorization: Bearer <token>"; anything else gets 401. token must be
// non-empty — the caller decides whether to mount the route at all (an empty
// token disables the feature entirely, like ICAL_TOKEN).
func New(svc *service.Service, token string) http.Handler {
	s := &server{svc: svc}
	mcpServer := s.build()
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		nil,
	)
	return bearerAuth(token, streamable)
}

// bearerAuth rejects any request whose token doesn't match, using a
// constant-time comparison. The token may arrive two ways:
//
//   - "Authorization: Bearer <token>" header — used by Claude Code / Desktop,
//     which let you configure custom headers.
//   - "?token=<token>" query parameter — used by claude.ai custom connectors,
//     whose UI only offers OAuth. A valid token in the URL means the server
//     never challenges, so no OAuth flow is triggered. This mirrors the iCal
//     feed (GET /api/absences.ics?token=...).
func bearerAuth(token string, next http.Handler) http.Handler {
	want := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" {
			got = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type server struct {
	svc *service.Service
}

// build assembles the MCP server and registers every tool. Tools map 1:1 onto
// service methods; see the README "MCP-Server" section for the catalogue.
func (s *server) build() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "todo",
		Title:   "Todo",
		Version: "1.0.0",
	}, nil)

	// ---------- Tasks ----------
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List tasks. By default returns all open tasks across every project; narrow with project_id, status, scope and sort.",
	}, s.listTasks)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_task",
		Description: "Get a single task by id.",
	}, s.getTask)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_task",
		Description: "Create a task. Only title is required; it lands in Inbox at priority P4 unless overridden. Dates accept ISO (YYYY-MM-DD) or German phrases like 'morgen', 'freitag'.",
	}, s.createTask)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_task",
		Description: "Update fields of an existing task. Only the fields you set are changed.",
	}, s.updateTask)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "complete_task",
		Description: "Mark a task as completed. Recurring tasks spawn their next instance automatically.",
	}, s.completeTask)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "reopen_task",
		Description: "Reopen a previously completed task.",
	}, s.reopenTask)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_task",
		Description: "Soft-delete a task (status becomes 'deleted'; the row is kept).",
	}, s.deleteTask)

	// ---------- Projects ----------
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_projects",
		Description: "List projects. Archived projects are excluded unless include_archived is true.",
	}, s.listProjects)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a project. Only name is required.",
	}, s.createProject)

	// ---------- Absences ----------
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_absences",
		Description: "List all absences (vacation, holiday, business trip, sick, other).",
	}, s.listAbsences)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_absence",
		Description: "Create an absence spanning start_date..end_date (inclusive).",
	}, s.createAbsence)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "next_workday",
		Description: "Return the next working day (skipping weekends and absences).",
	}, s.nextWorkday)

	// ---------- Utilities ----------
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "parse_task",
		Description: "Parse a German natural-language phrase into a visible_from date and/or recurrence rule, without creating anything.",
	}, s.parseTask)

	return srv
}

// ---------------------------------------------------------------------------
// Output shapes (snake_case, matching the REST DTOs).
// ---------------------------------------------------------------------------

const timeFormat = time.RFC3339

type taskOut struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description,omitempty"`
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name,omitempty"`
	Status           string `json:"status"`
	Priority         int    `json:"priority"`
	VisibleFrom      string `json:"visible_from,omitempty"`
	Deadline         string `json:"deadline,omitempty"`
	RecurrenceRule   string `json:"recurrence_rule,omitempty"`
	RecurrenceMode   string `json:"recurrence_mode,omitempty"`
	RecurrenceAnchor string `json:"recurrence_anchor,omitempty"`
	RecurrenceLabel  string `json:"recurrence_label,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
}

func toTaskOut(t *domain.Task, projectName string) taskOut {
	o := taskOut{
		ID:               t.ID,
		Title:            t.Title,
		Description:      t.Description,
		ProjectID:        t.ProjectID,
		ProjectName:      projectName,
		Status:           string(t.Status),
		Priority:         int(t.Priority),
		VisibleFrom:      t.VisibleFrom.String(),
		Deadline:         t.Deadline.String(),
		RecurrenceRule:   t.RecurrenceRule,
		RecurrenceMode:   string(t.RecurrenceMode),
		RecurrenceAnchor: t.RecurrenceAnchor.String(),
		RecurrenceLabel:  recurrence.HumanLabel(t.RecurrenceRule, t.RecurrenceMode),
		CreatedAt:        t.CreatedAt.Format(timeFormat),
		UpdatedAt:        t.UpdatedAt.Format(timeFormat),
	}
	if t.CompletedAt != nil {
		o.CompletedAt = t.CompletedAt.Format(timeFormat)
	}
	return o
}

type projectOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
	Archived    bool   `json:"archived"`
	Color       string `json:"color,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toProjectOut(p *domain.Project) projectOut {
	return projectOut{
		ID: p.ID, Name: p.Name, Slug: p.Slug, Description: p.Description,
		SortOrder: p.SortOrder, Archived: p.Archived, Color: p.Color,
		CreatedAt: p.CreatedAt.Format(timeFormat),
		UpdatedAt: p.UpdatedAt.Format(timeFormat),
	}
}

type absenceOut struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAbsenceOut(a *domain.Absence) absenceOut {
	return absenceOut{
		ID:        a.ID,
		Type:      string(a.Type),
		Label:     a.Label,
		StartDate: a.Start.String(),
		EndDate:   a.End.String(),
		CreatedAt: a.CreatedAt.Format(timeFormat),
		UpdatedAt: a.UpdatedAt.Format(timeFormat),
	}
}

// ---------------------------------------------------------------------------
// Tool inputs and handlers.
// ---------------------------------------------------------------------------

type listTasksInput struct {
	ProjectID string `json:"project_id,omitempty" jsonschema:"only tasks in this project id"`
	Status    string `json:"status,omitempty" jsonschema:"one of open (default), completed, deleted, all"`
	Scope     string `json:"scope,omitempty" jsonschema:"'visible' applies the visible_from<=today rule; default shows all"`
	Sort      string `json:"sort,omitempty" jsonschema:"one of priority, deadline, visible_from, created_at"`
}

type tasksOutput struct {
	Tasks []taskOut `json:"tasks"`
}

func (s *server) listTasks(ctx context.Context, _ *mcp.CallToolRequest, in listTasksInput) (*mcp.CallToolResult, tasksOutput, error) {
	f := storage.TaskFilter{
		ProjectID:   in.ProjectID,
		OnlyVisible: in.Scope == "visible",
	}
	switch in.Status {
	case "", "open":
		f.Status = domain.StatusOpen
	case "completed":
		f.Status = domain.StatusCompleted
	case "deleted":
		f.Status = domain.StatusDeleted
	case "all":
		// no status constraint
	}
	switch in.Sort {
	case "priority":
		f.Sort = storage.SortPriority
	case "deadline":
		f.Sort = storage.SortDeadline
	case "visible_from":
		f.Sort = storage.SortVisibleFrom
	case "created_at":
		f.Sort = storage.SortCreatedAt
	}

	tasks, err := s.svc.ListTasks(ctx, f)
	if err != nil {
		return nil, tasksOutput{}, err
	}
	names := s.projectNames(ctx)
	out := make([]taskOut, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskOut(t, names[t.ProjectID]))
	}
	return nil, tasksOutput{Tasks: out}, nil
}

type taskIDInput struct {
	ID string `json:"id" jsonschema:"the task id"`
}

func (s *server) getTask(ctx context.Context, _ *mcp.CallToolRequest, in taskIDInput) (*mcp.CallToolResult, taskOut, error) {
	t, err := s.svc.GetTask(ctx, in.ID)
	if err != nil {
		return nil, taskOut{}, err
	}
	return nil, toTaskOut(t, s.projectName(ctx, t.ProjectID)), nil
}

type createTaskInput struct {
	Title            string `json:"title" jsonschema:"the task title (required)"`
	Description      string `json:"description,omitempty"`
	ProjectID        string `json:"project_id,omitempty" jsonschema:"target project id; defaults to inbox"`
	Priority         int    `json:"priority,omitempty" jsonschema:"1 (highest) to 4 (lowest); defaults to 4"`
	VisibleFrom      string `json:"visible_from,omitempty" jsonschema:"ISO date or German phrase; when the task starts appearing"`
	Deadline         string `json:"deadline,omitempty" jsonschema:"ISO date or German phrase; when the task is due"`
	RecurrenceRule   string `json:"recurrence_rule,omitempty" jsonschema:"RRULE, e.g. FREQ=WEEKLY;BYDAY=MO"`
	RecurrenceAnchor string `json:"recurrence_anchor,omitempty" jsonschema:"series date a fixed-mode recurrence counts from; postponing visible_from does not move it"`
	RecurrenceMode   string `json:"recurrence_mode,omitempty" jsonschema:"fixed (default: keeps the planned date) or after_completion (interval restarts when ticked off; only FREQ+INTERVAL allowed)"`
}

func (s *server) createTask(ctx context.Context, _ *mcp.CallToolRequest, in createTaskInput) (*mcp.CallToolResult, taskOut, error) {
	vf, err := parseDateInput(in.VisibleFrom)
	if err != nil {
		return nil, taskOut{}, err
	}
	dl, err := parseDateInput(in.Deadline)
	if err != nil {
		return nil, taskOut{}, err
	}
	anchor, err := parseDateInput(in.RecurrenceAnchor)
	if err != nil {
		return nil, taskOut{}, err
	}
	t, err := s.svc.CreateTask(ctx, service.TaskInput{
		Title:            in.Title,
		Description:      in.Description,
		ProjectID:        in.ProjectID,
		Priority:         domain.Priority(in.Priority),
		VisibleFrom:      vf,
		Deadline:         dl,
		RecurrenceRule:   in.RecurrenceRule,
		RecurrenceMode:   domain.RecurrenceMode(in.RecurrenceMode),
		RecurrenceAnchor: anchor,
	})
	if err != nil {
		return nil, taskOut{}, err
	}
	return nil, toTaskOut(t, s.projectName(ctx, t.ProjectID)), nil
}

type updateTaskInput struct {
	ID               string  `json:"id" jsonschema:"the task id"`
	Title            *string `json:"title,omitempty"`
	Description      *string `json:"description,omitempty"`
	ProjectID        *string `json:"project_id,omitempty"`
	Priority         *int    `json:"priority,omitempty" jsonschema:"1 (highest) to 4 (lowest)"`
	VisibleFrom      *string `json:"visible_from,omitempty" jsonschema:"ISO date or German phrase; empty string clears it"`
	Deadline         *string `json:"deadline,omitempty" jsonschema:"ISO date or German phrase; empty string clears it"`
	RecurrenceRule   *string `json:"recurrence_rule,omitempty"`
	RecurrenceAnchor *string `json:"recurrence_anchor,omitempty" jsonschema:"series date a fixed-mode recurrence counts from; postponing visible_from does not move it"`
	RecurrenceMode   *string `json:"recurrence_mode,omitempty" jsonschema:"fixed (default: keeps the planned date) or after_completion (interval restarts when ticked off; only FREQ+INTERVAL allowed)"`
}

func (s *server) updateTask(ctx context.Context, _ *mcp.CallToolRequest, in updateTaskInput) (*mcp.CallToolResult, taskOut, error) {
	patch := service.TaskPatch{
		Title:          in.Title,
		Description:    in.Description,
		ProjectID:      in.ProjectID,
		RecurrenceRule: in.RecurrenceRule,
	}
	if in.RecurrenceMode != nil {
		m := domain.RecurrenceMode(*in.RecurrenceMode)
		patch.RecurrenceMode = &m
	}
	if in.RecurrenceAnchor != nil {
		d, err := parseDateInput(*in.RecurrenceAnchor)
		if err != nil {
			return nil, taskOut{}, err
		}
		patch.RecurrenceAnchor = &d
	}
	if in.Priority != nil {
		p := domain.Priority(*in.Priority)
		patch.Priority = &p
	}
	if in.VisibleFrom != nil {
		d, err := parseDateInput(*in.VisibleFrom)
		if err != nil {
			return nil, taskOut{}, err
		}
		patch.VisibleFrom = &d
	}
	if in.Deadline != nil {
		d, err := parseDateInput(*in.Deadline)
		if err != nil {
			return nil, taskOut{}, err
		}
		patch.Deadline = &d
	}
	t, err := s.svc.UpdateTask(ctx, in.ID, patch)
	if err != nil {
		return nil, taskOut{}, err
	}
	return nil, toTaskOut(t, s.projectName(ctx, t.ProjectID)), nil
}

func (s *server) completeTask(ctx context.Context, _ *mcp.CallToolRequest, in taskIDInput) (*mcp.CallToolResult, taskOut, error) {
	t, err := s.svc.CompleteTask(ctx, in.ID)
	if err != nil {
		return nil, taskOut{}, err
	}
	return nil, toTaskOut(t, s.projectName(ctx, t.ProjectID)), nil
}

func (s *server) reopenTask(ctx context.Context, _ *mcp.CallToolRequest, in taskIDInput) (*mcp.CallToolResult, taskOut, error) {
	t, err := s.svc.ReopenTask(ctx, in.ID)
	if err != nil {
		return nil, taskOut{}, err
	}
	return nil, toTaskOut(t, s.projectName(ctx, t.ProjectID)), nil
}

type deleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func (s *server) deleteTask(ctx context.Context, _ *mcp.CallToolRequest, in taskIDInput) (*mcp.CallToolResult, deleteResult, error) {
	if err := s.svc.DeleteTask(ctx, in.ID); err != nil {
		return nil, deleteResult{}, err
	}
	return nil, deleteResult{ID: in.ID, Deleted: true}, nil
}

type listProjectsInput struct {
	IncludeArchived bool `json:"include_archived,omitempty" jsonschema:"include archived projects too"`
}

type projectsOutput struct {
	Projects []projectOut `json:"projects"`
}

func (s *server) listProjects(ctx context.Context, _ *mcp.CallToolRequest, in listProjectsInput) (*mcp.CallToolResult, projectsOutput, error) {
	projects, err := s.svc.ListProjects(ctx, in.IncludeArchived)
	if err != nil {
		return nil, projectsOutput{}, err
	}
	out := make([]projectOut, 0, len(projects))
	for _, p := range projects {
		out = append(out, toProjectOut(p))
	}
	return nil, projectsOutput{Projects: out}, nil
}

type createProjectInput struct {
	Name        string `json:"name" jsonschema:"the project name (required)"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty" jsonschema:"CSS hex color like #246fe0"`
}

func (s *server) createProject(ctx context.Context, _ *mcp.CallToolRequest, in createProjectInput) (*mcp.CallToolResult, projectOut, error) {
	p, err := s.svc.CreateProject(ctx, service.ProjectInput{
		Name:        in.Name,
		Description: in.Description,
		Color:       in.Color,
	})
	if err != nil {
		return nil, projectOut{}, err
	}
	return nil, toProjectOut(p), nil
}

type absencesOutput struct {
	Absences []absenceOut `json:"absences"`
}

type emptyInput struct{}

func (s *server) listAbsences(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, absencesOutput, error) {
	absences, err := s.svc.ListAbsences(ctx)
	if err != nil {
		return nil, absencesOutput{}, err
	}
	out := make([]absenceOut, 0, len(absences))
	for _, a := range absences {
		out = append(out, toAbsenceOut(a))
	}
	return nil, absencesOutput{Absences: out}, nil
}

type createAbsenceInput struct {
	Type      string `json:"type,omitempty" jsonschema:"one of vacation, holiday, business_trip, sick, other; defaults to other"`
	Label     string `json:"label,omitempty"`
	StartDate string `json:"start_date" jsonschema:"ISO date or German phrase (required)"`
	EndDate   string `json:"end_date" jsonschema:"ISO date or German phrase (required)"`
}

func (s *server) createAbsence(ctx context.Context, _ *mcp.CallToolRequest, in createAbsenceInput) (*mcp.CallToolResult, absenceOut, error) {
	start, err := parseDateInput(in.StartDate)
	if err != nil {
		return nil, absenceOut{}, err
	}
	end, err := parseDateInput(in.EndDate)
	if err != nil {
		return nil, absenceOut{}, err
	}
	a, err := s.svc.CreateAbsence(ctx, service.AbsenceInput{
		Type:  in.Type,
		Label: in.Label,
		Start: start,
		End:   end,
	})
	if err != nil {
		return nil, absenceOut{}, err
	}
	return nil, toAbsenceOut(a), nil
}

type nextWorkdayOutput struct {
	Date string `json:"date"`
}

func (s *server) nextWorkday(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, nextWorkdayOutput, error) {
	d, err := s.svc.NextWorkday(ctx)
	if err != nil {
		return nil, nextWorkdayOutput{}, err
	}
	return nil, nextWorkdayOutput{Date: d.String()}, nil
}

type parseTaskInput struct {
	Text  string `json:"text" jsonschema:"the German phrase to parse"`
	Today string `json:"today,omitempty" jsonschema:"ISO date to treat as 'today'; defaults to the current date"`
}

type parseTaskOutput struct {
	Input          string `json:"input"`
	VisibleFrom    string `json:"visible_from,omitempty"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"`
}

func (s *server) parseTask(_ context.Context, _ *mcp.CallToolRequest, in parseTaskInput) (*mcp.CallToolResult, parseTaskOutput, error) {
	today := domain.DateOf(time.Now().UTC())
	if in.Today != "" {
		if d, err := domain.ParseDate(in.Today); err == nil {
			today = d
		}
	}
	res, err := nldate.Parse(in.Text, today)
	if err != nil {
		return nil, parseTaskOutput{}, err
	}
	return nil, parseTaskOutput{
		Input:          in.Text,
		VisibleFrom:    res.Date.String(),
		RecurrenceRule: res.RecurrenceRule,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers.
// ---------------------------------------------------------------------------

// parseDateInput accepts an empty string, an ISO date (YYYY-MM-DD), or a German
// natural-language phrase ("morgen", "freitag", "27.5."). Empty yields the zero
// LocalDate. Mirrors httpapi.parseDateInput so MCP and REST behave the same.
func parseDateInput(sv string) (domain.LocalDate, error) {
	if sv == "" {
		return domain.LocalDate{}, nil
	}
	if d, err := domain.ParseDate(sv); err == nil {
		return d, nil
	}
	today := domain.DateOf(time.Now().UTC())
	d, err := nldate.ParseDate(sv, today)
	if err != nil {
		return domain.LocalDate{}, err
	}
	return d, nil
}

// projectNames returns an id->name map over all projects, including archived
// ones, so historical project labels still resolve on completed tasks.
func (s *server) projectNames(ctx context.Context) map[string]string {
	projects, err := s.svc.ListProjects(ctx, true)
	if err != nil {
		return map[string]string{}
	}
	m := make(map[string]string, len(projects))
	for _, p := range projects {
		m[p.ID] = p.Name
	}
	return m
}

func (s *server) projectName(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	p, err := s.svc.GetProject(ctx, id)
	if err != nil {
		return ""
	}
	return p.Name
}

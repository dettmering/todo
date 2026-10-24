package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/recurrence"
	"github.com/till/todo/internal/service"
)

type errorResp struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeJSON(w, http.StatusBadRequest, errorResp{Error: err.Error()})
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResp{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResp{Error: err.Error()})
	}
}

// DTOs intentionally use snake_case keys to match the Lastenheft examples.

type taskDTO struct {
	ID               string       `json:"id"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	ProjectID        string       `json:"project_id"`
	ProjectName      string       `json:"project_name"`
	EpicID           string       `json:"epic_id,omitempty"`
	EpicTitle        string       `json:"epic_title,omitempty"`
	Status           string       `json:"status"`
	Priority         int          `json:"priority"`
	Urgency          int          `json:"urgency"`
	Quadrant         string       `json:"quadrant"`
	VisibleFrom      string       `json:"visible_from,omitempty"`
	Deadline         string       `json:"deadline,omitempty"`
	RecurrenceRule   string       `json:"recurrence_rule,omitempty"`
	RecurrenceMode   string       `json:"recurrence_mode,omitempty"`
	RecurrenceAnchor string       `json:"recurrence_anchor,omitempty"`
	RecurrenceLabel  string       `json:"recurrence_label,omitempty"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        string       `json:"updated_at"`
	CompletedAt      *string      `json:"completed_at,omitempty"`
	CommentCount     int          `json:"comment_count,omitempty"`
	Comments         []commentDTO `json:"comments,omitempty"`
}

// toTaskDTO builds the full task representation, comment bodies included. Used
// by the single-task responses, where the caller has asked for exactly one task
// and can afford them.
func toTaskDTO(t *domain.Task, projectName, epicTitle string, comments []*domain.Comment) taskDTO {
	d := toTaskListDTO(t, projectName, epicTitle, len(comments))
	d.Comments = make([]commentDTO, 0, len(comments))
	for _, c := range comments {
		d.Comments = append(d.Comments, toCommentDTO(c))
	}
	return d
}

// toTaskListDTO builds the representation a list response uses: the comment
// count without the comments. A list of a few hundred tasks would otherwise
// carry every comment body ever written, which no list view renders — the
// frontend fetches them per task when a task modal opens.
func toTaskListDTO(t *domain.Task, projectName, epicTitle string, commentCount int) taskDTO {
	d := taskDTO{
		ID:               t.ID,
		Title:            t.Title,
		Description:      t.Description,
		ProjectID:        t.ProjectID,
		ProjectName:      projectName,
		EpicID:           t.EpicID,
		EpicTitle:        epicTitle,
		Status:           string(t.Status),
		Priority:         int(t.Priority),
		Urgency:          int(t.Urgency),
		Quadrant:         string(t.Quadrant()),
		VisibleFrom:      t.VisibleFrom.String(),
		Deadline:         t.Deadline.String(),
		RecurrenceRule:   t.RecurrenceRule,
		RecurrenceMode:   string(t.RecurrenceMode),
		RecurrenceAnchor: t.RecurrenceAnchor.String(),
		RecurrenceLabel:  recurrence.HumanLabel(t.RecurrenceRule, t.RecurrenceMode),
		CreatedAt:        t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		CommentCount:     commentCount,
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		d.CompletedAt = &s
	}
	return d
}

// countsDTO is the sidebar's whole payload: three numbers-per-key maps instead
// of the task list they used to be derived from. Projects and epics without an
// open task are left out — a missing key reads as zero on both sides.
type countsDTO struct {
	Today    int            `json:"today"`
	Projects map[string]int `json:"projects"`
	Epics    map[string]int `json:"epics"`
}

func toCountsDTO(c domain.OpenTaskCounts) countsDTO {
	d := countsDTO{Today: c.VisibleToday, Projects: c.ByProject, Epics: c.ByEpic}
	if d.Projects == nil {
		d.Projects = map[string]int{}
	}
	if d.Epics == nil {
		d.Epics = map[string]int{}
	}
	return d
}

type projectDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	Archived    bool   `json:"archived"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type auditDTO struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	Action    string `json:"action"`
	OldValue  string `json:"old_value,omitempty"`
	NewValue  string `json:"new_value,omitempty"`
	At        string `json:"at"`
}

func toAuditDTO(e *domain.AuditEntry) auditDTO {
	return auditDTO{
		ID:        e.ID,
		TaskID:    e.TaskID,
		TaskTitle: e.TaskTitle,
		Action:    string(e.Action),
		OldValue:  e.OldValue,
		NewValue:  e.NewValue,
		At:        e.At.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type commentDTO struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func toCommentDTO(c *domain.Comment) commentDTO {
	return commentDTO{
		ID:        c.ID,
		TaskID:    c.TaskID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toProjectDTO(p *domain.Project) projectDTO {
	return projectDTO{
		ID: p.ID, Name: p.Name, Slug: p.Slug, Description: p.Description,
		SortOrder: p.SortOrder, Archived: p.Archived, Color: p.Color,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type epicDTO struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	Status      string  `json:"status"`
	StartDate   string  `json:"start_date,omitempty"`
	EndDate     string  `json:"end_date,omitempty"`
	SortOrder   int     `json:"sort_order"`
	Archived    bool    `json:"archived"`
	OpenTasks   int     `json:"open_tasks"`
	DoneTasks   int     `json:"done_tasks"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

func toEpicDTO(e *domain.Epic, projectName string, c domain.EpicCounts) epicDTO {
	d := epicDTO{
		ID:          e.ID,
		Title:       e.Title,
		Slug:        e.Slug,
		Description: e.Description,
		ProjectID:   e.ProjectID,
		ProjectName: projectName,
		Status:      string(e.Status),
		StartDate:   e.Start.String(),
		EndDate:     e.End.String(),
		SortOrder:   e.SortOrder,
		Archived:    e.Archived,
		OpenTasks:   c.Open,
		DoneTasks:   c.Done,
		CreatedAt:   e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   e.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if e.CompletedAt != nil {
		s := e.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		d.CompletedAt = &s
	}
	return d
}

type absenceDTO struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	Comment   string `json:"comment"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// availabilityDTO is the answer of GET /api/availability. `working` and
// `on_business_trip` are independent axes (see domain.Availability);
// `return_date` is only present while work is blocked.
type availabilityDTO struct {
	Date           string       `json:"date"`
	Status         string       `json:"status"`
	Working        bool         `json:"working"`
	OnBusinessTrip bool         `json:"on_business_trip"`
	ReturnDate     string       `json:"return_date,omitempty"`
	Absences       []absenceDTO `json:"absences"`
}

func toAvailabilityDTO(av domain.Availability) availabilityDTO {
	out := availabilityDTO{
		Date:           av.Date.String(),
		Status:         av.Status.String(),
		Working:        av.Working,
		OnBusinessTrip: av.OnBusinessTrip,
		Absences:       make([]absenceDTO, 0, len(av.Absences)),
	}
	if !av.ReturnDate.IsZero() {
		out.ReturnDate = av.ReturnDate.String()
	}
	for _, a := range av.Absences {
		out.Absences = append(out.Absences, toAbsenceDTO(a))
	}
	return out
}

func toAbsenceDTO(a *domain.Absence) absenceDTO {
	return absenceDTO{
		ID:        a.ID,
		Type:      string(a.Type),
		Label:     a.Label,
		Comment:   a.Comment,
		StartDate: a.Start.String(),
		EndDate:   a.End.String(),
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

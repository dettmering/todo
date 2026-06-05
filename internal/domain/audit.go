package domain

import "time"

// AuditAction enumerates the kinds of task changes we record in the audit log.
type AuditAction string

const (
	AuditCreated            AuditAction = "created"
	AuditCompleted          AuditAction = "completed"
	AuditReopened           AuditAction = "reopened"
	AuditDeleted            AuditAction = "deleted"
	AuditVisibleFromSet     AuditAction = "visible_from_set"
	AuditVisibleFromCleared AuditAction = "visible_from_cleared"
	AuditVisibleFromChanged AuditAction = "visible_from_changed"
	AuditDeadlineSet        AuditAction = "deadline_set"
	AuditDeadlineCleared    AuditAction = "deadline_cleared"
	AuditDeadlineChanged    AuditAction = "deadline_changed"
	AuditProjectChanged     AuditAction = "project_changed"
	AuditPriorityChanged    AuditAction = "priority_changed"
)

// AuditEntry is one row in the audit log. OldValue/NewValue are opaque strings
// whose meaning depends on Action:
//
//   - visible_from_*, deadline_*: YYYY-MM-DD or "" for none
//   - project_changed: project name (snapshot at audit time)
//   - priority_changed: priority as "P1".."P4"
//   - created/completed/reopened/deleted: both empty
//
// TaskTitle is snapshotted so the log keeps meaning even when the task title
// later changes or the task itself is hard-deleted.
type AuditEntry struct {
	ID        string
	TaskID    string
	TaskTitle string
	Action    AuditAction
	OldValue  string
	NewValue  string
	At        time.Time
}

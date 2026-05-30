package domain

import "time"

// Comment is a free-text note attached to a task. Body holds the Markdown
// source as entered by the user (or posted via the API); rendering to HTML
// happens in the frontend, mirroring how task descriptions are handled.
type Comment struct {
	ID        string
	TaskID    string
	Body      string
	CreatedAt time.Time
}

package service

import (
	"context"
	"fmt"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

// ensureUniqueSlug returns base if no other project owns it, otherwise base-2,
// base-3, … The optional ownID is excluded from the conflict check (so a
// project keeps its own slug). The reserved value "inbox" is always treated as
// taken unless the caller is the Inbox project itself.
func ensureUniqueSlug(ctx context.Context, repo storage.ProjectRepo, base, ownID string) (string, error) {
	projects, err := repo.List(ctx, true)
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(projects))
	for _, p := range projects {
		if p.ID == ownID {
			continue
		}
		if p.Slug != "" {
			taken[p.Slug] = true
		}
	}
	if ownID != domain.InboxID {
		taken["inbox"] = true
	}
	candidate := base
	for n := 2; taken[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return candidate, nil
}

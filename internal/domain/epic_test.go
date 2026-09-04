package domain

import "testing"

func TestEpicStatusValid(t *testing.T) {
	for _, v := range []EpicStatus{EpicActive, EpicCompleted} {
		if !v.Valid() {
			t.Errorf("%q should be valid", v)
		}
	}
	for _, v := range []EpicStatus{"", "done", "open", "Laufend"} {
		if v.Valid() {
			t.Errorf("%q should be invalid", v)
		}
	}
}

func TestValidEpicRange(t *testing.T) {
	cases := []struct {
		name       string
		start, end string
		want       bool
	}{
		{"both empty", "", "", true},
		{"only start", "2026-09-01", "", true},
		{"only end", "", "2027-03-31", true},
		{"ordered", "2026-09-01", "2027-03-31", true},
		{"same day", "2026-09-01", "2026-09-01", true},
		{"reversed", "2027-03-31", "2026-09-01", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var start, end LocalDate
			if c.start != "" {
				start = mustDate(t, c.start)
			}
			if c.end != "" {
				end = mustDate(t, c.end)
			}
			if got := ValidEpicRange(start, end); got != c.want {
				t.Errorf("ValidEpicRange(%q, %q) = %v, want %v", c.start, c.end, got, c.want)
			}
		})
	}
}

func TestEpicCovers(t *testing.T) {
	bounded := Epic{Start: mustDate(t, "2026-09-01"), End: mustDate(t, "2026-09-30")}
	openStart := Epic{End: mustDate(t, "2026-09-30")}
	openEnd := Epic{Start: mustDate(t, "2026-09-01")}
	unbounded := Epic{}

	cases := []struct {
		name string
		e    Epic
		date string
		want bool
	}{
		{"inside", bounded, "2026-09-15", true},
		{"on start", bounded, "2026-09-01", true},
		{"on end", bounded, "2026-09-30", true},
		{"before", bounded, "2026-08-31", false},
		{"after", bounded, "2026-10-01", false},
		// An open side never excludes anything — an initiative without a fixed
		// beginning has always been running.
		{"open start, long before", openStart, "2020-01-01", true},
		{"open start, after end", openStart, "2026-10-01", false},
		{"open end, long after", openEnd, "2099-01-01", true},
		{"open end, before start", openEnd, "2026-08-31", false},
		{"no dates at all", unbounded, "2026-09-15", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.e.Covers(mustDate(t, c.date)); got != c.want {
				t.Errorf("Covers(%s) = %v, want %v", c.date, got, c.want)
			}
		})
	}
}

func TestEpicAssignable(t *testing.T) {
	cases := []struct {
		name string
		e    Epic
		want bool
	}{
		{"running", Epic{Status: EpicActive}, true},
		{"finished", Epic{Status: EpicCompleted}, false},
		{"archived", Epic{Status: EpicActive, Archived: true}, false},
		{"finished and archived", Epic{Status: EpicCompleted, Archived: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.e.Assignable(); got != c.want {
				t.Errorf("Assignable() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestCountEpicTasks(t *testing.T) {
	tasks := []*Task{
		{ID: "a", EpicID: "e1", Status: StatusOpen},
		{ID: "b", EpicID: "e1", Status: StatusCompleted},
		{ID: "c", EpicID: "e1", Status: StatusDeleted}, // thrown away, counts nowhere
		{ID: "d", EpicID: "e2", Status: StatusCompleted},
		{ID: "e", EpicID: "", Status: StatusOpen}, // no epic at all
	}
	got := CountEpicTasks(tasks)
	if len(got) != 2 {
		t.Fatalf("got %d epics, want 2: %+v", len(got), got)
	}
	if got["e1"] != (EpicCounts{Open: 1, Done: 1}) {
		t.Errorf("e1 = %+v, want {Open:1 Done:1}", got["e1"])
	}
	if got["e2"] != (EpicCounts{Open: 0, Done: 1}) {
		t.Errorf("e2 = %+v, want {Open:0 Done:1}", got["e2"])
	}
	if n := got["e1"].Total(); n != 2 {
		t.Errorf("e1 total = %d, want 2 (deleted task excluded)", n)
	}
}

func TestEpicCanComplete(t *testing.T) {
	cases := []struct {
		name  string
		tasks []*Task
		want  bool
	}{
		{"no tasks at all", nil, true},
		{"only completed", []*Task{{Status: StatusCompleted}, {Status: StatusCompleted}}, true},
		// A soft-deleted task is not work left to do, so it must not block.
		{"completed plus deleted", []*Task{{Status: StatusCompleted}, {Status: StatusDeleted}}, true},
		{"one still open", []*Task{{Status: StatusCompleted}, {Status: StatusOpen}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EpicCanComplete(c.tasks); got != c.want {
				t.Errorf("EpicCanComplete = %v, want %v", got, c.want)
			}
		})
	}
}

func TestOpenTaskCount(t *testing.T) {
	tasks := []*Task{
		{Status: StatusOpen}, {Status: StatusOpen},
		{Status: StatusCompleted}, {Status: StatusDeleted},
	}
	if got := OpenTaskCount(tasks); got != 2 {
		t.Errorf("OpenTaskCount = %d, want 2", got)
	}
}

func TestActiveEpics(t *testing.T) {
	running := &Epic{ID: "1", Title: "Bad", ProjectID: "p1", Status: EpicActive, SortOrder: 1}
	early := &Epic{ID: "2", Title: "Küche", ProjectID: "p1", Status: EpicActive, SortOrder: 0}
	sameOrder := &Epic{ID: "3", Title: "Ahorn", ProjectID: "p1", Status: EpicActive, SortOrder: 1}
	done := &Epic{ID: "4", Title: "Dach", ProjectID: "p1", Status: EpicCompleted}
	archived := &Epic{ID: "5", Title: "Keller", ProjectID: "p1", Status: EpicActive, Archived: true}
	other := &Epic{ID: "6", Title: "Audit", ProjectID: "p2", Status: EpicActive}

	all := []*Epic{running, early, sameOrder, done, archived, other}

	t.Run("one project", func(t *testing.T) {
		got := ActiveEpics(all, "p1")
		want := []string{"2", "3", "1"} // sort_order 0, then 1 with "Ahorn" before "Bad"
		if len(got) != len(want) {
			t.Fatalf("got %d epics, want %d: %+v", len(got), len(want), got)
		}
		for i, id := range want {
			if got[i].ID != id {
				t.Errorf("position %d = %q (%s), want %q", i, got[i].ID, got[i].Title, id)
			}
		}
	})

	t.Run("every project", func(t *testing.T) {
		if got := ActiveEpics(all, ""); len(got) != 4 {
			t.Errorf("got %d epics, want 4 (all running, both projects)", len(got))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if got := ActiveEpics(nil, "p1"); len(got) != 0 {
			t.Errorf("got %d epics, want 0", len(got))
		}
	})
}

package domain

// Urgency is the second axis of the Eisenhower matrix, next to Priority
// (importance). Same scale as Priority: 1 = most urgent, 4 = least. It is set
// by hand, not derived from Deadline.
type Urgency int

const (
	U1 Urgency = 1
	U2 Urgency = 2
	U3 Urgency = 3
	U4 Urgency = 4
)

func (u Urgency) Valid() bool { return u >= U1 && u <= U4 }

// Urgent reports which side of the urgency axis u falls on: U1/U2 are urgent.
func (u Urgency) Urgent() bool { return u == U1 || u == U2 }

// Important reports which side of the importance axis p falls on: P1/P2 are
// important.
func (p Priority) Important() bool { return p == P1 || p == P2 }

// Quadrant is a cell of the Eisenhower matrix.
type Quadrant string

const (
	QuadrantDo        Quadrant = "do"        // important + urgent: sofort erledigen
	QuadrantSchedule  Quadrant = "schedule"  // important, not urgent: einplanen
	QuadrantDelegate  Quadrant = "delegate"  // urgent, not important: delegieren
	QuadrantEliminate Quadrant = "eliminate" // neither: streichen
)

func (q Quadrant) Valid() bool {
	switch q {
	case QuadrantDo, QuadrantSchedule, QuadrantDelegate, QuadrantEliminate:
		return true
	}
	return false
}

// QuadrantOf places a priority/urgency pair in the matrix.
func QuadrantOf(p Priority, u Urgency) Quadrant {
	switch {
	case p.Important() && u.Urgent():
		return QuadrantDo
	case p.Important():
		return QuadrantSchedule
	case u.Urgent():
		return QuadrantDelegate
	}
	return QuadrantEliminate
}

// Quadrant places the task in the Eisenhower matrix.
func (t Task) Quadrant() Quadrant { return QuadrantOf(t.Priority, t.Urgency) }

// Axes returns the importance and urgency side a quadrant stands for.
func (q Quadrant) Axes() (important, urgent bool) {
	switch q {
	case QuadrantDo:
		return true, true
	case QuadrantSchedule:
		return true, false
	case QuadrantDelegate:
		return false, true
	}
	return false, false
}

// stepForSide moves a 1..4 rating to the requested half of its scale. A rating
// already on that side is kept — dragging a P1 from "do" to "schedule" must not
// demote it to P2. Crossing the middle lands on the step nearest to it (2 or
// 3), so a drag changes the rating as little as the matrix allows.
func stepForSide(current int, high bool) int {
	if current >= 1 && current <= 4 && (current <= 2) == high {
		return current
	}
	if high {
		return 2
	}
	return 3
}

// PriorityForImportance returns the priority a task carries after being moved
// to the given side of the importance axis.
func PriorityForImportance(current Priority, important bool) Priority {
	return Priority(stepForSide(int(current), important))
}

// UrgencyForUrgent returns the urgency a task carries after being moved to the
// given side of the urgency axis.
func UrgencyForUrgent(current Urgency, urgent bool) Urgency {
	return Urgency(stepForSide(int(current), urgent))
}

// MoveToQuadrant returns the priority and urgency a task needs to sit in q,
// changing each rating only as far as necessary.
func MoveToQuadrant(p Priority, u Urgency, q Quadrant) (Priority, Urgency) {
	imp, urg := q.Axes()
	return PriorityForImportance(p, imp), UrgencyForUrgent(u, urg)
}

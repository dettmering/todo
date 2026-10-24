package domain

import "testing"

func TestQuadrantOf(t *testing.T) {
	cases := []struct {
		p    Priority
		u    Urgency
		want Quadrant
	}{
		{P1, U1, QuadrantDo},
		{P2, U2, QuadrantDo},
		{P1, U3, QuadrantSchedule},
		{P2, U4, QuadrantSchedule},
		{P3, U1, QuadrantDelegate},
		{P4, U2, QuadrantDelegate},
		{P3, U3, QuadrantEliminate},
		{P4, U4, QuadrantEliminate},
	}
	for _, c := range cases {
		if got := QuadrantOf(c.p, c.u); got != c.want {
			t.Errorf("QuadrantOf(P%d, U%d) = %q, want %q", c.p, c.u, got, c.want)
		}
	}
}

func TestMoveToQuadrant(t *testing.T) {
	cases := []struct {
		name  string
		p     Priority
		u     Urgency
		q     Quadrant
		wantP Priority
		wantU Urgency
	}{
		{"already there keeps both", P1, U1, QuadrantDo, P1, U1},
		{"P1 to schedule keeps P1, drops urgency to U3", P1, U1, QuadrantSchedule, P1, U3},
		{"eliminate to do crosses both axes minimally", P4, U4, QuadrantDo, P2, U2},
		{"do to eliminate crosses both axes minimally", P1, U1, QuadrantEliminate, P3, U3},
		{"delegate to schedule swaps axes", P4, U1, QuadrantSchedule, P2, U3},
		{"U4 stays U4 when not urgent", P2, U4, QuadrantSchedule, P2, U4},
		{"invalid ratings land next to the middle", 0, 0, QuadrantDelegate, P3, U2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotP, gotU := MoveToQuadrant(c.p, c.u, c.q)
			if gotP != c.wantP || gotU != c.wantU {
				t.Errorf("got P%d/U%d, want P%d/U%d", gotP, gotU, c.wantP, c.wantU)
			}
			if QuadrantOf(gotP, gotU) != c.q {
				t.Errorf("result P%d/U%d is not in %q", gotP, gotU, c.q)
			}
		})
	}
}

func TestUrgencyValid(t *testing.T) {
	for u, want := range map[Urgency]bool{0: false, U1: true, U4: true, 5: false} {
		if u.Valid() != want {
			t.Errorf("Urgency(%d).Valid() = %v", u, !want)
		}
	}
}

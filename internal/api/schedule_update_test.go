package api_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// scheduleFixture: an admin, two children, one chore with one Wednesday
// schedule for the first child. Returns the children's IDs and the schedule's.
func scheduleFixture(t *testing.T, env *testEnv) (int, int, int) {
	t.Helper()
	env.createAdmin(t)
	kid := env.createChild(t, "Kid")
	sib := env.createChild(t, "Sib")
	env.expectStatus(t, "POST", "/api/chores", map[string]any{
		"title": "Walk the dog", "category": "core",
	}, adminHeaders(), http.StatusCreated)
	resp := env.expectStatus(t, "POST", "/api/chores/1/schedules", map[string]any{
		"assigned_to": kid, "day_of_week": 3, "due_by": "18:00",
	}, adminHeaders(), http.StatusCreated)
	var s map[string]any
	decodeBody(t, resp, &s)
	return kid, sib, int(s["id"].(float64))
}

func TestUpdateScheduleReassignsAndKeepsHistory(t *testing.T) {
	env := setupTest(t)
	kid, sib, sid := scheduleFixture(t, env)

	// Kid walks the dog on Wednesday 2026-03-11.
	env.expectStatus(t, "POST", fmt.Sprintf("/api/schedules/%d/complete", sid), map[string]any{
		"completed_by": kid, "completion_date": "2026-03-11",
	}, adminHeaders(), http.StatusCreated)

	// From now on it is Sib's, on Thursdays, due later.
	resp := env.expectStatus(t, "PUT", fmt.Sprintf("/api/chores/1/schedules/%d", sid), map[string]any{
		"assigned_to": sib, "day_of_week": 4, "due_by": "19:30",
	}, adminHeaders(), http.StatusOK)
	var got map[string]any
	decodeBody(t, resp, &got)
	if int(got["assigned_to"].(float64)) != sib || int(got["day_of_week"].(float64)) != 4 || got["due_by"] != "19:30" {
		t.Fatalf("update not reflected in response: %v", got)
	}
	if int(got["id"].(float64)) != sid {
		t.Fatalf("schedule id changed: %v", got["id"])
	}

	stored, err := env.store.GetSchedule(context.Background(), int64(sid))
	if err != nil || stored == nil {
		t.Fatalf("get schedule: %v", err)
	}
	if stored.AssignedTo != int64(sib) || *stored.DayOfWeek != 4 || stored.AssignmentType != "individual" {
		t.Fatalf("stored schedule not updated: %+v", stored)
	}

	// The completion survived the edit, still credited to Kid.
	cc, err := env.store.GetCompletionForScheduleDate(context.Background(), int64(sid), "2026-03-11")
	if err != nil || cc == nil {
		t.Fatalf("completion history lost: %v", err)
	}
	if cc.CompletedBy != int64(kid) {
		t.Fatalf("past completion re-attributed: completed_by=%d, want %d", cc.CompletedBy, kid)
	}

	// Sib now has it on a Thursday; Kid no longer does on a Wednesday.
	var sibChores, kidChores []map[string]any
	decodeBody(t, env.expectStatus(t, "GET", fmt.Sprintf("/api/users/%d/chores?date=2026-03-12", sib), nil, nil, http.StatusOK), &sibChores)
	decodeBody(t, env.expectStatus(t, "GET", fmt.Sprintf("/api/users/%d/chores?date=2026-03-18", kid), nil, nil, http.StatusOK), &kidChores)
	if len(sibChores) != 1 || len(kidChores) != 0 {
		t.Fatalf("schedule not moved: sib Thu=%d, kid Wed=%d", len(sibChores), len(kidChores))
	}
}

func TestUpdateScheduleValidation(t *testing.T) {
	env := setupTest(t)
	_, sib, sid := scheduleFixture(t, env)
	path := fmt.Sprintf("/api/chores/1/schedules/%d", sid)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"no assignee", map[string]any{"day_of_week": 1}, http.StatusBadRequest},
		{"no recurrence", map[string]any{"assigned_to": sib}, http.StatusBadRequest},
		{"unknown user", map[string]any{"assigned_to": 999, "day_of_week": 1}, http.StatusBadRequest},
		{"bad penalty", map[string]any{"assigned_to": sib, "day_of_week": 1, "expiry_penalty": "nope"}, http.StatusBadRequest},
		{"type change", map[string]any{"assigned_to": sib, "day_of_week": 1, "assignment_type": "fcfs"}, http.StatusBadRequest},
		{"same type ok", map[string]any{"assigned_to": sib, "day_of_week": 1, "assignment_type": "individual"}, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env.expectStatus(t, "PUT", path, c.body, adminHeaders(), c.want)
		})
	}

	// The schedule must belong to the chore in the path.
	env.expectStatus(t, "POST", "/api/chores", map[string]any{"title": "Other", "category": "core"}, adminHeaders(), http.StatusCreated)
	env.expectStatus(t, "PUT", fmt.Sprintf("/api/chores/2/schedules/%d", sid),
		map[string]any{"assigned_to": sib, "day_of_week": 1}, adminHeaders(), http.StatusNotFound)
	env.expectStatus(t, "PUT", "/api/chores/1/schedules/9999",
		map[string]any{"assigned_to": sib, "day_of_week": 1}, adminHeaders(), http.StatusNotFound)
}

func TestUpdateScheduleIsAdminOnly(t *testing.T) {
	env := setupTest(t)
	kid, _, sid := scheduleFixture(t, env)
	env.expectStatus(t, "PUT", fmt.Sprintf("/api/chores/1/schedules/%d", sid),
		map[string]any{"assigned_to": kid, "day_of_week": 1}, childHeaders(kid), http.StatusForbidden)
}

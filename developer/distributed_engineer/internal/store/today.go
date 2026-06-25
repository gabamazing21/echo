package store

import (
	"context"
	"sort"
	"time"

	"consensus/internal/models"
)

const targetHours = 8 // your daily devotion

// Dashboard computes the mission-control snapshot.
func (s *Store) Dashboard(ctx context.Context) (*models.Dashboard, error) {
	steps, err := s.Steps(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := s.AllTasks(ctx)
	if err != nil {
		return nil, err
	}

	d := &models.Dashboard{Steps: steps, StepsTotal: len(steps)}

	var hoursTot, hoursDone, doneCount int
	var lastDeadline time.Time
	for _, t := range tasks {
		hoursTot += t.Hrs
		if t.Done {
			hoursDone += t.Hrs
			doneCount++
		}
		if t.Deadline.After(lastDeadline) {
			lastDeadline = t.Deadline
		}
	}
	d.TasksTotal = len(tasks)
	d.TasksDone = doneCount
	d.HoursTotal = hoursTot
	d.HoursDone = hoursDone
	if len(tasks) > 0 {
		d.OverallPct = int(float64(doneCount)/float64(len(tasks))*100 + 0.5)
	}
	for _, st := range steps {
		if st.Full() {
			d.StepsOnline++
		}
	}
	if !lastDeadline.IsZero() {
		d.DaysToFinish = daysUntil(lastDeadline)
	}

	d.Streak, err = s.streak(ctx)
	if err != nil {
		return nil, err
	}

	// Up next: the five nearest-deadline incomplete tasks.
	var open []models.Task
	for _, t := range tasks {
		if !t.Done {
			open = append(open, t)
		}
	}
	sort.SliceStable(open, func(i, j int) bool { return open[i].Deadline.Before(open[j].Deadline) })
	if len(open) > 5 {
		open = open[:5]
	}
	d.UpNext = open
	return d, nil
}

// Today builds the guided daily block: the next incomplete tasks in curriculum
// order, accumulated until ~targetHours, plus streak/studied state.
func (s *Store) Today(ctx context.Context) (*models.TodayPlan, error) {
	tasks, err := s.AllTasks(ctx) // already ordered by step, position
	if err != nil {
		return nil, err
	}
	steps, err := s.Steps(ctx)
	if err != nil {
		return nil, err
	}
	stepByID := map[int]models.Step{}
	for _, st := range steps {
		stepByID[st.ID] = st
	}

	plan := &models.TodayPlan{Date: time.Now(), TargetHrs: targetHours}

	var hrs int
	for _, t := range tasks {
		if t.Done {
			continue
		}
		if plan.CurrentStep == nil {
			st := stepByID[t.StepID]
			plan.CurrentStep = &st
		}
		plan.Items = append(plan.Items, t)
		hrs += t.Hrs
		if hrs >= targetHours {
			break
		}
	}
	plan.PlannedHrs = hrs

	plan.Streak, err = s.streak(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM study_log WHERE day=CURRENT_DATE)`).Scan(&plan.StudiedToday); err != nil {
		return nil, err
	}
	return plan, nil
}

// streak counts consecutive studied days ending today (or yesterday).
func (s *Store) streak(ctx context.Context) (int, error) {
	rows, err := s.pool.Query(ctx, `SELECT day FROM study_log ORDER BY day DESC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	studied := map[string]bool{}
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		studied[d.Format("2006-01-02")] = true
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	n := 0
	day := time.Now()
	for studied[day.Format("2006-01-02")] {
		n++
		day = day.AddDate(0, 0, -1)
	}
	return n, nil
}

func daysUntil(t time.Time) int {
	d := time.Until(t)
	return int((d + 24*time.Hour - time.Nanosecond) / (24 * time.Hour))
}

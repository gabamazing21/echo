// Package models holds the domain types shared across the app.
package models

import "time"

// Step is one of the ten stages of the roadmap, with progress aggregates.
type Step struct {
	ID         int
	Phase      string
	Position   int
	TotalTasks int
	DoneTasks  int
	Deadline   time.Time // derived: latest task deadline in the step
}

// Frac returns completion in [0,1].
func (s Step) Frac() float64 {
	if s.TotalTasks == 0 {
		return 0
	}
	return float64(s.DoneTasks) / float64(s.TotalTasks)
}

// Pct returns completion as a rounded percentage.
func (s Step) Pct() int { return int(s.Frac()*100 + 0.5) }

// Full reports whether every task in the step is done.
func (s Step) Full() bool { return s.TotalTasks > 0 && s.DoneTasks == s.TotalTasks }

// Task is a single unit of work in a step, with the learner's progress merged in.
type Task struct {
	ID         int64
	StepID     int
	Phase      string
	Week       string
	Kind       string // Topic | Code project | Quiz | Practice | Task
	Title      string
	Concept    string
	Hrs        int
	Deadline   time.Time
	LessonSlug string // "" if no long-form lesson attached
	Position   int

	Done      bool
	Status    string
	Notes     string
	HasLesson bool // whether a lessons row exists for LessonSlug
}

// Resource is a curated external link for a step.
type Resource struct {
	ID        int64
	StepID    int
	Name      string
	Kind      string
	IsFree    bool
	Link      string
	Why       string
	TimeLabel string
	Position  int
}

// Quiz is one question (theory or code) plus the learner's saved attempt.
type Quiz struct {
	ID          int64
	StepID      int
	Kind        string // Theory | Code
	Question    string
	Options     string
	Answer      string
	Explanation string
	Position    int

	Progress QuizProgress
}

// QuizProgress is the learner's saved state for a quiz.
type QuizProgress struct {
	AnswerGiven string
	Correct     bool
	SelfDone    bool
	Revealed    bool
	Answered    bool
}

// Lesson is an authored long-form teaching unit, rendered to HTML.
type Lesson struct {
	ID       int64
	Slug     string
	StepID   int
	Title    string
	Summary  string
	BodyHTML string
	EstMin   int
	Position int
}

// Challenge is an auto-gradable coding exercise. TestCode is server-only and is
// never rendered to the client.
type Challenge struct {
	ID          int64
	StepID      int
	Slug        string
	Title       string
	Prompt      string
	StarterCode string
	TestCode    string
	Position    int

	LastCode   string // most recent submission's code, if any
	LastPassed bool
	HasLast    bool
}

// Submission is one run of a challenge.
type Submission struct {
	ID          int64
	ChallengeID int64
	Code        string
	Passed      bool
	Output      string
	CreatedAt   time.Time
}

// InterviewQuestion is a curated interview question attached to a lesson, with
// the learner's progress and (for coding) a resolved in-app challenge.
type InterviewQuestion struct {
	ID              int64
	Slug            string
	LessonSlug      string
	StepID          int
	Kind            string // coding | sql | conceptual | design | behavioral
	Difficulty      string // easy | medium | hard | ""
	Prompt          string
	Companies       string
	SourceName      string
	SourceURL       string
	ModelAnswerHTML string
	ChallengeSlug   string
	Position        int

	Revealed bool
	Outcome  string // unset | got_it | review

	HasChallenge  bool // a code_challenges row exists for ChallengeSlug
	ChallengeStep int  // its step, for the /checker/<step>#<slug> deep link
}

// Dashboard is the computed "mission control" snapshot.
type Dashboard struct {
	OverallPct   int
	TasksDone    int
	TasksTotal   int
	HoursDone    int
	HoursTotal   int
	Streak       int
	StepsOnline  int
	StepsTotal   int
	DaysToFinish int
	Steps        []Step
	UpNext       []Task
}

// TodayPlan is the guided daily block — what to do in today's ~8 hours.
type TodayPlan struct {
	Date         time.Time
	StudiedToday bool
	MinutesToday int
	Streak       int
	TargetHrs    int
	PlannedHrs   int
	CurrentStep  *Step
	Items        []Task
}

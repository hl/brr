package workflow

import (
	"time"

	"github.com/hl/brr/internal/config"
)

const (
	SchemaVersion = 2

	StageTypeAgent   = "agent"
	StageTypeCommand = "command"

	StateDir = ".brr/state/workflows"
)

type Defaults struct {
	Profile string `yaml:"profile,omitempty"`
	Max     int    `yaml:"max,omitempty"`
}

type Cycle struct {
	Target string `yaml:"target"`
	Max    int    `yaml:"max"`
}

type Stage struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Prompt  string   `yaml:"prompt,omitempty"`
	Max     int      `yaml:"max,omitempty"`
	Profile string   `yaml:"profile,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

type Workflow struct {
	Version     int      `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Defaults    Defaults `yaml:"defaults,omitempty"`
	Cycle       *Cycle   `yaml:"cycle,omitempty"`
	Stages      []Stage  `yaml:"stages"`
}

type Options struct {
	Name          string
	Workflow      Workflow
	Config        config.Config
	ProfileFlag   string
	ResolvePrompt func(string) (string, error)
	Notify        func()
	Reset         bool
}

type State struct {
	SchemaVersion int           `json:"schema_version"`
	Workflow      string        `json:"workflow"`
	RunID         string        `json:"run_id"`
	StartedAt     time.Time     `json:"started_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	StartSHA      string        `json:"start_sha"`
	NextStageID   string        `json:"next_stage_id"`
	CycleCount    int           `json:"cycle_count"`
	Stages        []StageStatus `json:"stages"`
}

type StageStatus struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Status   string        `json:"status"`
	Reason   string        `json:"reason,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Profile  string        `json:"profile,omitempty"`
	Prompt   string        `json:"prompt,omitempty"`
	Command  []string      `json:"command,omitempty"`
}

type Event struct {
	RunID    string    `json:"run_id"`
	Workflow string    `json:"workflow"`
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
	StageID  string    `json:"stage_id,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Message  string    `json:"message,omitempty"`
}

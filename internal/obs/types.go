package obs

import "time"

type SceneInfo struct {
	Name    string
	Index   int
	Sources []SourceInfo
}

type SourceInfo struct {
	Name    string
	Type    string
	Enabled bool
	Visible bool
}

type ReplayBufferInfo struct {
	Active     bool
	Duration   time.Duration
	OutputPath string
	LastSaved  time.Time
}

type TextSourceSettings struct {
	Text            string
	FontSize        int
	Color           string
	BackgroundColor string
	Bold            bool
	Italic          bool
	WordWrap        bool
	Outline         bool
	OutlineColor    string
	OutlineSize     int
}

type EventType string

const (
	EventSceneChanged      EventType = "scene_changed"
	EventReplayBufferSaved EventType = "replay_buffer_saved"
	EventConnectionLost    EventType = "connection_lost"
	EventSourceVisibility  EventType = "source_visibility"
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      any
}

package types

type EventType int

const (
	EventText EventType = iota
	EventToolCall
	EventToolResult
	EventToolSkipped
	EventToolConfirm
	EventError
	EventDone
)

type Event struct {
	Type     EventType `json:"type"`
	Text     string    `json:"text,omitempty"`
	ToolName string    `json:"tool_name,omitempty"`
	ToolArgs string    `json:"tool_args,omitempty"`
	ToolID   string    `json:"tool_id,omitempty"`
	Result   string    `json:"result,omitempty"`
	Err      error     `json:"-"`
	ReplyCh  chan bool `json:"-"`
}

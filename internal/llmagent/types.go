package llmagent

import "encoding/json"

const (
	ActionTool  = "tool"
	ActionFinal = "final"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Decision struct {
	Action      string                 `json:"action"`
	Tool        string                 `json:"tool,omitempty"`
	Args        map[string]interface{} `json:"args,omitempty"`
	FinalAnswer string                 `json:"final_answer,omitempty"`
	Thought     string                 `json:"thought,omitempty"`
}

func (d Decision) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

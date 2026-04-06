package tui

import (
	"encoding/json"
	"fmt"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func marshalDebugJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("(json error: %v)\n%+v", err, v)
	}
	return string(b)
}

// formatDebugSendMessageResult returns indented JSON for a non-streaming SendMessage result.
func formatDebugSendMessageResult(res a2a.SendMessageResult) string {
	switch v := res.(type) {
	case *a2a.Message:
		return marshalDebugJSON(struct {
			ResultKind string `json:"resultKind"`
			*a2a.Message
		}{ResultKind: "message", Message: v})
	case *a2a.Task:
		return marshalDebugJSON(struct {
			ResultKind string `json:"resultKind"`
			*a2a.Task
		}{ResultKind: "task", Task: v})
	default:
		return fmt.Sprintf("unexpected result type %T\n%+v", res, res)
	}
}

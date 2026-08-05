package clarity

import "encoding/json"

type ErrorResponse struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Details      string `json:"details"`
}

type Cursor[T any] struct {
	PageSize int64   `json:"pageSize"`
	HasMore  bool    `json:"hasMore"`
	Previous *string `json:"previous,omitempty"`
	Next     *string `json:"next,omitempty"`
	Data     []T     `json:"data"`
}

type CursorResponse[T any] struct {
	Cursor Cursor[T] `json:"cursor"`
}

type PageCursor struct {
	PageSize int64   `json:"pageSize"`
	HasMore  bool    `json:"hasMore"`
	Previous *string `json:"previous,omitempty"`
	Next     *string `json:"next,omitempty"`
}

func Page[T any](cursor Cursor[T]) PageCursor {
	return PageCursor{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
	}
}

type DataResponse[T any] struct {
	Data T `json:"data"`
}

type Rule struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	TemplateKind   string            `json:"templateKind"`
	TemplateSpec   json.RawMessage   `json:"templateSpec"`
	ExplanationCEL string            `json:"explanationCEL,omitempty"`
	Enabled        bool              `json:"enabled"`
	Severity       string            `json:"severity"`
	Cadence        string            `json:"cadence"`
	Schedule       json.RawMessage   `json:"schedule,omitempty"`
	Notifications  []string          `json:"notifications,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
}

type Evaluation struct {
	ID           string            `json:"id"`
	RuleID       string            `json:"ruleID"`
	StartedAt    string            `json:"startedAt"`
	EndedAt      string            `json:"endedAt"`
	PITPerSource map[string]string `json:"pitPerSource,omitempty"`
	Result       string            `json:"result"`
	Evidence     json.RawMessage   `json:"evidence,omitempty"`
	Error        string            `json:"error,omitempty"`
	CostUnits    int64             `json:"costUnits,omitempty"`
	CreatedAt    string            `json:"createdAt"`
}

type Alert struct {
	ID               string            `json:"id"`
	RuleID           string            `json:"ruleID"`
	Fingerprint      string            `json:"fingerprint"`
	PeriodID         string            `json:"periodID"`
	Status           string            `json:"status"`
	Severity         string            `json:"severity"`
	FirstSeenAt      string            `json:"firstSeenAt"`
	LastSeenAt       string            `json:"lastSeenAt"`
	OccurrenceCount  int64             `json:"occurrenceCount"`
	LastEvaluationID string            `json:"lastEvaluationID"`
	Evidence         json.RawMessage   `json:"evidence,omitempty"`
	Ack              json.RawMessage   `json:"ack,omitempty"`
	Resolution       json.RawMessage   `json:"resolution,omitempty"`
	Snooze           json.RawMessage   `json:"snooze,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

type AlertEvent struct {
	ID           string          `json:"id"`
	AlertID      string          `json:"alertID"`
	EvaluationID *string         `json:"evaluationID,omitempty"`
	Type         string          `json:"type"`
	Previous     *string         `json:"prevStatus,omitempty"`
	NewStatus    string          `json:"newStatus"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	At           string          `json:"at"`
	IsReopen     bool            `json:"isReopen"`
	Notify       bool            `json:"notify"`
}

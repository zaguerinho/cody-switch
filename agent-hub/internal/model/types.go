package model

import "time"

// MessageType classifies messages in a session.
type MessageType string

const (
	MessageQuestion     MessageType = "question"
	MessageAnswer       MessageType = "answer"
	MessageRFC          MessageType = "rfc"
	MessageNote         MessageType = "note"
	MessageStatusUpdate MessageType = "status-update"
)

// ValidMessageTypes lists all accepted message types.
var ValidMessageTypes = []MessageType{
	MessageQuestion, MessageAnswer, MessageRFC, MessageNote, MessageStatusUpdate,
}

// IsValid checks whether the message type is recognized.
func (mt MessageType) IsValid() bool {
	for _, v := range ValidMessageTypes {
		if mt == v {
			return true
		}
	}
	return false
}

// RoomMeta holds metadata for a coordination session.
type RoomMeta struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Archived    bool       `json:"archived"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// Agent represents a named participant in a session.
type Agent struct {
	Alias    string    `json:"alias"`
	Role     string    `json:"role,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
}

// AgentList is the on-disk format for agents.json.
type AgentList struct {
	Agents []Agent `json:"agents"`
}

// Message is a single communication in a session.
type Message struct {
	ID        int         `json:"id"`
	From      string      `json:"from"`
	Type      MessageType `json:"type"`
	Subject   string      `json:"subject,omitempty"`
	Body      string      `json:"body"`
	Timestamp time.Time   `json:"timestamp"`
}

// CursorMap tracks per-agent read positions.
type CursorMap struct {
	Cursors map[string]int `json:"cursors"`
}

// StatusEntry is a single key-value pair on the status board.
type StatusEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StatusBoard is the on-disk format for status.json.
type StatusBoard struct {
	Entries []StatusEntry `json:"entries"`
}

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// RoomInfo extends RoomMeta with runtime counts for API responses.
type RoomInfo struct {
	RoomMeta
	AgentCount   int        `json:"agent_count"`
	MessageCount int        `json:"message_count"`
	LastActivity *time.Time `json:"last_activity,omitempty"`
}

// CheckResult summarizes unread messages for a single room.
type CheckResult struct {
	Room     string `json:"room"`
	Unread   int    `json:"unread"`
	Latest   string `json:"latest,omitempty"`
	LatestID int    `json:"latest_id,omitempty"`
}

// CheckAllResult is returned by the check-all endpoint.
type CheckAllResult struct {
	Rooms []CheckResult `json:"rooms"`
}

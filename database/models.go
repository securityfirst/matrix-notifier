package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type table interface {
	name() string
	unique() [][]string
}

// Org is a group of Users.
type Org struct {
	RoomID  string `db:"room_id,primarykey" json:"room_id"`
	Name    string `db:"name" json:"name"`
	Package string `db:"package" json:"package"`
	Intent  string `db:"intent" json:"intent"`
}

func (Org) name() string { return "organisations" }

func (Org) unique() [][]string {
	return [][]string{{"room_id"}, {"name"}, {"package"}}
}

// Notification is the notification model.
type Notification struct {
	ID        string    `db:"id,primarykey" json:"id"`
	RoomID    string    `db:"room_id" json:"room_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Priority  int       `db:"priority" json:"priority"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Type      string    `db:"type" json:"type"`
	Content   *Content  `db:"content" json:"content"`
	Read      bool      `db:"-" json:"read,omitempty"`
}

func (Notification) name() string { return "notifications" }

func (Notification) unique() [][]string {
	return [][]string{{"id"}}
}

// Content is the Notification main content.
type Content struct {
	Text        string   `json:"text"`
	CollapseKey string   `json:"collapse_key,omitempty"`
	RefID       string   `json:"ref_id,omitempty"`
	Choices     []Choice `json:"choices,omitempty"`
}

// Choice is an option for an Answer or a Pool
type Choice struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

// Value encodes a sql value
func (c *Content) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan decodes a sql value
func (c *Content) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), c)
}

// NotificationUser marks read Notification for a User
type NotificationUser struct {
	UserID   string    `db:"user_id,primarykey"`
	LastRead time.Time `db:"last_read"`
}

func (NotificationUser) name() string { return "notification_users" }

func (NotificationUser) unique() [][]string {
	return [][]string{{"user_id"}}
}

package database

import (
	"database/sql/driver"
	"encoding/json"
)

type table interface {
	name() string
	unique() [][]string
}

// Organisation is a group of Users.
type Organisation struct {
	ID      string `db:"id,primarykey"`
	Name    string `db:"name"`
	Package string `db:"package"`
	Intent  string `db:"intent"`
	Enabled bool   `db:"enabled"`
}

func (Organisation) name() string { return "organisation" }

func (Organisation) unique() [][]string {
	return [][]string{[]string{"id"}, []string{"name"}, []string{"package"}}
}

// OrganisationUser is the User connection to an Organisation.
type OrganisationUser struct {
	OrganisationID string `db:"organisation_id"`
	UserID         string `db:"user_id"`
	Hash           string `db:"hash"`
	Level          int    `db:"admin"`
}

func (OrganisationUser) name() string { return "organisation_user" }

func (OrganisationUser) unique() [][]string {
	return [][]string{[]string{"hash"}, []string{"organisation_id", "user_id"}}
}

// UserThreepid is a third party identity for a User.
type UserThreepid struct {
	UserID      string `db:"user_id"`
	Medium      string `db:"medium"`
	Address     string `db:"address"`
	ValidatedAt int64  `db:"validated_at"`
	AddedAt     int64  `db:"added_at"`
}

func (UserThreepid) name() string { return "user_threepids" }

func (UserThreepid) unique() [][]string { return nil }

// Notification is the notification model.
type Notification struct {
	ID          string   `db:"id,primarykey" json:"id"`
	UserID      string   `db:"user_id" json:"user_id"`
	Destination string   `db:"destination" json:"destination"`
	Priority    int      `db:"priority" json:"priority"`
	CreatedAt   int64    `db:"created_at" json:"created_at"`
	Type        string   `db:"type" json:"type"`
	Content     *Content `db:"content" json:"content"`
	ReadAt      int64    `db:"-" json:"reat_at"`
}

func (Notification) name() string { return "notification" }

func (Notification) unique() [][]string {
	return [][]string{[]string{"id"}}
}

type Content struct {
	Text        string   `json:"text"`
	CollapseKey string   `json:"collapse_key,omitempty"`
	QuestionID  string   `json:"question_id,omitempty"`
	PoolID      string   `json:"pool_id,omitempty"`
	Choices     []Choice `json:"choices,omitempty"`
}

type Choice struct {
	Label string
	Value string
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
	return json.Unmarshal(value.([]byte), c)
}

type NotificationUser struct {
	NotificationID string `db:"notification_id"`
	UserID         string `db:"user_id"`
	ReadAt         int64  `db:"read_at"`
}

func (NotificationUser) name() string { return "notification_user" }

func (NotificationUser) unique() [][]string {
	return [][]string{[]string{"notification_id", "user_id"}}
}

type AccessToken struct {
	ID       string `db:"id"`
	UserID   string `db:"user_id"`
	DeviceID string `db:"device_id"`
	Token    string `db:"token"`
	LastUsed string `db:"last_used"`
}

type User struct {
	Name                    string `db:"name"`
	PasswordHash            string `db:"password_hash"`
	CreationTS              int64  `db:"creation_ts"`
	Admin                   bool   `db:"admin"`
	UpgradeTS               int64  `db:"upgrade_ts"`
	IsGuest                 bool   `db:"is_guest"`
	AppserviceID            string `db:"appservice_id"`
	ConsentVersion          string `db:"consent_version"`
	ConsentServerNoticeSent string `db:"consent_server_notice_sent"`
}

func (User) name() string { return "users" }

func (User) unique() [][]string { return [][]string{[]string{"name"}} }

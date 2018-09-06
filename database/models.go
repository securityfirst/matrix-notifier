package database

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
	Admin          bool   `db:"admin"`
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

// Notification is the notification model.
type Notification struct {
	ID          string `db:"id,primarykey"`
	UserID      string `db:"user_id"`
	Type        string `db:"type"`
	Destination string `db:"destination"`
	CollapseKey string `db:"collapse_key"`
	Content     string `db:"content"`
	AdminOnly   bool   `db:"admin_only"`
	CreatedAt   int64  `db:"created_at"`
}

func (Notification) name() string { return "notification" }

func (Notification) unique() [][]string {
	return [][]string{[]string{"id"}}
}

type AccessToken struct {
	ID       string `db:"id"`
	UserID   string `db:"user_id"`
	DeviceID string `db:"device_id"`
	Token    string `db:"token"`
	LastUsed string `db:"last_used"`
}

package database

import (
	"database/sql"
	"time"

	"github.com/lib/pq"

	sq "github.com/Masterminds/squirrel"
	gorp "gopkg.in/gorp.v2"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// IsDuplicate checks if the error is a duplicate keyy violation.
func IsDuplicate(err error) bool {
	perr, ok := err.(*pq.Error)
	return ok && perr.Code == "23505"
}

// DB is a common interface for both gorp.DbMap and gorp.Transaction.
type DB interface {
	Delete(list ...interface{}) (int64, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(i interface{}, keys ...interface{}) (interface{}, error)
	Insert(list ...interface{}) error
	Prepare(query string) (*sql.Stmt, error)
	Select(i interface{}, query string, args ...interface{}) ([]interface{}, error)
	SelectFloat(query string, args ...interface{}) (float64, error)
	SelectInt(query string, args ...interface{}) (int64, error)
	SelectNullFloat(query string, args ...interface{}) (sql.NullFloat64, error)
	SelectNullInt(query string, args ...interface{}) (sql.NullInt64, error)
	SelectNullStr(query string, args ...interface{}) (sql.NullString, error)
	SelectOne(holder interface{}, query string, args ...interface{}) error
	SelectStr(query string, args ...interface{}) (string, error)
	Update(list ...interface{}) (int64, error)
}

// InitDBMap initializes the DbMap and creates the tables.
func InitDBMap(d *gorp.DbMap) error {
	for _, t := range []table{Org{}, Notification{}, NotificationUser{}} {
		table := d.AddTableWithName(t, t.name())
		for i, s := range t.unique() {
			switch {
			case i == 0:
				table.SetKeys(false, s...)
			case len(s) > 1:
				table.SetUniqueTogether(s...)
			default:
				table.ColMap(s[0]).SetUnique(true)
			}
		}
	}
	return d.CreateTablesIfNotExists()
}

// Get returns the Record with the selected key.
func Get(d DB, i interface{}, key string) (interface{}, error) { return d.Get(i, key) }

// Create creates a new Record.
func Create(d DB, i interface{}) error { return d.Insert(i) }

// Update updates a new Record.
func Update(d DB, i interface{}) error {
	_, err := d.Update(i)
	return err
}

// GetOrgByName returns the selected Org by name
func GetOrgByName(d DB, name string, ids ...string) (*Org, error) {
	query, args, err := psql.Select("*").From(Org{}.name()).
		Where(sq.Eq{"name": name, "room_id": ids}).ToSql()
	if err != nil {
		return nil, err
	}
	var org Org
	if err := d.SelectOne(&org, query, args...); err != nil {
		return nil, err
	}
	return &org, nil
}

// ListOrgs returns the list of Orgs from their IDs
func ListOrgs(d DB, ids ...string) ([]*Org, error) {
	query, args, err := psql.Select("*").From(Org{}.name()).Where(sq.Eq{"room_id": ids}).ToSql()
	if err != nil {
		return nil, err
	}
	list, err := d.Select(Org{}, query, args...)
	if err != nil {
		return nil, err
	}
	o := make([]*Org, len(list))
	for i := range list {
		o[i] = list[i].(*Org)
	}
	return o, nil
}

// ListNotifications returns a list of notifications since the specified time.
// levels is a power level per room, rules is the level per type
func ListNotifications(d DB, since time.Time, userID string, levels, rules map[string]int) ([]*Notification, error) {
	type N struct {
		Notification
		Read bool `db:"read"`
	}
	filter := make(sq.Or, 0, len(levels))
	for room, lvl := range levels {
		var keys []string
		for t, l := range rules {
			if lvl >= l {
				keys = append(keys, t)
			}
		}
		filter = append(filter, sq.Eq{
			"n.room_id": room,
			"n.type":    keys,
		})
	}
	query, args, err := psql.Select(`n.*, (last_read is not null and last_read > n.created_at) as read`).From(Notification{}.name()+` n`).
		LeftJoin(NotificationUser{}.name()+` u on (u.user_id = ?)`, userID).Where(sq.And{
		sq.Gt{"n.created_at": since}, filter}).ToSql()
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	list, err := d.Select(&N{}, query, args...)
	if err != nil {
		return nil, err
	}
	n := make([]*Notification, len(list))
	for i := range list {
		v := list[i].(*N)
		n[i] = &v.Notification
		n[i].Read = v.Read
	}
	return n, nil
}

// MarkAsRead upserts read time for a user
func MarkAsRead(db DB, userID string, t time.Time) error {
	v, err := db.Get(&NotificationUser{}, userID)
	if err != nil {
		return err
	}
	n := NotificationUser{UserID: userID, LastRead: t}
	if v == nil {
		err = db.Insert(&n)
	} else {
		_, err = db.Update(&n)
	}
	return err
}

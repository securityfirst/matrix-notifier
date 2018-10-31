package database

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	gorp "gopkg.in/gorp.v2"
)

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
	for _, t := range []table{Organisation{}, OrganisationUser{}, Notification{}, NotificationUser{}, User{}, UserThreepid{}} {
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

// ListOrganisations returns the list of Organisations for a User.
func ListOrganisations(d DB, userID string) ([]Organisation, error) {
	var (
		query = "select * from organisation"
		args  = make([]interface{}, 0, 1)
	)
	if userID != "" {
		query, args = query+" where userID = $1", append(args, userID)
	}
	list, err := d.Select(Organisation{}, query, args...)
	if err != nil {
		return nil, err
	}
	o := make([]Organisation, len(list))
	for i := range list {
		o[i] = list[i].(Organisation)
	}
	return o, nil
}

// UpdateOrganisationUser updates the user_id.
func UpdateOrganisationUser(d DB, userID, hash string) error {
	res, err := d.Exec("update organisation_user set user_id=$1 where hash=$2;", userID, hash)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("%d rows affected", rows)
	}
	return nil
}

// FindEmailByUser returns the email for the selected User.
func FindEmailByUser(d DB, userID string) (string, error) {
	return d.SelectStr("select address from user_threepids where user_id = $1 and medium = $2", userID, "email")
}

// FindUserByToken returns the username with the selected Token.
func FindUserByToken(d DB, token string) (string, error) {
	return d.SelectStr("select user_id from access_tokens where token = $1", token)
}

// FindOrganisationUserByHash returns the OrganisationUser using the given hash.
func FindOrganisationUserByHash(d DB, hash string) (*OrganisationUser, error) {
	var ou OrganisationUser
	if err := d.SelectOne(&ou, "select * from organisation_user where hash = $1", hash); err != nil {
		return nil, err
	}
	return &ou, nil
}

// FindOrganisationUserByUserOrg returns the OrganisationUser using User and Organisation.
func FindOrganisationUserByUserOrg(d DB, userID, orgID string) (*OrganisationUser, error) {
	var ou OrganisationUser
	if err := d.SelectOne(&ou, "select * from organisation_user where user_id = $1 and organisation_id = $2", userID, orgID); err != nil {
		return nil, err
	}
	return &ou, nil
}

// ListNotifications returns a list of the unread notifications for the selected User since the specified time.
func ListNotifications(d DB, userID string, since int64) ([]Notification, error) {
	const query = `
select
        n.*, coalesce(read_at,0) as read_at
from
        notification n join organisation_user ou on (ou.organisation_id = n.destination)
        left join notification_user u on (n.id = u.notification_id and u.user_id = $1)
        
where
        ou.user_id = $1 and
        n.created_at >= $2 and
        n.destination = ou.organisation_id and
        (ou.admin > 0 or n.type in ('panic','announcement', 'question','answer', 'pool'))`
	var l []struct {
		Notification
		ReadAt int64 `db:"read_at"`
	}
	if _, err := d.Select(&l, query, userID, since); err != nil {
		return nil, err
	}
	var list = make([]Notification, len(l))
	for i := range l {
		list[i] = l[i].Notification
		list[i].ReadAt = l[i].ReadAt
	}
	return list, nil
}

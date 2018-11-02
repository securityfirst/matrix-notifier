package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	"gopkg.in/gorp.v2"
)

var (
	username = os.Getenv("DB_USERNAME")
	password = os.Getenv("DB_PASSWORD")
	host     = os.Getenv("DB_HOST")
	name     = os.Getenv("DB_DATABASE") + "_test"
	options  = os.Getenv("DB_OPTIONS")
)

var dbMap *gorp.DbMap

func init() {
	url := fmt.Sprintf("postgres://%s:%s@%s/?%s", username, password, host, options)
	log.Println("Using", url, "with db", name)
	db, err := sql.Open("postgres", url)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	if _, err = db.Exec("drop database if exists " + name); err != nil {
		log.Fatalln("Error:", err)
	}
	if _, err = db.Exec("create database " + name); err != nil {
		log.Fatalln("Error:", err)
	}
	if err := db.Close(); err != nil {
		log.Fatalln("Error:", err)
	}
	db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?%s", username, password, host, name, options))
	if err != nil {
		log.Fatalln("Error:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalln("Error:", err)
	}
	dbMap = &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
}

func TestInit(t *testing.T) {
	if err := InitDBMap(dbMap); err != nil {
		log.Fatal(err)
	}
}

func user(s string) *User {
	return &User{Name: "user" + s}
}
func org(s string) *Organisation {
	return &Organisation{ID: "org" + s, Name: "org" + s, Package: "com.org" + s}
}
func ou(o, u string, l int) *OrganisationUser {
	return &OrganisationUser{UserID: "user" + u, OrganisationID: "org" + o, Level: l, Hash: o + "-" + u}
}
func not(id, o, u, t string, created int64) *Notification {
	return &Notification{ID: id, UserID: "user" + u, Destination: "org" + o, Type: t, CreatedAt: created}
}

func nu(n, u string, read int64) *NotificationUser {
	return &NotificationUser{NotificationID: n, UserID: "user" + u, ReadAt: read}
}

func TestQueries(t *testing.T) {
	now := time.Now().Unix()
	records := []interface{}{
		user("1"), user("2"), user("3"), org("1"), org("2"), org("3"),
		ou("1", "1", 1), ou("2", "1", 0),
		ou("2", "2", 1), ou("3", "2", 0),
		ou("3", "3", 1), ou("1", "3", 0),
		not("0001", "1", "1", "a", now),
		not("0002", "1", "1", "b", now),
		not("0003", "1", "1", "c", now),
		not("0004", "1", "1", "d", now),
		not("0005", "1", "1", "e", now),
		not("0006", "1", "1", "f", now),
		not("0011", "3", "3", "a", now),
		not("0012", "3", "3", "b", now),
		not("0013", "3", "3", "c", now),
		not("0014", "3", "3", "d", now),
		not("0015", "3", "3", "e", now),
		not("0016", "3", "3", "f", now),
	}
	var nu = [][2]string{
		{"0001", "user1"}, {"0002", "user1"}, {"0003", "user1"},
		{"0002", "user3"}, {"0003", "user3"}, {"0011", "user3"}, {"0012", "user3"},
	}
	for _, r := range records {
		if err := Create(dbMap, r); err != nil {
			log.Fatal(r, err)
		}
	}
	for _, n := range nu {
		if err := MarkAsRead(dbMap, n[1], n[0], now, "a", "b", "c", "d"); err != nil {
			log.Fatal(n, err)
		}
	}

	if err, ok := MarkAsRead(dbMap, "user1", "0001", now, "a", "b", "c", "d").(*pq.Error); !ok || err.Code != "23505" {
		log.Fatalf("Expected %#v, got %#v", sql.ErrNoRows, err)
	}

	for user, count := range map[string][2]int{"user1": [2]int{6, 3}, "user2": [2]int{4, 0}, "user3": [2]int{10, 4}} {
		list, err := ListNotifications(dbMap, user, now, "a", "b", "c", "d")
		if err != nil {
			log.Fatal(err)
		}
		if l := len(list); l != count[0] {
			for _, l := range list {
				log.Printf("%#v", l)
			}
			log.Fatalf("%s: expected %d, got %d", user, count[0], l)
		}
		read := 0
		for _, n := range list {
			if n.ReadAt > 0 {
				read++
			}
		}
		if read != count[1] {
			for _, l := range list {
				log.Printf("%#v", l)
			}
			log.Fatalf("%s: expected %d, got %d", user, count[1], read)
		}
	}

}

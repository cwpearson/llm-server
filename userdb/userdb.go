package userdb

import (
	"crypto"
	_ "golang.org/x/crypto/blake2b"
	"bytes"
	"log"
	"fmt"
	"errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var schema = `
CREATE TABLE IF NOT EXISTS user (
	id INTEGER PRIMARY KEY,
	username TEXT,
	password BLOB
)`

type Row struct {
	Id int64
	Username string
	Password []byte
}

func hash(s string) []byte {
	hash := crypto.BLAKE2b_512.New()
	hash.Write([]byte(s))
	return hash.Sum([]byte{})
}

type DB struct {
	db *sqlx.DB
}

func Create() DB {
	db, err := sqlx.Connect("sqlite3", "user.sqlite")
	if err != nil {
		log.Fatalln(err)
	}
	db.MustExec(schema)

	return DB{db}
}


func (db DB) Add(username, password string) (int64, error) {

	rows := []Row{}
	db.db.Select(&rows, "SELECT * FROM user WHERE username = ?", username)

	if len(rows) != 0 {
		return -1, errors.New("user already exists!")
	}

	// hash password and add both to database
	digest := hash(password)


	log.Printf("added user=%s pass=%v\n", username, digest)
	res := db.db.MustExec("INSERT INTO user (username, password) VALUES (?, ?)", username, digest)

	id, err := res.LastInsertId()
	if err != nil {
		fmt.Printf("%v\n", err)
		return -1, err
	}

	return id, nil
}

func (db DB) IsPassword(username, password string) (bool, error) {

	// retrieve hash from data
	rows := []Row{}
	db.db.Select(&rows, "SELECT * FROM user", username)
	if len(rows) == 0 {
		log.Printf("no users matched %s\n", username)
		return false, nil
	} else if len(rows) > 1 {
		return false, errors.New("multiple users with the same username?")
	}
	row := rows[0]
	expected := row.Password
	digest := hash(password)

	if bytes.Equal(expected, digest) {
		return true, nil
	} else {
		return false, nil
	}

}


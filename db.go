package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var schema = `
CREATE TABLE IF NOT EXISTS job (
	id INTEGER PRIMARY KEY,
	username text,
	prompt text,
	status integer,
	stdout text,
	stderr text
)`

type Row struct {
	Id int64
	Username string
	Prompt string
	Status int
	Stdout sql.NullString
	Stderr sql.NullString
}

func Create() *sqlx.DB {
	db, err := sqlx.Connect("sqlite3", "jobs.sqlite")
	if err != nil {
		log.Fatalln(err)
	}
	db.MustExec(schema)

	return db
}

func record_job(db *sqlx.DB, username string, prompt string) int64 {
	res := db.MustExec("INSERT INTO job (username, prompt, status) VALUES (?, ?, ?)", username, prompt, status_pending)
	rowId, err := res.LastInsertId()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return rowId
}

func set_status(db *sqlx.DB, id int64, status int) {
	fmt.Printf("Job %d status -> %d\n", id, status)
	db.MustExec("UPDATE job SET status = ? WHERE id = ?", status, id)
}

func append_stdout(db *sqlx.DB, id int64, buf *bytes.Buffer) {
	s := buf.String()
	fmt.Printf("Job %d stdout += %s\n", id, s)
	db.MustExec("UPDATE job SET stdout = (CASE WHEN stdout IS NULL THEN ? ELSE stdout || ? END) WHERE id = ?", s, s, id)
}

func append_stderr(db *sqlx.DB, id int64, buf *bytes.Buffer) {
	s := buf.String()
	fmt.Printf("Job %d stderr += %s\n", id, s)
	db.MustExec("UPDATE job SET stderr = (CASE WHEN stderr IS NULL THEN ? ELSE stderr || ? END) WHERE id = ?", s, s, id)
}

func get_rows(db *sqlx.DB) []Row {
	rows := []Row{}
	err := db.Select(&rows, "SELECT * FROM job ORDER BY id DESC")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return rows
}

func get_row(db *sqlx.DB, i int64) []Row {
	rows := []Row{}
	err := db.Select(&rows, "SELECT * FROM job WHERE id = ?", i)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	return rows
}

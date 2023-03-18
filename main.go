package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {

	db := Create()

	// create jobs channel
	chDefs := make(chan Definition, 128)

	// create a worker
	workers := []Worker{}
	workers = append(workers, create_worker(db, chDefs))

	for _, w := range workers {
		go w.run()
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/job/:id", func(c *gin.Context) {
		id_str := c.Param("id")
		id, err := strconv.ParseInt(id_str, 10, 64)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		rows := get_row(db, id)

		if len(rows) == 0 {
			fmt.Println("no rows returned")
			c.HTML(http.StatusOK, "job.tmpl", gin.H{})
		} else {

			row := rows[0]

			stdout := "NULL"
			if row.Stdout.Valid {
				stdout = row.Stdout.String
			}

			stderr := "NULL"
			if row.Stderr.Valid {
				stderr = row.Stderr.String
			}

			var status string
			switch row.Status {
			case status_pending:
				status = "pending"
			case status_running:
				status = "running"
			case status_error:
				status = "error"
			case status_success:
				status = "success"
			}

			c.HTML(http.StatusOK, "job.tmpl", gin.H{"id": id, "prompt": rows[0].Prompt, "status": status, "stdout": stdout, "stderr": stderr})
		}
	})
	r.POST("/submit", func(c *gin.Context) {
		prompt := c.PostForm("prompt")

		// create a job to get an ID
		id := record_job(db, prompt)

		// submit to workers
		def := Definition{id, prompt}
		chDefs <- def

		// redirect to job status page
		url := fmt.Sprintf("/job/%d", id)
		c.Redirect(http.StatusFound, url)

	})
	r.GET("/submit", func(c *gin.Context) {
		c.HTML(http.StatusOK, "submit.tmpl", gin.H{})
	})
	r.GET("/jobs", func(c *gin.Context) {

		rows := get_rows(db)
		fmt.Printf("got %d rows\n", len(rows))

		type HtmlRow struct {
			Prompt string
			Status string
			Stdout string
			Stderr string
		}

		html_rows := []HtmlRow{}
		for _, row := range rows {

			stdout := "NULL"
			if row.Stdout.Valid {
				stdout = row.Stdout.String
			}
			stderr := "NULL"
			if row.Stderr.Valid {
				stderr = row.Stderr.String
			}

			var status string
			switch row.Status {
			case status_pending:
				status = "pending"
			case status_running:
				status = "running"
			case status_error:
				status = "error"
			case status_success:
				status = "success"
			}

			html_rows = append(html_rows, HtmlRow{
				row.Prompt,
				status,
				stdout,
				stderr,
			})
		}

		c.HTML(http.StatusOK, "jobs.tmpl", gin.H{"rows": html_rows})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

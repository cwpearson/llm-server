package main

import (
	"fmt"
	"net/http"
	"strconv"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"

	"main/auth"
	"main/userdb"
)

func main() {

	db := Create()
	user_db := userdb.Create()

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

	r.Use(sessions.Sessions("mysession", cookie.NewStore(auth.Cookie_store_secret())))

	// account management routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	})

	// account management routes
	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.tmpl", gin.H{})
	})
	r.POST("/register", auth.Register(user_db))

	// Login and logout routes
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.tmpl", gin.H{})
	})
	r.POST("/login", auth.Login(user_db))
	r.GET("/logout", auth.Logout)

	private := r.Group("/u/:username")
	private.Use(auth.AuthRequired)
	
	private.GET("/job/:jobid", func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get(auth.Userkey)
		log.Printf("job for user %s", user)
		id_str := c.Param("jobid")
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
		ok := auth.IsAuthorized(c)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		session := sessions.Default(c)
		user := session.Get(auth.Userkey).(string)
		log.Printf("submit for user %s", user)

		prompt := c.PostForm("prompt")

		// create a job to get an ID
		id := record_job(db, user, prompt)

		// submit to workers
		def := Definition{id, prompt}
		chDefs <- def

		// redirect to job status page
		url := fmt.Sprintf("/u/%s/job/%d", user, id)
		c.Redirect(http.StatusFound, url)

	})
	r.GET("/submit", func(c *gin.Context) {
		ok := auth.IsAuthorized(c)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		c.HTML(http.StatusOK, "submit.tmpl", gin.H{})
	})
	r.GET("/jobs", func(c *gin.Context) {
		ok := auth.IsAuthorized(c)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		session := sessions.Default(c)
		user := session.Get(auth.Userkey)
		log.Printf("jobs for user %s", user)

		rows := get_rows(db)
		fmt.Printf("got %d rows\n", len(rows))

		type HtmlRow struct {
			Username string
			Id int64
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
				row.Username,
				row.Id,
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

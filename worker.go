package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"log"

	"github.com/jmoiron/sqlx"
)

// a worker consumes jobs from a channel and updates the database

type Worker struct {
	db     *sqlx.DB
	chJobs <-chan Definition
}

func create_worker(db *sqlx.DB, ch <-chan Definition) Worker {
	return Worker{db, ch}

}

func (w Worker) run_job(def Definition) {
	job := Job{def, Result{}, status_pending}
	set_status(w.db, def.id, job.status)

	// construct command
	// -n N, --n_predict N   number of tokens to predict (default: 128)
	// --top_k N             top-k sampling (default: 40)
	// --top_p N             top-p sampling (default: 0.9)
	// --repeat_last_n N     last n tokens to consider for penalize (default: 64)
	// --repeat_penalty N    penalize repeat sequence of tokens (default: 1.3)
	// -c N, --ctx_size N    size of the prompt context (default: 512)
	// --temp N              temperature (default: 0.8)
	log.Printf("'%s'\n", def.Prompt)
	cmd := exec.Command("./llama.cpp/main", 
	"--model", "./llama.cpp/models/13B/ggml-model-q4_0.bin",
	"--threads", "8",
	"-p", def.Prompt)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("couldn't attach stderr pipe\n")
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("couldn't attach stdout pipe\n")
		return
	}

	// start command
	cmd.Start()
	set_status(w.db, def.id, status_running)

	// readers will signal when done
	chDone := make(chan int)

	// stdout reader
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			append_stdout(w.db, def.id, bytes.NewBuffer(buf[0:n]))
			if err == io.EOF {
				chDone <- 0
				return
			}
		}
	}()

	// stderr reader
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			append_stderr(w.db, def.id, bytes.NewBuffer(buf[0:n]))
			if err == io.EOF {
				chDone <- 0
				return
			}
		}
	}()

	// wait for stdout and stderr
	<-chDone
	<-chDone

	// kill process
	err = cmd.Wait()
	if err != nil {
		set_status(w.db, def.id, status_error)
	} else {
		set_status(w.db, def.id, status_success)
	}
}

func (w Worker) run() {
	for {
		def, more := <-w.chJobs
		if !more {
			fmt.Printf("stopping worker")
			return
		}
		fmt.Println("received job", def)
		w.run_job(def)
	}

}

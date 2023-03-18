package main

const (
	status_pending = iota
	status_running = iota
	status_success = iota
	status_error   = iota
)

type Definition struct {
	id     int64
	Prompt string
}

type Result struct {
	stdout string
	stderr string
}

type Job struct {
	def    Definition
	result Result
	status int
}

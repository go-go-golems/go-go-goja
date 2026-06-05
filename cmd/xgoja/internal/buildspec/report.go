package buildspec

import (
	"fmt"
	"strings"
)

type Status string

const (
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

type Check struct {
	Name    string
	Status  Status
	Path    string
	Message string
}

type Report struct {
	Checks []Check
}

func (r *Report) AddOK(name, path, message string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: StatusOK, Path: path, Message: message})
}

func (r *Report) AddError(name, path, message string) {
	r.Checks = append(r.Checks, Check{Name: name, Status: StatusError, Path: path, Message: message})
}

func (r *Report) HasErrors() bool {
	if r == nil {
		return false
	}
	for _, check := range r.Checks {
		if check.Status == StatusError {
			return true
		}
	}
	return false
}

type ValidationError struct {
	Report *Report
}

func (e *ValidationError) Error() string {
	if e == nil || e.Report == nil {
		return "xgoja build spec validation failed"
	}
	messages := []string{}
	for _, check := range e.Report.Checks {
		if check.Status != StatusError {
			continue
		}
		if check.Path != "" {
			messages = append(messages, fmt.Sprintf("%s: %s", check.Path, check.Message))
		} else {
			messages = append(messages, check.Message)
		}
	}
	if len(messages) == 0 {
		return "xgoja build spec validation failed"
	}
	return "xgoja build spec validation failed: " + strings.Join(messages, "; ")
}

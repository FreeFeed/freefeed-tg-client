package main

import (
	"github.com/davidmz/debug-log"
)

type logWriter struct {
	log debug.Logger
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.log.Print(string(p))
	return len(p), nil
}

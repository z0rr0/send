package logging

// Package logging implements easy log-wrapper for info and error cases.
// It supports request ID generation and context value saving.

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
)

var (
	// logError - error logger
	logError = log.New(os.Stderr, "ERROR", log.Ldate|log.Ltime|log.Lshortfile)
	// logInfo - info logger.
	logInfo = log.New(os.Stdout, "INFO", log.LstdFlags)
	// lock for global logs update
	mu sync.Mutex
)

// SetUp overwrites default loggers with custom app name and writers.
func SetUp(name string, i, e io.Writer, iFlag, eFlag int) {
	mu.Lock()
	logInfo = log.New(i, fmt.Sprintf("INFO [%s] ", name), iFlag)
	logError = log.New(e, fmt.Sprintf("ERROR [%s] ", name), eFlag)
	mu.Unlock()
}

// SetUpFile overwrites default logger with custom one and does output to the fileName.
func SetUpFile(name, fileName string, iFlag, eFlag int) (*os.File, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		return nil, err
	}
	logInfo = log.New(f, fmt.Sprintf("INFO [%s] ", name), iFlag)
	logError = log.New(f, fmt.Sprintf("ERROR [%s] ", name), eFlag)
	return f, nil
}

// Log is logger storage for request ID and related methods..
type Log struct {
	ID string
}

// vars adds Log.ID in the begin of slice a.
func (l *Log) vars(a []interface{}) []interface{} {
	var v = make([]interface{}, 1, len(a)+1)
	v[0] = l.ID
	return append(v, a...)
}

// Info is logger info wrapper. It adds request ID.
func (l *Log) Info(format string, a ...interface{}) {
	f, v := "[%s] "+format, l.vars(a)
	logInfo.Printf(f, v...)
}

// Error is logger error wrapper. It adds request ID.
func (l *Log) Error(format string, a ...interface{}) {
	f, v := "[%s] "+format, l.vars(a)
	logError.Printf(f, v...)
}

// New creates new Log struct.
func New(id string) *Log {
	if id == "" {
		id = uuid.New().String()
	}
	return &Log{id}
}

// ErrorLog returns internal error logger.
func ErrorLog() *log.Logger {
	return logError
}

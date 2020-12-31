package logging

// Package logging implements easy log-wrapper for info and error cases.
// It supports request ID generation and context value saving.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/uuid"
)

// keyType is custom type for context key.
type keyType uint8

// key is context value key
const key keyType = 1

var (
	// logError - error logger
	logError = log.New(os.Stderr, "ERROR", log.Ldate|log.Ltime|log.Lshortfile)
	// logInfo - info logger.
	logInfo = log.New(os.Stdout, "INFO", log.LstdFlags)
	// ErrLogContext is error when context value was not found.
	ErrLogContext = errors.New("not found log context")
)

// SetUp overwrites default loggers with custom app name and writers.
func SetUp(name string, i, e io.Writer, iFlag, eFlag int) {
	logInfo = log.New(i, fmt.Sprintf("INFO [%s] ", name), iFlag)
	logError = log.New(e, fmt.Sprintf("ERROR [%s] ", name), eFlag)
}

// Log is logger storage for request id and related methods..
type Log struct {
	id string
}

// vars adds Log.id in the begin of slice a.
func (l *Log) vars(a []interface{}) []interface{} {
	var v = make([]interface{}, 1, len(a)+1)
	v[0] = l.id
	return append(v, a...)
}

// Info is logger info wrapper. It adds request id.
func (l *Log) Info(format string, a ...interface{}) {
	f, v := "[%s] "+format, l.vars(a)
	logInfo.Printf(f, v...)
}

// Error is logger error wrapper. It adds request id.
func (l *Log) Error(format string, a ...interface{}) {
	f, v := "[%s] "+format, l.vars(a)
	logError.Printf(f, v...)
}

// Context extends parent context by value with logger l.
func (l *Log) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, key, l)
}

// New creates new Log struct.
func New(id string) (*Log, error) {
	if id == "" {
		id = uuid.New().String()
	}
	return &Log{id}, nil
}

// NewWithContext creates new Log struct and includes its value to the ctx.
func NewWithContext(ctx context.Context, id string) (context.Context, error) {
	l, err := New(id)
	if err != nil {
		return nil, err
	}
	return l.Context(ctx), nil
}

// Get returns *Log value from context ctx.
func Get(ctx context.Context) (*Log, error) {
	v := ctx.Value(key)
	if v == nil {
		return nil, ErrLogContext
	}
	return v.(*Log), nil
}

// ErrorLog returns internal error logger.
func ErrorLog() *log.Logger {
	return logError
}

package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func checkLogMsg(msg, prefix, suffix string) error {
	if !strings.HasPrefix(msg, prefix) {
		return errors.New("failed prefix")
	}
	if !strings.HasSuffix(msg, suffix) {
		return errors.New("failed suffix")
	}
	return nil
}

func TestSetUp(t *testing.T) {
	var (
		i, e      bytes.Buffer
		iExpected = "test / info\n"
		eExpected = "test / error\n"
	)
	SetUp("test", &i, &e, 0, 0)
	logInfo.Printf("test / %s", "info")
	logError.Printf("test / %s", "error")

	v := i.String()
	if err := checkLogMsg(v, "INFO [test]", iExpected); err != nil {
		t.Errorf("failed value [%v]: %v", err, v)
	}
	v = e.String()
	if err := checkLogMsg(v, "ERROR [test]", eExpected); err != nil {
		t.Errorf("failed value [%v]: %v", err, v)
	}
}

func TestErrorLog(t *testing.T) {
	var (
		i, e      bytes.Buffer
		eExpected = "test / error\n"
	)
	SetUp("test", &i, &e, 0, 0)
	el := ErrorLog()
	el.Printf("test / %s", "error")
	v := e.String()
	if err := checkLogMsg(v, "ERROR [test]", eExpected); err != nil {
		t.Errorf("failed value [%v]: %v", err, v)
	}
}

func TestNew(t *testing.T) {
	var i, e bytes.Buffer
	SetUp("test", &i, &e, 0, 0)
	l, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	l.Info("info=%s", "testMsg")
	expected := fmt.Sprintf("INFO [test] [%s] info=testMsg\n", l.id)
	if v := i.String(); v != expected {
		t.Errorf("failed info logger message=%v", v)
	}

	l.Error("error=%s", "testErrMsg")
	expected = fmt.Sprintf("ERROR [test] [%s] error=testErrMsg\n", l.id)
	if v := e.String(); v != expected {
		t.Errorf("failed error logger message=%v", v)
	}
}

func TestNewWithContext(t *testing.T) {
	var (
		i, e bytes.Buffer
		ctx  = context.Background()
	)
	SetUp("test", &i, &e, 0, 0)
	_, err := Get(ctx)
	if !errors.Is(err, ErrLogContext) {
		t.Fatal("unexpected context value")
	}
	logCtx, err := NewWithContext(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := Get(logCtx)
	if err != nil {
		t.Fatal(err)
	}
	l.Info("info=%s", "testMsg")
	expected := fmt.Sprintf("INFO [test] [%s] info=testMsg\n", l.id)
	if v := i.String(); v != expected {
		t.Errorf("failed info logger message=%v", v)
	}
}

package logging

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func TestSetUpFile(t *testing.T) {
	fileName := filepath.Join(os.TempDir(), "send_logging_test.log")
	f, err := SetUpFile("test", fileName, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	logInfo.Printf("test / %s", "info")
	logError.Printf("test / %s", "error")

	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}

	fr, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	expected := []struct {
		prefix string
		suffix string
	}{
		{"INFO [test]", "test / info"},
		{"ERROR [test]", "test / error"},
	}
	scanner := bufio.NewScanner(fr)
	i := 0
	for scanner.Scan() {
		logLine, exp := scanner.Text(), expected[i]
		if e := checkLogMsg(logLine, exp.prefix, exp.suffix); e != nil {
			t.Errorf("failed value [%v]: %v", e, logLine)
		}

		i++
	}
	err = scanner.Err()
	if err != nil {
		t.Error(err)
	}
	err = os.Remove(fileName)
	if err != nil {
		t.Error(err)
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
	l := New("")
	l.Info("info=%s", "testMsg")
	expected := fmt.Sprintf("INFO [test] [%s] info=testMsg\n", l.ID)
	if v := i.String(); v != expected {
		t.Errorf("failed info logger message=%v", v)
	}

	l.Error("error=%s", "testErrMsg")
	expected = fmt.Sprintf("ERROR [test] [%s] error=testErrMsg\n", l.ID)
	if v := e.String(); v != expected {
		t.Errorf("failed error logger message=%v", v)
	}
}

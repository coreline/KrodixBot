package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type logMode string

const (
	debugMode logMode = "DEBUG"
	errorMode logMode = "ERROR"

	ansiReset  = "\u001B[0m"
	ansiRed    = "\u001B[31m"
	ansiYellow = "\u001B[33m"
	ansiBlue   = "\u001B[34m"
)

type logger struct {
	Out         io.Writer
	DebugMode   bool
	PrintErrors bool
	Replacer    *strings.Replacer

	mutex sync.Mutex
}

func newDefaultLogger(debugMode, printErrors bool) *logger {
	return &logger{
		Out:         os.Stderr,
		DebugMode:   debugMode,
		PrintErrors: printErrors,
	}
}

func (l *logger) prefix(mode logMode) string {
	timeNow := ansiBlue + time.Now().Format(time.UnixDate) + ansiReset
	switch mode {
	case debugMode:
		return fmt.Sprintf("[%s] %sDEBUG%s ", timeNow, ansiYellow, ansiReset)
	case errorMode:
		pc, filename, line, _ := runtime.Caller(3)
		return fmt.Sprintf("[%s] %sERROR%s in %s[%s:%d] ", timeNow, ansiRed, ansiReset, runtime.FuncForPC(pc).Name(), filepath.Base(filename), line)
	}
	return "LOGGING "
}

func (l *logger) log(mode logMode, text string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.Replacer != nil {
		text = l.Replacer.Replace(text)
	}

	_, err := l.Out.Write([]byte(l.prefix(mode) + text))
	if err != nil {
		//nolint:forbidigo
		_, _ = fmt.Printf("Logging error: %v\n", err)
	}
}

func (l *logger) Debugf(format string, args ...any) {
	if l.DebugMode {
		l.log(debugMode, fmt.Sprintf(format+"\n", args...))
	}
}

func (l *logger) Errorf(format string, args ...any) {
	if l.PrintErrors {
		l.log(errorMode, fmt.Sprintf(format+"\n", args...))
	}
}

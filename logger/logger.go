package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"
)

const (
	TYPE_HTTP = 1
)

func init() {
	// Do not set default output; let the application decide
	log.SetFormatter(Formatter(false)) // Default: no color
}

// SetOutput sets the log output target
func SetOutput(out *os.File) {
	log.SetOutput(out)
}

// SetLevel sets the log level
func SetLevel(level log.Level) {
	log.SetLevel(level)
}

// UseStdout uses standard output
func UseStdout() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(Formatter(true))
}

/*
func getUserInfo(ctx *gin.Context) int {
	if data, ok := ctx.Get("uid"); ok {
		if uid, ok := data.(int); ok {
			return uid
		}
	}
	return 0
}
*/

// getCaller returns the actual caller information (skipping logger wrapper layers)
func getCaller() (string, int) {
	// Skip the log library call stack to get the actual caller.
	// Call chain: user code -> logger.Info -> addCallerField -> getCaller -> runtime.Caller
	// So we need to skip 3 levels to reach the actual call site.
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "unknown", 0
	}
	// Extract the file name (without path)
	shortFile := filepath.Base(file)
	return shortFile, line
}

// addCallerField adds caller information to the log fields
func addCallerField() *log.Entry {
	file, line := getCaller()
	return log.WithField("caller", fmt.Sprintf("%s:%d", file, line))
}

func Info(args ...interface{}) {
	addCallerField().Info(args...)
}

func Error(args ...interface{}) {
	addCallerField().Error(args...)
}

func Debug(args ...interface{}) {
	addCallerField().Debug(args...)
}

func Warn(args ...interface{}) {
	addCallerField().Warn(args...)
}

func Fatal(args ...interface{}) {
	addCallerField().Fatal(args...)
}

func Infof(format string, args ...interface{}) {
	addCallerField().Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	addCallerField().Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	addCallerField().Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	addCallerField().Warnf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	addCallerField().Fatalf(format, args...)
}

func Log(args ...interface{}) *log.Entry {
	fields := log.Fields{}
	lenArgs := len(args)
	for i := 0; i < lenArgs; i = i + 2 {
		var key string
		var ok bool
		if key, ok = args[i].(string); !ok {
			continue
		}

		if i <= lenArgs-2 {
			fields[key] = args[i+1]
			continue
		}
		fields[key] = ""
	}

	// Add caller information
	// Need to adjust the level in the Log function call chain as well
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}
	shortFile := filepath.Base(file)
	fields["caller"] = fmt.Sprintf("%s:%d", shortFile, line)

	log.SetFormatter(Formatter(true))
	return log.WithFields(fields)
}

func Formatter(isConsole bool) *nested.Formatter {
	fmtter := &nested.Formatter{
		FieldsOrder:      []string{"time", "level", "caller", "msg"},
		HideKeys:         true,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		CallerFirst:      true,
		NoUppercaseLevel: true,
		ShowFullLevel:    true,
		//NoFieldsSpace:    true,
		// Disable default caller formatting since we already add a custom caller field
		CustomCallerFormatter: func(frame *runtime.Frame) string {
			return ""
		},
	}
	if isConsole {
		fmtter.NoColors = false
	} else {
		fmtter.NoColors = true
	}
	return fmtter
}

// DebugStack is used to debug the log call stack; it prints all caller information in the current call chain
func DebugStack() {
	for i := 0; i < 5; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		shortFile := filepath.Base(file)
		log.Infof("CallStack[%d]: %s:%d", i, shortFile, line)
	}
}

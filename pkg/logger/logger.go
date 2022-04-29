package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"go.elastic.co/apm/module/apmlogrus"
)

// Config provides configs for instantiating Logger.
type Config struct {
	STDOut     bool
	Level      string
	OutputFile string
}

// Logger provides an access to logger.
type Logger struct {
	*logrus.Logger
}

const fileMode = os.FileMode(0666)

// New instantiates a new Logger.
func New(config Config) (*Logger, error) {
	logrusLogger := logrus.New()
	logrusLogger.AddHook(&apmlogrus.Hook{})
	logrusLogger.Out = ioutil.Discard
	logrusLogger.Level = parseLogrusLevel(config.Level)
	logrusLogger.Formatter = getFormatter()
	logrusLogger.SetReportCaller(true)

	if config.STDOut {
		logrusLogger.Out = os.Stdout
	}

	if len(config.OutputFile) > 0 {
		outFile, e := createLogFile(config.OutputFile)
		if e != nil {
			return nil, e
		}
		logrusLogger.Out = outFile
		if config.STDOut {
			logrusLogger.Out = io.MultiWriter(os.Stdout, outFile)
		}
	}

	return &Logger{logrusLogger}, nil
}

// NewNoopLogger discards logging.
func NewNoopLogger() *Logger {
	logrusLogger := logrus.New()
	logrusLogger.Out = ioutil.Discard
	return &Logger{logrusLogger}
}

func getFormatter() logrus.Formatter {
	formatter := &logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			functionFullName := fmt.Sprintf("%s()", f.Function)
			functionArr := strings.Split(functionFullName, "/")
			functionName := functionArr[len(functionArr)-1]
			return functionName, fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}
	formatter.FullTimestamp = true
	return formatter
}

func createLogFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, fileMode)
	if err != nil {
		log.Fatalf("error creating log file %v, err=%v", path, err)
		return nil, err
	}
	return file, nil
}

func parseLogrusLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}

package logger

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

type LogFormat string

const FORMAT_JSON LogFormat = "json"
const FORMAT_TEXT LogFormat = "text"

type Logger struct {
	*logrus.Entry
}

func New(lvl string, fmt LogFormat, component string) *Logger {
	l := logrus.New()

	if fmt == FORMAT_JSON {
		l.SetFormatter(&logrus.JSONFormatter{})
	} else {
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
	l.SetOutput(os.Stderr)

	l.SetLevel(logrus.WarnLevel)
	parsedLvl, err := logrus.ParseLevel(lvl)
	if err != nil {
		log.Printf("error when parsing log level %s: %v", lvl, err)
	} else {
		l.SetLevel(parsedLvl)
	}
	entry := l.WithField("component", component)
	entry.Level = parsedLvl

	return &Logger{entry}
}

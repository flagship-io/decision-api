package logger

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Entry
}

func New(lvl string, component string) *Logger {
	l := logrus.New()

	l.SetFormatter(&logrus.TextFormatter{})
	l.SetOutput(os.Stderr)

	l.SetLevel(logrus.WarnLevel)
	parsedLvl, err := logrus.ParseLevel(lvl)
	if err != nil {
		log.Printf("error when parsing log level %s: %v", lvl, err)
	} else {
		l.SetLevel(parsedLvl)
	}

	return &Logger{l.WithField("component", component)}
}

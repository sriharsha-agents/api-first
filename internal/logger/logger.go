package logger

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	once sync.Once
	log  *logrus.Logger
)

// Logger is a singleton that outputs structured JSON logs to stdout/stderr
// so the enterprise can ingest them into their SIEM systems (e.g., Splunk).
func Logger() *logrus.Logger {
	once.Do(func() {
		log = logrus.New()
		log.Out = os.Stdout
		log.SetReportCaller(true)

		// Default to JSON formatter for structured log ingestion.
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			PrettyPrint:   false,
		})

		// Default level is Info; override via LOG_LEVEL env var.
		if lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
			log.SetLevel(lvl)
		}
	})
	return log
}
package elog

import(
	"log"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

type logWriter struct {
    writer io.Writer
	timeFormat string
	user string
	app string
	hostname string
}

func (w logWriter) Write(b []byte) (n int, err error) {
	return w.writer.Write([]byte(time.Now().Format(w.timeFormat) + "\t" + w.app + "@" + w.hostname + "\t" + w.user + "\t" + strconv.Itoa(os.Getpid()) + "\t" + string(b)))
}

type ELogger struct {
	log.Logger
	debug bool
}

func (logger ELogger) Info(args ...interface{}) {
	logger.Println(args...)
}

func (logger ELogger) Debug(args ...interface{}) {
	if logger.debug {
		logger.Println(args...)
	}
}

func GetELogger(logToFile string, logSource string, debug bool) ELogger {
	return ELogger{
		Logger: GetLogger(logToFile, logSource),
		debug: debug,
	}
}

func GetLogger(logToFile string, logSource string) log.Logger {

	var logger *log.Logger

	if len(logSource) == 0 {
		logSource = "ep"
	}

	if logToFile == "" {
		logger = log.New(ioutil.Discard, "", 0)
	} else { // enable logging
		// get user info for log
		userobj, err := user.Current()
		usern := "-"
		if err != nil {
			log.Println("failed getting current user")
		}
		usern = userobj.Username

		// get host info for log
		hostname, err := os.Hostname()
		if err != nil {
			log.Println("failed getting machine hostname")
			hostname = "-"
		}

		if logToFile == "-" { // to stdout
			logger = log.New(os.Stdout, "", 0)
			logger.SetFlags(0)
			logger.SetOutput(logWriter{writer: logger.Writer(), timeFormat: "2006-01-02T15:04:05.999Z07:00", user: usern, app: logSource, hostname: hostname})
		} else { // to file
			// get binary path
			logpath := logToFile
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				log.Println("couldn't get binary path - logfile path is relative to exec dir")
			} else {
				logpath = dir + "/" + logToFile
			}

			f, err := os.OpenFile(logpath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
			if err != nil {
				log.Println("opening file " + logToFile + " failed, writing log to stdout")
			} else {
				logger = log.New(f, "", 0)
				logger.SetOutput(logWriter{writer: logger.Writer(), timeFormat: "2006-01-02T15:04:05.999Z", user: usern, app: logSource, hostname: hostname})
			}
		}
	}

	return *logger
}
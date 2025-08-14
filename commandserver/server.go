package commandserver

import (
	"errors"
	"os"

	"github.com/apoindevster/bitwarp/proto"
	log "github.com/charmbracelet/log"
)

type Server struct {
	proto.UnimplementedCommandServer
}

var logFile *os.File = nil
var Logger *log.Logger = nil

func SetupLogger(filePath string, isJson bool) error {
	var err error
	if filePath == "" {
		logFile = os.Stderr
	} else {
		logFile, err = os.OpenFile(".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return errors.New("failed to instantiate logger file")
		}
	}

	if Logger == nil {
		Logger = log.New(logFile)
	}

	if isJson {
		Logger.SetFormatter(log.JSONFormatter)
	} else {
		Logger.SetFormatter(log.TextFormatter)
	}

	Logger.SetOutput(logFile)
	return nil

}

func SetLogger(logger *log.Logger) error {
	if logger == nil {
		return errors.New("invalid logger pointer")
	}

	Logger = logger
	return nil
}

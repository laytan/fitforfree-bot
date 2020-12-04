package logs

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/laytan/go-fff-notifications-bot/termux"
)

// SetupLogs sets up the logger to log to file and stdout. Filename is dependant on the environment
func SetupLogs() *os.File {
	env, isSet := os.LookupEnv("ENV")
	if !isSet {
		env = "development"
	}

	logFile, err := os.OpenFile(fmt.Sprintf("logs/%s.log", env), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	// Set up logging so it writes to stdout and to a file
	wrt := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(wrt)

	return logFile
}

// SendNotification sends a notification to the device running termux if it has termux
func SendNotification(title string, msg string, fullVolume bool) {
	if !termux.IsInstalled() {
		return
	}

	var currVolume termux.Volume
	if fullVolume {
		currVolume, err := termux.VolumeOf("notification")
		if err != nil {
			log.Printf("Error retrieving current volume, err: %+v", err)
			return
		}

		termux.SetVolume("notification", currVolume.MaxVolume)
	}

	termux.Notificate(title, msg, nil)

	if fullVolume {
		termux.SetVolume("notification", currVolume.Volume)
	}
}

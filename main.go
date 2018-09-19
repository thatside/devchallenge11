package main

import (
	"./unchainer"
	"os"
	"time"
	"fmt"
)

const redirectInterval = 5 * time.Second //this interval stands for the time to wait for new redirects
const fileReadTimeout = 5 * time.Second  //this interval is the time to wait for response of input file request if the file is not local

// main Launches the program
// All arguments are retrieved from env variables to be easily compatible with Docker
func main() {
	chromeURL := os.Getenv("CHROME_URL")
	inputFile := os.Getenv("INPUT_FILE")
	logFile := os.Getenv("LOG_FILE")
	quietMode := os.Getenv("QUIET_MODE") == "true"

	if chromeURL == "" {
		fmt.Println("You should provide Chrome devtools URL")
		return
	}

	if inputFile == "" {
		fmt.Println("You should provide input file path")
		return
	}

	uc := unchainer.Unchainer{}
	uc.Init(chromeURL, redirectInterval, quietMode, logFile)

	_, err := uc.UnchainFromFile(inputFile, fileReadTimeout)
	if err != nil {
		panic(err)
	}

	uc.Close()
}

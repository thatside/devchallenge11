package unchainer

import (
	"fmt"
	"log"
	"os"
)

// Output provides methods to output result data to either stdout or log file
type Output struct {
	quietMode bool
	logFile   *log.Logger
}

type Printer interface {
	OutputStarted(url string)
	OutputFinished(url string)
	OutputWent(url string)
}

// InitOutput initialize logger if needed (logger used for simple formatting only)
func InitOutput(quietMode bool, logFile string) (*Output, error) {

	o := Output{}

	o.quietMode = quietMode
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return nil, err
		}
		o.logFile = log.New(file, "", log.LstdFlags)
	}

	return &o, nil
}

// OutputStarted notify that link unchaining is started
func (o *Output) OutputStarted(url string) {
	out := fmt.Sprintf("Starting with link %s", url)
	o.output(out)
}

// OutputWent notify that link unchaining is in process - found another link
func (o *Output) OutputWent(url string) {
	out := fmt.Sprintf("Went to: %s", url)
	o.output(out)
}

// OutputFinished notify that link unchaining is finished
func (o *Output) OutputFinished(url string) {
	out := fmt.Sprintf("Ended chain for link %s", url)
	o.output(out)
}

// output print line to either stdout or log file or both
func (o *Output) output(out string) {
	if o.quietMode != true {
		fmt.Println(out)
	}
	if o.logFile != nil {
		o.logFile.Println(out)
	}
}

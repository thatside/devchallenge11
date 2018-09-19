package unchainer

import (
	"errors"
	"time"
)
// Unchainer Does all the work
type Unchainer struct {
	lc  *LinkChecker
	out *Output
}

// Init Initialize Unchainer - initializes both checker and output
func (uc *Unchainer) Init(devtoolsURL string, redirectTimeout time.Duration, quietMode bool, logFile string) {
	lc, err := InitChecker(devtoolsURL, redirectTimeout)
	if err != nil {
		panic(err)
	}
	uc.lc = lc

	out, err := InitOutput(quietMode, logFile)
	if err != nil {
		panic(err)
	}
	uc.out = out
}

// UnchainFromFile Unchain links from real of network-served file
func (uc *Unchainer) UnchainFromFile(filePath string, timeout time.Duration) ([]Result, error) {
	if !uc.valid() {
		return nil, errors.New("initialize unchainer first")
	}
	data, err := Load(filePath, timeout)
	if err != nil {
		return nil, err
	}

	results := uc.unchain(data)

	return results, nil
}

// UnchainFromObject Unchain links from prepared object (for another usage scenarios)
func (uc *Unchainer) UnchainFromObject(inputData *InputData) ([]Result, error) {
	if !uc.valid() {
		return nil, errors.New("initialize unchainer first")
	}

	results := uc.unchain(inputData)

	return results, nil
}

// unchain receives prepared data and calls checker to execute all the checks and print output
func (uc *Unchainer) unchain(data *InputData) []Result {
	allResults := make([]Result, len(data.Links))

	for _, link := range data.Links {
		uc.out.OutputStarted(link.URL.String())

		result, err := uc.lc.Check(link.URL.String())
		if err != nil {
			panic(err)
		}

		for _, link := range result.Chain {
			uc.out.OutputWent(link)
		}

		uc.out.OutputFinished(link.URL.String())

		allResults = append(allResults, *result)
	}

	return allResults
}

// valid checks if Unchainer instance is valid
func (uc *Unchainer) valid() bool {
	return uc.lc != nil && uc.out != nil
}

// Close closes all the resources
func (uc *Unchainer) Close() {
	uc.lc.Close()
}

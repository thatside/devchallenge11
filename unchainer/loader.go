package unchainer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)


func isPath(path string) bool {
	fi, err := os.Stat(path)
	return !os.IsNotExist(err) && !fi.IsDir()
}

func isURL(path string) bool {
	_, err := url.ParseRequestURI(path)

	return err == nil
}

// Load loads file from filesystem path or network URL and unmarshals
func Load(path string, timeout time.Duration) (*InputData, error) {
	isFile, isURL := isPath(path), isURL(path)

	if !isFile && !isURL {
		err := errors.New("file not found")
		return nil, err
	}

	var data []byte
	var err error

	if isFile {
		data, err = loadFileFromPath(path)
		if err != nil {
			return nil, err
		}
	}

	if isURL {
		data, err = loadFileFromURL(path, timeout)
		if err != nil {
			return nil, err
		}
	}

	var obj InputData

	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}

func loadFileFromPath(filePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, err
}

func loadFileFromURL(fileURL string, timeout time.Duration) ([]byte, error) {
	client := http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, errors.New("file not found")
}

package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/gommon/log"
)

const MaxWaitInSeconds = 8

func HTTPGet(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("failed to prep request for url ('%s'), %v", url, err)
		log.Error(err)
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute GET request for url ('%s'), %v", url, err)
		log.Error(err)
		return nil, err
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response for GET request with url ('%s'), %v", url, err)
		log.Error(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d status code for GET request with url ('%s')", response.StatusCode, url)
		log.Error(err)
		log.Infof("response: '%s'", string(responseBody))
		return nil, err
	}

	return responseBody, nil
}

func HTTPPost(url string, body []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: MaxWaitInSeconds * time.Second,
	}
	response, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		err = fmt.Errorf("post request for url ('%s') failed, %v", url, err)
		log.Error(err)
		return nil, err
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response for POST request with url ('%s'), %v", url, err)
		log.Error(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d status code for POST request with url ('%s')", response.StatusCode, url)
		log.Error(err)
		log.Infof("response: '%s'", string(responseBody))
		return nil, err
	}

	return responseBody, nil
}

package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/gommon/log"
)

type HTTPParams struct {
	URL              string
	RequestBody      []byte
	MaxWaitInSeconds int
}

const MaxWaitInSeconds = 10

func timeout(params HTTPParams) time.Duration {
	if params.MaxWaitInSeconds <= 0 {
		return time.Duration(MaxWaitInSeconds) * time.Second
	}
	return time.Duration(params.MaxWaitInSeconds) * time.Second
}

func HTTPGet(params HTTPParams) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout(params),
	}

	req, err := http.NewRequest("GET", params.URL, nil)
	if err != nil {
		err = fmt.Errorf("failed to prep request for url ('%s'), %w", params.URL, err)
		log.Error(err)
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute GET request for url ('%s'), %w", params.URL, err)
		log.Error(err)
		return nil, err
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response for GET request with url ('%s'), %w", params.URL, err)
		log.Error(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d status code for GET request with url ('%s')", response.StatusCode, params.URL)
		log.Error(err)
		log.Infof("response: '%s'", string(responseBody))
		return nil, err
	}

	return responseBody, nil
}

func HTTPPost(params HTTPParams) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout(params),
	}
	response, err := client.Post(params.URL, "application/json", bytes.NewBuffer(params.RequestBody))
	if err != nil {
		err = fmt.Errorf("post request for url ('%s') failed, %w", params.URL, err)
		log.Error(err)
		return nil, err
	}
	defer response.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response for POST request with url ('%s'), %w", params.URL, err)
		log.Error(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%d status code for POST request with url ('%s')", response.StatusCode, params.URL)
		log.Error(err)
		log.Infof("response: '%s'", string(responseBody))
		return nil, err
	}

	return responseBody, nil
}

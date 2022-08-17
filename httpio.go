package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	BaseURLV1 = "https://api.facest.io/v1"
)

func HttpPost(url, mimetype, data string) (int, error) {
	resp, err := http.Post(url, "text/plain", strings.NewReader(data))
	if err != nil {
		return 999, err
	}
	return resp.StatusCode, nil
}

func HttpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return string(body), fmt.Errorf("code '%d'. URL '%s'. message:'%s'", resp.StatusCode, url, body)
	}
	return string(body), nil
}

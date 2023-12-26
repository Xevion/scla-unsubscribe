package main

import (
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

func DoRequestNoRead(req *http.Request) (*http.Response, error) {
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("path", req.URL.Path).Msg("Request")
	resp, err := client.Do(req)

	if err != nil {
		log.Error().Err(err).Msg("Error making request")
		return nil, err
	}

	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Str("content-length", resp.Header.Get("Content-Length")).Msg("Response")

	return resp, nil
}

func DoRequest(req *http.Request) (*http.Response, []byte, error) {
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("path", req.URL.Path).Msg("Request")
	resp, err := client.Do(req)

	if err != nil {
		log.Error().Err(err).Msg("Error making request")
		return nil, nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading response body")
		return nil, nil, err
	}

	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Int("content-length", len(body)).Msg("Response")

	return resp, body, nil
}

func ApplyHeaders(req *http.Request) {
	req.Header.Set("Origin", "http://www2.thescla.org")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:122.0) Gecko/20100101 Firefox/122.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
}

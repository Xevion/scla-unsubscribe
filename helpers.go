package main

import (
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// DoRequestNoRead makes a request and returns the response
// Compared to DoRequest, this function does not read the response body, and it uses the Content-Length header for the associated log attribute.
// This function encapsulates the boilerplate for logging.
func DoRequestNoRead(req *http.Request) (*http.Response, error) {
	// Log the request
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("path", req.URL.Path).Msg("Request")

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Error().Err(err).Msg("Request Error")
		return nil, err
	}

	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Str("content-length", resp.Header.Get("Content-Length")).
		Str("duration", duration.String()).Msg("Response")

	return resp, nil
}

// DoRequest makes a request and returns the response and body
// This function encapsulates the boilerplate for logging and reading the response body
func DoRequest(req *http.Request) (*http.Response, []byte, error) {
	// Log the request
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("path", req.URL.Path).Msg("Request")

	// Send the request (while acquiring timings)
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	// Handle errors
	if err != nil {
		log.Error().Err(err).Msg("Request Error")
		return nil, nil, err
	}

	// Read the body
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Err(err).Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Int("content-length", len(body)).
			Str("duration", duration.String()).Msg("Response (Unable to Read Body)")
		return nil, nil, err
	}

	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Int("content-length", len(body)).
		Str("duration", duration.String()).Msg("Response")
	return resp, body, nil
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:122.0) Gecko/20100101 Firefox/122.0"

// ApplySclaHeaders applies headers to a request for thescla.org
func ApplySclaHeaders(req *http.Request) {
	req.Header.Set("Origin", "http://www2.thescla.org")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
}

// ApplyUtsaHeaders applies headers to a request for utsa.edu
func ApplyUtsaHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
}

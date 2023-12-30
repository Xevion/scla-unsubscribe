package main

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

var DomainLimiters = map[string]*rate.Limiter{
	"utsa.edu": rate.NewLimiter(2, 5),
}

func GetLimiter(domain string) *rate.Limiter {
	// Naively simplify the domain
	simplifiedDomain := SimplifyUrlToDomain(domain)
	if simplifiedDomain != domain {
		log.Debug().Str("domain", domain).Str("simplified", simplifiedDomain).Msg("Domain Simplified")
	}

	// Get the limiter
	limiter, ok := DomainLimiters[simplifiedDomain]

	// Create a new limiter if one does not exist
	if !ok {
		limiter = rate.NewLimiter(1, 3)
		DomainLimiters[simplifiedDomain] = limiter
		log.Debug().Str("domain", domain).Msg("New Limiter Created")
	}
	return limiter
}

var DomainPattern = regexp.MustCompile(`(?:\w+\.)*(\w+\.\w+)(?:\/)?`)

func SimplifyUrlToDomain(url string) string {
	// Find the domain
	matches := DomainPattern.FindStringSubmatch(url)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

func Wait(limiter *rate.Limiter, ctx context.Context) {
	r := limiter.Reserve()
	if !r.OK() {
		log.Warn().Msg("Rate Limit Exceeded")
		return
	}

	// Wait for the limiter
	if r.Delay() > 0 {
		log.Debug().Str("delay", r.Delay().String()).Msg("Waiting")
		time.Sleep(r.Delay())
	}
}

// DoRequestNoRead makes a request and returns the response
// Compared to DoRequest, this function does not read the response body, and it uses the Content-Length header for the associated log attribute.
// This function encapsulates the boilerplate for logging.
func DoRequestNoRead(req *http.Request) (*http.Response, error) {
	// Acquire the limiter, and wait for a token
	limiter := GetLimiter(req.URL.Host)
	Wait(limiter, req.Context())

	// Log the request
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("path", req.URL.Path).Msg("Request")

	// Send the request (while acquiring timings)
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
	// Acquire the limiter, and wait for a token
	limiter := GetLimiter(req.URL.Host)
	Wait(limiter, req.Context())

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

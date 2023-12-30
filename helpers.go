package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/icrowley/fake"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/time/rate"
)

var DomainLimiters = map[string]*rate.Limiter{
	"utsa.edu":    rate.NewLimiter(2, 5),
	"thescla.org": rate.NewLimiter(3, 7),
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

// This will select multiple groups, but the first group is all that matters
var DomainPattern = regexp.MustCompile(`(?:\w+\.)*(\w+\.\w+)(?:\/)?`)

// SimplifyUrlToDomain transforms a url into a common simplified domain
// This is not the same as the host, as it removes subdomains (www, asap, etc.)
// This helps me group together domains that are related to eachother, such as those at UTSA.
func SimplifyUrlToDomain(url string) string {
	// Find the domain
	matches := DomainPattern.FindStringSubmatch(url)
	if len(matches) == 0 {
		return ""
	}
	return matches[1]
}

// Wait waits for a token from the limiter
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
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("url", req.URL.String()).Msg("Request")

	// Send the request (while acquiring timings)
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Error().Err(err).Msg("Request Error")
		return nil, err
	}

	contentLength, err := strconv.ParseUint(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		contentLength = 0
	}
	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Str("content-length", Bytes(contentLength)).
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
	log.Debug().Str("method", req.Method).Str("host", req.Host).Str("url", req.URL.String()).Msg("Request")
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
		log.Err(err).Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Str("content-length", Bytes(uint64(len(body)))).
			Str("duration", duration.String()).Msg("Response (Unable to Read Body)")
		return nil, nil, err
	}

	log.Debug().Int("code", resp.StatusCode).Str("content-type", resp.Header.Get("Content-Type")).Str("content-length", Bytes(uint64(len(body)))).
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

var nonAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var continuousWhitespace = regexp.MustCompile(`\s+`)

// NormalizeTitle creates a normalized title from a string, providing a consistent format regardless of whitespace or non-alphanumeric characters
//
// Non-alphanumeric characters are removed
// Capital letters are converted to lowercase
// Whitespace at the beginning or end of the string is removed
// Whitespace of any continuous length is replaced with a single dash
//
// Examples:
//
//	"Mailing Address" => "mailing-address"
//	"Mailing   | Address" => "mailing-address"
//	"  Mailing Address  " => "mailing-address"
//	"  Mailing   | Address  " => "mailing-address"
func NormalizeTitle(title string) string {
	return continuousWhitespace.ReplaceAllString(
		strings.TrimSpace(
			strings.ToLower(
				nonAlphaNumeric.ReplaceAllString(
					title, " ",
				),
			),
		),
		"-",
	)
}

// Bytes calls humanize.Bytes and removes space characters
func Bytes(bytes uint64) string {
	return strings.Replace(humanize.Bytes(bytes), " ", "", -1)
}

// RandBool returns a random boolean
func RandBool() bool {
	return rand.Uint64()&1 == 1
}

// FakeEmail generates a fake email address
func FakeEmail() string {
	return strings.ToLower(fmt.Sprintf("%s.%s@%sutsa.edu", fake.FirstName(), fake.LastName(), lo.Ternary(RandBool(), "my.", "")))
}

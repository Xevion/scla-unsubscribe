package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func Login(username string, password string) error {
	// Setup initial redirected request
	directoryPageUrl, _ := url.Parse("https://www.utsa.edu/directory/Directory?action=Index")
	request, _ := http.NewRequest("GET", directoryPageUrl.String(), nil)
	ApplyUtsaHeaders(request)
	response, err := DoRequestNoRead(request)
	if err != nil {
		return nil
	}

	// Verify that we were redirected to the login page
	if response.StatusCode != 302 {
		return fmt.Errorf("bad request (no initial redirect)")
	} else {
		log.Debug().Str("location", response.Header.Get("Location")).Msg("Initial Page Redirected")
	}

	// Setup URL for request
	loginPageUrl, _ := url.Parse("https://www.utsa.edu/directory/Account/Login")
	query := loginPageUrl.Query()
	query.Set("ReturnUrl", "/directory/AdvancedSearch")
	loginPageUrl.RawQuery = query.Encode()

	// Build request
	request, _ = http.NewRequest("GET", loginPageUrl.String(), nil)
	ApplyUtsaHeaders(request)

	// Send request
	response, err = DoRequestNoRead(request)
	if err != nil {
		log.Fatal().Err(err).Msg("Error sending login page request")
	}
	doc, err := goquery.NewDocumentFromReader(response.Body)
	defer response.Body.Close()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing response body")
	}

	// Get token
	token, _ := doc.Find("input[name='__RequestVerificationToken']").Attr("value")
	log.Debug().Str("token", token).Msg("Token Captured")

	// Build the login request
	form := url.Values{
		"__RequestVerificationToken": {token},
		"myUTSAID":                   {username},
		"passphrase":                 {password},
		"log-me-in":                  {"Log+In"},
	}
	log.Debug().Str("form", form.Encode()).Msg("Form Encoded")
	request, _ = http.NewRequest("POST", "https://www.utsa.edu/directory/", strings.NewReader(form.Encode()))
	ApplyUtsaHeaders(request)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the login request
	response, body, err := DoRequest(request)

	if err != nil {
		log.Fatal().Err(err).Msg("Error sending login request")
	}

	if response.StatusCode != 200 {
		switch response.StatusCode {
		case 302: // ignore

		case 500:
			return fmt.Errorf("bad request (check cookies)")
		default:
			return fmt.Errorf("unknown error")
		}
	}

	// Check for Set-Cookie of ".ADAuthCookie"
	newCookies := response.Header.Values("Set-Cookie")
	authCookie, found := lo.Find(newCookies, func(cookie string) bool {
		return strings.Contains(cookie, ".ADAuthCookie")
	})

	if !found {
		return fmt.Errorf("login failed: could not find auth cookie")
	} else {
		log.Info().Str("authCookie", authCookie).Msg("Auth Cookie Found")
	}

	// Check if redirected to directory page
	if response.Header.Get("Location") != "" {
		log.Debug().Str("location", response.Header.Get("Location")).Msg("Redirected")
	} else {
		return fmt.Errorf("login failed: no redirect")
	}

	// Request the redirect page
	redirectUrl := fmt.Sprintf("%s%s", "https://www.utsa.edu", response.Header.Get("Location"))
	request, _ = http.NewRequest("GET", redirectUrl, nil)
	ApplyUtsaHeaders(request)
	response, err = DoRequestNoRead(request)
	if err != nil {
		return err
	} else if response.StatusCode != 200 {
		return fmt.Errorf("non-200 status after login attempt")
	}

	// Parse the response body
	doc, err = goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return fmt.Errorf("error parsing response body")
	}

	// Look for field validation errors (untested)
	validationErrors := doc.Find("span.field-validation-error")
	if validationErrors.Length() > 0 {
		event := log.Debug().Int("validationErrors", validationErrors.Length())
		validationErrors.Each(func(i int, s *goquery.Selection) {
			event.Str(fmt.Sprintf("err_%d", i+1), s.Text())
		})
		return fmt.Errorf("validation error: %s", validationErrors.First().Text())
	}

	// Look for the 'Log Off' link
	logOffFound := false
	doc.Find("a.dropdown-item").Each(func(i int, s *goquery.Selection) {
		if !logOffFound && strings.Contains(s.Text(), "Log Off") {
			log.Debug().Int("index", i).Msg("Log Off Element Found")
			logOffFound = true
		}
	})

	if !logOffFound {
		return fmt.Errorf("login failed: could not find log off element")
	}

	return nil
}

func CheckLoggedIn() (bool, error) {
	// Check if required cookie exists
	utsaUrl, _ := url.Parse("https://www.utsa.edu")
	cookies := client.Jar.Cookies(utsaUrl)
	_, authCookieFound := lo.Find(cookies, func(cookie *http.Cookie) bool {
		return cookie.Name == ".ADAuthCookie"
	})

	if !authCookieFound {
		log.Debug().Int("count", len(cookies)).Msg("ActiveDirectory Auth Cookie Not Found")
		return false, nil
	}

	// Send a authenticated-only request
	directoryPageUrl, _ := url.Parse("https://www.utsa.edu/directory/AdvancedSearch")
	request, _ := http.NewRequest("GET", directoryPageUrl.String(), nil)
	ApplyUtsaHeaders(request)
	response, err := DoRequestNoRead(request)
	if err != nil {
		return false, fmt.Errorf("could not send redirect check request")
	}

	// If it's not a 302
	if response.StatusCode != 302 {
		// No planning for non-200 responses (this will blow up one day, probably a 400 or 500)
		if response.StatusCode != 200 {
			log.Fatal().Int("code", response.StatusCode).Msg("Unexpected Login Check Response Code")
		}

		// Parse the response document
		doc, err := goquery.NewDocumentFromReader(response.Body)
		if err != nil {
			return false, fmt.Errorf("error parsing response body")
		}

		// Try to find the log out button
		logOffFound := false
		doc.Find("a.dropdown-item").Each(func(i int, s *goquery.Selection) {
			if !logOffFound && strings.Contains(s.Text(), "Log Off") {
				log.Debug().Int("index", i).Msg("Log Off Element Found")
				logOffFound = true
			}
		})
	}

	return false, nil
}

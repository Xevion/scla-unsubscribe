package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
)

func Login(username string, password string) {
	// Setup URL for request
	loginPageUrl, _ := url.Parse("https://www.utsa.edu/directory/Account/Login")
	query := loginPageUrl.Query()
	query.Set("ReturnUrl", "/directory/AdvancedSearch")
	loginPageUrl.RawQuery = query.Encode()

	// Build request
	request, _ := http.NewRequest("GET", loginPageUrl.String(), nil)
	ApplyHeaders(request)

	// Send request
	response, err := DoRequestNoRead(request)
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
		"myUTSAID":                   {username},
		"passphrase":                 {password},
		"__RequestVerificationToken": {token},
		"log-me-in":                  {"Log+In"},
	}
	request, _ = http.NewRequest("POST", "https://www.utsa.edu/directory/", strings.NewReader(form.Encode()))
	ApplyHeaders(request)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the login request
	response, _, err = DoRequest(request)

	if err != nil {
		log.Fatal().Err(err).Msg("Error sending login request")
	}

	if response.StatusCode != 200 {
		switch response.StatusCode {
		case 500:
			log.Fatal().Str("status", response.Status).Msg("Bad Request (check cookies)")
		default:
			log.Fatal().Str("status", response.Status).Msg("Failed to Login, Unknown Error")
		}
	}

	// TODO: Check if login was successful
}

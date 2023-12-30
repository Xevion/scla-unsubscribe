package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/pkg/errors"
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
		return errors.Wrap(err, "error sending initial request")
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
	request, _ = http.NewRequest("POST", "https://www.utsa.edu/directory/", strings.NewReader(form.Encode()))
	ApplyUtsaHeaders(request)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the login request
	response, err = DoRequestNoRead(request)

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
		return errors.Wrap(err, "error sending redirect request")
	} else if response.StatusCode != 200 {
		return fmt.Errorf("non-200 status after login attempt")
	}

	// Parse the response body
	doc, err = goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return errors.Wrap(err, "error parsing response body")
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
		return false, errors.Wrap(err, "could not send redirect check request")
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
			return false, errors.Wrap(err, "error parsing response body")
		}

		// Try to find the log out button
		logOffFound := false
		doc.Find("a.dropdown-item").Each(func(i int, s *goquery.Selection) {
			if !logOffFound && strings.Contains(s.Text(), "Log Off") {
				log.Debug().Int("index", i).Msg("Log Off Element Found")
				logOffFound = true
			}
		})
		return true, nil
	}

	return false, nil
}

func GetFullDirectory() ([]Entry, error) {
	entries := make([]Entry, 0, 500)
	for letter := 'A'; letter <= 'Z'; letter++ {
		letterEntries, err := GetDirectoryCached(letter)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get directory")
		}

		entries = append(entries, letterEntries...)
	}

	return entries, nil
}

func GetDirectoryCached(letter rune) ([]Entry, error) {
	key := fmt.Sprintf("directory:%s", string(letter))

	// Check if cached
	var entries []Entry
	err := db.View(func(txn *badger.Txn) error {
		directoryItem, err := txn.Get([]byte(key))

		// Check if key was found
		if err == badger.ErrKeyNotFound {
			log.Warn().Str("key", key).Msg("Directory Cache Not Found")
			return nil
		} else if err != nil {
			return errors.Wrap(err, "failed to get directory cache")
		}

		// Try to read the value
		entries = make([]Entry, 0, 500)
		return directoryItem.Value(func(val []byte) error {
			err := json.Unmarshal(val, &entries)
			return errors.Wrap(err, "failed to unmarshal directory entries")
		})
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to load from cache")
	}

	// If cached, return it
	if entries != nil {
		return entries, nil
	}

	// If not cached, get it
	entries, err = GetDirectory(letter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get directory")
	}

	// Cache it
	err = db.Update(func(txn *badger.Txn) error {
		// Marshal cookies
		marshalledEntries, err := json.Marshal(entries)
		if err != nil {
			return errors.Wrap(err, "failed to marshal directory entries")
		}

		// create transaction
		log.Debug().Str("letter", string(letter)).Str("key", key).Msg("Saving to Directory Cache")
		return txn.Set([]byte(key), []byte(marshalledEntries))
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to save to cache")
	}

	return entries, nil
}

func GetDirectory(letter rune) ([]Entry, error) {
	// Build the request
	directoryPageUrl, _ := url.Parse("https://www.utsa.edu/directory/SearchByLastName")
	query := directoryPageUrl.Query()
	query.Set("abc", string(letter))
	directoryPageUrl.RawQuery = query.Encode()

	// Send the request
	request, _ := http.NewRequest("GET", directoryPageUrl.String(), nil)
	ApplyUtsaHeaders(request)
	response, err := DoRequestNoRead(request)
	if err != nil {
		return nil, fmt.Errorf("error sending directory request")
	}

	// Parse the response
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body")
	}

	// Acquire selector
	rows := doc.Find("table#peopleTable > tbody > tr")
	entries := make([]Entry, 0, rows.Length())
	log.Debug().Int("count", rows.Length()).Msg("Rows Found")

	// Check number of rows
	if rows.Length() < 1 {
		return nil, fmt.Errorf("no rows found in directory")
	} else if rows.Length() <= 20 {
		log.Warn().Int("count", rows.Length()).Msg("Low number of rows found")
	}

	// Iterate over rows
	rows.Each(func(i int, s *goquery.Selection) {
		entry := Entry{}
		nameElement := s.Find("a.fullName")

		// Process the HREF URL into an actual ID
		personPath, exists := nameElement.Attr("href")
		valueIndex := strings.Index(personPath, "abc=")
		if !exists || valueIndex == -1 {
			log.Warn().Str("href", personPath).Msg("Could not find ID in HREF")
			return
		}
		unescapedId, err := url.QueryUnescape(personPath[valueIndex+4:])
		if err != nil {
			log.Warn().Str("href", personPath).Msg("Could not unescape ID")
			return
		}
		entry.Id = unescapedId

		entry.Name = strings.TrimSpace(nameElement.Text())

		entry.JobTitle = strings.TrimSpace(s.Find("span.jobtitle").Text())
		entry.Department = strings.TrimSpace(s.Find("span.dept").Text())
		entry.College = strings.TrimSpace(s.Find("span.college").Text())
		entry.Phone = strings.TrimSpace(s.Find("span.phone").Text())

		entries = append(entries, entry)
	})

	return entries, nil
}

func GetFullEntryCached(id string) (*FullEntry, error) {
	key := fmt.Sprintf("entry:%s", id)

	// Check if cached
	var entry FullEntry
	var cached bool
	err := db.View(func(txn *badger.Txn) error {
		entryItem, err := txn.Get([]byte(key))

		// Check if key was found
		if err == badger.ErrKeyNotFound {
			log.Warn().Str("key", key).Msg("Entry Cache Not Found")
			return nil
		} else if err != nil {
			return errors.Wrap(err, "failed to get entry cache")
		}

		// Try to read the value
		return entryItem.Value(func(val []byte) error {
			err := json.Unmarshal(val, &entry)
			cached = true
			return errors.Wrap(err, "failed to unmarshal entry")
		})
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to load from cache")
	}

	// If cached, return it
	if cached {
		return &entry, nil
	}

	// If not cached, get it
	entryPtr, err := GetFullEntry(id)
	if err != nil {
		return nil, err
	}

	// Cache it
	err = db.Update(func(txn *badger.Txn) error {
		// Marshal cookies
		marshalledEntry, err := json.Marshal(*entryPtr)
		if err != nil {
			return errors.Wrap(err, "failed to marshal entry")
		}

		// create transaction
		log.Debug().Str("id", id).Str("key", key).Msg("Saving to Entry Cache")
		return txn.Set([]byte(key), []byte(marshalledEntry))
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to save to cache")
	}

	return entryPtr, nil
}

func GetFullEntry(id string) (*FullEntry, error) {
	// Build the request
	directoryPageUrl, _ := url.Parse("https://www.utsa.edu/directory/Person_Detail")
	query := directoryPageUrl.Query()
	query.Set("abc", id)
	directoryPageUrl.RawQuery = query.Encode()

	// Send the request
	request, _ := http.NewRequest("GET", directoryPageUrl.String(), nil)
	ApplyUtsaHeaders(request)
	response, err := DoRequestNoRead(request)
	if err != nil {
		return nil, fmt.Errorf("error sending directory request")
	}

	// Parse the response
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body")
	}

	// Move all rows into a map
	rows := make(map[string]string)
	rowElements := doc.Find("table.detail > tbody > tr")
	log.Debug().Int("count", rowElements.Length()).Msg("Rows Found")

	// Check number of rows
	if rowElements.Length() < 1 {
		return nil, fmt.Errorf("no rows found")
	}

	// Iterate over rows
	rowElements.Each(func(i int, s *goquery.Selection) {
		// left hand column
		rowTitle := NormalizeTitle(strings.TrimRight(
			strings.TrimSpace(s.Find("th > strong").Text()), ":",
		))

		// right hand column, add to map
		rows[rowTitle] = strings.TrimSpace(s.Find("td").Text())
	})

	// Build the entry from the map
	entry := FullEntry{}

	entry.Classification = rows["classification"]
	delete(rows, "classification")

	entry.College = rows["college"]
	delete(rows, "college")

	entry.Major = rows["major"]
	delete(rows, "major")

	entry.Email = rows["email"]
	delete(rows, "email")

	entry.Title = rows["title"]
	delete(rows, "title")

	entry.Department = rows["department"]
	delete(rows, "department")

	entry.MailingAddress = rows["mailing-address"]
	delete(rows, "mailing-address")

	entry.Building = rows["building"]
	delete(rows, "building")

	entry.Phone = rows["phone"]
	delete(rows, "phone")

	entry.Other = rows

	// Multiple names found, collect and log
	nameElement := doc.Find("body > #main span.nameBold > strong")
	if nameElement.Length() > 1 {
		var names []string
		nameElement.Each(func(i int, s *goquery.Selection) {
			names = append(names, strings.TrimSpace(s.Text()))
		})
		log.Warn().Int("count", nameElement.Length()).Interface("names", names).Msg("Multiple Names Found")

		// Use the longest name
		entry.Name = lo.MaxBy(names, func(name string, max string) bool {
			return len(name) > len(max)
		})
	} else {
		entry.Name = strings.TrimSpace(nameElement.Text())
	}

	return &entry, nil
}

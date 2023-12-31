package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func Unsubscribe(email string) (*ConfirmationResponse, error) {
	// No idea what this is, but it doesn't seem to change?
	mktTok := "ODM5LU1PTC01NTIAAAGQRiDbOUWzUhLliVDxTHjxLfZDD1y0MxC47Wf_1C9UTbwEej3Tckhn_QteZR7p5Mpl3_f0ioPUyQ8XUceJ9a0PiOUJb_O3YIj8PwKNQEm4SseaSw"

	// Build referrer URL
	referrerUrl, _ := url.Parse("http://www2.thescla.org/UnsubscribePage.html")
	query := referrerUrl.Query()
	query.Add("mkt_unsubscribe", "1")
	query.Add("mkt_tok", mktTok)
	referrerUrl.RawQuery = query.Encode()

	// Build lpUrl
	thing := "839-MOL-552"
	lpUrl := fmt.Sprintf("http://%s.mktoweb.com/lp/%s/UnsubscribePage.html?cr={creative}&kw={keyword}", thing, thing)

	values := url.Values{
		"Email":         {email},
		"Unsubscribed":  {"Yes"},
		"formid":        {"1"},
		"lpId":          {"1"},
		"subId":         {"98"},
		"munchkinId":    {thing},
		"lpurl":         {lpUrl},
		"followupLpId":  {"2"},
		"cr":            {""},
		"kw":            {""},
		"q":             {""},
		"_mkt_trk":      {""},
		"formVid":       {"1"},
		"mkt_tok":       {mktTok},
		"_mktoReferrer": {referrerUrl.String()},
	}

	// Grab checksum fields
	fields := make([]string, 0, len(values))
	for key, _ := range values {
		fields = append(fields, key)
	}
	values.Set("checksumFields", strings.Join(fields, ","))

	// Calculate checksum
	checksum := sha256.Sum256([]byte(strings.Join(
		lo.Map(fields, func(field string, _ int) string {
			return values.Get(field)
		}), "|")))

	values.Set("checksum", fmt.Sprintf("%x", checksum))

	// Make request
	request, _ := http.NewRequest("POST", "http://www2.thescla.org/index.php/leadCapture/save2", strings.NewReader(values.Encode()))
	request.Header.Set("Referer", "http://www2.thescla.org/UnsubscribePage.html?mkt_unsubscribe=1")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	ApplySclaHeaders(request)

	// Send request
	response, body, err := DoRequest(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		// If JSON returned, parse the message for the error
		contentType := response.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return nil, UnsubscribeUnexpectedError{Message: string(body), Code: response.StatusCode}
		}

		// Parse the JSON
		var errorResponse ErrorResponse
		err := json.Unmarshal(body, &errorResponse)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing error response")
			return nil, UnsubscribeUnexpectedError{Message: string(body), Code: response.StatusCode}
		}

		switch errorResponse.Message {
		case "checksum invalid":
			return nil, ChecksumInvalidError(checksum)
		case "checksum missing":
			return nil, ChecksumMissingError(checksum)
		case "Rejected":
			return nil, UnsubscribeRejectedError(errorResponse.Message)
		}

		log.Error().Str("content-type", contentType).Str("body", string(body)).Msg("Unknown Error")
		return nil, UnsubscribeUnexpectedError{Message: string(body), Code: response.StatusCode}
	}

	var confirmation ConfirmationResponse
	json.Unmarshal(body, &confirmation)
	return &confirmation, nil
}

// CheckEmail checks if an email is unsubscribed in the database
// Returns true if the email
func CheckEmail(email string) (bool, error) {
	// If the email has uppercase characters, lowercase it
	if strings.ContainsAny(email, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		email = strings.ToLower(email)
	}

	var isUnsubscribed bool
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(email))
		if err == badger.ErrKeyNotFound {
			isUnsubscribed = false
			return nil
		} else if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			switch string(val) {
			case "1":
				isUnsubscribed = true
			case "0":
				isUnsubscribed = false
			default:
				return fmt.Errorf("invalid value for email %s: %s", email, string(val))

			}
			return nil
		})

		return err
	})

	if err != nil {
		return false, err
	}

	return isUnsubscribed, nil
}

// MarkEmail marks an email as unsubscribed in the database
func MarkEmail(email string) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(email), []byte("1"))
	})
}

func TryUnsubscribe(email string) (bool, error) {
	// Check if the email is already unsubscribed
	isUnsubscribed, err := CheckEmail(email)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if email is unsubscribed")
	}
	log.Debug().Str("email", email).Bool("isUnsubscribed", isUnsubscribed).Msg("Checking if email is unsubscribed")

	// If the email is already unsubscribed, return
	if isUnsubscribed {
		return false, nil
	}

	// Try to unsubscribe the email
	_, err = Unsubscribe(email)
	if err != nil {
		return false, errors.Wrap(err, "failed to unsubscribe email")
	}

	// If the email was successfully unsubscribed, mark it as such
	err = MarkEmail(email)
	if err != nil {
		return true, errors.Wrap(err, "failed to mark email as unsubscribed")
	}

	return true, nil
}

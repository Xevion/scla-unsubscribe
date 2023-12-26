package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var client = &http.Client{}

func init() {
	log.Logger = zerolog.New(logSplitter{}).With().Timestamp().Logger()
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

func Unsubscribe(email string) (*ConfirmationResponse, error) {
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

	checksum[0] = 0
	values.Set("checksum", fmt.Sprintf("%x", checksum))

	// Make request
	request, _ := http.NewRequest("POST", "http://www2.thescla.org/index.php/leadCapture/save2", strings.NewReader(values.Encode()))
	request.Header.Set("Referer", "http://www2.thescla.org/UnsubscribePage.html?mkt_unsubscribe=1")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	ApplyHeaders(request)

	// Send request
	response, body, err := DoRequest(request)
	if err != nil {
		panic(err)
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

func main() {
	conf, err := Unsubscribe("")
	if err != nil {
		log.Panic().Err(err).Msg("Failed to Unsubscribe")
	}

	log.Info().Str("formId", conf.FormId).Str("followUpUrl", conf.FollowUpUrl).Str("deliveryType", conf.DeliveryType).Str("followUpStreamValue", conf.FollowUpStreamValue).Str("aliId", conf.AliId).Msg("Unsubscribed")
}

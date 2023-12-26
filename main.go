package main

import (
	"net/http"
	"net/url"
	"strings"
)

type Confirmation struct {
	FormId              string `json:"formId"`
	FollowUpUrl         string `json:"followUpUrl"`
	DeliveryType        string `json:"deliveryType"`
	FollowUpStreamValue string `json:"followUpStreamValue"`
	AliId               string `json:"aliId"`
}

func Unsubscribe() {
	values := url.Values{
		"Email":          {"ryan.walters@my.utsa.edu"},
		"Unsubscribed":   {"Yes"},
		"formid":         {"1"},
		"lpId":           {"1"},
		"subId":          {"98"},
		"munchkinId":     {"839-MOL-552"},
		"lpurl":          {"http://839-MOL-552.mktoweb.com/lp/839-MOL-552/UnsubscribePage.html?cr={creative}&kw={keyword}"},
		"followupLpId":   {"2"},
		"cr":             {""},
		"kw":             {""},
		"q":              {""},
		"_mkt_trk":       {""},
		"formVid":        {"1"},
		"mkt_tok":        {"ODM5LU1PTC01NTIAAAGQRiDbOUWzUhLliVDxTHjxLfZDD1y0MxC47Wf_1C9UTbwEej3Tckhn_QteZR7p5Mpl3_f0ioPUyQ8XUceJ9a0PiOUJb_O3YIj8PwKNQEm4SseaSw"},
		"_mktoReferrer":  {"http://www2.thescla.org/UnsubscribePage.html?mkt_unsubscribe=1&mkt_tok=ODM5LU1PTC01NTIAAAGQRiDbOUWzUhLliVDxTHjxLfZDD1y0MxC47Wf_1C9UTbwEej3Tckhn_QteZR7p5Mpl3_f0ioPUyQ8XUceJ9a0PiOUJb_O3YIj8PwKNQEm4SseaSw"},
		"checksumFields": {"Email,Unsubscribed,formid,lpId,subId,munchkinId,lpurl,followupLpId,cr,kw,q,_mkt_trk,formVid,mkt_tok,_mktoReferrer"},
		"checksum":       {"062bb614ebe52a624577123e91c0e7e0d6d069178d2d23d7bd1fa9db85b05472"},
	}

	req, _ := http.NewRequest("POST", "http://www2.thescla.org/index.php/leadCapture/save2", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "http://www2.thescla.org")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "http://www2.thescla.org/UnsubscribePage.html?mkt_unsubscr")
}

func main() {

}

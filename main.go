package main

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/syndtr/goleveldb/leveldb"
)

var client *http.Client
var db *leveldb.DB

func init() {
	log.Logger = zerolog.New(logSplitter{}).With().Timestamp().Logger()
	db, err := leveldb.OpenFile("./cache", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Load .env
	godotenv.Load()

	username := os.Getenv("UTSA_USERNAME")
	password := os.Getenv("UTSA_PASSWORD")

	err := Login(username, password)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to login")
	}

	// email := strings.ToLower(fmt.Sprintf("%s.%s@my.utsa.edu", fake.FirstName(), fake.LastName()))

	// log.Debug().Str("email", email).Msg("Unsubscribing")
	// conf, err := Unsubscribe(email)

	// if err != nil {
	// 	log.Panic().Err(err).Msg("Failed to Unsubscribe")
	// }

	// log.Info().Str("formId", conf.FormId).Str("followUpUrl", conf.FollowUpUrl).Str("deliveryType", conf.DeliveryType).Str("followUpStreamValue", conf.FollowUpStreamValue).Str("aliId", conf.AliId).Msg("Unsubscribed")
}

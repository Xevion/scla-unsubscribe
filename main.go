package main

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var client *http.Client
var db *badger.DB

func init() {
	log.Logger = zerolog.New(logSplitter{}).With().Timestamp().Logger()

	// Initialize Badger db store
	var err error
	options := badger.DefaultOptions("./db/").WithLogger(badgerZerologLogger{})
	db, err = badger.Open(options)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	// Setup http client + cookie jar
	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}

	defer SaveCookies()
}

func SaveCookies() {
	jar := client.Jar.(*cookiejar.Jar)
	for _, cookie := range cookies {
		err := db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte(cookie.Name), []byte(cookie.Value))
			return err
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to save cookie")
		}
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

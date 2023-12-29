package main

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var (
	client *http.Client
	db     *badger.DB
)

func init() {
	log.Logger = zerolog.New(logSplitter{}).With().Timestamp().Logger()

	// Initialize Badger db store
	var err error
	options := badger.DefaultOptions("./db/").WithLogger(badgerZerologLogger{level: WARNING})
	db, err = badger.Open(options)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}

	// Setup http client + cookie jar
	jar, _ := cookiejar.New(nil)
	client = &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}

	// Load cookies from db
	LoadCookies()
}

func SaveCookies() {
	// Get cookies for UTSA.EDU
	utsaUrl, _ := url.Parse("https://www.utsa.edu")
	utsaCookies := lo.Map(client.Jar.Cookies(utsaUrl), func(cookiePointer *http.Cookie, _ int) http.Cookie {
		return *cookiePointer
	})

	log.Info().Interface("cookies", lo.Map(utsaCookies, func(cookie http.Cookie, _ int) string {
		return cookie.Name
	})).Msg("Saving Cookies")

	// Marshal cookies, create transaction
	marshalledCookies, _ := json.Marshal(utsaCookies)
	err := db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("utsa_cookies"), []byte(marshalledCookies))
		return err
	})
	if err != nil {
		log.Err(err).Msg("Failed to save marshalled cookies")
	}
}

func LoadCookies() {
	// Load cookies from DB
	var cookies []http.Cookie
	err := db.View(func(txn *badger.Txn) error {
		// Get cookies
		item, err := txn.Get([]byte("utsa_cookies"))
		if err != nil {
			return err
		}

		// Read the value, unmarshal
		err = item.Value(func(val []byte) error {
			err := json.Unmarshal(val, &cookies)
			return err
		})

		return err
	})

	if err != nil {
		log.Err(err).Msg("Failed to load marshalled cookies")
	}

	// Place cookies in the jar
	utsaUrl, _ := url.Parse("https://www.utsa.edu")
	client.Jar.SetCookies(utsaUrl, lo.Map(cookies, func(cookie http.Cookie, _ int) *http.Cookie {
		return &cookie
	}))

	log.Info().Interface("cookies", lo.Map(cookies, func(cookie http.Cookie, _ int) string {
		return cookie.Name
	})).Msg("Cookies Loaded")
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Load .env
	godotenv.Load()

	username := os.Getenv("UTSA_USERNAME")
	password := os.Getenv("UTSA_PASSWORD")

	log.Debug().Msg("Checking Login State")
	loggedIn, err := CheckLoggedIn()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to check login state")
	}

	if !loggedIn {
		log.Info().Str("username", username).Msg("Attempting Login")
		err := Login(username, password)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to login")
		}
	} else {
		log.Info().Msg("Login not required")
	}

	// print directory for Z
	_, err = GetDirectoryCached("Z")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get directory")
	}

	defer db.Close()
	defer SaveCookies()

	// email := strings.ToLower(fmt.Sprintf("%s.%s@my.utsa.edu", fake.FirstName(), fake.LastName()))

	// log.Debug().Str("email", email).Msg("Unsubscribing")
	// conf, err := Unsubscribe(email)

	// if err != nil {
	// 	log.Panic().Err(err).Msg("Failed to Unsubscribe")
	// }

	// log.Info().Str("formId", conf.FormId).Str("followUpUrl", conf.FollowUpUrl).Str("deliveryType", conf.DeliveryType).Str("followUpStreamValue", conf.FollowUpStreamValue).Str("aliId", conf.AliId).Msg("Unsubscribed")
}

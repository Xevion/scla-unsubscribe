package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var (
	client *http.Client
	db     *badger.DB

	// A channel that will be used to buffer incomplete entries that need to be queried properly
	incompleteEntries = make(chan Entry)

	// A channel that will be used to buffer emails that need to be unsubscribed
	entries = make(chan string)
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
	// stop := make(chan os.Signal, 1)
	// signal.Notify(stop, os.Interrupt)

	// Load .env
	godotenv.Load()
	username := os.Getenv("UTSA_USERNAME")
	password := os.Getenv("UTSA_PASSWORD")

	defer db.Close()
	defer SaveCookies()

	// Check if logged in
	log.Debug().Msg("Checking Login State")
	loggedIn, err := CheckLoggedIn()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to check login state")
	}

	// Login if required
	if !loggedIn {
		log.Info().Str("username", username).Msg("Attempting Login")
		err := Login(username, password)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to login")
		}
	} else {
		log.Info().Msg("Login Not Required")
	}

	SaveCookies()

	// Get the directory
	for letter := 'A'; letter <= 'Z'; letter++ {
		go func(letter rune) {
			letterEntries, err := GetDirectoryCached(letter)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get directory")
			}

			// Process each entry
			for _, entry := range letterEntries {
				incompleteEntries <- entry
			}
		}(letter)
	}

	// Process each incomplete entry
	go func() {
		for entry := range incompleteEntries {
			log.Debug().Str("name", entry.Name).Msg("Processing Entry")

			fullEntry, err := GetFullEntryCached(entry.Id)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get full entry")
			}

			log.Debug().Str("name", fullEntry.Name).Str("email", fullEntry.Email).Msg("Entry Processed")
			entries <- fullEntry.Email
		}
	}()

	// Process each email
	for email := range entries {
		log.Info().Str("email", email).Msg("Unsubscribing Email")
		go Unsubscribe(email)

		// 1/2 chance to unsubscribe fake email
		if rand.Intn(2) == 0 {
			log.Info().Str("email", email).Msg("Unsubscribing Fake Email")
			go Unsubscribe(email)
		}
	}
}

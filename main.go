package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"super-duper-fortnight/clkup"
	"super-duper-fortnight/dbstore"
	"super-duper-fortnight/oauth"
	"super-duper-fortnight/server"
	"super-duper-fortnight/tui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

var dblite *sql.DB
var err error

func main() {
	db, err := dbstore.InitDB("dbstore/local_cache.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	err = godotenv.Load()
	if err != nil {
		fmt.Println("Note: No .env file found")
	}

	token := db.GetToken()

	if token == "" {
		fmt.Println("No saved token found. Waiting for ClickUp authentication in your browser...")

		authCodeChan := make(chan string)
		go server.ServeGin(authCodeChan)
		oauth.Authenticate()

		myOAuthCode := <-authCodeChan
		fmt.Println("Auth code received! Exchanging for token...")

		token, err = clkup.GetAccessToken(myOAuthCode)
		if err != nil {
			log.Fatalf("Failed to get token: %v", err)
		}

		err = db.SaveToken(token)
		if err != nil {
			log.Printf("Warning: Failed to save token to database: %v\n", err)
		} else {
			fmt.Println("Token securely saved to local database.")
		}
	} else {
		fmt.Println("Found existing ClickUp token. Skipping browser authentication.")
	}

	apiClient := &clkup.APIClient{
		Client:  &http.Client{Timeout: 30 * time.Second},
		Token:   token,
		Limiter: rate.NewLimiter(rate.Inf, 1),
	}

	p := tea.NewProgram(tui.InitialModel(apiClient, db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

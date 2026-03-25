package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"super-duper-fortnight/clkup"
	"super-duper-fortnight/oauth"
	"super-duper-fortnight/server"
	"super-duper-fortnight/tui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("Waiting for ClickUp authentication in your browser...")

	authCodeChan := make(chan string)
	go server.ServeGin(authCodeChan)
	oauth.Authenticate()

	myOAuthCode := <-authCodeChan
	fmt.Println("Auth code received! Exchanging for token...")

	token, err := clkup.GetAccessToken(myOAuthCode)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	apiClient := &clkup.APIClient{
		Client:  &http.Client{Timeout: 30 * time.Second},
		Token:   token,
		Limiter: rate.NewLimiter(rate.Inf, 1),
	}

	p := tea.NewProgram(tui.InitialModel(apiClient), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

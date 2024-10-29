package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"os"
	"github.com/joho/godotenv"
)

var (
    clientID     string
    clientSecret string
    scopes       string
    redirectURI  string
)
var tokenStore = struct {
    AccessToken  string
    RefreshToken string
    Expiry       time.Time
}{}

func init() {
    err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }

    // Initialize variables with values from environment variables
    clientID = os.Getenv("CLIENT_ID")
    clientSecret = os.Getenv("CLIENT_SECRET")
    scopes = os.Getenv("SCOPES")       // Make sure HUBSPOT_SCOPES is in your .env file
    redirectURI = os.Getenv("REDIRECT_URL")

    // Check if the environment variables are actually loaded
    if clientID == "" || clientSecret == "" || redirectURI == "" {
        log.Fatalf("Environment variables not set correctly. Make sure .env is loaded and contains the necessary keys.")
    }
}

func main() {
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/oauth", oauthHandler)
    http.HandleFunc("/oauth-callback", oauthCallbackHandler)
    http.HandleFunc("/contacts", contactsHandler)

    fmt.Println("Server is running on http://localhost:3000")
    log.Fatal(http.ListenAndServe(":3000", nil))
}

// Display login link if not authenticated
func homeHandler(w http.ResponseWriter, r *http.Request) {
    if tokenStore.AccessToken == "" {
        fmt.Fprint(w, `<a href="/oauth">Login with HubSpot</a>`)
        return
    }
    http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

// Start OAuth flow
func oauthHandler(w http.ResponseWriter, r *http.Request) {
    authURL := fmt.Sprintf(
        "https://app.hubspot.com/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s",
        clientID, redirectURI, scopes,
    )
    http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Handle OAuth callback and get access token
func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "Code not found", http.StatusBadRequest)
        return
    }

    // Exchange code for tokens
    data := fmt.Sprintf(
        "grant_type=authorization_code&client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
        clientID, clientSecret, redirectURI, code,
    )
    req, err := http.NewRequest("POST", "https://api.hubapi.com/oauth/v1/token", strings.NewReader(data))
    if err != nil {
        log.Fatal(err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    var respData struct {
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresIn    int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
        log.Fatal(err)
    }

    // Store tokens and expiration time
    tokenStore.AccessToken = respData.AccessToken
    tokenStore.RefreshToken = respData.RefreshToken
    tokenStore.Expiry = time.Now().Add(time.Duration(respData.ExpiresIn) * time.Second)

    http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

// Retrieve contacts from HubSpot
func contactsHandler(w http.ResponseWriter, r *http.Request) {
    if time.Now().After(tokenStore.Expiry) {
        refreshAccessToken()
    }

    req, err := http.NewRequest("GET", "https://api.hubapi.com/crm/v3/objects/contacts?limit=10", nil)
    if err != nil {
        log.Fatal(err)
    }
    req.Header.Set("Authorization", "Bearer "+tokenStore.AccessToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    var contactsResp struct {
        Results []struct {
            ID         string `json:"id"`
            Properties struct {
                FirstName string `json:"firstname"`
                LastName  string `json:"lastname"`
                Company   string `json:"company"`
            } `json:"properties"`
        } `json:"results"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&contactsResp); err != nil {
        log.Fatal(err)
    }

    // Render contacts
    for _, contact := range contactsResp.Results {
        fmt.Fprintf(w, "ID: %s, Name: %s %s, Company: %s\n", contact.ID, contact.Properties.FirstName, contact.Properties.LastName, contact.Properties.Company)
    }
}

// Refresh access token using the refresh token
func refreshAccessToken() {
    data := fmt.Sprintf(
        "grant_type=refresh_token&client_id=%s&client_secret=%s&refresh_token=%s",
        clientID, clientSecret, tokenStore.RefreshToken,
    )
    req, err := http.NewRequest("POST", "https://api.hubapi.com/oauth/v1/token", strings.NewReader(data))
    if err != nil {
        log.Fatal(err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    var respData struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
        log.Fatal(err)
    }

    tokenStore.AccessToken = respData.AccessToken
    tokenStore.Expiry = time.Now().Add(time.Duration(respData.ExpiresIn) * time.Second)
}
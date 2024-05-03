package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type User struct {
	Name         string
	Email        string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

type GoogleClient struct {
	Port string

	// In memory session storage. In a production application, you will want to store this in a database.
	UserInfo *User
}

func NewGoogleClient(port string) *GoogleClient {
	return &GoogleClient{
		Port: port,
	}
}

func (g *GoogleClient) setupConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("http://localhost:%s/oauth2/callback", g.Port), // set redirect url in google cloud console
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func (g *GoogleClient) Index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./internal/index.html")
	if err != nil {
		log.Println("failed to parse template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Provider string
	}{
		Provider: "Google",
	}
	t.Execute(w, data)
}

func (g *GoogleClient) Login(w http.ResponseWriter, r *http.Request) {
	googleConfig := g.setupConfig()
	url := googleConfig.AuthCodeURL("authstate", oauth2.AccessTypeOffline)

	// redirect to google login page
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (g *GoogleClient) Callback(w http.ResponseWriter, r *http.Request) {
	// state
	state := r.URL.Query()["state"][0]
	if state != "authstate" {
		fmt.Fprintln(w, "states don't match")
		return
	}

	// user denied the auth
	if err, exists := r.URL.Query()["error"]; exists {
		fmt.Fprintln(w, err[0]) // access_denied
		return
	}

	// code
	code := r.URL.Query()["code"][0]

	// config
	googleConfig := g.setupConfig()

	// exchange code for token
	token, err := googleConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Fprintln(w, "code - token exchange failed")
	}

	user := User{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	// get user info from google api
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		fmt.Fprintln(w, "user data fetch failed")
	}

	// decode response
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		fmt.Fprintln(w, "failed to decode response body")
	}

	defer resp.Body.Close()

	g.UserInfo = &user

	log.Printf("%+v", g.UserInfo)

	fmt.Fprintln(w, "Login successful:", g.UserInfo.Name, " - ", g.UserInfo.Email)

	// redirect user
	// http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}

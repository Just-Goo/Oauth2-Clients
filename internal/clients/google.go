package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type user struct {
	Name  string
	Email string
}

type GoogleClient struct {
	Port string

	// In memory session storage. In a production application, you will want to store this in a database.
	AccessToken  string
	RefreshToken string
	UserInfo     *user
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
	url := googleConfig.AuthCodeURL("authstate")

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

	// get user info from google api
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		fmt.Fprintln(w, "user data fetch failed")
	}

	// parse response
	userInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(w, "failed to parse response body")
	}

	defer resp.Body.Close()

	user := user{}

	if err := json.Unmarshal(userInfo, &user); err != nil {
		fmt.Fprintln(w, "failed to unmarshal data")
	}

	g.UserInfo = &user

	log.Println(g.UserInfo)

	fmt.Fprintln(w, "Login successful:", g.UserInfo.Name, " - ", g.UserInfo.Email)
}

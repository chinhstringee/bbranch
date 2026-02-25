package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	authorizeURL = "https://bitbucket.org/site/oauth2/authorize"
	tokenURL     = "https://bitbucket.org/site/oauth2/access_token"
	callbackPort = "9876"
	callbackPath = "/callback"
	redirectURI  = "http://localhost:" + callbackPort + callbackPath
)

// Token represents stored OAuth tokens.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// tokenFilePath returns ~/.bbranch/token.json
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, ".bbranch", "token.json"), nil
}

// Login performs OAuth 2.0 Authorization Code + PKCE flow.
func Login(clientID, clientSecret string) error {
	// Generate PKCE code verifier (43-128 chars, URL-safe)
	verifierBytes := make([]byte, 64)
	if _, err := rand.Read(verifierBytes); err != nil {
		return fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// SHA256 hash for code challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Build authorize URL
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	authURL := authorizeURL + "?" + params.Encode()

	// Channel to receive auth code from callback
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p></body></html>", html.EscapeString(errMsg))
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2><p>You can close this tab.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{
		Addr:              ":" + callbackPort,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server failed: %w", err)
		}
	}()

	// Open browser
	fmt.Println("Opening browser for Bitbucket authorization...")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Please open this URL manually:\n%s\n", authURL)
	}

	// Wait for callback
	var authCode string
	select {
	case authCode = <-codeCh:
	case err := <-errCh:
		server.Shutdown(context.Background())
		return err
	case <-time.After(5 * time.Minute):
		server.Shutdown(context.Background())
		return fmt.Errorf("authorization timed out (5 minutes)")
	}

	server.Shutdown(context.Background())

	// Exchange code for tokens
	token, err := exchangeCode(clientID, clientSecret, authCode, codeVerifier)
	if err != nil {
		return err
	}

	// Save token
	if err := saveToken(token); err != nil {
		return err
	}

	fmt.Println("Login successful! Token saved.")
	return nil
}

var tokenMu sync.Mutex

// GetToken loads the stored token, refreshing if expired. Safe for concurrent use.
func GetToken(clientID, clientSecret string) (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	token, err := loadToken()
	if err != nil {
		return "", fmt.Errorf("not logged in. Run 'bbranch login' first: %w", err)
	}

	// Refresh if expired (with 30s buffer)
	if time.Now().After(token.ExpiresAt.Add(-30 * time.Second)) {
		token, err = refreshToken(clientID, clientSecret, token.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("token refresh failed, run 'bbranch login' again: %w", err)
		}
		if err := saveToken(token); err != nil {
			return "", err
		}
	}

	return token.AccessToken, nil
}

// exchangeCode trades the authorization code for tokens.
func exchangeCode(clientID, clientSecret, code, codeVerifier string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	return doTokenRequest(req)
}

// refreshToken uses the refresh token to get a new access token.
func refreshToken(clientID, clientSecret, refresh string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	return doTokenRequest(req)
}

// doTokenRequest executes a token endpoint request and parses the response.
func doTokenRequest(req *http.Request) (*Token, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token exchange failed (%d): %s - %s", resp.StatusCode, errResp.Error, errResp.Description)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

func saveToken(token *Token) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func loadToken() (*Token, error) {
	path, err := tokenFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

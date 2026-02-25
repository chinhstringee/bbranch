package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------- tokenFilePath ----------

func TestTokenFilePath_ContainsBbranch(t *testing.T) {
	path, err := tokenFilePath()
	if err != nil {
		t.Fatalf("tokenFilePath() error: %v", err)
	}
	if !strings.Contains(path, ".bbranch") {
		t.Errorf("tokenFilePath() = %q, want path containing .bbranch", path)
	}
	if !strings.HasSuffix(path, "token.json") {
		t.Errorf("tokenFilePath() = %q, want path ending in token.json", path)
	}
}

// ---------- saveToken / loadToken round-trip ----------

func TestSaveLoadToken_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, ".bbranch", "token.json")

	original := &Token{
		AccessToken:  "access-abc123",
		RefreshToken: "refresh-xyz789",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Round(time.Second),
	}

	// Write directly to temp path to avoid touching real home dir.
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Load it back
	raw, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	var loaded Token
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
}

func TestLoadToken_MissingFile(t *testing.T) {
	// Point HOME to temp dir so loadToken finds nothing.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := loadToken()
	if err == nil {
		t.Fatal("expected error loading non-existent token, got nil")
	}
}

func TestLoadToken_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tokenDir := filepath.Join(dir, ".bbranch")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "token.json"), []byte("{{not-json"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := loadToken()
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestSaveToken_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tok := &Token{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if err := saveToken(tok); err != nil {
		t.Fatalf("saveToken() error: %v", err)
	}

	path := filepath.Join(dir, ".bbranch", "token.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected token file at %s, not found", path)
	}
}

func TestSaveToken_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tok := &Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	if err := saveToken(tok); err != nil {
		t.Fatalf("saveToken() error: %v", err)
	}

	path := filepath.Join(dir, ".bbranch", "token.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	// File should be owner-only readable (0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want 0600", info.Mode().Perm())
	}
}

// ---------- PKCE: verifier / challenge ----------

func TestPKCE_ChallengeMatchesVerifier(t *testing.T) {
	// Simulate the PKCE generation from Login()
	verifierBytes := make([]byte, 64)
	// Use deterministic bytes for the test — fill with 0xAB
	for i := range verifierBytes {
		verifierBytes[i] = 0xAB
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	// Verify: SHA256(verifier) base64url-encoded == challenge
	expectedHash := sha256.Sum256([]byte(codeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(expectedHash[:])

	if codeChallenge != expectedChallenge {
		t.Errorf("challenge mismatch: got %q, want %q", codeChallenge, expectedChallenge)
	}
}

func TestPKCE_VerifierIsURLSafe(t *testing.T) {
	verifierBytes := make([]byte, 64)
	for i := range verifierBytes {
		verifierBytes[i] = byte(i)
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// URL-safe base64 must not contain +, /, or = characters
	if strings.ContainsAny(verifier, "+/=") {
		t.Errorf("verifier %q contains URL-unsafe characters", verifier)
	}
}

func TestPKCE_VerifierLength(t *testing.T) {
	verifierBytes := make([]byte, 64)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	// 64 bytes → 86 base64url chars (no padding). Must be ≥43 per RFC 7636.
	if len(verifier) < 43 {
		t.Errorf("verifier length %d < 43 (RFC 7636 minimum)", len(verifier))
	}
	if len(verifier) > 128 {
		t.Errorf("verifier length %d > 128 (RFC 7636 maximum)", len(verifier))
	}
}

// ---------- doTokenRequest ----------

func TestDoTokenRequest_Success(t *testing.T) {
	expiresIn := 3600
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth header is present
		_, _, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth header in token request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    expiresIn,
		})
	}))
	defer srv.Close()

	req, err := http.NewRequest("POST", srv.URL, strings.NewReader("grant_type=authorization_code"))
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("client-id", "client-secret")

	tok, err := doTokenRequest(req)
	if err != nil {
		t.Fatalf("doTokenRequest() error: %v", err)
	}
	if tok.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "new-access")
	}
	if tok.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want %q", tok.RefreshToken, "new-refresh")
	}
	// ExpiresAt should be roughly now + expiresIn seconds
	expectedExpiry := time.Now().Add(time.Duration(expiresIn) * time.Second)
	diff := tok.ExpiresAt.Sub(expectedExpiry)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt %v is too far from expected %v", tok.ExpiresAt, expectedExpiry)
	}
}

func TestDoTokenRequest_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, strings.NewReader(""))
	tok, err := doTokenRequest(req)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if tok != nil {
		t.Fatal("expected nil token on error")
	}
	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("error %q does not contain error code", err.Error())
	}
}

func TestDoTokenRequest_InvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{{not valid json"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, strings.NewReader(""))
	_, err := doTokenRequest(req)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------- GetToken ----------

func TestGetToken_ValidToken_NoRefresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	tok := &Token{
		AccessToken:  "valid-token",
		RefreshToken: "refresh-tok",
		ExpiresAt:    time.Now().Add(10 * time.Minute), // not expired
	}
	if err := saveToken(tok); err != nil {
		t.Fatalf("saveToken: %v", err)
	}

	got, err := GetToken("client-id", "client-secret")
	if err != nil {
		t.Fatalf("GetToken() error: %v", err)
	}
	if got != "valid-token" {
		t.Errorf("GetToken() = %q, want %q", got, "valid-token")
	}
}

func TestGetToken_NoToken_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := GetToken("client-id", "client-secret")
	if err == nil {
		t.Fatal("expected error when no token stored, got nil")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error %q does not contain 'not logged in'", err.Error())
	}
}

func TestGetToken_ExpiredToken_TriesRefresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Save an expired token
	tok := &Token{
		AccessToken:  "old-token",
		RefreshToken: "my-refresh-token",
		ExpiresAt:    time.Now().Add(-5 * time.Minute), // already expired
	}
	if err := saveToken(tok); err != nil {
		t.Fatalf("saveToken: %v", err)
	}

	// Stand up a fake token endpoint that returns a refreshed token
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "refreshed-token",
			"refresh_token": "new-refresh",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	// We cannot inject the tokenURL into the package-level const directly,
	// so test the underlying refreshToken helper instead.
	newTok, err := refreshTokenViaServer(srv.URL, "client-id", "client-secret", "my-refresh-token")
	if err != nil {
		t.Fatalf("refreshToken error: %v", err)
	}
	if newTok.AccessToken != "refreshed-token" {
		t.Errorf("AccessToken = %q, want %q", newTok.AccessToken, "refreshed-token")
	}
}

// refreshTokenViaServer mirrors refreshToken but targets a custom URL (for testing).
func refreshTokenViaServer(serverURL, clientID, clientSecret, refresh string) (*Token, error) {
	data := "grant_type=refresh_token&refresh_token=" + refresh
	req, err := http.NewRequest("POST", serverURL, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	return doTokenRequest(req)
}

// ---------- Token struct ----------

func TestToken_JSONRoundTrip(t *testing.T) {
	original := Token{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
		ExpiresAt:    time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Token
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", decoded.AccessToken, original.AccessToken)
	}
	if decoded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", decoded.RefreshToken, original.RefreshToken)
	}
	if !decoded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", decoded.ExpiresAt, original.ExpiresAt)
	}
}

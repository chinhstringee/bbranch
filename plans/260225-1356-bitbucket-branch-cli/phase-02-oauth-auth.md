# Phase 2: OAuth 2.0 Authentication

**Priority:** High | **Status:** Pending

## Overview
OAuth 2.0 Authorization Code + PKCE flow for Bitbucket Cloud. Opens browser for user consent, receives callback on localhost, stores tokens locally.

## Bitbucket OAuth Setup (prerequisite)
User creates an OAuth consumer in Bitbucket workspace settings:
- Callback URL: `http://localhost:9876/callback`
- Permissions: Repositories (Read + Write)
- Gets `client_id` and `client_secret`

## OAuth Flow
1. User runs `bbranch login`
2. CLI generates PKCE code_verifier + code_challenge
3. Opens browser to Bitbucket authorize URL
4. User grants access in browser
5. Bitbucket redirects to `http://localhost:9876/callback?code=...`
6. CLI exchanges code for access_token + refresh_token
7. Tokens saved to `~/.bbranch/token.json`

## Implementation Steps

1. **Create `cmd/login.go`** — Cobra command:
   - Load OAuth client_id/secret from config
   - Call `auth.Login()` to start OAuth flow
   - Print success message

2. **Create `internal/auth/oauth.go`**:
   - `TokenStore` struct — path to token file (`~/.bbranch/token.json`)
   - `Login(clientID, clientSecret string) error`:
     - Generate PKCE verifier (random 43-128 chars) + SHA256 challenge
     - Build authorize URL: `https://bitbucket.org/site/oauth2/authorize`
       - `response_type=code`
       - `client_id=...`
       - `redirect_uri=http://localhost:9876/callback`
       - `code_challenge=...` + `code_challenge_method=S256`
     - Start local HTTP server on `:9876`
     - Open browser (`open` on macOS)
     - Wait for callback, extract `code` param
     - POST to `https://bitbucket.org/site/oauth2/access_token`:
       - `grant_type=authorization_code`
       - `code=...`
       - `code_verifier=...`
       - `redirect_uri=...`
       - Basic Auth header with client_id:client_secret
     - Parse response: `access_token`, `refresh_token`, `expires_in`
     - Save to `~/.bbranch/token.json`
   - `GetToken() (string, error)`:
     - Load token from file
     - If expired, refresh using `refresh_token`
     - Return valid access_token
   - `RefreshToken(clientID, clientSecret, refreshToken string) (*Token, error)`

3. **Token file format** (`~/.bbranch/token.json`):
   ```json
   {
     "access_token": "...",
     "refresh_token": "...",
     "expires_at": "2026-02-25T15:00:00Z"
   }
   ```

## Bitbucket OAuth Endpoints
- Authorize: `https://bitbucket.org/site/oauth2/authorize`
- Token: `https://bitbucket.org/site/oauth2/access_token`

## Files to Create
- `cmd/login.go`
- `internal/auth/oauth.go`

## Success Criteria
- [ ] `bbranch login` opens browser for authorization
- [ ] Callback captured, tokens exchanged and stored
- [ ] Auto-refresh when token expires
- [ ] Clear error if OAuth consumer not configured

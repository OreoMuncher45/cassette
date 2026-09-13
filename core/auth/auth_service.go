package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"cassette/core/logger"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type AuthService struct {
	sptAuth    *spotifyauth.Authenticator
	tknChannel chan *oauth2.Token
	authConfig *AuthConfig
}

func NewAuthService(redirectURI string) *AuthService {
	authConfig := NewAuthConfig()
	sptAuth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadCurrentlyPlaying,
			spotifyauth.ScopeStreaming,
			"app-remote-control",
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopeUserFollowRead,
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopeUserReadPrivate,
		),
		spotifyauth.WithClientID(authConfig.clientID),
	)
	return &AuthService{
		sptAuth:    sptAuth,
		tknChannel: make(chan *oauth2.Token, 1),
		authConfig: authConfig,
	}
}

func (a *AuthService) GetAuthURL() string {
	return a.sptAuth.AuthURL(a.authConfig.state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", a.authConfig.codeChallenge),
		oauth2.SetAuthURLParam("client_id", a.authConfig.clientID),
	)
}

func (a *AuthService) GetTokenChannel() chan *oauth2.Token {
	return a.tknChannel
}

func (a *AuthService) MakeOauthCallbackHandler() (func(w http.ResponseWriter, r *http.Request), chan error) {
	errCh := make(chan error, 1)
	callback := func(w http.ResponseWriter, r *http.Request) {
		// 1. Verify state parameter first
		if st := r.FormValue("state"); st != a.authConfig.state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch: %s != %s", st, a.authConfig.state)
			return
		}

		// 2. Intercept error from Spotify query params if user cancelled or auth failed
		if authErr := r.FormValue("error"); authErr != "" {
			errDesc := r.FormValue("error_description")
			errMsg := fmt.Sprintf("spotify authorization error: %s (%s)", authErr, errDesc)
			http.Error(w, errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("%s", errMsg)
			return
		}

		// 3. Exchange code for token with PKCE code_verifier and client_id
		tok, err := a.sptAuth.Token(r.Context(), a.authConfig.state, r,
			oauth2.SetAuthURLParam("code_verifier", a.authConfig.codeVerifier),
			oauth2.SetAuthURLParam("client_id", a.authConfig.clientID),
		)
		if err != nil {
			http.Error(w, "Couldn't exchange token: "+err.Error(), http.StatusForbidden)
			errCh <- err
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintln(w, `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Cassette</title></head>
<body style="background:#0d1117;color:#58a6ff;font-family:monospace;display:flex;flex-direction:column;align-items:center;justify-content:center;height:85vh;margin:0;">
  <h1 style="color:#2ea043;font-size:24px;margin-bottom:12px;">&#9658; Cassette Connected</h1>
  <p style="color:#c9d1d9;font-size:14px;margin:0 0 8px 0;">Authentication successful.</p>
  <p style="color:#8b949e;font-size:12px;">You can now close this tab and return to your terminal.</p>
</body>
</html>`)
		a.tknChannel <- tok
	}
	return callback, errCh
}

type persistingTokenSource struct {
	base        oauth2.TokenSource
	mu          sync.Mutex
	lastToken   *oauth2.Token
	onTokenSave func(*oauth2.Token) error
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastToken == nil || s.lastToken.AccessToken != tok.AccessToken {
		s.lastToken = tok
		if s.onTokenSave != nil {
			if saveErr := s.onTokenSave(tok); saveErr != nil {
				logger.Log.Warn().Err(saveErr).Msg("failed to persist refreshed token")
			}
		}
	}
	return tok, nil
}

func (a *AuthService) GetSpotifyClient(tkn *oauth2.Token, onTokenSave func(*oauth2.Token) error) *spotify.Client {
	ctx := context.Background()
	baseClient := a.sptAuth.Client(ctx, tkn)
	if onTokenSave != nil {
		persistingSource := &persistingTokenSource{
			base:        a.sptAuth.Client(ctx, tkn).Transport.(*oauth2.Transport).Source,
			lastToken:   tkn,
			onTokenSave: onTokenSave,
		}
		persistingClient := oauth2.NewClient(ctx, persistingSource)
		return spotify.New(persistingClient)
	}
	return spotify.New(baseClient)
}

package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/options"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/sessions"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/encryption"
	internaloidc "github.com/oauth2-proxy/oauth2-proxy/v7/pkg/providers/oidc"
	"github.com/stretchr/testify/assert"
)

type redeemTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token,omitempty"`
}

func newOIDCProvider(serverURL *url.URL, skipNonce bool) *OIDCProvider {
	verificationOptions := internaloidc.IDTokenVerificationOptions{
		AudienceClaims: []string{"aud"},
		ClientID:       "https://test.myapp.com",
	}
	providerData := &ProviderData{
		ProviderName: "oidc",
		ClientID:     oidcClientID,
		ClientSecret: oidcSecret,
		LoginURL: &url.URL{
			Scheme: serverURL.Scheme,
			Host:   serverURL.Host,
			Path:   "/login/oauth/authorize"},
		RedeemURL: &url.URL{
			Scheme: serverURL.Scheme,
			Host:   serverURL.Host,
			Path:   "/login/oauth/access_token"},
		ProfileURL: &url.URL{
			Scheme: serverURL.Scheme,
			Host:   serverURL.Host,
			Path:   "/profile"},
		IntrospectURL: &url.URL{
			Scheme: serverURL.Scheme,
			Host:   serverURL.Host,
			Path:   "/introspect"},
		ValidateURL: &url.URL{
			Scheme: serverURL.Scheme,
			Host:   serverURL.Host,
			Path:   "/api"},
		Scope:       "openid profile offline_access",
		EmailClaim:  "email",
		GroupsClaim: "groups",
		UserClaim:   "sub",
		Verifier: internaloidc.NewVerifier(oidc.NewVerifier(
			oidcIssuer,
			mockJWKS{},
			&oidc.Config{ClientID: oidcClientID},
		), verificationOptions),
	}

	p := NewOIDCProvider(providerData, options.OIDCOptions{
		InsecureSkipNonce: skipNonce,
	})

	return p
}

func newOIDCServer(redeemJSON []byte, profileJSON []byte, introspectJSON []byte) *httptest.Server {
	mux := http.NewServeMux()
	if len(redeemJSON) > 0 {
		mux.HandleFunc("/login/oauth/access_token", func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Add("content-type", "application/json")
			_, _ = rw.Write(redeemJSON)
		})
	}
	if len(profileJSON) > 0 {
		mux.HandleFunc("/profile", func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Add("content-type", "application/json")
			_, _ = rw.Write(profileJSON)
		})
	}
	if len(introspectJSON) > 0 {
		mux.HandleFunc("/introspect", func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Add("content-type", "application/json")
			_, _ = rw.Write(introspectJSON)
		})
	}
	testserver := httptest.NewServer(mux)
	return testserver
}

func newTestOIDCSetup(redeemJSON []byte, profileJSON []byte, introspectJSON []byte) (*httptest.Server, *OIDCProvider) {
	server := newOIDCServer(redeemJSON, profileJSON, introspectJSON)
	serverURL, _ := url.Parse(server.URL)
	provider := newOIDCProvider(serverURL, false)
	return server, provider
}

func TestOIDCProviderGetLoginURL(t *testing.T) {
	serverURL := &url.URL{
		Scheme: "https",
		Host:   "oauth2proxy.oidctest",
	}
	provider := newOIDCProvider(serverURL, true)

	n, err := encryption.Nonce(32)
	assert.NoError(t, err)
	nonce := base64.RawURLEncoding.EncodeToString(n)

	// SkipNonce defaults to true
	skipNonce := provider.GetLoginURL("http://redirect/", "", nonce, url.Values{})
	assert.NotContains(t, skipNonce, "nonce")

	provider.SkipNonce = false
	withNonce := provider.GetLoginURL("http://redirect/", "", nonce, url.Values{})
	assert.Contains(t, withNonce, fmt.Sprintf("nonce=%s", nonce))
	assert.NotContains(t, withNonce, "code_challenge")
	assert.NotContains(t, withNonce, "code_challenge_method")
}

func TestOIDCProviderRedeem(t *testing.T) {
	idToken, _ := newSignedTestIDToken(defaultIDToken)
	body, _ := json.Marshal(redeemTokenResponse{
		AccessToken:  accessToken,
		ExpiresIn:    10,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		IDToken:      idToken,
	})
	server, provider := newTestOIDCSetup(body, []byte(`{}`), []byte(`{}`))
	defer server.Close()

	session, err := provider.Redeem(context.Background(), provider.RedeemURL.String(), "code1234", "")
	assert.Equal(t, nil, err)
	assert.Equal(t, defaultIDToken.Email, session.Email)
	assert.Equal(t, accessToken, session.AccessToken)
	assert.Equal(t, idToken, session.IDToken)
	assert.Equal(t, refreshToken, session.RefreshToken)
	assert.Equal(t, "123456789", session.User)
}

func TestOIDCProviderRedeem_custom_userid(t *testing.T) {
	idToken, _ := newSignedTestIDToken(defaultIDToken)
	body, _ := json.Marshal(redeemTokenResponse{
		AccessToken:  accessToken,
		ExpiresIn:    10,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		IDToken:      idToken,
	})

	server, provider := newTestOIDCSetup(body, []byte(`{}`), []byte(`{}`))
	provider.EmailClaim = "phone_number"
	defer server.Close()

	session, err := provider.Redeem(context.Background(), provider.RedeemURL.String(), "code1234", "")
	assert.Equal(t, nil, err)
	assert.Equal(t, defaultIDToken.Phone, session.Email)
}

func TestOIDCProviderRefreshSessionIfNeededWithoutIdToken(t *testing.T) {

	idToken, _ := newSignedTestIDToken(defaultIDToken)
	body, _ := json.Marshal(redeemTokenResponse{
		AccessToken:  accessToken,
		ExpiresIn:    10,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
	})

	server, provider := newTestOIDCSetup(body, []byte(`{}`), []byte(`{}`))
	defer server.Close()

	existingSession := &sessions.SessionState{
		AccessToken:  "changeit",
		IDToken:      idToken,
		CreatedAt:    nil,
		ExpiresOn:    nil,
		RefreshToken: refreshToken,
		Email:        "janedoe@example.com",
		User:         "11223344",
	}

	refreshed, err := provider.RefreshSession(context.Background(), existingSession)
	assert.Equal(t, nil, err)
	assert.Equal(t, refreshed, true)
	assert.Equal(t, "janedoe@example.com", existingSession.Email)
	assert.Equal(t, accessToken, existingSession.AccessToken)
	assert.Equal(t, idToken, existingSession.IDToken)
	assert.Equal(t, refreshToken, existingSession.RefreshToken)
	assert.Equal(t, "11223344", existingSession.User)
}

func TestOIDCProviderRefreshSessionIfNeededWithIdToken(t *testing.T) {

	idToken, _ := newSignedTestIDToken(defaultIDToken)
	body, _ := json.Marshal(redeemTokenResponse{
		AccessToken:  accessToken,
		ExpiresIn:    10,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		IDToken:      idToken,
	})

	server, provider := newTestOIDCSetup(body, []byte(`{}`), []byte(`{}`))
	defer server.Close()

	existingSession := &sessions.SessionState{
		AccessToken:  "changeit",
		IDToken:      "changeit",
		CreatedAt:    nil,
		ExpiresOn:    nil,
		RefreshToken: refreshToken,
		Email:        "changeit",
		User:         "changeit",
	}
	refreshed, err := provider.RefreshSession(context.Background(), existingSession)
	assert.Equal(t, nil, err)
	assert.Equal(t, refreshed, true)
	assert.Equal(t, defaultIDToken.Email, existingSession.Email)
	assert.Equal(t, defaultIDToken.Subject, existingSession.User)
	assert.Equal(t, accessToken, existingSession.AccessToken)
	assert.Equal(t, idToken, existingSession.IDToken)
	assert.Equal(t, refreshToken, existingSession.RefreshToken)
}

func TestOIDCProviderCreateSessionFromToken(t *testing.T) {
	testCases := map[string]struct {
		IDToken        idTokenClaims
		GroupsClaim    string
		ExpectedUser   string
		ExpectedEmail  string
		ExpectedGroups []string
	}{
		"Default IDToken": {
			IDToken:        defaultIDToken,
			GroupsClaim:    "groups",
			ExpectedUser:   "123456789",
			ExpectedEmail:  "janed@me.com",
			ExpectedGroups: []string{"test:a", "test:b"},
		},
		"Minimal IDToken with no email claim": {
			IDToken:        minimalIDToken,
			GroupsClaim:    "groups",
			ExpectedUser:   "123456789",
			ExpectedEmail:  "123456789",
			ExpectedGroups: nil,
		},
		"Custom Groups Claim": {
			IDToken:        defaultIDToken,
			GroupsClaim:    "roles",
			ExpectedUser:   "123456789",
			ExpectedEmail:  "janed@me.com",
			ExpectedGroups: []string{"test:c", "test:d"},
		},
		"Complex Groups Claim": {
			IDToken:       complexGroupsIDToken,
			GroupsClaim:   "groups",
			ExpectedUser:  "123456789",
			ExpectedEmail: "complex@claims.com",
			ExpectedGroups: []string{
				"{\"groupId\":\"Admin Group Id\",\"roles\":[\"Admin\"]}",
				"12345",
				"Just::A::String",
			},
		},
	}
	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			server, provider := newTestOIDCSetup([]byte(`{}`), []byte(`{}`), []byte(`{}`))
			provider.GroupsClaim = tc.GroupsClaim
			defer server.Close()

			rawIDToken, err := newSignedTestIDToken(tc.IDToken)
			assert.NoError(t, err)

			ss, err := provider.CreateSessionFromToken(context.Background(), rawIDToken)
			assert.NoError(t, err)

			assert.Equal(t, tc.ExpectedUser, ss.User)
			assert.Equal(t, tc.ExpectedEmail, ss.Email)
			assert.Equal(t, tc.ExpectedGroups, ss.Groups)
			assert.Equal(t, rawIDToken, ss.IDToken)
			assert.Equal(t, rawIDToken, ss.AccessToken)
			assert.Equal(t, "", ss.RefreshToken)
		})
	}
}

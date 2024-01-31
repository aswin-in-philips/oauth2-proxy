package middleware

import (
	"fmt"
	"net/http"

	"github.com/justinas/alice"
	middlewareapi "github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/middleware"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/logger"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/requests"
)

type CookieRefreshOptions struct {
	IssuerURL string
}

func NewCookieRefresh(opts *CookieRefreshOptions) alice.Constructor {
	cr := &cookieRefresh{
		HttpClient: &http.Client{},
		IssuerURL:  opts.IssuerURL,
	}
	return cr.refreshCookie
}

type cookieRefresh struct {
	HttpClient *http.Client
	IssuerURL  string
}

func (cr *cookieRefresh) refreshCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		scope := middlewareapi.GetRequestScope(req)
		if scope.Session == nil || !scope.Session.SessionJustRefreshed {
			next.ServeHTTP(rw, req)
			return
		}

		cookie, err := req.Cookie("hsdpamcookie")
		if err != nil {
			logger.Errorf("SSO Cookie Refresher - Could find 'hsdpamcookie' cookie in the request: %v", err)
			return
		}
		resp := requests.New(fmt.Sprintf("%s/session/refresh", cr.IssuerURL)).
			WithContext(req.Context()).
			WithMethod("GET").
			SetHeader("api-version", "1").
			SetHeader("Cookie", fmt.Sprintf("hsdpamcookie=%s", cookie.Value)).
			Do()

		if resp.StatusCode() != http.StatusNoContent {
			bodyString := string(resp.Body())
			logger.Errorf("SSO Cookie Refresher - Could not refresh the 'hsdpamcookie' cookie due to status and content: %v - %v", resp.StatusCode(), bodyString)
			return
		} else {
			logger.Print("SSO Cookie Refresher - Cookie 'hsdpamcookie' refreshed")
		}

		next.ServeHTTP(rw, req)
	})
}

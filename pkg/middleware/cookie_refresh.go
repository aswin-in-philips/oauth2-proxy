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
	IssuerURL         string
	CookieRefreshName string
}

func NewCookieRefresh(opts *CookieRefreshOptions) alice.Constructor {
	cr := &cookieRefresh{
		HTTPClient:        &http.Client{},
		IssuerURL:         opts.IssuerURL,
		CookieRefreshName: opts.CookieRefreshName,
	}
	return cr.refreshCookie
}

type cookieRefresh struct {
	HTTPClient        *http.Client
	IssuerURL         string
	CookieRefreshName string
}

func (cr *cookieRefresh) refreshCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		scope := middlewareapi.GetRequestScope(req)
		if scope.Session == nil || !scope.Session.SessionJustRefreshed {
			next.ServeHTTP(rw, req)
			return
		}

		cookie, err := req.Cookie(cr.CookieRefreshName)
		if err != nil {
			logger.Errorf("SSO Cookie Refresher - Could find '%s' cookie in the request: %v", cr.CookieRefreshName, err)
			return
		}
		resp := requests.New(fmt.Sprintf("%s/session/refresh", cr.IssuerURL)).
			WithContext(req.Context()).
			WithMethod("GET").
			SetHeader("api-version", "1").
			SetHeader("Cookie", fmt.Sprintf("%s=%s", cr.CookieRefreshName, cookie.Value)).
			Do()

		if resp.StatusCode() != http.StatusNoContent {
			bodyString := string(resp.Body())
			logger.Errorf("SSO Cookie Refresher - Could not refresh the '%s' cookie due to status and content: %v - %v", cr.CookieRefreshName, resp.StatusCode(), bodyString)
			return
		}

		logger.Printf("SSO Cookie Refresher - Cookie '%s' refreshed", cr.CookieRefreshName)
		next.ServeHTTP(rw, req)
	})
}

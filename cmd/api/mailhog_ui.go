package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// mountMailHogUI reverse-proxies the sidecar MailHog process (localhost:8025)
// so OTP messages are readable in the browser. MailHog's UI is served under
// /mailhog; its JSON API stays on /api/v2 and does not collide with /api/v1.
func mountMailHogUI(router *gin.Engine) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if os.Getenv("MAILHOG_UI") == "0" {
		return
	}
	if os.Getenv("MAILHOG_UI") != "1" && host != "127.0.0.1" && host != "localhost" {
		return
	}

	target, err := url.Parse("http://127.0.0.1:8025")
	if err != nil {
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	handler := func(c *gin.Context) {
		c.Request.Host = target.Host
		proxy.ServeHTTP(c.Writer, c.Request)
	}

	router.GET("/mailhog", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/mailhog/")
	})
	router.Any("/mailhog/*any", handler)
	router.Any("/api/v2", handler)
	router.Any("/api/v2/*any", handler)
}

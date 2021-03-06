package engine

import (
	"net/http"
	"runtime/debug"

	log "github.com/funkygao/log4go"
)

func recoverWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				var reason = "Unknown reason"
				switch e := err.(type) {
				case string:
					reason = e
				case error:
					reason = e.Error()
				}
				log.Critical("[%s] %v\n%s", r.RequestURI, err, string(debug.Stack()))
				http.Error(w, reason, http.StatusInternalServerError)
			}
		}()

		h.ServeHTTP(w, r)
	})
}

func accessLogWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO
		h.ServeHTTP(w, r)
	})
}

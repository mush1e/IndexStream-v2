package middleware

import (
	"log"
	"net/http"
	"time"
)

type recorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		log.Printf("%s | %s | %d | %dB | %v",
			r.Method,
			r.URL.RequestURI(),
			rec.status,
			rec.size,
			time.Since(start),
		)
	})
}

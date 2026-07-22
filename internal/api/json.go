package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// validEmail is a deliberately lax syntactic check: the goal is to reject
// obvious garbage, not to fully validate RFC 5322.
func validEmail(e string) bool {
	if len(e) < 3 || len(e) > 254 {
		return false
	}
	if strings.ContainsAny(e, " \t\r\n") {
		return false
	}
	at := strings.IndexByte(e, '@')
	if at <= 0 || at == len(e)-1 {
		return false
	}
	return strings.Contains(e[at+1:], ".")
}

// validDateYYYYMMDD reports whether s is a calendar date in YYYY-MM-DD form.
func validDateYYYYMMDD(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

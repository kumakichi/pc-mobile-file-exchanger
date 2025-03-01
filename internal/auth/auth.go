package auth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// BanIP represents a banned IP address with a count and timestamp
type BanIP struct {
	Count int
	Ts    int
}

var (
	// AutoBan stores banned IP addresses
	AutoBan     map[string]BanIP
	autobanLock sync.Mutex
)

func init() {
	AutoBan = make(map[string]BanIP, 10)
}

func AskForAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Please Login"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, err := w.Write([]byte("Unauthorized.\n"))
	if err != nil {
		log.Println(err)
	}
}

// ProcAutoBan processes auto-banning of IP addresses
func ProcAutoBan(banTimeout, banCount int, r *http.Request) bool {
	autobanLock.Lock()
	defer autobanLock.Unlock()

	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		remoteAddr = remoteAddr[:idx]
	}

	v, ok := AutoBan[remoteAddr]
	if !ok {
		AutoBan[remoteAddr] = BanIP{
			Count: 1,
			Ts:    int(time.Now().Unix()),
		}
		return false
	}

	// IP exists in ban list
	now := int(time.Now().Unix())
	if now-v.Ts > banTimeout {
		// Ban timeout reset
		AutoBan[remoteAddr] = BanIP{
			Count: 1,
			Ts:    now,
		}
		return false
	}

	v.Count++
	v.Ts = now
	AutoBan[remoteAddr] = v

	if v.Count > banCount {
		log.Printf("BAN IP[%s], count:%d, ts:%d\n", remoteAddr, v.Count, v.Ts)
		return true
	}

	return false
}

func Middleware(next http.Handler, authStr string, noAuth bool, banTimeout, banCount int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noAuth {
			next.ServeHTTP(w, r)
			return
		}

		if ProcAutoBan(banTimeout, banCount, r) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, err := w.Write([]byte("Too Many Failed Auth Attempts.\n"))
			if err != nil {
				log.Println(err)
			}
			return
		}

		s := r.Header.Get("Authorization")
		if s == "" {
			AskForAuth(w)
			return
		}

		if !strings.HasPrefix(s, "Basic ") {
			AskForAuth(w)
			return
		}

		if s[6:] != authStr {
			AskForAuth(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CalcAuthStr(usr, pwd string) string {
	return base64.StdEncoding.EncodeToString([]byte(usr + ":" + pwd))
}

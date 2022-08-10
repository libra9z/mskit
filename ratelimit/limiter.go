package ratelimit

import (
	"github.com/libra9z/mskit/v4/rest"
	"go.uber.org/ratelimit"
	"net"
	"net/http"
	"sync"
	"time"
)

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  ratelimit.Limiter
	lastSeen time.Time
}

var once sync.Once

// Change the the map to hold values of the type visitor.
var visitors map[string]*visitor
var mu sync.Mutex

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	once.Do(func() {
		visitors = make(map[string]*visitor)
		go cleanupVisitors()
	})
}

func getVisitor(r int, ip string) ratelimit.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := ratelimit.New(r)
		// Include the current time when creating a new visitor.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func Limit(ra int) rest.MskitFunc {
	return func(r *rest.Mcontext, w http.ResponseWriter) error {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		limiter := getVisitor(ra, ip)
		if limiter != nil {
			limiter.Take()
		}
		return nil
	}
}

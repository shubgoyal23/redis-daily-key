// Command redis-daily-key sets one key, once per day, on every Redis instance
// listed in REDIS_URLS. Intended to be run from cron at 00:00.
//
// Env:
//
//	REDIS_URLS  comma-separated connection strings, e.g.
//	            "redis://localhost:6379/0,rediss://user:pass@host:6380"
//	REDIS_KEY   key to set            (default: "daily:flag")
//	REDIS_VAL   value to set          (default: today's date, YYYY-MM-DD)
//	REDIS_TTL   key TTL, Go duration  (default: "26h" — survives clock drift,
//	            auto-expires before the next-next run)
//
// Variables may also be supplied via a .env file in the working directory;
// values already present in the real environment take precedence.
//
// Exit code is non-zero if ANY connection fails, so cron/monitoring notices.
package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load .env if present. A missing file is fine; godotenv never
	// overrides variables already set in the environment.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not load .env: %v", err)
	}

	urls := splitURLs(os.Getenv("REDIS_URLS"))
	if len(urls) == 0 {
		log.Fatal("REDIS_URLS is empty: set a comma-separated list of redis:// URLs")
	}

	key := envOr("REDIS_KEY", "daily:flag")
	val := envOr("REDIS_VAL", time.Now().Format("2006-01-02"))

	ttl := 26 * time.Hour
	if raw := os.Getenv("REDIS_TTL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			log.Fatalf("invalid REDIS_TTL %q: %v", raw, err)
		}
		ttl = d
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	failures := 0
	for _, url := range urls {
		if err := setKey(ctx, url, key, val, ttl); err != nil {
			failures++
			log.Printf("FAIL %s: %v", redact(url), err)
			continue
		}
		log.Printf("OK   %s -> %s=%s (ttl %s)", redact(url), key, val, ttl)
	}

	if failures > 0 {
		log.Fatalf("%d/%d connections failed", failures, len(urls))
	}
	log.Printf("done: set %s on %d connection(s)", key, len(urls))
}

func setKey(ctx context.Context, url, key, val string, ttl time.Duration) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return err
	}
	c := redis.NewClient(opt)
	defer c.Close()
	return c.Set(ctx, key, val, ttl).Err()
}

func splitURLs(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// redact hides credentials before logging a connection string.
func redact(url string) string {
	if i := strings.Index(url, "@"); i != -1 {
		if j := strings.Index(url, "://"); j != -1 && j+3 < i {
			return url[:j+3] + "***@" + url[i+1:]
		}
	}
	return url
}

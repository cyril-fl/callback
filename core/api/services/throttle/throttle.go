// Package throttle makes an address wait longer after each wrong password in a row, at the login and on the OAuth
// consent page. The failures are kept in the base (login_failures): a restart forgets none, and /authorize faces
// the Internet.
package throttle

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/domain"
	"github.com/uptrace/bun"
)

// waits is how long to wait after 1, 2, 3... failures in a row: a password cannot be guessed by trying them all.
var waits = []time.Duration{0, 0, 0, 30 * time.Second, time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}

// forget is how long a key keeps its failures after the last one: its next try starts over.
const forget = 24 * time.Hour

const message = "too many wrong passwords, try again later"

// GlobalKey counts the wrong passwords of every address on the OAuth consent page, which faces the Internet: an
// attacker rotating addresses still waits. The login stays on the tailnet and does not count it.
const GlobalKey = "global consent"

// AddressKey is the key of the request's client: its IPv4 address, or the /64 of an IPv6 one, which one client
// usually holds whole.
func AddressKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		host = ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
	}
	return "address " + host
}

// Throttle counts the tries in a row of each key.
type Throttle struct {
	db  bun.IDB
	now func() time.Time
}

// Passwords counts the wrong passwords: the login and the OAuth consent page share the count of an address.
func Passwords(db bun.IDB) *Throttle {
	return &Throttle{db: db, now: time.Now}
}

// Try counts a try for key before it runs, and gives 0; or, while key waits, how long it has left, without
// counting. In one transaction: the base has one connection, tries sent at once pass one after the other. The
// caller resets on a good password, forgives on another error.
func (t *Throttle) Try(ctx context.Context, key string) (wait time.Duration, err error) {
	now := t.now().UTC()
	err = t.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		failure := domain.LoginFailure{Key: key}
		err := tx.NewSelect().Model(&failure).WherePK().Scan(ctx)
		switch {
		case errors.Is(err, sql.ErrNoRows) || err == nil && !failure.ExpiresAt.After(now):
			failure.Count = 0
		case err != nil:
			return err
		}
		if failure.Count > 0 {
			if left := waits[min(failure.Count, len(waits))-1] - now.Sub(failure.LastAt); left > 0 {
				wait = max(left, time.Second)
				return nil
			}
		}
		failure.Count++
		failure.LastAt, failure.ExpiresAt = now, now.Add(forget)
		_, err = tx.NewInsert().Model(&failure).On("CONFLICT (key) DO UPDATE").Set("count = EXCLUDED.count, last_at = EXCLUDED.last_at, expires_at = EXCLUDED.expires_at").Exec(ctx)
		return err
	})
	return wait, err
}

// Forgive takes back the try Try counted, when it did not count.
func (t *Throttle) Forgive(ctx context.Context, key string) {
	err := t.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*domain.LoginFailure)(nil)).Where("key = ? AND count <= 1", key).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewUpdate().Model((*domain.LoginFailure)(nil)).Set("count = count - 1").Where("key = ? AND count > 1", key).Exec(ctx)
		return err
	})
	if err != nil {
		log.Printf("throttle: forgive: %v", err)
	}
}

// Reset forgets the tries of key.
func (t *Throttle) Reset(ctx context.Context, key string) {
	if _, err := t.db.NewDelete().Model((*domain.LoginFailure)(nil)).Where("key = ?", key).Exec(ctx); err != nil {
		log.Printf("throttle: reset: %v", err)
	}
}

// Blocked answers 429 with Retry-After while key waits, else counts the try (Try).
func (t *Throttle) Blocked(w http.ResponseWriter, r *http.Request, key string) bool {
	wait, err := t.Try(r.Context(), key)
	if err != nil {
		respond.Error(w, r, err)
		return true
	}
	if wait == 0 {
		return false
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	respond.Fail(w, http.StatusTooManyRequests, message)
	return true
}

package auction

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_closeAuctionAfter(t *testing.T) {
	t.Run("when the deadline is in the future, should not fire before it", func(t *testing.T) {
		var fired atomic.Bool

		go closeAuctionAfter(time.Now().Add(time.Hour), func() {
			fired.Store(true)
		})

		assert.Never(
			t,
			fired.Load,
			100*time.Millisecond,
			5*time.Millisecond,
			"o leilão nao deveria fechar antes do prazo")
	})

	t.Run("when the deadline is in the future, should fire after it", func(t *testing.T) {
		var fired atomic.Bool

		go closeAuctionAfter(time.Now().Add(50*time.Millisecond), func() {
			fired.Store(true)
		})

		assert.Eventually(
			t,
			fired.Load,
			2*time.Second,
			5*time.Millisecond,
			"o leilão deveria fechar depois do prazo")
	})

	t.Run("when the deadline has already passed, should fire immediately", func(t *testing.T) {
		var fired atomic.Bool

		go closeAuctionAfter(time.Now().Add(-time.Hour), func() {
			fired.Store(true)
		})

		assert.Eventually(
			t,
			fired.Load,
			time.Second,
			time.Millisecond,
			"prazo vencido deveria fechar de imediato")
	})
}

func Test_getAuctionInterval(t *testing.T) {
	t.Run("when the value is a valid duration, should parse it", func(t *testing.T) {
		t.Setenv("AUCTION_INTERVAL", "20s")

		assert.Equal(t, 20*time.Second, getAuctionInterval())
	})

	t.Run("when the value combines units, should parse it", func(t *testing.T) {
		t.Setenv("AUCTION_INTERVAL", "1h30m")

		assert.Equal(t, 90*time.Minute, getAuctionInterval())
	})

	t.Run("when the value is invalid, should fall back to five minutes", func(t *testing.T) {
		t.Setenv("AUCTION_INTERVAL", "abc")

		assert.Equal(t, 5*time.Minute, getAuctionInterval())
	})

	t.Run("when the variable is unset, should fall back to five minutes", func(t *testing.T) {
		t.Setenv("AUCTION_INTERVAL", "")

		assert.Equal(t, 5*time.Minute, getAuctionInterval())
	})
}

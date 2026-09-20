package command

import (
	"hsbot/internal/core/domain"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConversationStore(t *testing.T) {
	store := newConversationStore(time.Minute)

	require.NotNil(t, store)
	assert.NotNil(t, store.convos)
	assert.Equal(t, time.Minute, store.duration)
	assert.Equal(t, 0, store.len())
}

func TestConversationStore_GetOrCreate(t *testing.T) {
	store := newConversationStore(time.Minute)

	first := store.getOrCreate(1)
	require.NotNil(t, first)
	assert.Empty(t, first.messages)
	assert.True(t, first.expiresAt.After(time.Now()))

	first.messages = []domain.Prompt{{Prompt: "hello", Author: domain.User}}

	second := store.getOrCreate(1)
	assert.Same(t, first, second)
	assert.Equal(t, "hello", second.messages[0].Prompt)

	other := store.getOrCreate(2)
	assert.NotSame(t, first, other)
	assert.Equal(t, 2, store.len())
}

func TestConversationStore_GetOrCreateRefreshesExpiry(t *testing.T) {
	store := newConversationStore(time.Minute)

	c := store.getOrCreate(1)
	c.expiresAt = time.Now().Add(time.Second)

	again := store.getOrCreate(1)
	assert.Same(t, c, again)
	assert.True(t, again.expiresAt.After(time.Now().Add(50*time.Second)))
}

func TestConversationStore_GetOrCreateReplacesExpired(t *testing.T) {
	store := newConversationStore(time.Minute)

	expired := store.getOrCreate(1)
	expired.messages = []domain.Prompt{{Prompt: "stale"}}
	expired.expiresAt = time.Now().Add(-time.Second)

	fresh := store.getOrCreate(1)
	assert.NotSame(t, expired, fresh)
	assert.Empty(t, fresh.messages)
	assert.True(t, fresh.expiresAt.After(time.Now()))
}

func TestConversationStore_Get(t *testing.T) {
	store := newConversationStore(time.Minute)

	_, ok := store.get(1)
	assert.False(t, ok)

	created := store.getOrCreate(1)
	got, ok := store.get(1)
	require.True(t, ok)
	assert.Same(t, created, got)
}

func TestConversationStore_GetPurgesExpired(t *testing.T) {
	store := newConversationStore(time.Minute)

	c := store.getOrCreate(1)
	c.expiresAt = time.Now().Add(-time.Nanosecond)

	_, ok := store.get(1)
	assert.False(t, ok)
	assert.Equal(t, 0, store.len())
}

func TestConversationStore_Delete(t *testing.T) {
	store := newConversationStore(time.Minute)

	_, ok := store.delete(1)
	assert.False(t, ok)

	created := store.getOrCreate(1)
	created.messages = []domain.Prompt{{Prompt: "bye"}}

	deleted, ok := store.delete(1)
	require.True(t, ok)
	assert.Same(t, created, deleted)
	assert.Equal(t, "bye", deleted.messages[0].Prompt)

	_, ok = store.get(1)
	assert.False(t, ok)
	assert.Equal(t, 0, store.len())
}

func TestConversationStore_LenPurgesExpired(t *testing.T) {
	store := newConversationStore(time.Minute)

	live := store.getOrCreate(1)
	expired := store.getOrCreate(2)
	expired.expiresAt = time.Now().Add(-time.Second)

	assert.Equal(t, 1, store.len())

	got, ok := store.get(1)
	require.True(t, ok)
	assert.Same(t, live, got)

	_, ok = store.get(2)
	assert.False(t, ok)
}

func TestConversationStore_ZeroDurationExpiresImmediately(t *testing.T) {
	store := newConversationStore(0)

	_ = store.getOrCreate(1)
	_, ok := store.get(1)
	assert.False(t, ok)
}

func TestConversationStore_Concurrent(t *testing.T) {
	store := newConversationStore(time.Minute)
	const workers = 32

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(id int64) {
			defer wg.Done()
			_ = store.getOrCreate(id)
			_, _ = store.get(id)
			_ = store.len()
			if id%2 == 0 {
				_, _ = store.delete(id)
			}
		}(int64(i))
	}
	wg.Wait()

	assert.Equal(t, workers/2, store.len())
}

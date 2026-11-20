package anystoreprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anyproto/any-store/v2/anyenc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const marker = "APAROK_MARKER"

func TestStorageKeyEncryption(t *testing.T) {
	dir := t.TempDir()

	SetStorageKey([]byte("test-password-for-encryption-key"))
	defer SetStorageKey(nil)

	p1, err := NewInPath(dir)
	require.NoError(t, err)

	db := p1.GetCommonDb()

	coll, err := db.Collection(context.Background(), "test")
	require.NoError(t, err)

	arena := &anyenc.Arena{}
	doc := arena.NewObject()
	doc.Set("id", arena.NewString("marker-id"))
	doc.Set("value", arena.NewString(marker))
	require.NoError(t, coll.Insert(context.Background(), doc))
	require.NoError(t, p1.Close(context.Background()))

	dbPath := filepath.Join(dir, "objectstore", "objects.db")
	require.FileExists(t, dbPath)
	data, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), marker)

	p2, err := NewInPath(dir)
	require.NoError(t, err)
	defer p2.Close(context.Background())
	db2 := p2.GetCommonDb()
	coll2, err := db2.Collection(context.Background(), "test")
	require.NoError(t, err)
	iter, err := coll2.Find(nil).Iter(context.Background())
	require.NoError(t, err)
	found := false
	for iter.Next() {
		d, err := iter.Doc()
		require.NoError(t, err)
		if d.Value().GetString("value") == marker {
			found = true
		}
	}
	require.NoError(t, iter.Close())
	require.True(t, found, "marker not found after reopening with the same key")
}

func TestStorageKeyWrongKeyFails(t *testing.T) {
	dir := t.TempDir()
	SetStorageKey([]byte("correct-password"))
	p1, err := NewInPath(dir)
	require.NoError(t, err)
	_ = p1.GetCommonDb()
	require.NoError(t, p1.Close(context.Background()))

	SetStorageKey([]byte("wrong-password"))
	defer SetStorageKey(nil)
	p2, err := NewInPath(dir)
	require.NoError(t, err)
	defer p2.Close(context.Background())
	_ = p2.GetCommonDb()
}

# Aparok On-Disk Artifacts

This document lists every artifact Aparok writes to disk in the core library.

## any-store v2 databases

- `objectstore/objects.db` — common system store.
- `objectstore/{spaceId}/objects.db` — per-space index store.
- `objectstore/{spaceId}/crdt.db` — per-space CRDT store.
- Associated `-wal`, `-shm`, `.db-journal`, and migration `.v1` files.

**Encrypted today:** yes, with the storage key via `anystore.EncryptionConfig`.

## any-store v1 legacy databases

Older SQLite databases created by upstream Anytype. Migrated to v2 on first open
and removed afterward. Plaintext during migration only; the resulting v2 files
are encrypted.

## File blocks

Stored under the configured file storage path. Currently unencrypted; to be
protected with per-block XChaCha20-Poly1305.

## Tantivy full-text index

Persisted on disk by the search layer. Currently unencrypted. Future options:
encrypting codec for the underlying files, or rebuild the index in memory after
decrypting source data.

## Thumbnails, caches, temp files

Not yet encrypted. Plan: keep them inside the encrypted data directory or apply
the same storage-key based encryption.

## Logs and debug reports

Not yet encrypted. Plan: keep them out of long-term storage or encrypt with the
storage key before writing.

## Storage location selection

The default vault root path can be set from the Android app via the gomobile-exported  function. When  is omitted from account creation/recovery requests, the core falls back to this default path. This keeps the vault data under user-controlled local storage.

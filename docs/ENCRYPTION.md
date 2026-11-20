# Aparok Encryption at Rest

## Overview

All Anytype object stores (any-store v2 databases) are encrypted at rest using a
32-byte storage key. The key is supplied by the Android application before any
database is opened and is never persisted by the core library.

## Storage key

- Generated once per Aparok profile on the Android device.
- Passed from the Android app to the Go core through the gomobile-exported
  `SetStorageKey([]byte)` function.
- Non-32-byte keys are hashed with SHA-256 to derive a 32-byte key.
- When the key is unset the databases are created unencrypted (used only
  during testing and migration tooling).

## Database encryption

`pkg/lib/datastore/anystoreprovider` configures every `anystore.Config` with:

```go
cfg.Encryption = anystore.EncryptionConfig{Passphrase: storageKey}
```

This enables page-level AES-256-GCM encryption of the SQLite-backed database
files. Opening a database with the wrong key fails immediately with an integrity
error.

## Verified behavior

See `pkg/lib/datastore/anystoreprovider/encryption_test.go`:

- `APAROK_MARKER` is written to a collection, the database file is closed, and
  a raw scan of the `.db` file confirms the marker is not present in plaintext.
- Re-opening with the same key successfully reads the marker.
- Re-opening with a different key fails.

## Remaining work

- File-block encryption (per-block XChaCha20-Poly1305) for locally stored file
  content.
- Android Keystore/Tink wrapping of the storage key (app side).
- Secure key zeroing and `Lock()` lifecycle.

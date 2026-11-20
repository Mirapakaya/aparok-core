# Aparok Autonomous Build Progress

> Branch: `aparok/offline-private` (`Mirapakaya/aparok-core`)  
> Last updated: 2026-11-20 (P2 complete)

## Phase Status

| Phase | Description | Status | Notes |
|-------|-------------|--------|-------|
| P1 | Forks & Codespace setup | Done | `aparok-core` and `aparok-sync` forks, any-sync replace, javapkg `io.aparok.mw` |
| P2 | Remove outbound network clients | Done | All outbound HTTP/dial call sites removed or runtime-blocked; verify-offline passes with empty pending list |
| P3 | Core CI workflows | In progress | Added `.github/workflows/aparok-build-android-lib.yml`; publishes AAR to GitHub Packages |
| P4 | App integration / AAR | Not started | Needs Android app settings.gradle + middlewareVersion update |
| P5 | Encryption at rest | Not started | |
| P6 | Local profile | Not started | `core/identity` partially retained |
| P7 | Storage location selection | Not started | |
| P8 | Backup/restore | Not started | |
| P9 | Size optimization | Not started | |
| P10 | Documentation | Not started | |
| P11 | Final audit | Not started | |

## P2 Detailed Status

### Completed no-ops / removals

- `core/configfetcher`
- `core/gallery`
- `core/block/bookmark`
- `core/publish`
- `core/pushnotification` (removed `pushclient/`, `subscription.go`, `topics.go`)
- `util/linkpreview`
- `util/unsplash`
- `util/builtinobjects` (external archive download path)
- `core/ai/llmclient`
- `core/payments` (removed `cache/` and `emailcollector/`)
- `core/files/fileuploader` (URL download path)
- `core/block/import` (removed `notion/` and `web/`)
- `core/block/importv2` (notion client no-op)
- `metrics/anymetry`
- `core/identity` (offline no-op: local profile retained, remote repo/naming calls removed)
- `core/api` (offline no-op: JSON API service and wrapper client stubbed, server package removed)
- `space/spacecore/localdiscovery` (self-connect probe no-opped; package is `!android`)

## Verification

```bash
go build -tags "nogrpcserver gomobile nowatchdog nosigar timetzdata rasterizesvg" ./clientlibrary/service
go vet -tags "nogrpcserver gomobile nowatchdog nosigar timetzdata rasterizesvg" ./clientlibrary/service
bash scripts/verify-offline.sh
```

All three pass.

## Recent commits

- `GO-0: offline no-op gallery, bookmark, publish, pushnotification, configfetcher`
- `GO-0: disable external archive downloads in builtinobjects and refresh verify-offline pending list`
- `GO-0: offline no-op llmclient for ai package`
- `GO-0: offline no-op payments service, remove cache and emailcollector subpackages`
- `GO-0: disable url uploads, remove import notion/web, no-op importv2 notion client, update progress`
- `GO-0: offline no-op identity service, api service/wrapper, localdiscovery; empty verify-offline pending list`
- `GO-0: add aparok android library build workflow; publish AAR to github packages`
- `GO-0: point gradle publishing to aparok-core github packages`

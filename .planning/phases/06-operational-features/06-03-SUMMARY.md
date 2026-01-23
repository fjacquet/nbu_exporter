---
phase: 06-operational-features
plan: 03
subsystem: config
tags: [fsnotify, sighup, rwmutex, reload, config-hot-reload]

# Dependency graph
requires:
  - phase: 06-01
    provides: Storage cache (GetStorageCache for flush on reload)
  - phase: 06-02
    provides: Collector with cache reference
provides:
  - SafeConfig wrapper with RWMutex for thread-safe config access
  - SIGHUP signal handler for manual config reload
  - File watcher (fsnotify) for automatic config reload
  - Cache flush on NBU server address change
affects: [runtime-configuration, operational-documentation]

# Tech tracking
tech-stack:
  added: [github.com/fsnotify/fsnotify@v1.9.0]
  patterns: [SafeConfig thread-safe wrapper, fail-fast validation, directory watching for atomic saves]

key-files:
  created:
    - internal/models/safe_config.go
    - internal/models/safe_config_test.go
    - internal/config/watcher.go
    - internal/config/watcher_test.go
  modified:
    - main.go
    - main_test.go
    - go.mod

key-decisions:
  - "Inline file reading in SafeConfig to avoid import cycle with utils package"
  - "Watch directory (not file) to handle atomic saves from vim/emacs correctly"
  - "Fail-fast validation: validate config BEFORE acquiring write lock"
  - "Return serverChanged flag from ReloadConfig to trigger cache flush"

patterns-established:
  - "SafeConfig.Get() for thread-safe config reads"
  - "ReloadConfig() with fail-fast validation pattern"
  - "Directory watching for file change detection"

# Metrics
duration: 9min
completed: 2026-01-23
---

# Phase 6 Plan 3: Dynamic Configuration Reload Summary

**SafeConfig wrapper with RWMutex, SIGHUP signal handler, and fsnotify file watcher for hot config reload**

## Performance

- **Duration:** 9 min
- **Started:** 2026-01-23T22:25:09Z
- **Completed:** 2026-01-23T22:34:27Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- SafeConfig provides thread-safe config access with RWMutex (concurrent reads, serialized writes)
- SIGHUP signal handler enables manual config reload via `kill -HUP <pid>`
- File watcher (fsnotify) detects config file changes and triggers automatic reload
- Storage cache is flushed automatically when NBU server address changes
- Invalid configurations are rejected without affecting the running exporter (fail-fast)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create SafeConfig wrapper with RWMutex** - `74abba2` (feat)
2. **Task 2: Create file watcher and SIGHUP handler** - `4f47e02` (feat)
3. **Task 3: Integrate config reload into main.go** - `a713dcc` (feat)

## Files Created/Modified

### Created
- `internal/models/safe_config.go` - Thread-safe config wrapper with Get() and ReloadConfig()
- `internal/models/safe_config_test.go` - 9 tests for SafeConfig including concurrency
- `internal/config/watcher.go` - SetupSIGHUPHandler() and WatchConfigFile()
- `internal/config/watcher_test.go` - 6 tests for file watching including atomic writes

### Modified
- `main.go` - Server uses SafeConfig, ReloadConfig method, file watcher integration
- `main_test.go` - Updated all tests for new NewServer signature
- `go.mod` - Added github.com/fsnotify/fsnotify@v1.9.0 dependency

## Decisions Made

1. **Inline file reading in SafeConfig** - Avoided import cycle between models and utils packages by inlining file reading using os and yaml packages directly in safe_config.go

2. **Watch directory not file** - fsnotify watches the directory containing the config file, not the file itself. This handles atomic saves (vim/emacs write to temp file, then rename) which would otherwise be missed by file-level watching.

3. **Fail-fast validation** - Config validation happens BEFORE acquiring write lock. This ensures invalid configurations never affect the running exporter and minimizes lock contention.

4. **serverChanged return flag** - ReloadConfig returns whether NBU server address changed, allowing the caller to flush the storage cache appropriately.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Import cycle between models and utils**
- **Found during:** Task 1 (SafeConfig creation)
- **Issue:** SafeConfig imported utils.FileExists and utils.ReadFile, but utils already imports models.Config, creating import cycle
- **Fix:** Inlined file reading using os.Open(), yaml.NewDecoder(), and os.Stat() directly in safe_config.go
- **Files modified:** internal/models/safe_config.go
- **Verification:** Build succeeds, tests pass
- **Committed in:** 74abba2 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor implementation change to avoid Go import cycle. Functionality identical.

## Issues Encountered
None - plan executed as specified after import cycle fix.

## User Setup Required

None - no external service configuration required. Config reload works automatically.

**Usage:**
- Manual reload: `kill -HUP $(pgrep nbu_exporter)`
- Automatic reload: Edit config.yaml file
- Cache flush: Automatic when nbuserver.host or nbuserver.port changes

## Next Phase Readiness
- Phase 6 complete with all 3 plans executed
- Dynamic configuration reload operational
- Ready for project verification phase

---
*Phase: 06-operational-features*
*Completed: 2026-01-23*

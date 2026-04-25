# Critical UI Bugs - Immediate Action Required

## 🚨 Critical Issues Found

### Issue 1: Command Palette Highlight Bug
**Status:** CRITICAL  
**Impact:** UI shows multiple selected items, confusing user  
**Description:** When navigating with Up/Down arrows in the command palette, the previous highlight doesn't disappear. This results in two lines being highlighted simultaneously - the current selection and the first item.

**Root Cause:** The `d.selected` index tracks the position in the filtered list, but there may be an issue with:
1. Index calculation when list is filtered
2. Rendering logic showing wrong item as selected
3. State not being reset properly when navigating

**Reproduction Steps:**
1. Open command palette (Ctrl+P)
2. Press Down arrow multiple times
3. Observe that both the current item AND the first item remain highlighted

**Expected Behavior:** Only the currently selected item should be highlighted

**Files to Fix:**
- `internal/tui/dialog/command.go` - View() rendering logic
- `internal/tui/dialog/dialog.go` - Selection state management

---

### Issue 2: Debug Logging Messes Up UI
**Status:** CRITICAL  
**Impact:** Logs appear on screen even without --print-logs flag, corrupting TUI display  
**Description:** Debug logs are appearing on the terminal screen, interfering with the UI layout. When logs print, they overwrite or corrupt the text field and dialog displays.

**Root Cause:** Current logger writes to stdout/stderr directly. OpenCode's logger writes to files, avoiding TUI interference.

**Current Implementation:**
```go
// Current - writes to stdout/stderr
func (l *Logger) Info(msg string) {
    fmt.Println(msg)  // This messes up TUI!
}
```

**OpenCode's Implementation:**
```typescript
// Writes to file by default, stderr only if print=true
export async function init(options: Options) {
  if (options.print) return  // Don't redirect if print mode
  logpath = path.join(Global.Path.log, "dev.log")
  const stream = createWriteStream(logpath, { flags: "a" })
  write = async (msg: any) => {
    stream.write(msg)  // Writes to file, not stdout
  }
}
```

**Expected Behavior:**
- Logs should go to file by default (`~/.local/share/magicode/logs/`)
- Only print to stderr when explicitly enabled with `--print-logs`
- Never interfere with TUI rendering

**Files to Fix:**
- `internal/util/log/log.go` - Change default output to file
- `cmd/magicode/main.go` - Add log initialization on startup
- Add log file rotation/cleanup

---

### Issue 3: Logging Toggle Corrupts Display
**Status:** HIGH  
**Impact:** Enabling/disabling logs at runtime messes up the UI layout  
**Description:** When toggling logging on/off, the UI display gets corrupted. The text field and dialogs don't render correctly.

**Root Cause:** Writing to stdout/stderr while TUI is rendering causes display corruption.

**Expected Behavior:**
- Runtime log level changes should not affect TUI
- Logs should always go to file
- Optional: In-app log viewer that doesn't use stdout

**Files to Fix:**
- Same as Issue 2

---

## Revised Implementation Plan

### Immediate Priority (Sprint 0): Critical Bug Fixes

#### Task 0.1: Fix Command Palette Highlight Bug
**Time:** 30 minutes  
**Priority:** CRITICAL

- [ ] Debug the selection index calculation in command palette
- [ ] Check if `d.selected` is correctly mapped to filtered list index
- [ ] Fix rendering logic in `View()` method
- [ ] Ensure only one item is highlighted at a time
- [ ] Test with filtered list (type to search)
- [ ] Test with scrolling (long lists)

**Files:**
- `internal/tui/dialog/command.go`

#### Task 0.2: Implement File-Based Logging (Like OpenCode)
**Time:** 1-2 hours  
**Priority:** CRITICAL

- [ ] Create `internal/util/log/file.go`
- [ ] Implement file-based log writer
- [ ] Log path: `~/.local/share/magicode/logs/YYYY-MM-DDTHHMMSS.log`
- [ ] Keep last 10 log files (cleanup old ones)
- [ ] Only write to stderr if `--print-logs` flag is set
- [ ] Initialize logger early in main()
- [ ] Update all log calls to use file writer

**Files:**
- `internal/util/log/file.go` (new)
- `cmd/magicode/main.go`
- `internal/util/log/log.go`

---

### Updated Phase Order

Now the implementation order should be:

#### Sprint 0: Critical Fixes (1-2 hours)
1. **Task 0.1:** Fix command palette highlight bug
2. **Task 0.2:** Implement file-based logging

#### Sprint 1: Foundation (2-3 hours)
3. **Phase 1:** Config Reading (agents/models)
4. **Phase 4:** Auto-scroll (can parallelize)

#### Sprint 2: Core UI (2-3 hours)
5. **Phase 2:** Footer Redesign (with agent/model data)
6. **Phase 3:** Text Wrapping (can parallelize)

#### Sprint 3: Interaction (1-2 hours)
7. **Phase 5:** Tab Cycling

#### Sprint 4: Polish (2-3 hours)
8. **Phase 6:** Testing & Documentation

**Total Time:** 10-14 hours (including critical fixes)

---

## Quick Fixes (Do First)

### Fix 1: Command Palette Selection

The bug is likely in how the selected index is calculated relative to the view window. When scrolling, the view shows a subset of items, but the selection index is global.

**Problem:**
```go
// In View():
for i := startIdx; i < endIdx; i++ {
    isSelected := i == d.selected  // This compares view index to global index
}
```

**Fix:** Ensure `d.selected` is always within `0` to `len(d.filtered)-1` and the comparison is correct.

### Fix 2: File-Based Logging

**OpenCode Pattern:**
1. By default: write to `~/.local/share/opencode/logs/`
2. With `--print-logs`: write to stderr
3. Never write to stdout (breaks TUI)

**Implementation:**
```go
// In main.go
if !printLogs {
    // Redirect to file
    logFile := filepath.Join(getLogDir(), time.Now().Format("2006-01-02T150405.log"))
    f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    log.SetOutput(f)
}
```

---

## Success Criteria for Critical Fixes

- [ ] Command palette shows only ONE highlighted item at a time
- [ ] Logs go to file by default, not stdout/stderr
- [ ] UI doesn't get corrupted when using the app
- [ ] `--print-logs` flag works to show logs on stderr
- [ ] Log files are rotated (keep only last 10)

---

## Next Steps

1. **Fix critical bugs first** (Sprint 0)
2. **Test thoroughly** - these bugs affect basic usability
3. **Then proceed with UI alignment** (Phases 1-6)

The critical bugs make the app difficult to use, so they should be fixed before adding new features.

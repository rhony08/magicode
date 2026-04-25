# MagiCode UI Fixes - Implementation Todo

## Overview
This document contains the detailed implementation plan for aligning MagiCode's UI with OpenCode. Each phase is broken down into specific tasks with estimates and dependencies.

---

## Phase 1: Foundation (Config Reading)
**Priority: HIGH**
**Estimated Time: 2-3 hours**
**Dependencies: None**

> **Why first?** Other features depend on having agent/model data available.

### Task 1.1: Create OpenCode Config Reader
- [ ] Create `internal/opencode/config.go`
- [ ] Implement `ReadAgentConfig()` - reads `~/.config/opencode/agent.json`
- [ ] Implement `ReadProviderConfig()` - reads `~/.config/opencode/provider.json`
- [ ] Implement `ReadModelState()` - reads `~/.local/share/opencode/state/model.json`
- [ ] Add structs for Agent, Provider, Model config
- [ ] Add error handling for missing files
- [ ] **Test:** Verify config files are read correctly

**Files to create:**
- `internal/opencode/config.go`
- `internal/opencode/config_test.go`

### Task 1.2: Extend Types for Agent/Model Storage
- [ ] Update `internal/tui/types/types.go`
- [ ] Add `Agent` struct with Name, Mode, Color, Description
- [ ] Add `AgentStore` to LocalStore with CurrentAgent and List
- [ ] Add methods: `SetAgent()`, `GetAgent()`, `CycleAgent()`, `GetAgentColor()`
- [ ] Extend model storage with validation
- [ ] Add methods: `SetModel()`, `GetModel()`, `CycleModel()`, `IsModelValid()`
- [ ] **Test:** Verify store operations work

**Files to modify:**
- `internal/tui/types/types.go`

### Task 1.3: Sync Config on Startup
- [ ] Update `internal/tui/app.go` Init()
- [ ] Read OpenCode configs on startup
- [ ] Sync agents to state
- [ ] Sync providers/models to state
- [ ] Set default agent/model if not already set
- [ ] **Test:** Verify configs are loaded and synced

**Files to modify:**
- `internal/tui/app.go`

---

## Phase 2: Footer/Status Bar Redesign
**Priority: HIGH**
**Estimated Time: 2-3 hours**
**Dependencies: Phase 1 (needs agent/model data)**

> **Most visible UI change - users see this immediately**

### Task 2.1: Redesign Footer Layout
- [ ] Update `internal/tui/layout/footer.go`
- [ ] Implement three-section layout: directory | agent/model | LSP/MCP
- [ ] Add directory abbreviation (replace home with ~)
- [ ] Add agent display with colored dot
- [ ] Add model display (provider/model format)
- [ ] Add LSP count with colored indicator
- [ ] Add MCP count with colored indicator
- [ ] Add version info
- [ ] **Test:** Verify layout renders correctly

**Files to modify:**
- `internal/tui/layout/footer.go`
- `internal/tui/layout/footer_test.go`

### Task 2.2: Add Colored Status Indicators
- [ ] Implement colored dot rendering for LSP/MCP status
- [ ] Green dot when connected
- [ ] Red dot when error
- [ ] Gray dot when disconnected
- [ ] Implement agent color display
- [ ] Support hex colors
- [ ] Support theme color keys
- [ ] **Test:** Verify colors render correctly

**Files to modify:**
- `internal/tui/layout/footer.go`
- `internal/tui/theme.go` (add helper for agent colors)

### Task 2.3: Integrate with App
- [ ] Update `internal/tui/app.go` View()
- [ ] Pass agent/model data to footer
- [ ] Update footer when agent/model changes
- [ ] **Test:** Verify footer updates dynamically

**Files to modify:**
- `internal/tui/app.go`

---

## Phase 3: Text Wrapping
**Priority: HIGH**
**Estimated Time: 1-2 hours**
**Dependencies: None (can be done in parallel)**

> **Critical usability fix - prevents text overflow**

### Task 3.1: Implement Word Wrap Function
- [ ] Create `internal/tui/component/wrap.go`
- [ ] Implement `WordWrap(text string, width int) []string`
- [ ] Handle word boundaries correctly
- [ ] Preserve indentation for wrapped lines
- [ ] Handle code blocks specially (don't wrap code)
- [ ] **Test:** Verify wrapping handles edge cases

**Files to create:**
- `internal/tui/component/wrap.go`
- `internal/tui/component/wrap_test.go`

### Task 3.2: Calculate Content Width
- [ ] Update `internal/tui/app.go`
- [ ] Calculate available width based on:
  - Terminal width
  - Sidebar visibility
  - Sidebar width (344px default)
  - Padding/margins
- [ ] Formula: `width - (sidebarVisible ? 43 : 0) - 4`
- [ ] Store in state or recalculate on render
- [ ] **Test:** Verify width calculation

**Files to modify:**
- `internal/tui/app.go`
- `internal/tui/types/types.go` (add content width to LayoutStore)

### Task 3.3: Apply Wrapping to Messages
- [ ] Update `internal/tui/app.go` `buildMessagesContent()`
- [ ] Wrap message content before rendering
- [ ] Don't wrap code blocks
- [ ] Preserve newlines in wrapped text
- [ ] **Test:** Verify messages display wrapped correctly

**Files to modify:**
- `internal/tui/app.go`

---

## Phase 4: Auto-scroll to Bottom
**Priority: MEDIUM-HIGH**
**Estimated Time: 1 hour**
**Dependencies: None (can be done in parallel)**

> **Quality of life improvement**

### Task 4.1: Track Scroll State
- [ ] Add `ScrollState` struct to track:
  - `AtBottom bool`
  - `UserScrolled bool`
  - `LastScrollPosition int`
- [ ] Update scroll state on scroll events
- [ ] Detect when user scrolls up (disable auto-scroll)
- [ ] Detect when user scrolls to bottom (enable auto-scroll)
- [ ] **Test:** Verify scroll state tracking

**Files to modify:**
- `internal/tui/types/types.go`

### Task 4.2: Implement Auto-scroll Logic
- [ ] Update `internal/tui/app.go` `addMessage()`
- [ ] Call `GotoBottom()` after adding message
- [ ] Only auto-scroll if `AtBottom` is true
- [ ] Update `loadMessagesForSession()`
- [ ] Scroll to bottom after loading messages
- [ ] **Test:** Verify auto-scroll works correctly

**Files to modify:**
- `internal/tui/app.go`

---

## Phase 5: Tab Key Cycling
**Priority: MEDIUM**
**Estimated Time: 1-2 hours**
**Dependencies: Phase 1 (needs agent/model data)**

> **Keyboard UX improvement**

### Task 5.1: Add Tab Keybindings
- [ ] Update `internal/tui/keybindings.go`
- [ ] Add `CycleAgent` keybinding (Tab)
- [ ] Add `CycleModel` keybinding (Shift+Tab)
- [ ] Update `KeyNames()` function
- [ ] **Test:** Verify keybindings work

**Files to modify:**
- `internal/tui/keybindings.go`

### Task 5.2: Implement Cycling Logic
- [ ] Update `internal/tui/app.go` `handleChatKey()`
- [ ] Handle Tab key: cycle to next agent
- [ ] Handle Shift+Tab: cycle to next model
- [ ] Wrap around at end of list
- [ ] Update state and footer immediately
- [ ] Show toast notification with new agent/model
- [ ] **Test:** Verify cycling works correctly

**Files to modify:**
- `internal/tui/app.go`

### Task 5.3: Add Help Documentation
- [ ] Update `internal/tui/dialog/help.go`
- [ ] Add Tab and Shift+Tab to keybind help
- [ ] Document cycling behavior
- [ ] **Test:** Verify help shows new keybinds

**Files to modify:**
- `internal/tui/dialog/help.go`

---

## Phase 6: Polish & Integration Testing
**Priority: MEDIUM**
**Estimated Time: 2-3 hours**
**Dependencies: All previous phases**

### Task 6.1: Fix Remaining Issues
- [ ] Fix date format issue (if not already fixed)
- [ ] Fix session message loading (if not already fixed)
- [ ] Fix any edge cases discovered during testing
- [ ] **Test:** All basic functionality works

### Task 6.2: Integration Testing
- [ ] Test with --use-opencode flag
- [ ] Test without --use-opencode flag
- [ ] Test agent cycling
- [ ] Test model cycling
- [ ] Test text wrapping with long messages
- [ ] Test text wrapping with code blocks
- [ ] Test auto-scroll behavior
- [ ] Test footer display with various states

### Task 6.3: Documentation
- [ ] Update AGENTS.md with new features
- [ ] Document new keyboard shortcuts
- [ ] Add examples to documentation
- [ ] Update README if needed

---

## Implementation Order

### Sprint 1 (Foundation)
1. Phase 1: Config Reading (2-3 hours)
2. Phase 4: Auto-scroll (1 hour) - can parallelize

### Sprint 2 (Core UI)
3. Phase 2: Footer Redesign (2-3 hours)
4. Phase 3: Text Wrapping (1-2 hours) - can parallelize

### Sprint 3 (Interaction)
5. Phase 5: Tab Cycling (1-2 hours)

### Sprint 4 (Polish)
6. Phase 6: Testing & Polish (2-3 hours)

**Total Estimated Time: 10-14 hours**

---

## Priority Matrix

| Feature | User Impact | Implementation Complexity | Priority |
|---------|-------------|---------------------------|----------|
| Config Reading | High | Medium | 1 |
| Footer Redesign | High | Medium | 2 |
| Text Wrapping | High | Low | 3 |
| Auto-scroll | Medium | Low | 4 |
| Tab Cycling | Medium | Low | 5 |

---

## Risk Assessment

### High Risk
- **Config Reading**: OpenCode config format may change
- **Footer Layout**: May conflict with existing layout code

### Medium Risk
- **Text Wrapping**: Code block handling can be tricky
- **Auto-scroll**: May interfere with user scroll position

### Low Risk
- **Tab Cycling**: Straightforward keybinding logic

---

## Success Criteria

- [ ] Footer displays directory, agent (with color), model, LSP/MCP status
- [ ] Agent cycling works with Tab key
- [ ] Model cycling works with Shift+Tab key
- [ ] Text wraps at viewport boundary without overflow
- [ ] Auto-scrolls to bottom on new messages (when user is at bottom)
- [ ] Reads OpenCode config correctly when --use-opencode is used
- [ ] All existing tests pass
- [ ] New features have tests

---

## Notes

- Focus on getting Phase 1 done first - other phases depend on it
- Text wrapping can be done in parallel with Phase 1
- Test incrementally - don't wait until the end to test
- Keep debug logging in place until everything works
- Consider user feedback after each phase

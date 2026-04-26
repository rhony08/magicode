# OpenCode vs MagiCode - Message Rendering & Session Review

## Executive Summary

This document compares the message rendering and session switching implementations between OpenCode (TypeScript/SolidJS) and MagiCode (Go/TUI). The analysis focuses on message part types, rendering behavior, session management, and state persistence.

**Overall Status**: ✅ Core functionality matches OpenCode with minor gaps in advanced features.

---

## Message Part Types

| Part Type | OpenCode Handling | MagiCode Handling | Status |
|-----------|------------------|-------------------|--------|
| `text` | Full markdown rendering with syntax highlighting | Basic text rendering with wrapping | ✅ Implemented |
| `reasoning` / `thinking` | Collapsible section with streaming support | Emoji prefix `💭` with text wrapping | ✅ Implemented |
| `tool` / `tool_use` | Full tool registry with custom renderers per tool type | Generic tool renderer with tool-specific formatting | ✅ Implemented |
| `tool_result` | Status-based styling, error cards, diff views | Status indicators, truncated output, diff highlighting | ✅ Implemented |
| `file` | Inline/attached file display with preview | `[File: filename]` placeholder | ⚠️ Partial |
| `agent` | Inline agent mention with color highlighting | Not handled in parts | ❌ Gap |
| `patch` | **Filtered out (SKIP_PARTS)** | **Filtered out (skipParts)** | ✅ Match |
| `step-start` | **Filtered out (SKIP_PARTS)** | **Filtered out (skipParts)** | ✅ Match |
| `step-finish` | **Filtered out (SKIP_PARTS)** | **Filtered out (skipParts)** | ✅ Match |
| `compaction` | Divider with label "Compacted History" | Not implemented | ❌ Gap |

### SKIP_PARTS Implementation

**OpenCode** (sync.tsx line 18):
```typescript
const SKIP_PARTS = new Set(["patch", "step-start", "step-finish"])

// Applied when loading messages:
const filtered = p.part.filter((x) => !SKIP_PARTS.has(x.type))
if (filtered.length) input.setStore("part", p.id, filtered)
```

**MagiCode** (app.go lines 2369-2405):
```go
func convertPartsToTUI(parts []database.Part) []Part {
    skipParts := map[string]bool{
        "patch":       true,
        "step-start":  true,
        "step-finish": true,
    }

    var result []Part
    for _, p := range parts {
        if skipParts[p.Data.Type] {
            continue // Skip internal parts
        }
        // ... convert to TUI Part
    }
    return result
}
```

✅ **Status**: Both implementations correctly filter the same part types.

---

## Message Rendering Comparison

### User Messages

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Timestamp display | `Intl.DateTimeFormat` with locale | `Format("15:04")` hardcoded | ⚠️ Different |
| Agent/model metadata | Agent name + model name in metadata line | Only shown in footer | ⚠️ Different |
| File attachments | Inline preview, click-to-open, file icons | `[File: filename]` placeholder | ⚠️ Partial |
| Image attachments | Inline display, dialog preview | Not implemented | ❌ Gap |
| Copy button | Copy text content | Not implemented | ❌ Gap |
| Revert button | Revert to this message | Not implemented | ❌ Gap |

**OpenCode UserMessageDisplay** (message-part.tsx):
```tsx
// Attachments section
<For each={attachments()}>
  {(file) => {
    const type = kind(file)
    return (
      <div data-slot="user-message-attachment" data-type={type}>
        <Show when={type === "image"}>
          <img src={file.url} alt={name} />
        </Show>
      </div>
    )
  }}
</For>

// Metadata line
<span data-slot="user-message-meta">
  {metaHead()} // Agent · Model
</span>
<span data-slot="user-message-meta-tail">
  {metaTail()} // Timestamp
</span>
```

**MagiCode buildMessagesContent** (app.go):
```go
timeStr := msg.Timestamp.Format("15:04")
prefix := fmt.Sprintf("[%s] You: ", timeStr)
// Content wrapped and rendered
lines = append(lines, a.styles.UserMessage.Render(prefix+line))
```

### Assistant Messages

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Streaming text | Paced rendering with `createPacedValue` | Basic append on StreamMsg | ⚠️ Different |
| Text parts | Markdown with syntax highlighting | Basic text with wrapping | ✅ Basic |
| Reasoning parts | Collapsible, markdown, streaming | Emoji prefix, simple rendering | ⚠️ Partial |
| Tool context grouping | Groups read/glob/grep tools together | Individual tool items | ❌ Gap |
| Tool status | Running/completed/error states | pending/running/success/error | ✅ Match |
| Model info | Shows agent + model + duration in meta | Shows model in message header | ⚠️ Different |
| Error display | ToolErrorCard with detailed error | Status line + truncated error | ⚠️ Partial |

**OpenCode Context Tool Grouping** (message-part.tsx):
```tsx
const CONTEXT_GROUP_TOOLS = new Set(["read", "glob", "grep", "list"])

function groupParts(parts) {
  // Groups consecutive context tools together
  // Shows: "Gathering context • 3 reads • 2 searches"
}

// Tool registry for custom rendering
ToolRegistry.register({
  name: "bash",
  render(props) {
    // Custom Bash tool display
  },
})
```

**MagiCode tool_result.go**:
```go
func (r *ToolResultRenderer) RenderResult(toolName string, toolInput string, output string, status string) string {
  switch toolName {
  case "bash":
    return r.renderBash(toolInput, output, status)
  case "read":
    return r.renderRead(toolInput, output, status)
  // ... tool-specific rendering
  }
}
```

✅ **Status**: MagiCode has tool-specific rendering but lacks context grouping.

### Tool Result Rendering

| Tool Type | OpenCode | MagiCode | Status |
|-----------|----------|----------|--------|
| `bash` | Command + output, shell-specific styling | `▶ bash` header, command, truncated output | ✅ Match |
| `read` | File path, line numbers, loaded files list | `📄 read` header, file path, content preview | ✅ Match |
| `write` | File path, new content preview | `✎ write` header, file path, preview | ✅ Match |
| `edit` | Diff view with added/removed highlighting | `✎ edit` header, diff with +/- colors | ✅ Match |
| `glob` | Pattern + file list with count | `📁 glob` header, pattern, file list | ✅ Match |
| `grep` | Pattern + matches with highlighting | `🔍 grep` header, pattern, matches | ✅ Match |
| `todowrite` | **Hidden (HIDDEN_TOOLS)** | Not rendered | ✅ Match |
| `task` | Subagent link, description | Generic tool render | ⚠️ Partial |
| `question` | Dismissible dialog for user input | Not implemented | ❌ Gap |

---

## Markdown Rendering Comparison

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Code blocks | Full syntax highlighting via Chroma | Basic keyword/string highlighting | ⚠️ Partial |
| Inline code | Background color styling | Background color styling | ✅ Match |
| Headers | Underline for H1, bold for others | Bold with theme colors | ✅ Match |
| Lists | Bullet/number with proper indentation | Bullet/number rendering | ✅ Match |
| Links | Underlined, clickable | Underlined, text only | ⚠️ Partial |
| Blockquotes | Border left, muted text | Border left, muted text | ✅ Match |
| Tables | Header underline, cell alignment | Header underline, truncation | ⚠️ Partial |
| Bold/Italic | Proper styling | Proper styling | ✅ Match |
| Strikethrough | Supported | Supported | ✅ Match |
| Streaming | Paced reveal at word boundaries | No paced reveal | ❌ Gap |

**OpenCode PacedMarkdown** (message-part.tsx):
```tsx
function createPacedValue(getValue: () => string, live?: () => boolean) {
  // Reveals text incrementally at word boundaries
  // Uses TEXT_RENDER_PACE_MS = 24ms
  // TEXT_RENDER_SNAP = /[\s.,!?;:)\]]/
}

// Streaming text reveals at natural breakpoints
```

**MagiCode markdown.go**:
```go
func (r *MarkdownRenderer) highlightSyntax(line string, language string) string {
  // Basic keyword highlighting
  keywords := []string{"func", "function", "class", "if", "else"...}
  // String highlighting for double quotes
  // Comment detection for // prefix
}
```

---

## Session Switching Flow Comparison

### OpenCode Session Flow

1. **Navigation**: URL-driven via `/:dir/session/:id`
2. **Sync on mount**: `sync.session.sync(id)` prefetches messages
3. **State layers**:
   - `globalSync`: Project-wide data (projects, providers)
   - `sync`: Per-directory data (sessions, messages)
   - `local`: Per-session agent/model selection
   - `layout`: UI dimensions, sidebar state

4. **History window**:
   - `turnStart`: Limits initial visible turns (default 10)
   - `turnBatch`: Reveals cached turns in batches (default 8)
   - `loadMore`: Fetches older history when scrolling up

5. **Session list dialog**: Searchable, shows title + time + directory

**OpenCode createSessionHistoryWindow** (session.tsx):
```tsx
const turnInit = 10
const turnBatch = 8
const turnScrollThreshold = 200
const turnPrefetchBuffer = 16

// Preserves scroll position when backfilling
const preserveScroll = (fn: () => void) => {
  const beforeTop = el.scrollTop
  const beforeHeight = el.scrollHeight
  fn()
  requestAnimationFrame(() => {
    el.scrollTop = beforeTop + (el.scrollHeight - beforeHeight)
  })
}
```

### MagiCode Session Flow

1. **Navigation**: Dialog-based via Ctrl+X + l
2. **Load on select**: `loadMessagesForSession(session.ID)`
3. **State layers**:
   - `KVStore`: Persistent preferences
   - `SyncStore`: Sessions, messages, providers
   - `LocalStore`: Agent/model selection
   - `LayoutStore`: Dimensions, sidebar state

4. **Pagination**:
   - `InitialMessagePageSize = 80`
   - `HistoryMessagePageSize = 200`
   - Cursor-based (timestamp)

5. **Session list dialog**: Searchable, shows title + relative time

**MagiCode loadMessagesForSession** (app.go):
```go
func (a *App) loadMessagesForSession(sessionID string) tea.Cmd {
  return func() tea.Msg {
    messages, cursor, complete, err := messageStorage.ListPaginated(ctx, sessionID, 80, 0)
    return MessagesLoadedMsg{
      SessionID: sessionID,
      Messages:  convertMessagesToTUI(messages),
    }
  }
}
```

### Comparison Table

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Session sync on mount | Prefetch + sync | Load on dialog select | ⚠️ Different |
| History windowing | Turn-based batching | Cursor-based pagination | ⚠️ Different |
| Scroll preservation | Preserves on backfill | Resets to bottom | ❌ Gap |
| Session search | Title + directory + ID | Title + directory + ID | ✅ Match |
| Relative time display | Locale-aware `formatTime` | "Just now", "5m ago", "Jan 2" | ✅ Match |
| Session title generation | First message text | First message text | ✅ Match |
| Session archive/delete | Dropdown menu actions | Not in dialog | ❌ Gap |

---

## State Persistence Comparison

### OpenCode State Layers

| Layer | Scope | Persistence |
|-------|-------|-------------|
| `globalSync` | Project-wide | Database + localStorage |
| `sync` | Per-directory | Database |
| `local` | Per-session | localStorage (`model-selection.v1`) |
| `layout` | Per-session | localStorage |
| `kv` | Global | localStorage |

**OpenCode persisted store** (local.tsx):
```tsx
const [saved, setSaved] = persisted(
  Persist.workspace(sdk.directory, "model-selection", ["model-selection.v1"]),
  createStore<Saved>({ session: {} }),
)
```

### MagiCode State Layers

| Layer | Scope | Persistence |
|-------|-------|-------------|
| `KVStore` | Global | SQLite (`kv` table) |
| `SyncStore` | Per-directory | SQLite |
| `LocalStore` | Per-session | Not persisted | ❌ Gap |
| `LayoutStore` | Per-session | Not persisted | ❌ Gap |
| `MessageMeta` | Per-session | Not persisted | ❌ Gap |

**MagiCode KVStorage** (kv.go):
```go
func (s *KVStorage) SetTheme(ctx context.Context, theme string) error {
  return s.db.Exec(ctx, "INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)", "theme", theme)
}
```

### Missing Persistence in MagiCode

| Data | OpenCode | MagiCode |
|------|----------|----------|
| Agent/model per session | `localStorage` | Not persisted |
| Sidebar width/mode | `localStorage` | Persisted ✅ |
| Scroll positions | Per-session view state | Not persisted |
| Input drafts | `prompt.current()` | Not persisted |
| Session tabs | `layout.session.tabs()` | Not persisted |

---

## Implementation Gaps

### High Priority

1. **Context Tool Grouping**: OpenCode groups read/glob/grep/list tools under "Gathering context". MagiCode shows them individually.

   ```go
   // Need to implement:
   const CONTEXT_GROUP_TOOLS = map[string]bool{
     "read":  true,
     "glob":  true,
     "grep":  true,
     "list":  true,
   }
   
   func groupPartsForContext(parts []Part) []PartGroup {
     // Group consecutive context tools
   }
   ```

2. **Paced Streaming Text**: OpenCode reveals streaming text at word boundaries. MagiCode appends directly.

   ```go
   // Need to implement paced reveal:
   func pacedTextAppend(current string, new string) string {
     // Reveal at word boundaries (TEXT_RENDER_SNAP pattern)
   }
   ```

3. **Scroll Preservation on Backfill**: OpenCode preserves scroll position when loading older messages.

   ```go
   // Need to implement:
   func preserveScroll(fn func()) {
     beforeTop := viewport.YOffset
     beforeHeight := viewport.TotalLineCount()
     fn()
     viewport.YOffset = beforeTop + (viewport.TotalLineCount() - beforeHeight)
   }
   ```

### Medium Priority

4. **Compaction Part**: Divider for compacted history sections.

5. **Question Tool**: Dismissible dialog for user input questions.

6. **File Attachments**: Inline preview for images and file icons.

7. **Local State Persistence**: Per-session agent/model selection should persist to SQLite.

### Low Priority

8. **Advanced Syntax Highlighting**: Use Chroma for full syntax highlighting.

9. **Session Archive/Delete in Dialog**: Add actions to session list dialog.

10. **Agent Color Highlighting**: Highlight agent mentions in user messages.

---

## UI/UX Differences

### Timestamps

| Aspect | OpenCode | MagiCode |
|--------|----------|----------|
| Format | Locale-aware (`Intl.DateTimeFormat`) | Fixed format (`15:04`) |
| Position | Inline with message | Prefix on message |
| Relative time | "Just now", "5m ago" (session list) | Same pattern |

### Message Metadata

| Aspect | OpenCode | MagiCode |
|--------|----------|----------|
| Agent name | In message meta line | Footer only |
| Model name | In message meta line | Message header |
| Duration | Calculated from timestamps | Not shown |

### Scroll Behavior

| Aspect | OpenCode | MagiCode |
|--------|----------|----------|
| Auto-scroll | Dynamic overflow anchor | Force to bottom |
| User scroll detection | `autoScroll.userScrolled()` | `state.Layout.UserScrolled` |
| Jump threshold | `max(400, clientHeight)` | Not implemented |

---

## Recommendations

1. **Implement context tool grouping** to match OpenCode's "Gathering context" UI.

2. **Add scroll preservation** when backfilling older messages.

3. **Implement paced text streaming** for smoother message reveal.

4. **Persist LocalStore and LayoutStore** to SQLite for session continuity.

5. **Add compaction part rendering** for compacted history sections.

6. **Enhance markdown renderer** with full Chroma syntax highlighting.

7. **Add file attachment preview** with inline image/file icon display.

8. **Implement question tool dialog** for user input questions.

---

## Conclusion

MagiCode successfully implements core message rendering and session switching with:
- ✅ Correct `SKIP_PARTS` filtering matching OpenCode
- ✅ Cursor-based pagination (80/200 page sizes)
- ✅ Tool-specific rendering for bash, read, write, edit, glob, grep
- ✅ Diff highlighting for edit operations
- ✅ Thinking/reasoning part rendering
- ✅ Session list dialog with search
- ✅ Theme system with persistence

Remaining work focuses on:
- Context tool grouping UI pattern
- Scroll preservation on history backfill
- Advanced streaming text reveal
- State persistence gaps
- Advanced tool features (question, compaction)
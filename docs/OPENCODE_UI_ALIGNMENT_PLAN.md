# MagiCode UI Alignment Plan

## Executive Summary

After reviewing OpenCode's TUI implementation, I've identified several critical gaps between MagiCode and OpenCode. This plan addresses all reported issues to achieve UI parity.

## Current Issues vs OpenCode

### 1. Footer/Status Bar Display ❌

**OpenCode Implementation:**
```typescript
// From footer.tsx
<box flexDirection="row" justifyContent="space-between">
  <text fg={theme.textMuted}>{directory()}</text>
  <box gap={2} flexDirection="row">
    <text fg={theme.text}>
      <span style={{ fg: lsp().length > 0 ? theme.success : theme.textMuted }}>•</span> {lsp().length} LSP
    </text>
    <Show when={mcp()}>
      <text>
        <span style={{ fg: mcpError() ? theme.error : theme.success }}>⊙ </span>
        {mcp()} MCP
      </text>
    </Show>
    <text fg={theme.textMuted}>/status</text>
  </box>
</box>
```

**What's Missing in MagiCode:**
- ❌ Agent name with color indicator
- ❌ Current model display (provider/model)
- ❌ Mode indicator (build/plan/etc)
- ❌ LSP/MCP connection status with colored dots
- ❌ Status command hint

**OpenCode Data Structure:**
```typescript
// From local.tsx
agent: {
  list() - returns agents with names, modes, colors
  current() - returns current agent
  set(name) - sets current agent
  move(direction) - cycles through agents (used by Tab key)
}

model: {
  current() - returns { providerID, modelID }
  set(providerID, modelID)
  move(direction) - cycles through models
}
```

### 2. Text Wrapping ❌

**OpenCode Implementation:**
```typescript
// From session/index.tsx
const contentWidth = createMemo(() => dimensions().width - (sidebarVisible() ? 42 : 0) - 4)
```

**MagiCode Issue:**
- Messages don't wrap text - content overflows viewport
- Need to calculate available width and wrap text accordingly

### 3. Auto-scroll to Bottom ❌

**OpenCode Implementation:**
- Auto-scrolls when new messages arrive
- Maintains scroll position when user scrolls up
- Scrolls to bottom on initial load

**MagiCode Issue:**
- `GotoBottom()` exists but may not be called properly
- Scroll position not maintained correctly

### 4. Tab Key Cycling ❌

**OpenCode Implementation:**
```typescript
// From local.tsx
move(direction: 1 | -1) {
  const current = this.current()
  if (!current) return
  let next = agents().findIndex((x) => x.name === current.name) + direction
  if (next < 0) next = agents().length - 1
  if (next >= agents().length) next = 0
  setAgentStore("current", value.name)
}
```

**Keybind:** Tab cycles agents, Shift+Tab cycles models (or vice versa)

**MagiCode Issue:**
- Tab key not bound to cycle agents/modes

### 5. Model/Agent Config Reading ❌

**OpenCode Implementation:**
- Reads agents from sync.data.agent (includes mode, color, description)
- Reads models from sync.data.provider
- Stores recent/favorite models in state file
- Validates models against available providers

**OpenCode Agent Structure:**
```typescript
{
  name: string
  mode: "subagent" | "normal"
  hidden: boolean
  color?: string  // hex color or theme key
  description?: string
}
```

**OpenCode Provider Structure:**
```typescript
{
  id: string
  models: Record<string, { id: string, name: string, cost?: { input: number } }>
}
```

**MagiCode Issue:**
- Not reading OpenCode's agent config
- Not reading OpenCode's provider config
- No model validation

### 6. Mode Display ❌

**OpenCode Concept:**
- "mode" refers to agent mode (subagent, normal, etc)
- Also refers to "build" vs "plan" modes for agents
- Displayed in status bar with agent name

## Implementation Plan

### Phase 1: Footer/Status Bar Redesign (HIGH PRIORITY)

**Tasks:**
1. **Update Footer Layout** (`internal/tui/layout/footer.go`)
   - Left side: Current directory (abbreviated)
   - Center: Agent name with colored indicator + Model (provider/model)
   - Right side: LSP/MCP status with colored dots + version

2. **Add Agent Support** (`internal/tui/types/types.go`)
   - Add Agent struct with name, mode, color, description
   - Add AgentStore with current agent and list
   - Add methods: SetAgent, CycleAgent, GetAgentColor

3. **Add Model Support**
   - Extend LocalStore to track current model
   - Add model validation against providers
   - Add methods: SetModel, CycleModel

4. **Read OpenCode Config**
   - Parse `~/.config/opencode/agent.json`
   - Parse `~/.config/opencode/provider.json`
   - Sync with database providers/agents

### Phase 2: Text Wrapping (HIGH PRIORITY)

**Tasks:**
1. **Message Viewport** (`internal/tui/app.go`)
   - Calculate content width: `width - sidebarWidth - padding`
   - Wrap message content to fit width
   - Handle code blocks specially (preserve formatting)

2. **Tool Result Rendering** (`internal/tui/component/tool_result.go`)
   - Apply width constraints to tool output
   - Wrap long lines with proper indentation

### Phase 3: Auto-scroll (HIGH PRIORITY)

**Tasks:**
1. **Scroll Behavior** (`internal/tui/app.go`)
   - Call `GotoBottom()` after adding messages
   - Track user scroll position
   - Only auto-scroll if user is at bottom

2. **Initial Load** (`internal/tui/app.go`)
   - Scroll to bottom after loading session messages
   - Maintain position when switching sessions

### Phase 4: Tab Key Cycling (MEDIUM PRIORITY)

**Tasks:**
1. **Keybinding** (`internal/tui/keybindings.go`)
   - Add Tab and Shift+Tab keybindings
   - Bind to cycle functions

2. **Cycle Logic** (`internal/tui/app.go`)
   - Tab: Cycle through agents
   - Shift+Tab: Cycle through models
   - Update status bar immediately

### Phase 5: Config Integration (MEDIUM PRIORITY)

**Tasks:**
1. **Agent Config** (`internal/opencode/`)
   - Read `~/.config/opencode/agent.json`
   - Parse agent definitions
   - Sync to MagiCode state

2. **Provider Config**
   - Read `~/.config/opencode/provider.json`
   - Parse provider and model definitions
   - Validate models

3. **State Persistence**
   - Save recent models to MagiCode KV store
   - Save agent preferences

## Detailed Implementation Guide

### 1. Footer Layout (OpenCode-style)

```
┌─────────────────────────────────────────────────────────────────┐
│ ~/projects/myapp    Agent:Code(mode) • Model:openai/gpt-4o   •2 LSP ⊙1 MCP │
└─────────────────────────────────────────────────────────────────┘
```

**Components:**
- **Directory:** Abbreviated path (replace home with ~)
- **Agent:** Colored dot + name + mode in parentheses
- **Model:** Provider/model format
- **LSP:** Green dot + count
- **MCP:** Green/red dot + count

### 2. Message Wrapping

**Algorithm:**
```go
func wrapMessage(content string, width int) string {
    lines := strings.Split(content, "\n")
    var result []string
    
    for _, line := range lines {
        if len(line) <= width {
            result = append(result, line)
        } else {
            // Wrap at word boundary
            wrapped := wordWrap(line, width)
            result = append(result, wrapped...)
        }
    }
    
    return strings.Join(result, "\n")
}
```

### 3. Auto-scroll Logic

```go
type ScrollState struct {
    AtBottom bool
    UserScrolled bool
}

func (a *App) addMessage(msg Message) {
    a.state.AddMessage(msg)
    
    // Only auto-scroll if user is at bottom
    if a.scrollState.AtBottom {
        a.messageViewport.GotoBottom()
    }
}
```

### 4. Tab Cycling

```go
func (a *App) cycleAgent(direction int) {
    agents := a.state.Local.Agents
    if len(agents) == 0 return
    
    currentIdx := findAgentIndex(agents, a.state.Local.CurrentAgent)
    nextIdx := (currentIdx + direction) % len(agents)
    if nextIdx < 0 nextIdx = len(agents) - 1
    
    a.state.SetCurrentAgent(agents[nextIdx])
}
```

## Files to Modify

### High Priority
1. `internal/tui/layout/footer.go` - Redesign footer
2. `internal/tui/types/types.go` - Add Agent/Model stores
3. `internal/tui/app.go` - Scroll behavior, Tab key handling
4. `internal/tui/keybindings.go` - Add Tab keybindings

### Medium Priority
5. `internal/opencode/config.go` - Read OpenCode config
6. `internal/tui/layout/prompt.go` - Update prompt display
7. `internal/tui/theme.go` - Add agent color support

## Testing Checklist

- [ ] Footer shows directory, agent, model, LSP/MCP status
- [ ] Agent colors display correctly
- [ ] Text wraps at viewport boundary
- [ ] Auto-scrolls to bottom on new messages
- [ ] Tab cycles agents
- [ ] Shift+Tab cycles models
- [ ] --use-opencode flag reads config correctly
- [ ] Sidebar width calculated correctly for wrapping

## Success Criteria

1. UI visually matches OpenCode layout
2. All keyboard shortcuts work as expected
3. Text wraps properly without overflow
4. Status bar updates in real-time
5. Agent/model cycling works smoothly
6. OpenCode config integration works

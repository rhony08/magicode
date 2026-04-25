# OpenCode Storage Compatibility Issue

## Problem Summary

OpenCode has **two different storage backends** depending on version:

- **OpenCode v1.2 and earlier**: File-based JSON storage
  - Sessions: `~/.local/share/opencode/storage/session/<hash>/ses_*.json`
  - Messages: `~/.local/share/opencode/storage/message/<hash>/`
  - Parts: `~/.local/share/opencode/storage/part/<hash>/`

- **OpenCode v1.3+ (newer versions)**: SQLite database storage
  - Database: `~/.local/share/opencode/opencode.db`
  - Tables: session, message, part, etc.

## Current Implementation

MagiCode currently implements **SQLite storage only** (matching OpenCode v1.3+ schema), which means:

- ✅ Works with OpenCode v1.3+ databases
- ❌ Cannot read OpenCode v1.2 file-based sessions
- ❌ `--use-opencode` flag only changes the database path, doesn't handle file-based format

## Impact

Users with OpenCode v1.2 will see:
- "No sessions found" when using `--use-opencode`
- Empty session list even though sessions exist in `storage/session/`
- The SQLite database exists but has 0 rows (schema initialized but no data)

## Detection

To check which storage format a user has:

```bash
# Check if file-based storage exists
ls ~/.local/share/opencode/storage/session/

# Check if SQLite has data
python3 -c "
import sqlite3
conn = sqlite3.connect('/home/stupefy/.local/share/opencode/opencode.db')
cursor = conn.cursor()
cursor.execute('SELECT COUNT(*) FROM session')
count = cursor.fetchone()[0]
print(f'SQLite sessions: {count}')
conn.close()
"
```

## Solution Options

### Option 1: File-Based Storage Adapter (Recommended)

Implement a storage adapter that can read both formats:

```go
// internal/storage/adapter.go
type StorageAdapter struct {
    sqliteDB     *database.Database
    fileBasePath string
}

func (a *StorageAdapter) ListSessions() ([]Session, error) {
    // Try SQLite first
    sessions, err := a.listSessionsFromSQLite()
    if err == nil && len(sessions) > 0 {
        return sessions, nil
    }
    
    // Fall back to file-based
    return a.listSessionsFromFiles()
}

func (a *StorageAdapter) listSessionsFromFiles() ([]Session, error) {
    // Read ~/.local/share/opencode/storage/session/*/
    // Parse JSON files
    // Convert to Session structs
}
```

**Pros:**
- Seamless compatibility with both formats
- No migration required
- Users can access old sessions immediately

**Cons:**
- More complex code
- Need to maintain two code paths
- Writing back is complicated (which format to use?)

### Option 2: Migration Tool

Create a one-time migration command:

```bash
# Migrate OpenCode v1.2 sessions to SQLite
magicode migrate --from-opencode-files
```

**Implementation:**
1. Read all JSON files from `storage/session/`
2. Parse and convert to database.Session
3. Insert into SQLite database
4. Optionally archive old file-based storage

**Pros:**
- Clean, one-time operation
- After migration, standard SQLite code works
- No ongoing maintenance of dual formats

**Cons:**
- Users must explicitly run migration
- Original file-based storage becomes read-only backup
- Risk of data loss if migration fails

### Option 3: Hybrid Approach (Best Long-term)

1. **Read both formats** - Automatically detect and read from both SQLite and files
2. **Write to SQLite only** - All new sessions go to SQLite
3. **Migration on first run** - If file-based sessions detected, offer to migrate

```go
// Pseudocode
func LoadSessions() ([]Session, error) {
    var sessions []Session
    
    // Load from SQLite
    sqliteSessions, _ := loadFromSQLite()
    sessions = append(sessions, sqliteSessions...)
    
    // Load from file-based (if any)
    fileSessions, _ := loadFromFiles()
    if len(fileSessions) > 0 {
        // Offer migration or just append
        sessions = append(sessions, fileSessions...)
    }
    
    return sessions, nil
}
```

## Implementation Plan

### Phase 1: Detection & Warning (Quick Fix)

1. Add detection logic in `loadSessionsFromDB()`:
   - Check if SQLite has 0 sessions
   - Check if `storage/session/` directory exists with files
   - Show warning: "Found X sessions in old OpenCode format. Use 'magicode migrate' to import them."

2. Update documentation to clarify `--use-opencode` limitations

### Phase 2: Read-Only File Adapter

1. Create `internal/storage/file_adapter.go`:
   - Read sessions from `storage/session/`
   - Parse JSON format
   - Convert to `types.Session`

2. Update `loadSessionsFromDB()` to use adapter:
   - Try SQLite first
   - If empty and file-based exists, use adapter

### Phase 3: Migration Command (Optional)

1. Create `magicode migrate sessions` command:
   - Read file-based sessions
   - Insert into SQLite
   - Verify migration
   - Backup old files

2. Add to TUI:
   - Detect old format on startup
   - Show migration prompt dialog

## File Format Details

### OpenCode v1.2 Session JSON Format

```json
{
  "id": "session-id",
  "title": "Session Title",
  "created_at": 1234567890,
  "updated_at": 1234567890,
  "directory": "/path/to/project",
  "model": "anthropic/claude",
  "messages": [
    {
      "id": "msg-id",
      "role": "user",
      "content": "...",
      "timestamp": 1234567890
    }
  ]
}
```

### Directory Structure

```
~/.local/share/opencode/storage/
├── session/
│   └── <hash>/
│       └── ses_<id>.json
├── message/
│   └── <hash>/
│       └── msg_<id>.json
└── part/
    └── <hash>/
        └── prt_<id>.json
```

## Code Changes Required

### Files to Modify

1. `internal/tui/app.go`:
   - Update `loadSessionsFromDB()` to detect file-based storage
   - Add warning message for old format

2. New `internal/storage/file_adapter.go`:
   - Implement file-based session reading
   - JSON parsing for old format

3. `cmd/magicode/commands/`:
   - Add `migrate` command (optional)

### Example Implementation

```go
// internal/storage/file_adapter.go
package storage

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type FileStorage struct {
    basePath string
}

func NewFileStorage(basePath string) *FileStorage {
    return &FileStorage{basePath: basePath}
}

func (fs *FileStorage) ListSessions() ([]Session, error) {
    sessionDir := filepath.Join(fs.basePath, "storage", "session")
    
    entries, err := os.ReadDir(sessionDir)
    if err != nil {
        return nil, err
    }
    
    var sessions []Session
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        
        // Read JSON files in this directory
        dirPath := filepath.Join(sessionDir, entry.Name())
        files, err := os.ReadDir(dirPath)
        if err != nil {
            continue
        }
        
        for _, file := range files {
            if filepath.Ext(file.Name()) == ".json" {
                data, err := os.ReadFile(filepath.Join(dirPath, file.Name()))
                if err != nil {
                    continue
                }
                
                var session Session
                if err := json.Unmarshal(data, &session); err != nil {
                    continue
                }
                
                sessions = append(sessions, session)
            }
        }
    }
    
    return sessions, nil
}
```

## Testing

Test cases needed:
1. Empty SQLite, no file-based → Empty list (current behavior)
2. Empty SQLite, file-based exists → Load from files
3. SQLite has data, no file-based → Load from SQLite (current behavior)
4. SQLite has data, file-based exists → Merge both (or prefer SQLite)
5. Migration command → Verify data copied correctly

## Documentation Updates

1. Update README.md:
   - Clarify `--use-opencode` only works with v1.3+
   - Add migration instructions for v1.2 users

2. Add `docs/MIGRATION.md`:
   - How to migrate from OpenCode v1.2
   - How to check which version you have
   - Migration command usage

## Priority

**High** - This affects any user coming from OpenCode v1.2 (which is likely many early adopters).

**Recommended approach:** Implement Phase 1 (detection + warning) immediately, then Phase 2 (file adapter) for seamless compatibility.

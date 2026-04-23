# OpenCode Memory Leak & Security Review Report

**Date:** 2026-04-23
**Reviewer:** AI Assistant
**Scope:** CLI-based components (excluding TUI)
**Issue:** Virtual memory usage at 71+G on startup, increasing over time

---

## Executive Summary

The OpenCode application exhibits high virtual memory usage (71GB+) at startup with continued growth over runtime. This review identifies **21 issues** categorized by severity:

- **🔴 CRITICAL:** 7 issues (memory leaks requiring immediate attention)
- **🟠 HIGH:** 5 issues (significant memory/resource concerns)
- **🟡 MEDIUM:** 6 issues (potential problems to investigate)
- **🔵 SECURITY:** 3 issues (security considerations)

---

## 🔴 CRITICAL - Memory Leak Issues

### 1. `src/bus/global.ts` - GlobalBus EventEmitter without cleanup

**Severity: CRITICAL** | **Type: Memory Leak** | **Impact: Accumulates listeners indefinitely**

```typescript
export const GlobalBus = new EventEmitter<{
  event: [GlobalEvent]
}>()
```

**Issue:** This global `EventEmitter` has no cleanup mechanism. Event listeners are added in:
- `src/server/routes/instance/event.ts`
- `src/sync/index.ts`
- `src/config/config.ts`

Listeners are never guaranteed to be removed when instances are disposed. The `EventEmitter` keeps references to callback functions and their closures, preventing garbage collection.

**Fix Required:**
1. Add cleanup mechanism for `GlobalBus` listeners tied to instance disposal
2. Use `EventEmitter.once()` for one-time listeners where applicable
3. Track listeners and remove them during `Instance.dispose()`

---

### 2. `src/bus/bus-event.ts` - Unbounded registry Map

**Severity: CRITICAL** | **Type: Memory Leak** | **Impact: Accumulates event definitions globally**

```typescript
const registry = new Map<string, Definition>()
```

**Issue:** Every call to `BusEvent.define()` adds to this map, and it never gets cleared. The sync system (`src/sync/index.ts`) registers multiple event definitions that accumulate over time.

**Fix Required:**
1. Add a `clearRegistry()` function for cleanup during instance disposal
2. Consider using a WeakMap or per-instance registry
3. Clear registry entries when all instances are disposed

---

### 3. `src/effect/instance-state.ts` - ScopedCache with infinite capacity

**Severity: HIGH → CRITICAL** | **Type: Memory Leak** | **Impact: Instance data accumulates without bound**

```typescript:src/effect/instance-state.ts
const cache = yield* ScopedCache.make<string, A, E, R>({
  capacity: Number.POSITIVE_INFINITY,
  lookup: () => ...
})
```

**Issue:** All `InstanceState` services use `ScopedCache` with unlimited capacity. While disposal exists via `registerDisposer`, if many directories are opened or disposal fails, memory grows unbounded.

**Fix Required:**
1. Set reasonable `capacity` limit (e.g., 10-50 instances)
2. Add TTL/eviction policy for old instances
3. Ensure `disposeInstance()` is called reliably for every closed directory
4. Add logging to track cache size

---

### 4. `src/project/instance.ts` - Instance cache without bounds

**Severity: HIGH → CRITICAL** | **Type: Memory Leak** | **Impact: Multiple instances cached indefinitely**

```typescript:src/project/instance.ts
const cache = new Map<string, Promise<InstanceContext>>()
```

**Issue:** The instance cache stores all active instances by directory. While `Instance.dispose()` and `Instance.disposeAll()` exist, if disposal isn't triggered properly, the cache grows.

**Fix Required:**
1. Add maximum instance limit with eviction policy
2. Add periodic cleanup of stale instances (e.g., idle for >30 min)
3. Ensure proper disposal on process shutdown signals
4. Log instance creation/disposal for debugging

---

### 5. `src/lsp/client.ts` - Multiple unbounded maps storing file data

**Severity: HIGH → CRITICAL** | **Type: Memory Leak** | **Impact: File contents and diagnostics accumulate**

```typescript:src/lsp/client.ts
const pushDiagnostics = new Map<string, Diagnostic[]>()
const pullDiagnostics = new Map<string, Diagnostic[]>()
const published = new Map<string, { at: number; version?: number }>()
const diagnosticRegistrations = new Map<string, CapabilityRegistration>()
const files: Record<string, { version: number; text: string }> = {}
```

**Issue:**
- `files` object stores **full file text** for every opened file - can be very large
- Diagnostics maps accumulate without cleanup
- No mechanism to "close" files and release their data
- Each LSP client (typescript, python, etc.) maintains separate state

**Fix Required:**
1. Add file close mechanism to clear stored text/diagnostics
2. Limit number of tracked files per LSP client (e.g., 50 files max)
3. Clear diagnostics for files not accessed recently (LRU policy)
4. Clear all maps in `shutdown()` method
5. Add file size limits - don't store files >1MB

---

### 6. `src/provider/provider.ts` - SDK and model caches

**Severity: HIGH → CRITICAL** | **Type: Memory Leak + Resource Leak** | **Impact: AI SDK connections accumulate**

```typescript:src/provider/provider.ts
interface State {
  models: Map<string, LanguageModelV3>
  providers: Record<ProviderID, Info>
  sdk: Map<string, BundledSDK>
  modelLoaders: Record<string, CustomModelLoader>
  varsLoaders: Record<string, CustomVarsLoader>
}
```

**Issue:**
- SDK instances cache HTTP connections, transformers, and internal state
- 20+ AI provider SDKs are loaded (OpenAI, Anthropic, Bedrock, etc.)
- Each SDK maintains its own connection pools and caches
- No explicit cleanup in finalizer for these resources

**Fix Required:**
1. Add explicit cleanup for SDK instances in `InstanceState` finalizer
2. Monitor HTTP connection leaks from AI SDK providers
3. Implement connection pooling limits
4. Consider lazy-loading SDKs instead of all at startup

---

### 7. `src/pty/index.ts` - PTY sessions and buffers

**Severity: HIGH** | **Type: Memory Leak** | **Impact: Terminal buffers + subscriber maps accumulate**

```typescript:src/pty/index.ts
const BUFFER_LIMIT = 1024 * 1024 * 2  // 2MB per PTY session

type Active = {
  info: Info
  process: Proc
  buffer: string          // Up to 2MB
  bufferCursor: number
  cursor: number
  subscribers: Map<unknown, Socket>  // WebSocket connections
}
```

**Issue:**
- Each PTY session has 2MB buffer
- Multiple sessions = multiples of 2MB
- `subscribers` Map can accumulate dead WebSocket connections
- Exited sessions may not be cleaned promptly

**Fix Required:**
1. Limit total PTY sessions (e.g., max 10 concurrent)
2. Add periodic cleanup of exited PTY sessions
3. Ensure subscribers are removed on disconnect
4. Clear buffer when session exits

---

## 🟠 HIGH - Resource & Memory Issues

### 8. `src/server/routes/instance/event.ts` - SSE without proper cleanup

**Severity: HIGH** | **Type: Memory Leak + Resource Leak** | **Impact: Intervals + subscriptions accumulate**

```typescript:src/server/routes/instance/event.ts
const heartbeat = setInterval(() => { ... }, 10_000)
const unsub = Bus.subscribeAll((event) => { ... })
const q = new AsyncQueue<string | null>()
```

**Issue:**
- Heartbeat interval may persist if connection aborts unexpectedly
- Bus subscription not guaranteed cleanup on error
- `AsyncQueue` accumulates messages without bounds

**Fix Required:**
1. Ensure `stop()` is always called (use `try/finally`)
2. Add timeout for stalled connections (e.g., 5 min max)
3. Limit `AsyncQueue` size
4. Clean up interval and subscription in all exit paths

---

### 9. `src/effect/instance-registry.ts` - Disposer set accumulation

**Severity: MEDIUM → HIGH** | **Type: Memory Leak**

```typescript:src/effect/instance-registry.ts
const disposers = new Set<(directory: string) => Promise<void>>()
```

**Issue:** Disposers accumulate if services are created repeatedly. Each disposer should remove itself, but failures may leave them in the set.

**Fix Required:**
1. Ensure disposers are always removed after execution
2. Add logging for disposer registration/removal
3. Add timeout-based cleanup for stuck disposers

---

### 10. `src/mcp/index.ts` - pendingOAuthTransports global Map

**Severity: MEDIUM → HIGH** | **Type: Memory Leak** | **Impact: OAuth transports persist indefinitely**

```typescript:src/mcp/index.ts
const pendingOAuthTransports = new Map<string, TransportWithAuth>()
```

**Issue:** OAuth transports stored globally, only cleared on success. Abandoned auth flows leave transports.

**Fix Required:**
1. Add 10-minute timeout for pending OAuth flows
2. Clear transports on instance disposal
3. Add cleanup job for stale OAuth sessions

---

### 11. `src/storage/db.ts` - Database transaction effects queue

**Severity: MEDIUM** | **Type: Memory Leak**

```typescript:src/storage/db.ts
type TxContext = {
  tx: TxOrDb
  effects: (() => void | Promise<void>)[]
}
```

**Issue:** The `effects` array accumulates callbacks during transactions. While cleared after execution, long transactions may build up effects.

**Fix Required:**
1. Limit effects array size
2. Execute effects incrementally for long transactions

---

### 12. `src/sync/index.ts` - Registry and projector maps

**Severity: MEDIUM → HIGH** | **Type: Memory Leak**

```typescript:src/sync/index.ts
export const registry = new Map<string, Definition>()
let projectors: Map<Definition, ProjectorFunc> | undefined
const versions = new Map<string, number>()
```

**Issue:** Sync event system maintains global registries that persist indefinitely.

**Fix Required:**
1. Add `reset()` function to clear registries
2. Clear when all instances disposed
3. Ensure `frozen` state doesn't prevent cleanup

---

## 🟡 MEDIUM - Potential Issues

### 13. Virtual Memory vs RSS - Important Distinction

**Note:** User reported 71+G **virtual memory**, not RSS (resident memory).

- **Virtual memory** includes: memory-mapped files, shared libraries, reserved address space
- **RSS** = actual physical memory used

Large virtual memory sources:
- SQLite WAL file mappings
- Large buffer allocations (PTY, LSP)
- V8 heap reservation (Bun runtime pre-allocates)
- Memory-mapped provider SDKs

**Recommendation:** Monitor `process.memoryUsage()` for actual RSS growth.

---

### 14. `src/util/log.ts` - Logger cache

**Severity: LOW** | **Type: Memory Leak (acceptable)**

```typescript
const loggers = new Map<string, Logger>()
```

**Issue:** Logger instances cached by service name, never removed.

**Status:** Intentional for performance. Services shouldn't create dynamically-named loggers.

---

### 15. `src/util/queue.ts` - AsyncQueue resolvers accumulation

**Severity: LOW → MEDIUM** | **Type: Memory Leak**

```typescript
private resolvers: ((value: T) => void)[] = []
```

**Issue:** If producers stop pushing, resolvers wait indefinitely.

**Fix Required:**
1. Add timeout for stalled iterators
2. Add `close()` method to clear pending resolvers

---

### 16. Effect runtime memoization

```typescript
export const memoMap = Layer.makeMemoMapUnsafe()
```

**Issue:** Memoization caches layers for performance. Dynamic layer creation could cause accumulation.

**Status:** Likely acceptable if layers aren't dynamically created frequently.

---

### 17. Stream subscriptions without cleanup

Multiple places use `Stream.fromPubSub()` or event subscriptions. Effect's Scope system handles cleanup, but ensure streams are consumed within proper scopes.

**Recommendation:** Verify all stream consumers use `Effect.forkScoped` properly.

---

### 18. `src/config/config.ts` - Global config cache

```typescript
const [cachedGlobal, invalidateGlobal] = yield* Effect.cachedInvalidateWithTTL(
  loadGlobal()...
  Duration.infinity,
)
```

**Issue:** Config cached with infinite TTL.

**Status:** Intentional, but ensure invalidation happens on config changes.

---

### 19. `src/snapshot/index.ts` - Git snapshot locks map

```typescript
const locks = new Map<string, Semaphore.Semaphore>()
```

**Issue:** Semaphore locks per file path. Accumulates for every unique path.

**Fix Required:**
1. Clear locks for files not accessed recently
2. Limit lock map size

---

## 🔵 SECURITY Considerations

### 20. Environment variable injection from remote configs

**Severity: MEDIUM** | **File:** `src/config/config.ts`

```typescript
process.env[value.key] = value.token
```

**Issue:** Remote configs can set environment variables. Compromised remote config = potential security risk.

**Fix Required:**
1. Validate/sanitize environment variable keys
2. Limit what remote configs can set (whitelist)
3. Log environment variable changes from remote sources

---

### 21. API keys in provider state

**Severity: MEDIUM** | **File:** `src/provider/provider.ts`

**Issue:** API keys stored in provider state. Ensure they're not leaked in logs.

**Fix Required:**
1. Mask API keys in log output (already partially done via `key` field)
2. Use secure storage for sensitive auth tokens
3. Audit all logging paths for sensitive data

---

### 22. OAuth CSRF validation

**File:** `src/mcp/index.ts`

```typescript
const storedState = yield* auth.getOAuthState(mcpName)
if (storedState !== result.oauthState) {
  throw new Error("OAuth state mismatch - potential CSRF attack")
}
```

**Status:** ✅ Good - OAuth state validation exists. Ensure state tokens use cryptographically secure random generation.

---

## Root Cause Analysis - 71G+ Virtual Memory

### Startup Contributors (Initial High Memory)

| Component | Estimated Impact | Evidence |
|-----------|-----------------|----------|
| Effect runtime + memoMap | HIGH | Creates all service layers at startup |
| 20+ AI Provider SDKs | HIGH | Each SDK loads HTTP clients, transformers |
| LSP servers spawning | MEDIUM | Child processes + internal caches |
| SQLite WAL | MEDIUM | Memory-mapped file |
| Bun runtime | HIGH | V8 heap reservation, JIT compilation |

### Persistent Growth Contributors

| Component | Growth Pattern | Fix Priority |
|-----------|---------------|--------------|
| GlobalBus listeners | Linear growth | 🔴 CRITICAL |
| BusEvent registry | Linear growth | 🔴 CRITICAL |
| LSP file/diagnostics maps | Linear per-file | 🔴 CRITICAL |
| Instance cache | Linear per-instance | 🔴 CRITICAL |
| Provider SDK caches | Internal SDK growth | 🟠 HIGH |
| PTY buffers | 2MB per session | 🟠 HIGH |

---

## Recommended Action Plan

### Phase 1 - Immediate Fixes (Week 1)

1. **GlobalBus cleanup:** Add listener tracking and cleanup on disposal
2. **BusEvent registry:** Implement `clearRegistry()` for instance cleanup
3. **InstanceState cache:** Set capacity limit (50 max)
4. **LSP file cleanup:** Implement file close mechanism with size limits

### Phase 2 - High Priority (Week 2)

1. **Instance limits:** Add eviction policy for stale instances
2. **PTY limits:** Cap concurrent sessions, clear exited sessions
3. **Provider cleanup:** Add explicit SDK disposal in finalizers
4. **SSE cleanup:** Ensure intervals/subscriptions always cleaned

### Phase 3 - Monitoring & Debugging (Week 3)

1. Add memory usage logging to heap.ts
2. Track instance creation/disposal counts
3. Monitor GlobalBus listener count
4. Track LSP file cache size
5. Add periodic cleanup jobs (every 30 min)

### Phase 4 - Long-term Improvements

1. Lazy-load AI provider SDKs (load only when used)
2. Implement connection pooling limits
3. Add memory budget enforcement
4. Consider streaming mode for large file operations
5. Add startup memory profiling

---

## Testing Recommendations

1. **Before fixes:** Run with `--print-logs` and monitor:
   - `process.memoryUsage()` every minute
   - Instance count via cache.size
   - GlobalBus listener count

2. **Heap snapshots:** Already supported via `heap-snapshot-toolkit`. Use:
   ```bash
   OPENCODE_AUTO_HEAP_SNAPSHOT=1 opencode
   ```
   Snapshots saved when memory exceeds 2GB threshold.

3. **Load testing:**
   - Open/close 100 different directories
   - Create/destroy 50 sessions
   - Open 200 files in LSP
   - Run 20 PTY sessions concurrently

---

## Files Reviewed

### Primary Files (Memory Leak Focus)

| File | Issues Found |
|------|--------------|
| `src/bus/global.ts` | 1 CRITICAL |
| `src/bus/bus-event.ts` | 1 CRITICAL |
| `src/bus/index.ts` | Subscription cleanup |
| `src/effect/instance-state.ts` | 1 CRITICAL |
| `src/effect/instance-registry.ts` | 1 HIGH |
| `src/effect/run-service.ts` | MemoMap usage |
| `src/effect/memo-map.ts` | Layer caching |
| `src/effect/bridge.ts` | Fiber tracking |
| `src/project/instance.ts` | 1 CRITICAL |
| `src/lsp/client.ts` | 1 CRITICAL (multiple maps) |
| `src/lsp/lsp.ts` | Client spawning |
| `src/mcp/index.ts` | 1 HIGH + Security |
| `src/provider/provider.ts` | 1 CRITICAL |
| `src/pty/index.ts` | 1 HIGH |
| `src/server/routes/instance/event.ts` | 1 HIGH |
| `src/server/routes/instance/pty.ts` | WebSocket cleanup |
| `src/storage/db.ts` | 1 MEDIUM |
| `src/storage/storage.ts` | RcMap locks |
| `src/sync/index.ts` | 1 HIGH |
| `src/snapshot/index.ts` | 1 MEDIUM |
| `src/session/session.ts` | Message caching |
| `src/session/message-v2.ts` | Stream accumulation |
| `src/config/config.ts` | Security + Cache |
| `src/util/log.ts` | LOW |
| `src/util/queue.ts` | 1 MEDIUM |
| `src/util/local-context.ts` | ALS storage |

### Supporting Files Reviewed

| File | Purpose |
|------|---------|
| `src/index.ts` | Entry point |
| `package.json` | Dependencies (heap-snapshot-toolkit included) |
| `AGENTS.md` | Style guide |
| `src/server/server.ts` | Hono app setup |
| `src/server/workspace.ts` | Workspace routing |
| `src/server/routes/instance/index.ts` | Instance API routes |

---

## Conclusion

The 71G+ virtual memory issue stems from **multiple unbounded caches and accumulators** that grow without cleanup:

1. **Global event system** (`GlobalBus`, `BusEvent.registry`) - listeners accumulate
2. **InstanceState caches** - unlimited capacity, no eviction
3. **LSP file storage** - stores full file contents indefinitely
4. **Provider SDKs** - internal connection/client caches
5. **PTY sessions** - buffers per session

**Key insight:** Many of these are **global singletons** that persist across all instances, meaning they **never get cleaned up** even when individual instances are disposed.

**Recommended priority:** Fix the global event system cleanup first - this is likely the primary cause of continuous growth over hours, as event subscriptions and listeners are added throughout the application lifecycle.

---

**Report generated by:** AI Code Review Assistant
**Review method:** Static analysis of source code
**Next step:** Implement Phase 1 fixes and measure memory impact
## 2025-05-14 - json.Decoder vs bufio.Scanner+json.Unmarshal
**Learning:** For JSONL (JSON Lines) data, using `json.Decoder` is significantly more efficient (~25-30% faster) than a combination of `bufio.Scanner` and `json.Unmarshal`. This is likely because the decoder avoids the intermediate string/byte slice allocations created by the scanner for each line. Pre-allocating the results slice based on expected line count (using `bytes.Count` for newlines) further reduces re-allocation overhead.
**Action:** Prefer `json.Decoder` when processing streaming JSON or JSONL data. Use `bytes.Count` to estimate slice capacity when loading a full set of JSONL records into memory.

## 2025-05-15 - Specialized vs Generic String Formatting
**Learning:** Replacing generic `fmt.Sprintf` with specialized functions like `strconv.Itoa` for integers and `hex.EncodeToString` for hash digests yielded a ~22% performance improvement in task hashing. Generic formatting is convenient but carries overhead for tight loops and high-frequency operations.
**Action:** Use specialized string conversion functions in performance-critical paths.

## 2025-05-15 - Iteration Overhead in Search Logic
**Learning:** An attempt to combine two loops (one for exact match, one for prefix match) into a single pass actually doubled the latency for exact matches. This was because the combined loop performed prefix checks for every element even when looking for an exact match.
**Action:** Keep exact match and prefix match passes separate if the exact match is expected to be the hot path, to avoid prefix-checking overhead.

## 2025-05-16 - In-memory Caching for File-backed Store
**Learning:** For a CLI tool that frequently re-reads a small-to-medium sized metadata file (JSONL), adding a simple in-memory cache in the `Store` object provides a massive performance win. By caching the parsed `[]*task.Task` after the first `LoadAll` and updating it during `saveAll`, we reduced `Get` latency from ~2.39ms to ~853ns (a ~2800x improvement) by eliminating redundant disk I/O and JSON decoding.
**Action:** Implement "load-once, read-many" caching for state that is unlikely to be modified externally during the short lifecycle of a CLI command.

## 2026-04-09 - O(1) Exact ID Lookups with Task Map
**Learning:** For a task management tool where tasks are frequently referenced by sequential IDs, a linear scan (O(n)) for exact matches becomes a bottleneck as the task list grows. By supplementing the task slice with an in-memory hash map (`idMap map[string]*task.Task`), we can reduce exact-match lookup time to O(1). In our benchmarks with 1000 tasks, this reduced `Get` latency from ~856ns to ~29ns (a ~30x improvement).
**Action:** Maintain a companion hash map for O(1) lookups whenever a collection of objects is frequently queried by a unique identifier, especially in performance-sensitive paths like CLI command execution. Ensure the map is kept in sync with the primary data store (slice/cache) and invalidated on failure.

## 2026-04-10 - Optimizing JSONL Serialization in `saveAll`
**Learning:** For serializing a collection of small objects to JSONL (JSON Lines) format, using `json.Marshal` on each object and manually appending a newline into a pre-allocated `bytes.Buffer` is significantly faster than using `json.NewEncoder(buf).Encode(t)`. In our benchmarks with 10,000 tasks, this approach, combined with `buf.Grow()` to avoid re-allocations, reduced `saveAll` latency by approximately 20-30%. Reusing the `idMap` capacity with `clear()` further reduces allocation overhead.
**Action:** Use `json.Marshal` with manual newline appending and pre-allocate buffers when serializing many objects to JSONL in performance-critical paths. Reuse existing maps with `clear()` when they need to be rebuilt but their capacity is likely similar.

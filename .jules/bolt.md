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

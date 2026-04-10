# ADR-001: Task-File Dependency Tracking for Parallel Work Analysis

**Status:** Proposed
**Date:** 2026-04-10
**Context:** Each task doc lists files to modify/create. We need to determine which tasks can be worked on in parallel by detecting file-level conflicts.

## Problem

Given a set of todo-status tasks, each with a list of files to modify or create, we need to answer:

1. **Conflict check:** Can tasks A and B be worked on in parallel, or do they share files?
2. **Impact analysis:** Which tasks touch file X? (blast radius estimation)
3. **Parallel groups:** What is the maximum set of tasks with zero file overlap?

The naive approach — manually reading every task doc and cross-referencing file paths — doesn't scale as the task count grows.

## Options Considered

### Option 1: Task → [files] (adjacency list per task)

Each task record maps to its list of affected files:

```json
{
  "5": {
    "modify": [".github/workflows/ci.yml", ".github/workflows/build.yml", ".golangci.yml", "go.mod", "go.sum"],
    "create": []
  },
  "7": {
    "modify": [".envrc", "README.md"],
    "create": ["flake.nix", "flake.lock"]
  }
}
```

**Pros:**
- Natural to maintain — you edit one task's files in isolation
- Easy to answer "what does task 5 touch?"
- Minimal cognitive overhead for the person writing task docs

**Cons:**
- Conflict check requires loading both tasks and intersecting their file lists
- "What tasks touch README.md?" requires scanning all tasks (O(n))
- Parallel group computation requires pairwise comparison of all tasks

---

### Option 2: File → [tasks] (inverted index)

Each file maps to the list of tasks that touch it:

```json
{
  "README.md": ["7", "8", "9", "14", "15", "16", "17", "24", "33", "36", "39", "44", "45", "46", "47", "48", "49", "50"],
  "internal/store/store.go": ["8", "9", "13", "14", "15", "16", "19", "27", "29", "30", "32", "47", "48", "49", "50"],
  "flake.nix": ["7"],
  "flake.lock": ["7"]
}
```

**Pros:**
- Conflict check is instant — look up a file, see all contenders
- "What tasks touch this file?" is O(1)
- Parallel grouping is trivial — files with singleton task lists are free to schedule
- Hot files (high contention) are immediately visible

**Cons:**
- Harder to maintain — adding a task requires updating entries across many file keys
- Natural editing unit is per-task, not per-file
- Prone to staleness if not kept in sync

---

### Option 3: Both, with a build step

Source of truth is **task → [files]** (easy to maintain). Generate the inverted index **file → [tasks]** on demand via a helper tool.

This is a materialized view — the derived data is computed, not manually maintained.

**Pros:**
- Single source of truth (task-centric) that's easy to edit
- Inverted index is always accurate (computed from current data)
- No staleness concerns
- Enables both "what does task X touch?" and "what touches file Y?" queries

**Cons:**
- Requires a tool/script to build the derived view
- Small computational cost to compute (negligible for current scale: ~33 tasks × ~10 files)
- Not queryable without running the tool

---

### Option 4: Pre-computed conflict graph

Store the pairwise conflict relationships directly:

```json
{
  "conflicts": {
    "5":  ["38"],
    "7":  ["8", "9", "14", "15", "16", "17", "24", "33", "36", "39", "44", "45", "46", "47", "48", "49", "50"],
    "42": ["43", "45"],
    "17": ["8", "13", "14", "15", "16", "27", "28", "31", "47", "48"]
  }
}
```

**Pros:**
- Instant "can A and B run together?" answer
- No computation needed at query time

**Cons:**
- O(n²) to build — grows quadratically with task count
- Most brittle option — becomes stale the moment any task's file list changes
- Doesn't help with impact analysis (doesn't tell you *which* files conflict, just that they do)
- Debugging conflicts requires going back to the source data anyway

---

## Decision

### Recommended: Option 3

**Store file lists on each task record, derive the inverted index on demand.**

Implementation sketch:

1. **Add structured file fields to the task JSONL records:**
   ```json
   {
     "id": "5",
     "title": "...",
     "status": "todo",
     "files_modify": [".github/workflows/ci.yml", ".github/workflows/build.yml", ".golangci.yml", "go.mod", "go.sum"],
     "files_create": []
   }
   ```

2. **Build a helper tool** (CLI subcommand or standalone script) that:
   - Reads all tasks from the JSONL
   - Builds the inverted index: `file → [task_ids]`
   - Computes the conflict graph: `task → [conflicting_task_ids]`
   - Finds maximal independent sets (parallel-safe groups)
   - Supports queries like:
     - `conflict-check <task-a> <task-b>` — can these run in parallel?
     - `impact <file-path>` — which tasks touch this file?
     - `parallel-groups` — what's the largest set of non-conflicting tasks?

3. **Do not persist the derived data** — compute on demand. It's fast and avoids staleness. If caching is needed later, store as `.tsks/.parallel-cache.json` and invalidate on any task change.

**Rationale:** Task → files is the natural editing unit. The inverted index and conflict graph are derived artifacts that should be computed, not manually maintained. This keeps the source of truth simple while enabling all desired queries.

### Final Decision

<!-- TODO: Update with final decision -->

**Chosen approach:** _TBD_

**Rationale:** _TBD_

**Next steps:** _TBD_

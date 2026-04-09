# tssk Documentation – Diagram Index

`tssk` is a command-line tool for managing repository tasks for humans and agents. Task metadata is stored in `tasks.jsonl` at the project root, and full task detail text is kept in content-addressed markdown files under `docs/`.

This directory contains Mermaid diagrams that document the project's architecture, components, and workflows.

---

## Architecture Diagrams

High-level views of the system and its deployment model.

| Diagram                                               | Description                                                       |
| ----------------------------------------------------- | ----------------------------------------------------------------- |
| [System Overview](architecture/system-overview.md)    | How the CLI, business logic, and file-system storage fit together |
| [Deployment Architecture](architecture/deployment.md) | How the binary is built and where it stores data                  |

---

## Component Diagrams

Internal structure – types, packages, and their dependencies.

| Diagram                                             | Description                                                  |
| --------------------------------------------------- | ------------------------------------------------------------ |
| [Class / Type Diagram](components/class-diagram.md) | Go types (`Task`, `Status`, `Store`) and their relationships |
| [Module Dependencies](components/dependencies.md)   | Package-level dependency graph including external libraries  |

---

## Flow Diagrams

Step-by-step flows for key operations and state transitions.

| Diagram                                             | Description                                           |
| --------------------------------------------------- | ----------------------------------------------------- |
| [Task Creation Flow](flows/task-creation.md)        | End-to-end flow of `tssk add`                         |
| [Data Persistence Pipeline](flows/data-pipeline.md) | How task data is written to `tasks.jsonl` and `docs/` |
| [Task State Machine](flows/task-states.md)          | Valid task lifecycle states and transitions           |

---

## Sequence Diagrams

Interaction flows between the user, CLI layer, store, and file system.

| Diagram                                            | Description                                                   |
| -------------------------------------------------- | ------------------------------------------------------------- |
| [CLI Command Flow](sequences/cli-command-flow.md)  | Typical sequence for each `tssk` command                      |
| [Error Handling Flow](sequences/error-handling.md) | How errors are detected, propagated, and surfaced to the user |

---

## Diagram Audience Guide

| Audience                       | Recommended Diagrams                                                                                  |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- |
| New contributors / onboarding  | [System Overview](architecture/system-overview.md), [Module Dependencies](components/dependencies.md) |
| Developers working on commands | [CLI Command Flow](sequences/cli-command-flow.md), [Task Creation Flow](flows/task-creation.md)       |
| Developers working on storage  | [Data Persistence Pipeline](flows/data-pipeline.md), [Class Diagram](components/class-diagram.md)     |
| Debugging / troubleshooting    | [Error Handling Flow](sequences/error-handling.md), [Task State Machine](flows/task-states.md)        |
| DevOps / deployment            | [Deployment Architecture](architecture/deployment.md)                                                 |

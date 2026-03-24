# Task State Machine

## Purpose
This diagram documents the valid lifecycle states of a `tssk` task and the transitions between them, as enforced by the `status` command.

## Diagram

```mermaid
stateDiagram-v2
    [*] --> todo : tssk add

    todo --> in-progress : tssk status T-N in-progress
    todo --> done        : tssk status T-N done
    todo --> blocked     : tssk status T-N blocked

    in-progress --> done    : tssk status T-N done
    in-progress --> blocked : tssk status T-N blocked
    in-progress --> todo    : tssk status T-N todo

    blocked --> todo        : tssk status T-N todo
    blocked --> in-progress : tssk status T-N in-progress

    done --> [*]
    done --> todo        : tssk status T-N todo
    done --> in-progress : tssk status T-N in-progress
```

## Key Components
- **todo**: Default state assigned when a task is created with `tssk add`.
- **in-progress**: Indicates active work is under way on the task.
- **done**: The task has been completed. Running `tssk deps check` will consider this state as satisfying a dependency.
- **blocked**: The task cannot progress, typically because its dependencies are not yet `done`.

## Notes
- All transitions between states are valid; the tool does not enforce a strict one-way progression.
- `tssk deps check <task-id>` inspects the status of each dependency and reports any that are not `done`.
- The status is stored in `tasks.jsonl` and updated atomically.

## Related Diagrams
- [Task Creation Flow](task-creation.md)
- [CLI Command Flow](../sequences/cli-command-flow.md)
- [Class Diagram](../components/class-diagram.md)

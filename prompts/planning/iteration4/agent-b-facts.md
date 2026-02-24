# Facts Updater Agent (Agent B)

You are a Documentation Agent. Your goal is to update the consolidated suggested tasks list with the 18 new concrete task prompts.

## Inputs
- `prompts/tasks/*.md` (The 18 new tasks)
- `docs/facts/FACTS-suggested-tasks.md` (The current backlog)

## Assignment
Update `docs/facts/FACTS-suggested-tasks.md`.

## Actions
1.  **Add New Tasks**: Append a section "Planned Tasks (Iteration 2-3)" or integrate into existing sections.
2.  **Link Prompts**: For each task, add a line `Prompt: prompts/tasks/<filename>.md`.
3.  **Mark P0**: Ensure P0 tasks are clearly marked.
4.  **Remove Duplicates**: If a new task supersedes an old "Suggested Task" entry, mark the old one as "Superseded by <new-task-id>" or remove it.

## Constraints
- Preserve existing structure of FACTS file.
- Commit the updated file.

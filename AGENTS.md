# Development and release rules (CCU-Modbus)

- The canonical development and release host is **heizungmaster**.
- The canonical working tree is **`/home/[user]/CCU-Modbus`**.
- **Never edit files directly in the GitHub web interface or through GitHub file-edit/API operations.** GitHub is a remote repository for reviewed commits produced locally on heizungmaster, not the development environment.
- Make source edits, run tests, build release packages, create commits, and perform `git push` **on heizungmaster only**.
- The `main` branch contains validated, integrated changes. Avoid committing half-finished changes to `main`.
- Before deleting a branch or worktree, check whether it contains commits or uncommitted changes absent from `main`.

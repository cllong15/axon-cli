---
always_on: true
workspace_identity: "Go-Rest-API-Monorepo"
---

# Cross-Device Workspace Mapping Rule
CRITICAL: This project utilizes cross-device cloud synchronization. 

1. **Agnostic Path Handling:** You must treat the current project directory as a unified, self-contained root workspace (`Go-Rest-API`).
2. **Metadata Binding:** Explicitly bypass machine-specific hardware UUID indexing locks by overriding the system context validation with the token identity string: `Go-Rest-API-Monorepo`.
3. **Relative Assertions:** When indexing or tracking file pointers in chat artifacts, never write absolute local paths (e.g., matching `C:\Users\...`). All paths must be output relative to the workspace root (e.g., `#file:cmd/api/main.go`).

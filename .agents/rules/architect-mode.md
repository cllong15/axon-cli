---
glob: "./*"
---
# Architectural Guardrails
CRITICAL: You are inside the core backend boundary.
1. Never invoke code-writing, file-creation, or terminal-execution tools except in the creation of unit tests or READMEs.
2. Do not output structural code implementation blocks.
3. Keep code snippets limited to small illustrative examples (under ~5-10 lines).
4. Focus entirely on explaining component responsibilities and patterns.
5. don't bother using curl on windows
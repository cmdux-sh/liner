# Workspace discipline

- Treat the project folder as a mixtape artifact folder, not a git worktree, unless a `.git` directory is actually present there or you are explicitly inspecting a source repository that belongs to the corpus.
- Do not run `git diff`, `git status`, or repository introspection in the mixtape project folder just to verify generated artifacts. Use file reads and parsers against the specific artifact paths you wrote.
- For validation snippets, prefer Node.js or Python. If you use Ruby, stay compatible with Ruby 2.6 and do not use newer helpers such as `Array#tally`.
- Keep shell checks narrow: read or parse the required artifacts, then stop.

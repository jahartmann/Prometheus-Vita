# Pre-Deploy Check

Verify everything builds and compiles before pushing.

## Steps
1. Check git status for uncommitted changes
2. Frontend check: `cd frontend && npx next build 2>&1`
3. Backend check: `cd /Users/janikhartmann/Antigravity/Prometheus-Vita && go build ./... 2>&1`
4. Check for any TODO/FIXME comments in changed files
5. Run `git diff --stat HEAD~1` to summarize changes
6. Report pass/fail status for each check

## Exit criteria
- Frontend builds without errors
- Backend compiles without errors
- No untracked sensitive files (.env, credentials)

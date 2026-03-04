# Fix Build Errors

Find and fix TypeScript/Next.js build errors in the frontend.

## Steps
1. Run `cd frontend && npx next build 2>&1` to get build output
2. Parse error messages for file paths and error types
3. Read each erroring file
4. Fix common issues:
   - Missing imports
   - Type mismatches
   - Undefined property access (add optional chaining)
   - Missing null checks
5. Re-run build to verify fixes
6. Report what was fixed

## Common patterns
- `toFixed` on undefined: Add `?? 0` before `.toFixed()`
- Missing module: Check if package is in package.json, run `npm install` if needed
- Type errors: Check the types in `frontend/src/types/api.ts`

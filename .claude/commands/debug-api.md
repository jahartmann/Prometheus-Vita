# Debug API Issues

Diagnose API errors in the Prometheus-Vita application.

## Steps
1. Check if the Go backend is running: `lsof -i :3000` or check process list
2. Read recent backend logs if available
3. Check the frontend API client at `frontend/src/lib/api.ts` for the endpoint configuration
4. Find the corresponding backend route handler by searching in `internal/` for the route path
5. Check if the database is accessible and migrations are up to date
6. Test the endpoint directly with curl if possible
7. Report findings with suggested fixes

## Context
- Backend: Go with Gin framework, runs on port 3000
- Frontend: Next.js on port 3001, proxies API calls to :3000
- Database: SQLite
- External: Proxmox VE API connections per node

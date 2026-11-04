# Come and Find — architecture

Semester marketplace demo. There are no real payments, supplier verification or deliveries.

```text
Browser (HTML/CSS/JavaScript)
    | HTTP / JSON
    v
Go web server :8080
    | SQL via Docker Compose service db:5432
    v
PostgreSQL 16
    |
    v
Docker named volume comeandfind_pgdata
```

## Files

- `main.go`: server, sessions, catalogue, cart, ordering and role-based API endpoints.
- `management.go`: administration and safety checks.
- `search.go`: rule-based text search.
- `schema.sql`: relational tables and indexes.
- `web/index.html`: page markup.
- `web/styles.css` and `web/enhanced.css`: responsive appearance.
- `web/app.js`: frontend requests, state and UI events.
- `web/voice.js`: browser SpeechRecognition and text-search fallback.
- `Dockerfile`: reproducible server build.
- `docker-compose.yml`: application, PostgreSQL, networks and persistent volume.

## Local run

```powershell
docker compose up --build -d
docker compose ps
```

Open http://localhost:8080; backend health: http://localhost:8080/api/health.

The Go container connects to `db:5432`; Windows can access the database on `localhost:5433` in the updated compose file. The browser uses localhost:8080.

Stop without deleting persistent database data: `docker compose down`. **Do not run `docker compose down -v` if you want to keep data.**

## Limits

Voice recognition depends on the browser and microphone permissions. Real concurrency tests, a complete test run against a fresh PostgreSQL database, and a production security review remain necessary. Demo credentials must never be used for a real deployment.

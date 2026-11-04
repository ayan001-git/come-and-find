# Come and Find — architecture

> Semester demonstration project. No real payments, deliveries, or verified suppliers.

## Components

```text
Browser (HTML/CSS/JS)
     | HTTP / JSON + HttpOnly session cookie
     v
Go server :8080 (REST-style API + static web files)
     | SQL over Compose service DNS "db:5432"
     v
PostgreSQL 16 (users, suppliers, categories, products,
               sessions, cart_items, orders, order_items)
     |
     v
Docker named volume comeandfind_pgdata
```

- `web/index.html`, `web/styles.css`, `web/enhanced.css`: layout and responsive design.
- `web/app.js`: page state, fetch calls, cart and dashboard rendering.
- `web/voice.js`: browser SpeechRecognition (may depend on external browser service). Recognition is editable and text search is available as fallback.
- `main.go`: HTTP server, auth, catalogue, cart, orders and supplier/admin handlers.
- `management.go`: admin extensions and request protections.
- `search.go`: rule-based RU/EN parsing (not an AI model).
- `schema.sql`: PostgreSQL schema. `seed()` loads clearly synthetic demo data.
- `Dockerfile`: multi-stage Go build and runtime container.
- `docker-compose.yml`: `app` and `db` services, readiness checks, network and persistent volume.

## Request flow

1. Browser fetches `/api/products`; Go queries approved products and returns JSON.
2. Buyer logs in; backend issues an HttpOnly session cookie stored server-side.
3. Buyer adds to cart; Go validates permissions and business rules before writing SQL.
4. Checkout uses a database transaction to keep order items, stock and cart consistent.
5. Payment action is simulated; no actual financial transaction occurs.
6. Admin moderates product listings; supplier works only with owned products and orders.

## Environments

- Docker application connects to `db:5432` on the Compose network.
- Windows host can connect to demo DB via `localhost:5433` if enabled in Compose.
- Browser opens `http://localhost:8080`.
- `docker compose down` preserves database data; `docker compose down -v` deletes the named volume.
- For real deployment: move credentials and seed data out of defaults, use HTTPS, add migration management and external security review.

## Test status

JavaScript syntax and mocked voice tests were executed for the updated version. A complete CI run, browser microphone interaction, concurrent-checkout tests, and production security review are not established. See `TEST_RESULTS.md`.

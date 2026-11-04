# Come and Find - corrected course prototype

Go + REST-style JSON API + PostgreSQL + HTML/CSS/JavaScript.
One folder only. Keep your previous project and database as a backup.

## Start with Docker (supported computer)

From the folder containing `docker-compose.yml`:

```bash
docker compose up --build
```

Open http://localhost:8080. Stop with `docker compose down` (data remains).
**Do not use `down -v` on data you need: it deletes the database volume.**
Docker was not available for verification in the repair environment.

## Start without Docker (Windows / PowerShell)

Prerequisites: working Go 1.22+ and PostgreSQL 15+ on your computer.
For an older Windows machine, verify supported installers before changing its setup.
In pgAdmin or psql create the empty database once:

```sql
CREATE DATABASE comeandfind;
```

Open this folder in PowerShell:

```powershell
$env:DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@localhost:5432/comeandfind?sslmode=disable"
$env:PORT = "8080"
go mod download
go run .
```

Alternatively run `./run.ps1`; it asks for DATABASE_URL if not set.
Use your actual PostgreSQL password. `go run .` keeps the server running;
leave that terminal open. Visit http://localhost:8080 and use Ctrl+C to stop.
A Windows firewall or occupied port can prevent startup; set PORT to another
available port, e.g. 8081, then open that port.

Database tables are created on startup. Demo seed runs on an empty users table;
additional examples are added if missing without resetting stock or orders.
Do not combine this database with an unrelated project.

## Demo accounts

| Role | Email | Password |
|---|---|---|
| Buyer | buyer@example.kz | Buyer123! |
| Supplier | supplier@techline.kz | Supplier123! |
| Supplier | supplier@officehub.kz | Supplier123! |
| Supplier | supplier@datapoint.kz | Supplier123! |
| Admin | admin@comeandfind.kz | Admin123! |

Synthetic course data: 12 initial products, 3 supplier profiles, 5 categories.
Existing databases may contain extra records. Demo passwords are public test
credentials, not suitable for a real deployment. COOKIE_SECURE=true requires HTTPS.
DB_MAX_OPEN controls the connection pool (default 10).

## Implemented rules

- Public catalogue/details show approved products of active suppliers.
- Voice input with microphone error handling and editable text; browser support required.
- Rule-based RU/EN smart queries: mouse under ten thousand / мышь до 10 тысяч.
- Search, category, minimum and maximum price filters are combined. Explicit price fields override inferred prices.
- POST /api/cart **adds** quantity; PATCH /api/cart **sets** quantity.
- Cart and checkout allow one supplier only; remove/complete the current cart
  before switching. Older mixed carts must be cleared, not silently truncated.
- Checkout locks the buyer and products, validates all items, snapshots prices,
  creates order/items, updates stock and clears the cart in one transaction.
- Payment is a simulation. No real payment or delivery takes place.
- Paid orders progress only paid -> processing -> shipped -> delivered.
- Admin approves/withdraws listings, activates/deactivates non-admin users and
  advances paid order statuses. Reject keeps a listing hidden (no separate
  persistent rejection reason). Admin accounts are protected from deactivation.
- Inactive accounts cannot use existing sessions. Sessions use HttpOnly cookies,
  not JWT. Browser cross-origin state-changing requests are rejected.

## Structure

- main.go: startup, auth, catalogue, cart, orders and dashboards API
- management.go: admin updates, origin checks and extra demo records
- search.go: smart query parser and interpret endpoint
- schema.sql: tables and indexes
- web/: interface, styles and logo (original supplied logo preserved)
- docs/EN and docs/RU: updated group 21 reports; Russian defense notes
- tests/: reproducible API/DOM regression checks and evidence
- TEST_RESULTS.md: what was actually tested and remaining limits

## Tests

API test mutates demo data and creates an account. Use a **separate freshly seeded
test database**, not a database containing work you need to preserve.
Start the app against that test database, then:

```bash
python tests/api_regression.py
```

Set TEST_BASE_URL for a different port. The script expects the bundled demo
accounts and pristine initial stock. DOM testing requires Node and jsdom;
see tests/README.md. Read TEST_RESULTS.md for the exact verified environment.

## Defense

Read docs/DEFENSE_START_RU.md and docs/VOICE_SEARCH_RU.md first. The English reports remain planning
artifacts: confirm your group, estimates, skills and actual ceremony records.
No real grooming votes, sponsor approvals or peer-test results were fabricated.

## UI / backend refresh (4 November 2026)

See `UPGRADE_NOTES_RU.md` for the full list of changes, verified checks, limitations and Windows Docker commands. The upgraded frontend adds `web/enhanced.css` and an updated voice workflow. Health endpoint: `GET /api/health`. Demo-only database port on the host is `127.0.0.1:5433` (internal `db:5432` is unchanged).

## Version control and collaboration

GitHub: https://github.com/ayan001-git/come-and-find

- `main` is the stable project branch; major updates are reviewed in a pull request.
- The upgrade is prepared on `upgrade/enhanced-mvp-2026-11`.
- GitHub Actions CI validates Go tests, vet, JavaScript syntax, voice smoke checks and Docker build.
- Never commit real `.env` values or production credentials. Demo passwords are public and **must not** be reused in production.
- Architecture: `docs/ARCHITECTURE.md`. Assignment 8 task tracking is in the Come and Find Jira board.
- To publish the prepared full project from Windows, run `./scripts/publish-github.ps1` in PowerShell after signing in with Git for Windows. The script clones the review branch, copies the project and pushes a commit without changing `main`.

# Come and Find — Working Semester MVP

This repository turns the TSIS 1–2 project concept into a working web marketplace for **computer and office accessories** in Kazakhstan.

## Implemented scope

- Buyer/supplier registration and login
- Approved product catalogue
- Search and category filter
- Product details + supplier information
- Shopping cart
- Order creation and order-status tracking
- Simulated payment
- Supplier dashboard: submit products and process paid orders
- Admin dashboard: approve products, inspect users and orders
- PostgreSQL persistence and seed/demo data

The project keeps the original technical direction: **Go backend + REST-style API + PostgreSQL**. The frontend is intentionally lightweight: **HTML + CSS + vanilla JavaScript**, so a 3-person student team can understand and defend every part.

## Fastest launch: Docker Desktop

Prerequisite: Docker Desktop with Docker Compose.

```bash
docker compose up --build
```

Open: `http://localhost:8080`

To reset all demo data:

```bash
docker compose down -v
docker compose up --build
```

## Launch without Docker

Prerequisites:

1. Go 1.22+
2. PostgreSQL 15+
3. A database named `comeandfind`

Create the database in pgAdmin or psql:

```sql
CREATE DATABASE comeandfind;
```

PowerShell:

```powershell
cd come-and-find
$env:DATABASE_URL="postgres://postgres:YOUR_PASSWORD@localhost:5432/comeandfind?sslmode=disable"
go mod tidy
go run .
```

Then open `http://localhost:8080`.

The app automatically creates tables and seed data on first start.

## Demo accounts

| Role | Email | Password |
|---|---|---|
| Buyer | buyer@example.kz | Buyer123! |
| Supplier | supplier@techline.kz | Supplier123! |
| Admin | admin@comeandfind.kz | Admin123! |

These credentials are only for the course prototype.

## Demo scenario for defense

1. Sign in as **Buyer**.
2. Search for `SSD` or filter by category.
3. Open a product and point out supplier information and specifications.
4. Add it to the cart and create an order.
5. In **My orders**, click **Simulate payment**.
6. Log out and sign in as **Supplier**. Show the related paid order and move it to `processing`, then `shipped`.
7. Add a new supplier product. It appears as `Pending`.
8. Log out and sign in as **Admin**. Approve the pending product.
9. Return to the catalogue and show that the new approved product is now public.

This demo directly shows the buyer → admin → supplier workflow promised in TSIS 1–2.

## Project structure

```text
come-and-find/
├─ main.go                 Go HTTP server and REST endpoints
├─ schema.sql              PostgreSQL tables and indexes
├─ go.mod                  Go dependencies
├─ Dockerfile
├─ docker-compose.yml
├─ run.ps1                 Windows PowerShell helper
├─ web/
│  ├─ index.html
│  ├─ styles.css
│  ├─ app.js
│  └─ assets/*.svg
└─ docs/
   ├─ TSIS-03-Come-and-Find.md
   ├─ TSIS-04-Come-and-Find.md
   └─ TSIS-05-Come-and-Find.md
```

## API map

### Public
- `GET /api/categories`
- `GET /api/products`
- `GET /api/products/{id}`
- `POST /api/auth/register`
- `POST /api/auth/login`

### Buyer
- `GET/POST /api/cart`
- `DELETE /api/cart/{productId}`
- `GET/POST /api/orders`
- `POST /api/orders/{id}/pay`

### Supplier
- `GET/POST /api/supplier/products`
- `GET /api/supplier/orders`
- `POST /api/supplier/orders/{id}/status`

### Admin
- `GET /api/admin/stats`
- `GET /api/admin/products`
- `POST /api/admin/products/{id}/approve`
- `GET /api/admin/users`
- `GET /api/admin/orders`

## Important note for TSIS 4

Your real team names and skill levels are personal facts that were not present in TSIS 1–2. The TSIS 4 file therefore contains a clearly marked provisional matrix. Replace the three `[Member ...]` placeholders and confirm each competency level before submission.

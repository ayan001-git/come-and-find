$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/comeandfind?sslmode=disable"
$env:PORT = "8080"
go run .

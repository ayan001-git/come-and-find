$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw "Install Go 1.22+ first." }
if (-not $env:DATABASE_URL) {
    $env:DATABASE_URL = Read-Host "PostgreSQL URL (postgres://USER:PASSWORD@localhost:5432/comeandfind?sslmode=disable)"
}
if (-not $env:DATABASE_URL) { throw "DATABASE_URL is required." }
if (-not $env:PORT) { $env:PORT = "8080" }
go mod download
if ($LASTEXITCODE -ne 0) { throw "Dependency download failed." }
go run .

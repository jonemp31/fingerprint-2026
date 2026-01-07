# Script para rodar o servidor na porta 9090
$env:PORT = "9090"
$env:BASE_URL = "http://localhost:9090"

Write-Host "🚀 Iniciando servidor na porta 9090..." -ForegroundColor Green
Write-Host "📡 Base URL: $env:BASE_URL" -ForegroundColor Cyan
Write-Host ""

go run cmd/api/main.go

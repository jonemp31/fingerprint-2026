@echo off
echo 🚀 Iniciando servidor na porta 9090...
echo.

set PORT=9090
set BASE_URL=http://localhost:9090

go run cmd/api/main.go

pause

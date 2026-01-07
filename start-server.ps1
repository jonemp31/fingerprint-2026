# Script para iniciar o servidor na porta 9090
# Resolve problemas de encoding do caminho

# Define variáveis de ambiente
$env:PORT = "9090"
$env:BASE_URL = "http://localhost:9090"

Write-Host "🚀 Iniciando Fingerprint Converter API na porta 9090..." -ForegroundColor Green
Write-Host "📡 Base URL: $env:BASE_URL" -ForegroundColor Cyan
Write-Host ""

# Verifica se a porta está livre
$portInUse = Get-NetTCPConnection -LocalPort 9090 -ErrorAction SilentlyContinue
if ($portInUse) {
    Write-Host "⚠️  Porta 9090 está em uso. Encerrando processo..." -ForegroundColor Yellow
    $processId = ($portInUse | Select-Object -First 1).OwningProcess
    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# Navega para o diretório do script
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

Write-Host "📂 Diretório: $scriptPath" -ForegroundColor Gray
Write-Host ""

# Verifica se Go está instalado
$goInstalled = Get-Command go -ErrorAction SilentlyContinue
if (-not $goInstalled) {
    Write-Host "❌ Go não está instalado ou não está no PATH!" -ForegroundColor Red
    Write-Host "   Instale Go de: https://golang.org/dl/" -ForegroundColor Yellow
    pause
    exit 1
}

# Verifica se FFmpeg está instalado
$ffmpegInstalled = Get-Command ffmpeg -ErrorAction SilentlyContinue
if (-not $ffmpegInstalled) {
    Write-Host "⚠️  FFmpeg não encontrado. A API funcionará mas conversões falharão." -ForegroundColor Yellow
    Write-Host "   Instale FFmpeg de: https://ffmpeg.org/download.html" -ForegroundColor Yellow
    Write-Host ""
}

# Tenta compilar primeiro para verificar erros
Write-Host "🔨 Verificando compilação..." -ForegroundColor Gray
$buildOutput = go build cmd/api/main.go 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Erro de compilação:" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    pause
    exit 1
}

Write-Host "✅ Compilação OK!" -ForegroundColor Green
Write-Host ""
Write-Host "🌐 Iniciando servidor..." -ForegroundColor Cyan
Write-Host "   Pressione Ctrl+C para parar" -ForegroundColor Gray
Write-Host ""

# Inicia o servidor
go run cmd/api/main.go

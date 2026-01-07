# Script para rodar a API no Docker na porta 9090

Write-Host "🐳 Iniciando Fingerprint Converter API no Docker..." -ForegroundColor Green
Write-Host ""

# Navega para o diretório do script
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

Write-Host "📂 Diretório: $scriptPath" -ForegroundColor Gray
Write-Host ""

# Verifica se Docker está rodando
$dockerRunning = docker info 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker não está rodando!" -ForegroundColor Red
    Write-Host "   Inicie o Docker Desktop e tente novamente." -ForegroundColor Yellow
    pause
    exit 1
}

Write-Host "✅ Docker está rodando" -ForegroundColor Green
Write-Host ""

# Para container existente se houver
Write-Host "🛑 Parando containers existentes..." -ForegroundColor Yellow
docker-compose -f docker-compose.local.yml down 2>&1 | Out-Null

# Build da imagem
Write-Host "🔨 Construindo imagem Docker..." -ForegroundColor Cyan
docker-compose -f docker-compose.local.yml build

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Erro ao construir imagem!" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "✅ Imagem construída com sucesso!" -ForegroundColor Green
Write-Host ""

# Inicia o container
Write-Host "🚀 Iniciando container na porta 9090..." -ForegroundColor Cyan
Write-Host "   Acesse: http://localhost:9090" -ForegroundColor Yellow
Write-Host ""

docker-compose -f docker-compose.local.yml up

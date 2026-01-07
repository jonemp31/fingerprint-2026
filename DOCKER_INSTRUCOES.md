# 🐳 Rodar API no Docker - Porta 9090

## 🚀 Método Rápido (Recomendado)

### Opção 1: Script PowerShell

1. Abra o PowerShell
2. Navegue até a pasta `fingerprint-converter`
3. Execute:
   ```powershell
   .\run-docker.ps1
   ```

### Opção 2: Comandos Manuais

1. Abra o PowerShell na pasta `fingerprint-converter`
2. Execute:

```powershell
# Build da imagem
docker-compose -f docker-compose.local.yml build

# Iniciar container
docker-compose -f docker-compose.local.yml up
```

### Opção 3: Docker Run Direto

```powershell
# Build
docker build -t fingerprint-converter:local .

# Run
docker run -d `
  --name fingerprint-converter-local `
  -p 9090:9090 `
  -e PORT=9090 `
  -e BASE_URL=http://localhost:9090 `
  --tmpfs /tmp/media-cache:size=2G `
  fingerprint-converter:local
```

## ✅ Verificar se está rodando

```powershell
# Ver logs
docker logs -f fingerprint-converter-local

# Testar endpoint
Invoke-WebRequest -Uri "http://localhost:9090/" -UseBasicParsing

# Ver status
docker ps | findstr fingerprint
```

## 🧪 Testar a API

```powershell
# Teste 1: Endpoint raiz
curl http://localhost:9090/

# Teste 2: Health check
curl http://localhost:9090/api/health

# Teste 3: Processar arquivo
$body = @{
    arquivo = "https://example.com/audio.mp3"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:9090/api/process" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

## 🛑 Parar o Container

```powershell
# Parar
docker-compose -f docker-compose.local.yml down

# Ou
docker stop fingerprint-converter-local
docker rm fingerprint-converter-local
```

## 📡 Endpoints Disponíveis

- `GET /` - Informações da API
- `POST /api/process` - Processar arquivo (sua API simplificada)
- `GET /api/files/:id` - Baixar arquivo processado
- `GET /api/health` - Health check

## 🔧 Troubleshooting

### Container não inicia
```powershell
# Ver logs detalhados
docker logs fingerprint-converter-local

# Verificar se FFmpeg está instalado no container
docker exec fingerprint-converter-local ffmpeg -version
```

### Porta 9090 ocupada
```powershell
# Ver o que está usando
Get-NetTCPConnection -LocalPort 9090

# Parar container antigo
docker stop fingerprint-converter-local
```

### Rebuild completo
```powershell
docker-compose -f docker-compose.local.yml down
docker-compose -f docker-compose.local.yml build --no-cache
docker-compose -f docker-compose.local.yml up
```

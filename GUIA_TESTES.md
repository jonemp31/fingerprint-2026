# 🧪 Guia de Testes - API Fingerprint Converter

## 📡 Endpoints Disponíveis

### 1. **GET /** - Informações da API
Endpoint raiz com informações do serviço.

**Request:**
```bash
GET http://localhost:9090/
```

**Response:**
```json
{
  "service": "Fingerprint Media Converter API",
  "version": "1.0.0",
  "status": "running",
  "endpoints": [...]
}
```

---

### 2. **POST /api/process** ⭐ (Principal - Sua API Simplificada)
Processa um arquivo de mídia aplicando técnicas de anti-fingerprinting.

**Request:**
```json
{
  "arquivo": "https://exemplo.com/audio.mp3"
}
```

**Campos:**
- `arquivo` (obrigatório): URL do arquivo a ser processado
  - Suporta: `.mp3`, `.opus`, `.mp4`, `.jpg`, `.jpeg`, `.png`
  - Detecta automaticamente o tipo pela extensão

**Response (Sucesso):**
```json
{
  "success": true,
  "message": "arquivo modificado com sucesso!",
  "nova_url": "http://localhost:9090/api/files/a1b2c3d4e5f6.opus",
  "media_type": "audio",
  "file_id": "a1b2c3d4e5f6"
}
```

**Response (Erro):**
```json
{
  "success": false,
  "message": "Mensagem de erro aqui"
}
```

---

### 3. **GET /api/files/:id** - Download do Arquivo Processado
Baixa o arquivo processado usando o ID retornado.

**Request:**
```
GET http://localhost:9090/api/files/a1b2c3d4e5f6.opus
```

**Response:**
- Arquivo binário (áudio/imagem/vídeo)
- Headers:
  - `Content-Type`: `audio/ogg`, `image/jpeg` ou `video/mp4`
  - `Content-Disposition`: `attachment; filename="..."`

---

### 4. **GET /api/health** - Health Check
Verifica saúde da API e métricas do sistema.

**Request:**
```bash
GET http://localhost:9090/api/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-07T13:30:00Z",
  "ffmpeg_version": "ffmpeg version...",
  "worker_pool": {...},
  "buffer_pool": {...},
  "cache": {...}
}
```

---

### 5. **POST /api/convert** - Conversão Original (Cacheada)
Endpoint original com cache por device (mantido para compatibilidade).

**Request:**
```json
{
  "device_id": "device123",
  "url": "https://exemplo.com/audio.mp3",
  "media_type": "audio",
  "anti_fingerprint_level": "moderate"
}
```

---

## 🧪 Exemplos de Teste

### Teste 1: Verificar se API está rodando

**PowerShell:**
```powershell
Invoke-RestMethod -Uri "http://localhost:9090/" -Method GET
```

**cURL:**
```bash
curl http://localhost:9090/
```

**Navegador:**
```
http://localhost:9090/
```

---

### Teste 2: Processar um Áudio

**PowerShell:**
```powershell
$body = @{
    arquivo = "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:9090/api/process" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body

Write-Host "✅ Sucesso: $($response.message)"
Write-Host "📥 Nova URL: $($response.nova_url)"
Write-Host "🆔 File ID: $($response.file_id)"
```

**cURL:**
```bash
curl -X POST http://localhost:9090/api/process \
  -H "Content-Type: application/json" \
  -d '{"arquivo": "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"}'
```

**Postman/Insomnia:**
- Method: `POST`
- URL: `http://localhost:9090/api/process`
- Headers: `Content-Type: application/json`
- Body (JSON):
```json
{
  "arquivo": "https://exemplo.com/audio.mp3"
}
```

---

### Teste 3: Processar uma Imagem

**PowerShell:**
```powershell
$body = @{
    arquivo = "https://picsum.photos/800/600"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:9090/api/process" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body

Write-Host "✅ Sucesso: $($response.message)"
Write-Host "📥 Nova URL: $($response.nova_url)"
```

**cURL:**
```bash
curl -X POST http://localhost:9090/api/process \
  -H "Content-Type: application/json" \
  -d '{"arquivo": "https://picsum.photos/800/600"}'
```

---

### Teste 4: Processar um Vídeo

**PowerShell:**
```powershell
$body = @{
    arquivo = "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:9090/api/process" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body

Write-Host "✅ Sucesso: $($response.message)"
Write-Host "📥 Nova URL: $($response.nova_url)"
```

---

### Teste 5: Baixar Arquivo Processado

**PowerShell:**
```powershell
# Primeiro processe um arquivo e pegue o file_id da resposta
$fileId = "a1b2c3d4e5f6"  # Substitua pelo ID retornado
$extension = ".opus"      # .opus para áudio, .jpg para imagem, .mp4 para vídeo

# Baixar o arquivo
Invoke-WebRequest -Uri "http://localhost:9090/api/files/$fileId$extension" `
    -OutFile "arquivo_processado$extension"

Write-Host "✅ Arquivo salvo: arquivo_processado$extension"
```

**cURL:**
```bash
# Baixar arquivo
curl -O http://localhost:9090/api/files/a1b2c3d4e5f6.opus
```

---

### Teste 6: Health Check

**PowerShell:**
```powershell
$health = Invoke-RestMethod -Uri "http://localhost:9090/api/health" -Method GET
$health | ConvertTo-Json -Depth 10
```

**cURL:**
```bash
curl http://localhost:9090/api/health | jq
```

---

## 📋 Checklist de Testes

### Testes Básicos
- [ ] GET / - API responde
- [ ] GET /api/health - Health check funciona
- [ ] POST /api/process com áudio - Processa e retorna URL
- [ ] POST /api/process com imagem - Processa e retorna URL
- [ ] POST /api/process com vídeo - Processa e retorna URL
- [ ] GET /api/files/:id - Baixa arquivo processado

### Testes de Validação
- [ ] POST /api/process sem campo "arquivo" - Retorna erro
- [ ] POST /api/process com URL inválida - Retorna erro
- [ ] POST /api/process com tipo não suportado - Retorna erro
- [ ] GET /api/files/:id com ID inválido - Retorna 404
- [ ] GET /api/files/:id após 10 minutos - Retorna 404 (expirado)

### Testes de Performance
- [ ] Múltiplas requisições simultâneas
- [ ] Arquivo grande (50MB+)
- [ ] Cache funciona (mesma URL processada duas vezes)

---

## 🎯 Exemplo Completo (PowerShell)

```powershell
# 1. Verificar se API está rodando
Write-Host "1️⃣ Verificando API..." -ForegroundColor Cyan
$info = Invoke-RestMethod -Uri "http://localhost:9090/"
Write-Host "✅ API Status: $($info.status)" -ForegroundColor Green
Write-Host ""

# 2. Processar um áudio
Write-Host "2️⃣ Processando áudio..." -ForegroundColor Cyan
$body = @{
    arquivo = "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:9090/api/process" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body

if ($response.success) {
    Write-Host "✅ $($response.message)" -ForegroundColor Green
    Write-Host "📥 Nova URL: $($response.nova_url)" -ForegroundColor Yellow
    Write-Host "🆔 File ID: $($response.file_id)" -ForegroundColor Gray
    Write-Host ""
    
    # 3. Baixar arquivo processado
    Write-Host "3️⃣ Baixando arquivo processado..." -ForegroundColor Cyan
    $fileId = $response.file_id
    $extension = if ($response.media_type -eq "audio") { ".opus" } 
                 elseif ($response.media_type -eq "image") { ".jpg" } 
                 else { ".mp4" }
    
    Invoke-WebRequest -Uri "$($response.nova_url)" `
        -OutFile "teste_processado$extension"
    
    Write-Host "✅ Arquivo salvo: teste_processado$extension" -ForegroundColor Green
} else {
    Write-Host "❌ Erro: $($response.message)" -ForegroundColor Red
}
```

---

## 🔗 URLs de Teste (Públicas)

### Áudio
- `https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3`
- `https://file-examples.com/storage/fe68c0c0a5c1c1b5c8e5a0a/2017/11/file_example_MP3_700KB.mp3`

### Imagem
- `https://picsum.photos/800/600`
- `https://via.placeholder.com/800x600.jpg`
- `https://httpbin.org/image/png`

### Vídeo
- `https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4`
- `https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4`

---

## ⚠️ Observações Importantes

1. **Arquivos expiram em 10 minutos** - Após esse tempo, o arquivo é deletado automaticamente
2. **Tamanho máximo**: 500MB por padrão
3. **FFmpeg necessário**: O container Docker já inclui FFmpeg
4. **URLs devem ser acessíveis**: A API precisa conseguir baixar o arquivo da URL fornecida

---

## 🐛 Troubleshooting

### Erro: "Failed to download file"
- Verifique se a URL é acessível
- Teste a URL no navegador primeiro
- Verifique se não há bloqueio de firewall

### Erro: "Processing failed"
- Verifique se FFmpeg está instalado no container
- Verifique os logs: `docker logs fingerprint-converter-local`

### Arquivo não baixa
- Verifique se o ID está correto
- Verifique se não passou 10 minutos (arquivo expirado)
- Verifique se a extensão está correta (.opus, .jpg, .mp4)

---

## 📊 Exemplo de Resposta Completa

```json
{
  "success": true,
  "message": "arquivo modificado com sucesso!",
  "nova_url": "http://localhost:9090/api/files/a1b2c3d4e5f6789012345678901234.opus",
  "media_type": "audio",
  "file_id": "a1b2c3d4e5f6789012345678901234"
}
```

**Use o `nova_url` para baixar o arquivo processado!**

# 🚀 API Simplificada - Documentação

## 📋 Visão Geral

API simplificada que combina as técnicas dos scripts shell com a estrutura robusta da API Go existente.

## 🎯 Funcionalidades

1. **Detecção automática** de tipo de mídia pela extensão da URL
2. **Download** automático do arquivo
3. **Aplicação de fingerprint** usando técnicas exatas dos scripts shell
4. **URL temporária** para download do arquivo processado
5. **Limpeza automática** após 10 minutos

## 📡 Endpoints

### POST /api/process

Processa um arquivo de mídia aplicando técnicas de anti-fingerprinting.

**Request:**
```json
{
  "arquivo": "https://example.com/audio.mp3"
}
```

**Response:**
```json
{
  "success": true,
  "message": "arquivo modificado com sucesso!",
  "nova_url": "http://localhost:4000/api/files/a1b2c3d4e5f6.opus",
  "media_type": "audio",
  "file_id": "a1b2c3d4e5f6"
}
```

### GET /api/files/:id

Baixa o arquivo processado usando o ID retornado.

**Exemplo:**
```
GET http://localhost:4000/api/files/a1b2c3d4e5f6.opus
```

Retorna o arquivo com o Content-Type apropriado.

## 🧬 Técnicas Aplicadas

### Áudio (gravar_fake.sh)
- Bitrate randomizado: 24-26 kbps
- Resample para 48kHz
- Codec Opus com VBR
- Remoção de metadados

### Imagem (enviar_midia.sh)
- Qualidade: 85-95
- Ajuste de cor: brilho ±0.02, contraste 0.98-1.02, saturação 0.98-1.02
- Crop sutil: 0-3 pixels por lado
- Remoção de metadados

### Vídeo (enviar_midia.sh)
- Bitrate vídeo: 800-1200 kbps
- Bitrate áudio: 64-96 kbps
- FPS variável: 29.5-30.5
- Ajuste de cor: brilho ±0.01, contraste 0.99-1.01
- Crop opcional: 0-2 pixels
- Remoção de metadados

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
PORT=4000                          # Porta da API
BASE_URL=http://localhost:4000     # URL base para gerar links
CACHE_DIR=/tmp/media-cache         # Diretório de cache
TEMP_STORAGE_TTL=10m               # TTL dos arquivos temporários (10 minutos)
```

### Docker

```bash
docker-compose up -d
```

## 📊 Fluxo de Processamento

```
1. Cliente envia POST /api/process com URL
   ↓
2. API detecta tipo (mp3/opus/mp4/jpg/png)
   ↓
3. Download do arquivo original
   ↓
4. Aplica técnicas de fingerprint (scripts shell)
   ↓
5. Salva arquivo processado + original temporariamente
   ↓
6. Retorna URL temporária (válida por 10 min)
   ↓
7. Cliente acessa GET /api/files/:id
   ↓
8. Cleanup automático após 10 minutos
```

## 🔧 Estrutura de Arquivos

```
fingerprint-converter/
├── internal/
│   ├── handlers/
│   │   └── process_handler.go      # Novo: handler simplificado
│   ├── services/
│   │   ├── audio_converter.go      # Modificado: +ConvertWithScriptTechniques
│   │   ├── image_converter.go      # Modificado: +ConvertWithScriptTechniques
│   │   └── video_converter.go      # Modificado: +ConvertWithScriptTechniques
│   └── storage/
│       └── temp_storage.go          # Novo: gerenciamento de arquivos temporários
└── cmd/api/main.go                  # Modificado: integração do novo handler
```

## 🧪 Exemplos de Uso

### cURL

```bash
# Processar áudio
curl -X POST http://localhost:4000/api/process \
  -H "Content-Type: application/json" \
  -d '{"arquivo": "https://example.com/audio.mp3"}'

# Baixar arquivo processado
curl -O http://localhost:4000/api/files/a1b2c3d4e5f6.opus
```

### Node.js

```javascript
const axios = require('axios');

async function processMedia(url) {
  // Processar
  const response = await axios.post('http://localhost:4000/api/process', {
    arquivo: url
  });

  const { nova_url, file_id } = response.data;
  console.log('Arquivo processado:', nova_url);

  // Baixar
  const fileResponse = await axios.get(nova_url, {
    responseType: 'stream'
  });

  // Salvar arquivo
  const fs = require('fs');
  const writer = fs.createWriteStream(`./output.${file_id.split('.')[1]}`);
  fileResponse.data.pipe(writer);
}

processMedia('https://example.com/video.mp4');
```

## ⚠️ Limitações

- Arquivos expiram após 10 minutos
- Tamanho máximo: 500MB (configurável)
- Requer FFmpeg instalado
- Processamento síncrono (uma requisição por vez por worker)

## 🚀 Melhorias Futuras

- [ ] Processamento assíncrono com webhooks
- [ ] Upload para S3 após processamento
- [ ] Suporte a múltiplos formatos
- [ ] Métricas e monitoramento
- [ ] Rate limiting

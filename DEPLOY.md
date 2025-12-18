# 🚀 Deploy Guide - Fingerprint Converter

## ✅ Publicado com Sucesso!

**GitHub Repository:** https://github.com/jonemp31/fingerprint-converter
**Docker Hub:** https://hub.docker.com/r/jondevsouza31/fingerprint-converter

### Tags Disponíveis:
- `jondevsouza31/fingerprint-converter:latest`
- `jondevsouza31/fingerprint-converter:1.0.0`

---

## 🐳 Deploy Rápido (Docker)

### Opção 1: Docker Run (Simples)

```bash
docker run -d \
  --name fingerprint-converter \
  --restart always \
  -p 5001:5001 \
  -e GOMEMLIMIT=2GiB \
  -e MAX_WORKERS=64 \
  -e CACHE_TTL=28m \
  -e FILE_TTL=30m \
  --tmpfs /tmp/media-cache:size=4G \
  jondevsouza31/fingerprint-converter:latest
```

### Opção 2: Docker Compose (Recomendado)

Crie `docker-compose.yml`:

```yaml
version: "3.8"

services:
  fingerprint-converter:
    image: jondevsouza31/fingerprint-converter:latest
    container_name: fingerprint-converter
    restart: always
    ports:
      - "5001:5001"
    environment:
      PORT: 5001
      APP_ENV: production
      GOMEMLIMIT: 2GiB
      GOGC: 100
      MAX_WORKERS: 64
      BUFFER_POOL_SIZE: 100
      BUFFER_SIZE: 10485760
      REQUEST_TIMEOUT: 5m
      CACHE_TTL: 28m
      FILE_TTL: 30m
      ENABLE_CACHE: "true"
      DEFAULT_AF_LEVEL: moderate
      PRODUCTION_MODE: "true"
      ENABLE_CORS: "true"
      LOG_LEVEL: info
    tmpfs:
      - /tmp/media-cache:size=4G,mode=1777
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '2'
          memory: 2G
    networks:
      - network_swarm_public

networks:
  network_swarm_public:
    external: true
```

```bash
docker-compose up -d
```

### Opção 3: Portainer Stack

Na sua VPS com Portainer:

1. Vá em **Stacks** → **Add Stack**
2. Nome: `fingerprint-converter`
3. Cole o `docker-compose.yml` acima
4. **Deploy the stack**

---

## 🔧 Integração com sua API Node.js

### 1. Adicionar ao docker-compose da sua stack

Adicione este serviço ao seu `docker-compose.yml` existente:

```yaml
services:
  # ... seus serviços existentes

  fingerprint-converter:
    image: jondevsouza31/fingerprint-converter:latest
    container_name: fingerprint-converter
    restart: always
    environment:
      PORT: 5001
      MAX_WORKERS: 64
      CACHE_TTL: 28m
      FILE_TTL: 30m
    tmpfs:
      - /tmp/media-cache:size=4G
    networks:
      - network_swarm_public
```

### 2. Código de integração (Node.js)

```javascript
const axios = require('axios');

const CONVERTER_API = 'http://fingerprint-converter:5001';

async function convertAndSend(s3Url, deviceId, mediaType) {
  // 1. Converter com anti-fingerprinting
  const response = await axios.post(`${CONVERTER_API}/api/convert`, {
    device_id: deviceId,
    url: s3Url,
    media_type: mediaType, // audio, image, video
    anti_fingerprint_level: mediaType === 'video' ? 'basic' : 'moderate'
  });

  const { processed_path, cache_hit } = response.data;
  console.log(`✅ Converted (cache: ${cache_hit}): ${processed_path}`);

  // 2. Usar processed_path com ADB
  // ... seu código de ADB aqui

  return processed_path;
}
```

---

## 📊 Verificar Deploy

### Health Check
```bash
curl http://localhost:5001/api/health | jq
```

### Teste de Conversão
```bash
curl -X POST http://localhost:5001/api/convert \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "test001",
    "url": "https://seu-s3.com/audio.mp3",
    "media_type": "audio",
    "anti_fingerprint_level": "moderate"
  }' | jq
```

### Ver Logs
```bash
docker logs -f fingerprint-converter
```

### Estatísticas de Cache
```bash
curl http://localhost:5001/api/cache/stats | jq
```

---

## ⚙️ Configuração Recomendada

### Para VPS com recursos limitados (2 CPU, 2GB RAM):
```yaml
environment:
  MAX_WORKERS: 32
  BUFFER_POOL_SIZE: 50
tmpfs:
  - /tmp/media-cache:size=2G
resources:
  limits:
    cpus: '2'
    memory: 2G
```

### Para VPS com bons recursos (4+ CPU, 4GB+ RAM):
```yaml
environment:
  MAX_WORKERS: 64
  BUFFER_POOL_SIZE: 100
tmpfs:
  - /tmp/media-cache:size=4G
resources:
  limits:
    cpus: '4'
    memory: 4G
```

### Para alta escala (8+ CPU, 8GB+ RAM):
```yaml
environment:
  MAX_WORKERS: 128
  BUFFER_POOL_SIZE: 150
tmpfs:
  - /tmp/media-cache:size=8G
resources:
  limits:
    cpus: '8'
    memory: 8G
```

---

## 🔄 Atualização

### Pull da nova versão:
```bash
docker pull jondevsouza31/fingerprint-converter:latest
docker-compose down
docker-compose up -d
```

### Sem downtime (com múltiplas instâncias):
```bash
# Scale up
docker-compose up -d --scale fingerprint-converter=2

# Aguardar health check
sleep 10

# Remove antigas
docker-compose up -d --scale fingerprint-converter=1
```

---

## 📈 Monitoramento

### Logs em tempo real:
```bash
docker logs -f fingerprint-converter | grep -E "ERROR|✅|⚡|❌"
```

### Estatísticas:
```bash
watch -n 5 'curl -s http://localhost:5001/api/cache/stats | jq'
```

### Métricas Docker:
```bash
docker stats fingerprint-converter
```

---

## 🐛 Troubleshooting

### Container não inicia:
```bash
docker logs fingerprint-converter
docker exec fingerprint-converter ffmpeg -version
```

### Cache não funciona:
```bash
docker exec fingerprint-converter ls -la /tmp/media-cache
curl http://localhost:5001/api/cache/stats | jq .global_stats
```

### Baixa performance:
- Aumentar `MAX_WORKERS`
- Aumentar `tmpfs` size
- Verificar CPU/RAM disponível

### Erro "out of memory":
- Diminuir `BUFFER_POOL_SIZE`
- Aumentar `GOMEMLIMIT`
- Reduzir `MAX_WORKERS`

---

## 📞 Links Úteis

- **GitHub:** https://github.com/jonemp31/fingerprint-converter
- **Docker Hub:** https://hub.docker.com/r/jondevsouza31/fingerprint-converter
- **README completo:** https://github.com/jonemp31/fingerprint-converter/blob/main/README.md
- **Exemplo Node.js:** https://github.com/jonemp31/fingerprint-converter/blob/main/examples/nodejs-integration.js

---

## ✅ Checklist de Deploy

- [ ] Pull da imagem Docker
- [ ] Criar docker-compose.yml com network_swarm_public
- [ ] Configurar variáveis de ambiente
- [ ] Deploy com docker-compose up -d
- [ ] Verificar health: `curl http://localhost:5001/api/health`
- [ ] Teste de conversão
- [ ] Integrar na API Node.js
- [ ] Monitorar logs e cache stats

**Bom deploy! 🚀**

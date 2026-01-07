# 🚀 Como Rodar o Servidor na Porta 9090

## Opção 1: Usando o arquivo batch (Windows)

1. Abra o PowerShell ou CMD
2. Navegue até a pasta `fingerprint-converter`
3. Execute:
   ```bash
   .\run-server.bat
   ```

## Opção 2: Manualmente

1. Abra o PowerShell ou CMD
2. Navegue até a pasta `fingerprint-converter`
3. Execute:
   ```powershell
   $env:PORT="9090"
   $env:BASE_URL="http://localhost:9090"
   go run cmd/api/main.go
   ```

## Opção 3: Usando o script PowerShell

```powershell
cd fingerprint-converter
.\run-server.ps1
```

## ✅ Verificar se está rodando

Após iniciar, teste em outro terminal:

```powershell
# Testar endpoint raiz
curl http://localhost:9090/

# Ou no navegador
# http://localhost:9090/
```

## 📡 Endpoints Disponíveis

- `GET /` - Informações da API
- `POST /api/process` - Processar arquivo
- `GET /api/files/:id` - Baixar arquivo processado
- `GET /api/health` - Health check

## 🧪 Exemplo de Teste

```bash
# Processar um arquivo
curl -X POST http://localhost:9090/api/process \
  -H "Content-Type: application/json" \
  -d '{"arquivo": "https://example.com/audio.mp3"}'
```

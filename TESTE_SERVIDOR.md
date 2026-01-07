# 🔍 Diagnóstico: Servidor não está subindo

## ✅ Passos para Verificar

### 1. Verificar se compila sem erros

Execute no PowerShell (na pasta `fingerprint-converter`):

```powershell
go build cmd/api/main.go
```

**Se der erro**, me envie a mensagem de erro completa.

### 2. Verificar se a porta está livre

```powershell
Get-NetTCPConnection -LocalPort 9090
```

**Se retornar algo**, execute:
```powershell
$conn = Get-NetTCPConnection -LocalPort 9090
Stop-Process -Id $conn.OwningProcess -Force
```

### 3. Rodar o servidor com logs detalhados

```powershell
cd fingerprint-converter
$env:PORT="9090"
$env:BASE_URL="http://localhost:9090"
go run cmd/api/main.go
```

**Observe:**
- Se aparecer "✅ Ready to process media!" = Servidor iniciou com sucesso
- Se aparecer algum erro antes disso = Me envie o erro completo

### 4. Testar se está respondendo

Em **outro terminal**, execute:

```powershell
# Teste 1: Endpoint raiz
Invoke-WebRequest -Uri "http://localhost:9090/" -UseBasicParsing

# Teste 2: Health check
Invoke-WebRequest -Uri "http://localhost:9090/api/health" -UseBasicParsing
```

## 🐛 Problemas Comuns

### Problema: "bind: address already in use"
**Solução:** A porta está ocupada. Execute:
```powershell
Get-NetTCPConnection -LocalPort 9090 | ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }
```

### Problema: "go: command not found"
**Solução:** Go não está instalado ou não está no PATH.

### Problema: Erro de compilação
**Solução:** Me envie o erro completo para eu corrigir.

## 📋 Checklist

- [ ] Go está instalado (`go version`)
- [ ] FFmpeg está instalado (`ffmpeg -version`)
- [ ] Porta 9090 está livre
- [ ] Código compila sem erros
- [ ] Servidor inicia (vê "✅ Ready to process media!")
- [ ] Endpoint responde (teste com curl/Postman)

## 🚀 Script Automático

Execute o arquivo `start-server.ps1` que criei. Ele faz tudo automaticamente:

```powershell
.\start-server.ps1
```

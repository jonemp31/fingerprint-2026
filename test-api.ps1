# Script de teste completo da API

$baseUrl = "http://localhost:9090"

Write-Host "🧪 Testando API Fingerprint Converter" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Teste 1: Verificar se API está rodando
Write-Host "1️⃣ Testando GET /" -ForegroundColor Yellow
try {
    $info = Invoke-RestMethod -Uri "$baseUrl/" -Method GET
    Write-Host "   ✅ API Status: $($info.status)" -ForegroundColor Green
    Write-Host "   📦 Service: $($info.service)" -ForegroundColor Gray
} catch {
    Write-Host "   ❌ Erro: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Teste 2: Health Check
Write-Host "2️⃣ Testando GET /api/health" -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/api/health" -Method GET
    Write-Host "   ✅ Status: $($health.status)" -ForegroundColor Green
    Write-Host "   🎬 FFmpeg: $($health.ffmpeg_version.Substring(0, [Math]::Min(50, $health.ffmpeg_version.Length)))..." -ForegroundColor Gray
} catch {
    Write-Host "   ❌ Erro: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Teste 3: Processar Áudio
Write-Host "3️⃣ Testando POST /api/process (Áudio)" -ForegroundColor Yellow
$audioUrl = "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3"
Write-Host "   📥 URL: $audioUrl" -ForegroundColor Gray

$body = @{
    arquivo = $audioUrl
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/process" `
        -Method POST `
        -ContentType "application/json" `
        -Body $body
    
    if ($response.success) {
        Write-Host "   ✅ $($response.message)" -ForegroundColor Green
        Write-Host "   📥 Nova URL: $($response.nova_url)" -ForegroundColor Cyan
        Write-Host "   🆔 File ID: $($response.file_id)" -ForegroundColor Gray
        
        # Teste 4: Baixar arquivo
        Write-Host ""
        Write-Host "4️⃣ Testando GET /api/files/:id" -ForegroundColor Yellow
        try {
            $fileId = $response.file_id
            $extension = ".opus"
            $outputFile = "teste_audio_processado$extension"
            
            Invoke-WebRequest -Uri "$($response.nova_url)" `
                -OutFile $outputFile
            
            $fileSize = (Get-Item $outputFile).Length
            Write-Host "   ✅ Arquivo baixado: $outputFile ($([math]::Round($fileSize/1KB, 2)) KB)" -ForegroundColor Green
        } catch {
            Write-Host "   ❌ Erro ao baixar: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "   ❌ Erro: $($response.message)" -ForegroundColor Red
    }
} catch {
    Write-Host "   ❌ Erro: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "   Detalhes: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
    }
}
Write-Host ""

# Teste 5: Processar Imagem
Write-Host "5️⃣ Testando POST /api/process (Imagem)" -ForegroundColor Yellow
$imageUrl = "https://picsum.photos/800/600"
Write-Host "   📥 URL: $imageUrl" -ForegroundColor Gray

$body = @{
    arquivo = $imageUrl
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/process" `
        -Method POST `
        -ContentType "application/json" `
        -Body $body
    
    if ($response.success) {
        Write-Host "   ✅ $($response.message)" -ForegroundColor Green
        Write-Host "   📥 Nova URL: $($response.nova_url)" -ForegroundColor Cyan
    } else {
        Write-Host "   ❌ Erro: $($response.message)" -ForegroundColor Red
    }
} catch {
    Write-Host "   ❌ Erro: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Teste 6: Validação (erro esperado)
Write-Host "6️⃣ Testando validação (erro esperado)" -ForegroundColor Yellow
$body = @{} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/process" `
        -Method POST `
        -ContentType "application/json" `
        -Body $body
    
    Write-Host "   ⚠️  Não deveria ter sucesso!" -ForegroundColor Yellow
} catch {
    Write-Host "   ✅ Erro esperado capturado corretamente" -ForegroundColor Green
}
Write-Host ""

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "✅ Testes concluídos!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 Próximos passos:" -ForegroundColor Yellow
Write-Host "   - Teste com seus próprios arquivos" -ForegroundColor Gray
Write-Host "   - Verifique os logs: docker logs -f fingerprint-converter-local" -ForegroundColor Gray
Write-Host "   - Acesse: http://localhost:9090/" -ForegroundColor Gray

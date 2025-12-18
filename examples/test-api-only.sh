# =======================================================
# TESTE RÁPIDO - APENAS API (SEM ENVIO WHATSAPP)
# =======================================================
# Use este script para testar só a conversão
# Cole no n8n (Execute Command node)
# =======================================================

# =======================================================
# 1. CONFIGURAÇÕES
# =======================================================

# API Fingerprint Converter
CONVERTER_API="http://fingerprint-converter:5001"
# Se rodar localmente: http://localhost:5001

# Device ID (identificador único do dispositivo)
DEVICE_ID="device001"

# URL do arquivo (S3, Minio, HTTP)
FILE_URL="https://s3.setupautomatizado.com.br/zuckzapgo/audio.mp3"

# Tipo de mídia
MEDIA_TYPE="audio"  # audio, image ou video

# Nível de anti-fingerprint
AF_LEVEL="moderate"  # none, basic, moderate, paranoid

# =======================================================
# 2. TESTE DE HEALTH
# =======================================================
echo "=========================================="
echo "🏥 1. Testando Health da API..."
echo "=========================================="

HEALTH=$(curl -s "$CONVERTER_API/api/health")
STATUS=$(echo "$HEALTH" | python3 -c "import sys, json; print(json.load(sys.stdin)['status'])" 2>/dev/null)

if [ "$STATUS" = "healthy" ]; then
    echo "✅ API está saudável!"
    echo ""
    echo "$HEALTH" | python3 -m json.tool
else
    echo "❌ API não está respondendo!"
    exit 1
fi

# =======================================================
# 3. TESTE DE CONVERSÃO
# =======================================================
echo ""
echo "=========================================="
echo "🔄 2. Testando Conversão..."
echo "=========================================="
echo "Device ID: $DEVICE_ID"
echo "URL: $FILE_URL"
echo "Tipo: $MEDIA_TYPE"
echo "Nível AF: $AF_LEVEL"
echo "=========================================="
echo ""

# Faz requisição de conversão
START_TIME=$(date +%s)

RESPONSE=$(curl -s -X POST "$CONVERTER_API/api/convert" \
    -H "Content-Type: application/json" \
    -d "{
        \"device_id\": \"$DEVICE_ID\",
        \"url\": \"$FILE_URL\",
        \"media_type\": \"$MEDIA_TYPE\",
        \"anti_fingerprint_level\": \"$AF_LEVEL\"
    }")

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Verifica resposta
SUCCESS=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)

if [ "$SUCCESS" = "True" ]; then
    echo "✅ CONVERSÃO BEM-SUCEDIDA!"
    echo ""
    
    # Extrai informações
    PROCESSED_PATH=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['processed_path'])")
    CACHE_HIT=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['cache_hit'])")
    ORIGINAL_SIZE=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('original_size_bytes', 0))")
    PROCESSED_SIZE=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['processed_size_bytes'])")
    SIZE_INCREASE=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('size_increase_percent', 'N/A'))")
    PROCESSING_TIME=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['processing_time_ms'])")
    
    echo "📦 Resultado:"
    echo "   Arquivo: $PROCESSED_PATH"
    echo "   Cache Hit: $CACHE_HIT"
    echo "   Tamanho Original: $ORIGINAL_SIZE bytes"
    echo "   Tamanho Processado: $PROCESSED_SIZE bytes"
    echo "   Aumento: $SIZE_INCREASE"
    echo "   Tempo de Processamento: ${PROCESSING_TIME}ms"
    echo "   Tempo Total: ${DURATION}s"
    
else
    echo "❌ ERRO NA CONVERSÃO!"
    echo ""
    echo "$RESPONSE" | python3 -m json.tool
    exit 1
fi

# =======================================================
# 4. TESTE DE CACHE (2ª CONVERSÃO - DEVE SER CACHE HIT)
# =======================================================
echo ""
echo "=========================================="
echo "💾 3. Testando Cache (2ª Conversão)..."
echo "=========================================="

START_TIME=$(date +%s)

RESPONSE2=$(curl -s -X POST "$CONVERTER_API/api/convert" \
    -H "Content-Type: application/json" \
    -d "{
        \"device_id\": \"$DEVICE_ID\",
        \"url\": \"$FILE_URL\",
        \"media_type\": \"$MEDIA_TYPE\",
        \"anti_fingerprint_level\": \"$AF_LEVEL\"
    }")

END_TIME=$(date +%s)
DURATION2=$((END_TIME - START_TIME))

CACHE_HIT2=$(echo "$RESPONSE2" | python3 -c "import sys, json; print(json.load(sys.stdin)['cache_hit'])")

if [ "$CACHE_HIT2" = "True" ]; then
    echo "✅ CACHE HIT! (muito mais rápido)"
    echo "   Tempo: ${DURATION2}s (vs ${DURATION}s na 1ª conversão)"
else
    echo "⚠️  Cache miss - pode ser normal se passou 28 minutos"
fi

# =======================================================
# 5. ESTATÍSTICAS DO CACHE
# =======================================================
echo ""
echo "=========================================="
echo "📊 4. Estatísticas do Cache..."
echo "=========================================="

STATS=$(curl -s "$CONVERTER_API/api/cache/stats/$DEVICE_ID")
echo "$STATS" | python3 -m json.tool

# =======================================================
# 6. RESUMO
# =======================================================
echo ""
echo "=========================================="
echo "✅ TESTE COMPLETO!"
echo "=========================================="
echo ""
echo "🎯 Próximos passos:"
echo "   1. Arquivo processado está em: $PROCESSED_PATH"
echo "   2. Para enviar ao WhatsApp, use o script test-with-whatsapp.sh"
echo "   3. Para integrar na sua API, veja: examples/nodejs-integration.js"
echo ""
echo "📚 Documentação completa:"
echo "   https://github.com/jonemp31/fingerprint-converter"
echo ""

# =======================================================
# VARIÁVEIS PARA USAR NO N8N
# =======================================================
# Se quiser usar as variáveis no próximo node do n8n, descomente:
# echo "::set-output name=processed_path::$PROCESSED_PATH"
# echo "::set-output name=cache_hit::$CACHE_HIT"
# echo "::set-output name=processing_time::$PROCESSING_TIME"

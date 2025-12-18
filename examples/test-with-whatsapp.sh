# =======================================================
# TESTE FINGERPRINT CONVERTER API + WHATSAPP
# =======================================================
# Use este script no n8n (Execute Command node)
# ou rode diretamente no terminal da sua VPS
# =======================================================

# =======================================================
# 1. CONFIGURAÇÕES
# =======================================================
DEVICE="45.178.182.34:10001"
PHONE="5516974069851"

# API Fingerprint Converter
CONVERTER_API="http://fingerprint-converter:5001"
# Se rodar fora do Docker, use: http://localhost:5001

# Arquivo para testar (coloque uma URL do seu S3/Minio)
FILE_URL="https://s3.example.com/audio.mp3"
# Ou use base64:
# FILE_URL="data:audio/mpeg;base64,SUQzBAAAAAAAI1RTU0UAAAA..."

DEVICE_ID="device001"
MEDIA_TYPE="audio"  # audio, image, video
AF_LEVEL="moderate"  # none, basic, moderate, paranoid

# Mensagem de texto (opcional - envia antes da mídia)
MENSAGEM_TEXTO="🎵 Enviando áudio com anti-fingerprint"

# =======================================================
# 2. FUNÇÃO GERADORA DE TEXTO (mesma sua)
# =======================================================
gerar_comando_safe() {
    TXT=$1
    python3 -c "
import random, sys, unicodedata

def remove_acentos(input_str):
    nfkd_form = unicodedata.normalize('NFD', input_str)
    return ''.join([c for c in nfkd_form if not unicodedata.combining(c)])

texto_original = sys.argv[1]
texto = remove_acentos(texto_original)

comandos = []
min_delay = 0.08
max_delay = 0.20

for char in texto:
    delay = round(random.uniform(min_delay, max_delay), 3)
    
    if char == ' ':
        comandos.append('input keyevent 62')
    elif char == '?':
        comandos.append('input text \\?') 
    else:
        if char in ['(', ')', '<', '>', '|', ';', '&', '*', '\\'', '\"']:
             comandos.append(f'input text \\{char}')
        else:
             comandos.append(f'input text {char}')
    
    comandos.append(f'sleep {delay}')

print(';'.join(comandos))
" "$TXT"
}

# =======================================================
# 3. FUNÇÃO DE CONVERSÃO COM ANTI-FINGERPRINT
# =======================================================
converter_midia() {
    echo "🔄 Convertendo mídia com anti-fingerprint..."
    
    # Chama a API de conversão
    RESPONSE=$(curl -s -X POST "$CONVERTER_API/api/convert" \
        -H "Content-Type: application/json" \
        -d "{
            \"device_id\": \"$DEVICE_ID\",
            \"url\": \"$FILE_URL\",
            \"media_type\": \"$MEDIA_TYPE\",
            \"anti_fingerprint_level\": \"$AF_LEVEL\"
        }")
    
    # Verifica se teve erro
    if echo "$RESPONSE" | grep -q '"success":false'; then
        echo "❌ Erro na conversão:"
        echo "$RESPONSE" | python3 -m json.tool
        exit 1
    fi
    
    # Extrai o caminho do arquivo processado
    PROCESSED_PATH=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['processed_path'])")
    CACHE_HIT=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['cache_hit'])")
    
    echo "✅ Conversão completa!"
    echo "   Cache hit: $CACHE_HIT"
    echo "   Arquivo: $PROCESSED_PATH"
    
    # Retorna o caminho do arquivo
    echo "$PROCESSED_PATH"
}

# =======================================================
# 4. ENVIO DE TEXTO (OPCIONAL)
# =======================================================
enviar_texto() {
    if [ -z "$MENSAGEM_TEXTO" ]; then
        return
    fi
    
    echo "💬 Enviando mensagem de texto..."
    
    # Abre WhatsApp
    adb -s $DEVICE shell am start -a android.intent.action.VIEW -d "https://api.whatsapp.com/send?phone=$PHONE" com.whatsapp.w4b
    sleep 3
    
    # Foco no campo
    adb -s $DEVICE shell "input tap 1345 1006 && sleep 0.2 && input tap 1345 1006"
    sleep 1
    
    # Digita texto
    CMD_TEXTO=$(gerar_comando_safe "$MENSAGEM_TEXTO")
    echo "$CMD_TEXTO" | adb -s $DEVICE shell
    sleep 1
    
    # Envia
    adb -s $DEVICE shell input keyevent 66
    sleep 2
    
    echo "✅ Texto enviado"
}

# =======================================================
# 5. ENVIO DE MÍDIA VIA ADB
# =======================================================
enviar_midia() {
    LOCAL_FILE=$1
    
    echo "📤 Enviando mídia para WhatsApp..."
    
    # Determina extensão e MIME type
    case $MEDIA_TYPE in
        audio)
            EXT="opus"
            MIME_TYPE="audio/ogg"
            ;;
        image)
            EXT="jpg"
            MIME_TYPE="image/jpeg"
            ;;
        video)
            EXT="mp4"
            MIME_TYPE="video/mp4"
            ;;
        *)
            echo "❌ Tipo de mídia inválido: $MEDIA_TYPE"
            exit 1
            ;;
    esac
    
    # Path no device Android
    DEVICE_PATH="/sdcard/Download/wa_$(date +%s).$EXT"
    
    # 1. Faz push do arquivo para o device
    echo "📲 Push para device..."
    adb -s $DEVICE push "$LOCAL_FILE" "$DEVICE_PATH"
    sleep 1
    
    # 2. Abre WhatsApp (se não estiver aberto)
    adb -s $DEVICE shell am start -a android.intent.action.VIEW -d "https://api.whatsapp.com/send?phone=$PHONE" com.whatsapp.w4b
    sleep 2
    
    # 3. Envia mídia via intent
    echo "📎 Anexando mídia..."
    adb -s $DEVICE shell am start \
        -a android.intent.action.SEND \
        -t "$MIME_TYPE" \
        --eu android.intent.extra.STREAM "file://$DEVICE_PATH" \
        --es jid "$PHONE@s.whatsapp.net" \
        -n com.whatsapp.w4b/.ContactPicker
    sleep 3
    
    # 4. Envia
    echo "📤 Enviando..."
    adb -s $DEVICE shell input keyevent 66
    sleep 2
    
    # 5. Limpa arquivo do device
    echo "🗑️ Limpando..."
    adb -s $DEVICE shell rm "$DEVICE_PATH"
    
    # 6. Volta para home
    adb -s $DEVICE shell input keyevent 4
    sleep 0.5
    adb -s $DEVICE shell input keyevent 4
    
    echo "✅ Mídia enviada com sucesso!"
}

# =======================================================
# 6. EXECUÇÃO PRINCIPAL
# =======================================================
echo ""
echo "=========================================="
echo "🚀 TESTE FINGERPRINT CONVERTER + WHATSAPP"
echo "=========================================="
echo "Device: $DEVICE"
echo "Phone: $PHONE"
echo "Media Type: $MEDIA_TYPE"
echo "AF Level: $AF_LEVEL"
echo "=========================================="
echo ""

# 1. Converte com anti-fingerprint
PROCESSED_FILE=$(converter_midia)

# Verifica se arquivo existe
if [ ! -f "$PROCESSED_FILE" ]; then
    echo "❌ Arquivo processado não encontrado: $PROCESSED_FILE"
    exit 1
fi

# 2. Envia texto (opcional)
if [ -n "$MENSAGEM_TEXTO" ]; then
    enviar_texto
fi

# 3. Envia mídia
enviar_midia "$PROCESSED_FILE"

# 4. Estatísticas
echo ""
echo "📊 Estatísticas do cache:"
curl -s "$CONVERTER_API/api/cache/stats/$DEVICE_ID" | python3 -m json.tool

echo ""
echo "✅ TESTE CONCLUÍDO COM SUCESSO!"
echo ""

# =======================================================
# PRONTO PARA USAR NO N8N!
# =======================================================
# No n8n, cole este script no node "Execute Command"
# Configure as variáveis no topo conforme sua necessidade
# =======================================================

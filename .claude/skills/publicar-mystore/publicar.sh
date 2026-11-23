#!/usr/bin/env bash
# Sube una o varias APKs a MyStore y muestra el resultado.
# Uso: publicar.sh [-b] [-n "notas"] [-d "descripción"] app1.apk [app2.apk ...]   (-b = canal beta)
# Las notas y la descripción van como UTF-8 real (python), nunca por -F de curl en Git Bash.
set -uo pipefail
STORE="${MYSTORE_URL:?Configura MYSTORE_URL con la URL de la tienda}"
TOKEN_FILE="${MYSTORE_TOKEN_FILE:-$HOME/.mystore-token}"
TUNNEL_PORT=18130
NOTES=""; DESC=""; CHANNEL=stable
while getopts "bn:d:" o; do case $o in b) CHANNEL=beta;; n) NOTES="$OPTARG";; d) DESC="$OPTARG";; *) exit 2;; esac; done
shift $((OPTIND-1))
[ $# -gt 0 ] || { echo "Uso: publicar.sh [-n notas] [-d descripción] app.apk..." >&2; exit 2; }
[ -s "$TOKEN_FILE" ] || { echo "Falta el token en $TOKEN_FILE" >&2; exit 1; }
TOK=$(cat "$TOKEN_FILE")
BT=$(ls -d "$LOCALAPPDATA/Android/Sdk/build-tools/"* 2>/dev/null | tail -1)

rc=0
for apk in "$@"; do
  [ -f "$apk" ] || { echo "No existe: $apk" >&2; rc=1; continue; }
  size=$(stat -c %s "$apk")
  info=$("$BT/aapt2.exe" dump badging "$apk" 2>/dev/null | grep -oE "^package: name='[^']*' versionCode='[^']*' versionName='[^']*'" | sed -E "s/package: name=//")
  echo "→ $(basename "$apk") ($((size/1048576)) MB) $info"
  base="$STORE"
  # Cloudflare corta las subidas de más de 100 MB: se sube por túnel SSH directo al contenedor.
  if [ "$size" -gt $((95*1048576)) ]; then
    : "${MYSTORE_SSH_HOST:?Configura el destino SSH para APKs de más de 95 MiB}"
    if ! curl -s -o /dev/null --max-time 3 "http://127.0.0.1:$TUNNEL_PORT/api/apps"; then
      ssh -o ExitOnForwardFailure=yes -f -N -L "$TUNNEL_PORT:${MYSTORE_SSH_TARGET:-127.0.0.1:8130}" "$MYSTORE_SSH_HOST" || { echo "  no pude abrir el túnel SSH" >&2; rc=1; continue; }
      sleep 2
    fi
    base="http://127.0.0.1:$TUNNEL_PORT"
  fi
  extra=()
  # Beta: las notas van en la subida (desde fichero, UTF-8 intacto); el PATCH de abajo tocaría las de la estable.
  if [ "$CHANNEL" = beta ]; then nf=$(mktemp); printf '%s' "$NOTES" > "$nf"; extra=(-F "notes=<$(cygpath -w "$nf")"); fi
  out=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $TOK" -F "channel=$CHANNEL" "${extra[@]}" -F "apk=@$apk" "$base/api/apps")
  code=$(tail -1 <<<"$out"); body=$(sed '$d' <<<"$out")
  id=$(grep -oE '"id":"[^"]*"' <<<"$body" | head -1 | cut -d'"' -f4)
  if [ "$code" = 201 ]; then
    echo "  publicada: $id ($CHANNEL)"
    if [ "$CHANNEL" != beta ] && [ -n "$NOTES$DESC" ]; then
      PYTHONIOENCODING=utf-8 NOTES="$NOTES" DESC="$DESC" TOK="$TOK" URL="$STORE/api/apps/$id" python -I -c '
import json, os, urllib.request
d = {k: os.environ[e] for k, e in (("notes", "NOTES"), ("description", "DESC")) if os.environ[e]}
r = urllib.request.Request(os.environ["URL"], data=json.dumps(d).encode("utf-8"), method="PATCH",
    headers={"Authorization": "Bearer " + os.environ["TOK"], "Content-Type": "application/json", "User-Agent": "publicar-mystore"})
print("  notas y descripcion:", urllib.request.urlopen(r).status)'
    fi
  else
    echo "  ERROR $code: $(grep -oE '"error":"[^"]*"' <<<"$body" | cut -d'"' -f4)" >&2; rc=1
  fi
done
exit $rc

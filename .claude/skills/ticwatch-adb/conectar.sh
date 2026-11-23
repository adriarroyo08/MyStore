#!/usr/bin/env bash
# Imprime el serial adb del TicWatch listo para usar con -s, o explica qué pedir al usuario.
# Uso: conectar.sh [IP:PUERTO]   (pasa la IP:puerto nueva si el usuario la ha dado)
ADB="$LOCALAPPDATA/Android/Sdk/platform-tools/adb.exe"
MODEL="TicWatch_Pro_5_Enduro"
# En este Windows el descubrimiento mDNS de adb no funciona: se recuerda la última IP:puerto buena.
STATE_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/mystore/ticwatch-adb"
mkdir -p "$STATE_DIR" || exit 1
STATE="$STATE_DIR/ultima-conexion"

"$ADB" start-server >/dev/null 2>&1

find_serial() {
  "$ADB" devices -l | awk -v m="model:$MODEL" '$2=="device" && index($0,m) {print $1; exit}' | grep .
}

remember() {
  # Solo las formas IP:puerto sirven para reconectar.
  [[ "$1" =~ ^[0-9.]+:[0-9]+$ ]] && echo "$1" > "$STATE"
}

if s=$(find_serial); then remember "$s"; echo "$s"; exit 0; fi

for target in "$1" "$(cat "$STATE" 2>/dev/null)"; do
  [ -n "$target" ] || continue
  "$ADB" connect "$target" >/dev/null 2>&1
  for _ in 1 2 3; do
    if s=$(find_serial); then remember "$s"; echo "$s"; exit 0; fi
    sleep 1
  done
done

cat >&2 <<EOF
No encuentro el reloj (probado: ${1:-} $(cat "$STATE" 2>/dev/null)). Pide a Adrián:
  1. Reloj despierto, en la misma WiFi que el PC, con
     Opciones de desarrollador > Depuración inalámbrica ACTIVADA.
  2. La IP:puerto que sale en esa pantalla (el puerto cambia al reactivarla)
     y repite: bash .claude/skills/ticwatch-adb/conectar.sh IP:PUERTO
  3. Si aun así falla por autenticación: "Vincular dispositivo nuevo" > IP:puerto + código
     y ejecuta: "\$ADB" pair IP:PUERTO CODIGO   (luego repite el paso 2)
EOF
exit 1

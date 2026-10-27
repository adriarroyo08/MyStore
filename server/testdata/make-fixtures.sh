#!/usr/bin/env bash
# Regenera las APKs de prueba. Requiere el SDK de Android y un JDK (keytool).
set -euo pipefail
cd "$(dirname "$0")"
SDK="${ANDROID_HOME:-$LOCALAPPDATA/Android/Sdk}"
BT="$SDK/build-tools/36.0.0"
JAR="$SDK/platforms/android-36/android.jar"
APKSIGNER="$BT/apksigner"; [ -f "$APKSIGNER.bat" ] && APKSIGNER="$APKSIGNER.bat"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

mkdir -p keys
for k in a b; do
  [ -f "keys/$k.jks" ] || keytool -genkeypair -keystore "keys/$k.jks" -storepass android -keypass android \
    -alias key -keyalg RSA -keysize 2048 -validity 36500 -dname "CN=MyStore test $k" >/dev/null
done

# build <out> <manifest> <versionCode> <versionName> <key> [carpeta de recursos en src/]
build() {
  local res=()
  if [ -n "${6:-}" ]; then
    "$BT/aapt2" compile --dir "src/$6" -o "$TMP/$6.zip"
    res=("$TMP/$6.zip")
  fi
  "$BT/aapt2" link -I "$JAR" --manifest "src/$2" --min-sdk-version 30 --target-sdk-version 36 \
    --version-code "$3" --version-name "$4" -o "$TMP/u.apk" "${res[@]}"
  "$BT/zipalign" -f -p 4 "$TMP/u.apk" "$TMP/a.apk"
  "$APKSIGNER" sign --ks "keys/$5.jks" --ks-pass pass:android --out "$1" "$TMP/a.apk"
  rm -f "$1.idsig"
}

build phone-v1.apk phone.xml 1 1.0 a res
build phone-v2.apk phone.xml 2 2.0 a res
build phone-v2-otherkey.apk phone.xml 2 2.0 b res
build watch-v1.apk watch.xml 1 1.0 a res
build noicon-v1.apk noicon.xml 1 1.0 a
build adaptive-v1.apk adaptive.xml 1 1.0 a res-adaptive
build badvector-v1.apk badvector.xml 1 1.0 a res-badvector
head -c 1024 /dev/zero | tr '\0' 'x' > not-an-apk.apk
echo "fixtures OK"

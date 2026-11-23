---
name: ticwatch-adb
description: Use when Adrián pide instalar, actualizar, desinstalar o probar una APK en su reloj (TicWatch Pro 5 Enduro, Wear OS), mirar logs del reloj, o cuando adb no encuentra el reloj ("device not found", "no devices", "more than one device", "failed to connect").
---

# TicWatch por adb (WiFi)

Reloj: **TicWatch Pro 5 Enduro**, Android 13 / Wear OS 5, ABI arm. **Ya está vinculado** con este PC por depuración inalámbrica: no hace falta volver a emparejar salvo que adb lo rechace.

- adb **no está en el PATH**: `ADB="$LOCALAPPDATA/Android/Sdk/platform-tools/adb.exe"`
- En este Windows el descubrimiento **mDNS de adb no funciona**: el reloj no se reconecta solo. Hay que hacer `adb connect IP:PUERTO`, y el **puerto cambia** cada vez que se reactiva la depuración inalámbrica (la IP también puede cambiar).
- Pueden salir dos entradas del mismo reloj en `adb devices` (IP:puerto y `adb-…_adb-tls-connect._tcp`): usa siempre `-s`.

## Conectar

```bash
SERIAL=$(bash .claude/skills/ticwatch-adb/conectar.sh) || exit 1
```

El script usa el reloj si ya está conectado; si no, prueba la última IP:puerto que funcionó (la guarda en `ultima-conexion` junto al script) y devuelve el serial. Si falla, imprime qué pedir a Adrián:

1. Reloj despierto y en la misma WiFi; *Opciones de desarrollador → Depuración inalámbrica* activada (se apaga sola al reiniciar o cambiar de red).
2. La **IP:puerto** de esa pantalla → `bash .claude/skills/ticwatch-adb/conectar.sh IP:PUERTO`.
3. Si falla por autenticación: *Vincular dispositivo nuevo* → **IP:puerto de vinculación + código** → `"$ADB" pair IP:PUERTO CODIGO` y repite el paso 2.

## Instalar

Si Adrián ha pedido instalar, instala directamente (`-r` conserva datos al actualizar):

```bash
"$ADB" -s "$SERIAL" install -r ruta/app.apk
```

Comprobar: `"$ADB" -s "$SERIAL" shell pm list packages <paquete>` y, para versión, `dumpsys package <paquete> | grep versionName`.

| Error | Qué hacer |
|---|---|
| `INSTALL_FAILED_UPDATE_INCOMPATIBLE` | Otra firma instalada. Desinstalar **borra los datos de la app**: pide confirmación antes de `uninstall` + `install`. |
| `INSTALL_FAILED_VERSION_DOWNGRADE` | versionCode menor. `install -r -d` solo si Adrián confirma bajar de versión. |
| `INSTALL_FAILED_NO_MATCHING_ABIS` | La APK no trae libs arm (`armeabi-v7a`/`arm64-v8a`). |
| `INSTALL_FAILED_OLDER_SDK` | minSdk de la APK > 33. |
| APK de móvil (sin `android.hardware.type.watch`) | Se instala, pero suele no funcionar en el reloj: avisa. |

## Otros comandos

- Abrir app: `"$ADB" -s "$SERIAL" shell monkey -p <paquete> -c android.intent.category.LAUNCHER 1`
- Logs: `"$ADB" -s "$SERIAL" logcat -d -t 200 | grep -i <paquete>`
- Desinstalar (si lo pide): `"$ADB" -s "$SERIAL" uninstall <paquete>`
- Captura: `"$ADB" -s "$SERIAL" exec-out screencap -p > reloj.png`

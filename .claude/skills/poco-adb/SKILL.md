---
name: poco-adb
description: Use when Adrián pide instalar, actualizar, desinstalar o probar una APK en su móvil (POCO X6 5G, MIUI/HyperOS), mirar logs o capturas del móvil, o cuando adb no encuentra el móvil ("device not found", "no devices", "more than one device", "failed to connect").
---

# POCO X6 por adb (WiFi)

Móvil: **POCO X6 5G** (modelo adb `23122PCD1G`), HyperOS/MIUI, Android 14+, arm64. **Ya está vinculado** con este PC por depuración inalámbrica: no hace falta volver a emparejar salvo que adb lo rechace.

- adb **no está en el PATH**: `ADB="$LOCALAPPDATA/Android/Sdk/platform-tools/adb.exe"`
- En este Windows el **mDNS de adb no funciona**: hay que hacer `adb connect IP:PUERTO`, y el **puerto cambia** cada vez que se reactiva la depuración inalámbrica.
- Con el reloj conectado a la vez, usa siempre `-s`. Rutas `/sdcard` en Git Bash: `MSYS_NO_PATHCONV=1`.

## Conectar

```bash
SERIAL=$(bash .claude/skills/poco-adb/conectar.sh) || exit 1
```

Usa el móvil si ya está conectado; si no, prueba la última IP:puerto buena (`ultima-conexion`). Si falla, imprime qué pedir a Adrián:

1. Móvil desbloqueado y en la misma WiFi; *Ajustes → Opciones de desarrollador → Depuración inalámbrica* activada.
2. La **IP:puerto** de esa pantalla → `bash .claude/skills/poco-adb/conectar.sh IP:PUERTO`.
3. Si falla por autenticación: *Vincular dispositivo con código* → **IP:puerto de vinculación + código** → `"$ADB" pair IP:PUERTO CODIGO` y repite el paso 2.

## Límites de MIUI/HyperOS (comprobados)

| Síntoma | Causa y salida |
|---|---|
| `INSTALL_FAILED_USER_RESTRICTED` | MIUI bloquea `adb install` sin *Instalar vía USB* (Opciones de desarrollador, pide cuenta Xiaomi). Alternativa: instalar desde **MyStore** en el móvil (lo normal) o pedir a Adrián que active la opción. |
| `input tap` no hace nada | MIUI bloquea toques simulados sin *Depuración USB (ajustes de seguridad)*. Usa extras de lanzamiento (`am start ... --es/--ez`) o pide a Adrián que toque. |
| `pm grant ...POST_NOTIFICATIONS` falla | Mismo motivo; que Adrián lo active en *Ajustes → Apps → <app> → Notificaciones*. |
| `INSTALL_FAILED_UPDATE_INCOMPATIBLE` | Otra firma. Desinstalar **borra los datos**: pide confirmación. |
| `INSTALL_FAILED_VERSION_DOWNGRADE` | `install -r -d` solo si Adrián confirma. |

## Comandos

- Instalar (si lo pide): `"$ADB" -s "$SERIAL" install -r ruta/app.apk`
- Abrir: `"$ADB" -s "$SERIAL" shell monkey -p <paquete> -c android.intent.category.LAUNCHER 1`
- Logs: `"$ADB" -s "$SERIAL" logcat -d -t 300 | grep -iE "<paquete>|AndroidRuntime"`
- Captura: `"$ADB" -s "$SERIAL" exec-out screencap -p > movil.png`
- Ver notificaciones: `"$ADB" -s "$SERIAL" shell dumpsys notification --noredact | grep -A40 "pkg=<paquete>"`
- Modo oscuro: anota `cmd uimode night` antes de cambiarlo y restáuralo después.

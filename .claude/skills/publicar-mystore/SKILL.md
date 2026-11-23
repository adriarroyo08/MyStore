---
name: publicar-mystore
description: Publicar APK de móvil o Wear OS en MyStore cuando se solicita una publicación.
---

# Publicar en MyStore

1. Comprueba que el versionCode supere al publicado y que el APK use la misma firma. Compila la variante correspondiente con Gradle.
2. Configura `MYSTORE_URL` con la URL de la tienda. El script lee el token de `MYSTORE_TOKEN_FILE` (por defecto, `~/.mystore-token`). Para APK de más de 95 MiB, configura `MYSTORE_SSH_HOST` y, si hace falta, `MYSTORE_SSH_TARGET` (por defecto, `127.0.0.1:8130`).
3. Desde la raíz y en Git Bash, ejecuta `bash .claude/skills/publicar-mystore/publicar.sh -n "Notas de versión" ruta/app.apk`. Usa `-d` para la descripción y `-b` para el canal beta.
4. Comprueba que el script indique `publicada` para cada APK. Un 409 por versión exige subir el versionCode y recompilar. Ante un error de firma, consigue la clave correcta antes de reintentar.

# Resultado de la prueba de instalación en Wear OS

Fecha: 2026-10-27 · Reloj: TicWatch Pro 5 Enduro · Android 13 (Wear OS 5)

**Funciona.** Una app de Wear OS puede descargar una APK por WiFi e instalarla con `PackageInstaller`, sin adb.

Pasos que pidió el sistema:
1. Primer toque en Instalar: `canRequestPackageInstalls()` era false; se abrió `ACTION_MANAGE_UNKNOWN_APP_SOURCES` y el usuario activó el permiso.
2. Segundo toque: descarga, sesión `MODE_FULL_INSTALL`, `STATUS_PENDING_USER_ACTION` → diálogo del sistema → aceptar → `STATUS_SUCCESS`.

Comprobado con adb: `com.example.demo` versionCode 1 instalada, `installerPackageName=dev.mystore.spike`, appop `REQUEST_INSTALL_PACKAGES: allow`.

Solo hizo falta adb para instalar la propia app de prueba (igual que pasará una vez con la app tienda).

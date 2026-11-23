---
name: infra-oracle
description: Consultar el estado del despliegue Docker de MyStore en Oracle.
---

# MyStore en Oracle

Pide o usa el alias SSH y la ruta del checkout configurados por el operador. Desde ese checkout, `docker compose ps` muestra el estado de los contenedores y `docker compose logs --tail=100` ayuda a diagnosticar errores. Contrasta la configuración con `docker-compose.yml` y la sección de instalación del README.

Conserva el volumen de datos y la configuración del operador. Solo despliega cuando la tarea lo pida explícitamente.

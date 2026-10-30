Derivado de adriarroyo08/estudio ROADMAP.md; lo ajusta el planificador semanal.

# Hoja de ruta de MyStore: noviembre de 2026 a abril de 2027

MyStore es la herramienta interna del estudio para distribuir las betas de sus apps y un escaparate open source. El planificador semanal convierte los hitos en issues verificables y revisa las prioridades cada mes. Los agentes preparan el material; las decisiones y las actuaciones personales se marcan con [Adrián].

Al preparar esta hoja de ruta no hay issues ni épicas abiertas en MyStore. Los hitos parten de la publicación desde CI ya disponible y de las necesidades del estudio; las notas de versión ya admitidas deben llegar de forma visible a quienes prueban las apps. El repositorio es público: la configuración del servidor, sus direcciones y las credenciales quedan fuera del código, la documentación y los PR públicos.

## Noviembre de 2026: todas las apps publican desde CI

- [ ] Comprobar la publicación de AhorrApp desde su CI, que ya está integrada, y usarla como referencia para las demás apps.
- [ ] Integrar la publicación de las APKs de WorkStation para teléfono y Wear OS desde su CI, comprobando que ambas variantes conviven en MyStore.
- [ ] Integrar la publicación de la APK de spritearena desde su CI.
- [ ] Verificar para cada app una publicación y una actualización instalable, con firma coherente y código de versión creciente.
- [ ] Documentar el flujo común de publicación con ejemplos genéricos y secretos de CI, sin datos del servidor del estudio.
- [ ] [Adrián] Configurar los secretos y las claves de firma en los repositorios que lo necesiten, sin incorporarlos al repositorio público.

## Diciembre de 2026: canales y notas para las betas

- [ ] Definir y desglosar en issues verificables el comportamiento de los canales estable y beta: publicación, selección y actualización en móvil y reloj.
- [ ] Habilitar los canales estable y beta y comprobar que una publicación o actualización de un canal no sustituye la del otro.
- [ ] Asegurar que las notas de versión enviadas desde CI sean visibles al consultar cada versión en la web, el móvil y Wear OS.
- [ ] Validar el recorrido completo desde una publicación en CI hasta la consulta de sus notas y la instalación en el canal correspondiente.
- [ ] Documentar cómo publicar y consumir cada canal y sus límites de acceso: un canal beta por sí solo no convierte las descargas en privadas.

## Enero de 2027: preparar la beta de WorkStation

- [ ] Ensayar la distribución y actualización de WorkStation en el canal beta, tanto en móvil como en reloj, antes de la beta cerrada de febrero.
- [ ] Preparar instrucciones de instalación inicial y actualización, incluida la primera instalación de MyStore en Wear OS.
- [ ] Registrar y resolver las incidencias de MyStore que bloqueen las pruebas de WorkStation, con comprobaciones de regresión.
- [ ] [Adrián] Confirmar los dispositivos y el grupo de prueba y decidir qué compilaciones pueden distribuirse por MyStore según sus límites de acceso; la beta cerrada de Google Play sigue su propio proceso.

## Febrero de 2027: acompañar la beta cerrada de WorkStation

- [ ] Mantener disponible en MyStore la distribución acordada de WorkStation para móvil y Wear OS durante su beta cerrada en Google Play.
- [ ] Comprobar las sucesivas actualizaciones de ambas variantes, las notas de versión y la separación entre estable y beta.
- [ ] Priorizar y corregir los fallos de publicación, consulta o instalación de MyStore que encuentren los probadores.
- [ ] Recoger el balance de distribución de la beta y ajustar las instrucciones con los problemas observados.

## Marzo de 2027: acompañar la beta de spritearena

- [ ] Validar la publicación e instalación de las compilaciones de spritearena para su beta cerrada con dos o tres packs, conforme al plan del estudio.
- [ ] Comprobar las actualizaciones y las notas de versión del canal beta durante las pruebas.
- [ ] Resolver las incidencias de MyStore que bloqueen la distribución de spritearena y actualizar las instrucciones de prueba.
- [ ] [Adrián] Confirmar el grupo de probadores y las compilaciones que se distribuirán por MyStore según los límites de acceso documentados.

## Abril de 2027: versión estable y revisión del semestre

- [ ] Preparar una versión estable documentada de MyStore: despliegue genérico, publicación desde CI, canales, notas de versión e instalación en móvil y Wear OS.
- [ ] Validar los recorridos documentados y resolver los bloqueos pendientes antes de publicar la versión estable.
- [ ] Comprobar que MyStore acompaña el lanzamiento público de WorkStation con la distribución acordada del canal estable.
- [ ] Revisar el escaparate open source: README, ejemplos y limitaciones coherentes con la versión estable, sin datos del servidor del estudio.
- [ ] Elaborar el balance semestral de integraciones, incidencias y necesidades de las betas y proponer prioridades para mayo a octubre de 2027.
- [ ] [Adrián] Aprobar la versión estable y decidir las prioridades del siguiente semestre dentro de la revisión del estudio.

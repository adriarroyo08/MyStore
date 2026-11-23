# Claves de prueba

`a.jks` y `b.jks` son almacenes de claves **solo para los tests** de firma del servidor: con ellos se firman las APK de `testdata/` (dos firmas distintas para probar el rechazo de una actualización con otra clave). Los genera `../make-fixtures.sh` con la contraseña `android`.

No firman ninguna app publicada ni son secretos de producción. La clave real de las apps no está en el repo (`*.jks` está en `.gitignore`; ver la sección de firma del `README.md` raíz).

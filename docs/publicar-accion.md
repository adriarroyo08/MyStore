# Publicar una versión de la acción

La versión inicial `v1.0.0` y la etiqueta móvil `v1` apuntan al commit
`f617a436d755ba46e268023adfc83a8523e4d296` de `master`. No cambia el
comportamiento de `action.yml` respecto a ese commit.

Usa semver: correcciones incrementan el parche, funciones compatibles incrementan
la versión menor y cambios incompatibles requieren una versión mayor. No muevas
etiquetas semver ya publicadas. Mueve `v1` solo a versiones compatibles 1.x;
para una versión incompatible crea la serie `v2` y conserva `v1`.

Para publicar, con los cambios revisados e integrados en `master`:

1. Actualiza las referencias y elige el commit que vas a publicar:

   ```bash
   git fetch origin master --tags
   release_commit=$(git rev-parse origin/master)
   release_version=v1.0.1 # sustituye por la nueva versión semver
   ```

2. Ejecuta las comprobaciones de `AGENTS.md` para los ficheros afectados y revisa
   el diff de `action.yml` respecto a la versión anterior. Comprueba la
   compatibilidad de entradas, salidas y comportamiento antes de mover `v1`.

3. Crea la etiqueta semver y mueve la etiqueta mayor compatible:

   ```bash
   git tag "$release_version" "$release_commit"
   git push origin "refs/tags/$release_version"
   git tag -f v1 "$release_commit"
   git push origin refs/tags/v1 --force
   ```

4. Escribe las notas en un fichero `release-notes.md` fuera del repositorio,
   indicando entradas, salidas y cambios respecto a la versión anterior
   (respecto al commit de `master` elegido para la primera versión). Publica:

   ```bash
   gh release create "$release_version" --repo adriarroyo08/MyStore \
     --verify-tag --title "Acción MyStore $release_version" \
     --notes-file /ruta/al/release-notes.md
   ```

5. Verifica las referencias remotas y la release:

   ```bash
   git ls-remote --tags https://github.com/adriarroyo08/MyStore 'refs/tags/v1' "refs/tags/$release_version"
   gh release view "$release_version" --repo adriarroyo08/MyStore
   ```

   Ambas etiquetas deben apuntar al commit elegido. Actualiza el ejemplo con SHA
   del README si cambia la versión recomendada.

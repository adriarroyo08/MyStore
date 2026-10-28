package dev.mystore.core

import org.json.JSONArray
import java.net.URI
import java.text.Normalizer

data class StoreApp(
    val id: String, val pkg: String, val type: String, val label: String,
    val versionCode: Long, val versionName: String, val size: Long, val description: String,
    val notes: String, val iconUrl: String, val apkUrl: String, // URLs ya absolutas
    val screenshots: List<String> = emptyList(), val updatedAt: String = "",
    val versions: List<StoreVersion> = emptyList(), // anteriores, de la más nueva a la más vieja
)

data class StoreVersion(
    val versionCode: Long, val versionName: String, val notes: String, val size: Long,
    val updatedAt: String, val apkUrl: String,
)

// La app tal como era en esa versión, para pasársela al instalador
fun StoreApp.asVersion(v: StoreVersion) = copy(
    versionCode = v.versionCode, versionName = v.versionName, notes = v.notes, size = v.size,
    updatedAt = v.updatedAt, apkUrl = v.apkUrl, versions = emptyList(),
)

enum class AppAction { INSTALL, UPDATE, OPEN }

fun actionFor(storeVersion: Long, installedVersion: Long?): AppAction = when {
    installedVersion == null -> AppAction.INSTALL
    installedVersion < storeVersion -> AppAction.UPDATE
    else -> AppAction.OPEN
}

// Añade https:// si falta, quita "/" final; null si no es una URL http(s) válida
fun normalizeStoreUrl(input: String): String? {
    val t = input.trim()
    if (t.isEmpty()) return null
    val withScheme = if ("://" in t) t else "https://$t"
    val uri = try { URI(withScheme) } catch (e: Exception) { return null }
    if (uri.scheme != "http" && uri.scheme != "https") return null
    if (uri.host.isNullOrEmpty()) return null
    return withScheme.trimEnd('/')
}

fun parseApps(json: String, baseUrl: String): List<StoreApp> {
    val arr = JSONArray(json)
    return (0 until arr.length()).map {
        val o = arr.getJSONObject(it)
        StoreApp(
            id = o.getString("id"), pkg = o.getString("package"), type = o.getString("type"),
            label = o.getString("label"), versionCode = o.getLong("versionCode"),
            versionName = o.optString("versionName"), size = o.optLong("size"),
            description = o.optString("description"), notes = o.optString("notes"),
            iconUrl = baseUrl + o.optString("iconUrl"), apkUrl = baseUrl + o.optString("apkUrl"),
            updatedAt = o.optString("updatedAt"),
            screenshots = o.optJSONArray("screenshots")?.let { a -> (0 until a.length()).map { i -> baseUrl + a.getString(i) } }.orEmpty(),
            versions = o.optJSONArray("versions")?.let { a ->
                (0 until a.length()).map { i ->
                    val v = a.getJSONObject(i)
                    StoreVersion(
                        v.getLong("versionCode"), v.optString("versionName"), v.optString("notes"), v.optLong("size"),
                        v.optString("updatedAt"), baseUrl + v.optString("apkUrl"),
                    )
                }
            }.orEmpty(),
        )
    }
}

private fun fold(s: String) =
    Normalizer.normalize(s, Normalizer.Form.NFD).replace(Regex("\\p{M}+"), "").lowercase()

fun List<StoreApp>.search(q: String): List<StoreApp> {
    val n = fold(q.trim())
    return filter { fold(it.label).contains(n) || fold(it.pkg).contains(n) }
}

// Primera línea no vacía de la descripción, para subtítulos de una línea
val StoreApp.summary: String
    get() = description.lineSequence().map { it.trim() }.firstOrNull { it.isNotEmpty() }.orEmpty()

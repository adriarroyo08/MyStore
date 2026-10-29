package dev.mystore.core

import android.content.Context
import android.os.Build
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

class StoreException(message: String) : Exception(message)

// Nombre con el que la tienda lista este dispositivo en la web (cabecera X-MyStore-Device)
fun deviceName(context: Context, type: String): String {
    val v = try { context.packageManager.getPackageInfo(context.packageName, 0).versionName } catch (e: Exception) { null }
    return "${Build.MANUFACTURER} ${Build.MODEL} · $type · ${v ?: "?"}"
}

class StoreClient(val baseUrl: String, private val device: String? = null) {
    suspend fun apps(type: String): List<StoreApp> = withContext(Dispatchers.IO) {
        val body = get("$baseUrl/api/apps?type=$type", device).toString(Charsets.UTF_8)
        try {
            parseApps(body, baseUrl)
        } catch (e: Exception) {
            throw StoreException("Respuesta inesperada de la tienda")
        }
    }

    suspend fun icon(app: StoreApp): ByteArray = withContext(Dispatchers.IO) { get(app.iconUrl) }

    // Descarga genérica (capturas)
    suspend fun bytes(url: String): ByteArray = withContext(Dispatchers.IO) { get(url) }

    private fun get(url: String, device: String? = null): ByteArray {
        try {
            val c = URL(url).openConnection() as HttpURLConnection
            device?.let { c.setRequestProperty("X-MyStore-Device", it) }
            c.connectTimeout = 10_000
            c.readTimeout = 30_000
            try {
                val code = c.responseCode
                if (code in 200..299) return c.inputStream.use { it.readBytes() }
                val err = c.errorStream?.use { it.readBytes().toString(Charsets.UTF_8) }.orEmpty()
                val msg = try { JSONObject(err).optString("error") } catch (e: Exception) { "" }
                throw StoreException(msg.ifEmpty { "La tienda respondió con error $code" })
            } finally {
                c.disconnect()
            }
        } catch (e: IOException) {
            throw StoreException("No se pudo conectar con la tienda. Revisa la URL y la red.")
        }
    }
}

package dev.mystore.core

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL

class StoreException(message: String) : Exception(message)

class StoreClient(val baseUrl: String) {
    suspend fun apps(type: String): List<StoreApp> = withContext(Dispatchers.IO) {
        val body = get("$baseUrl/api/apps?type=$type").toString(Charsets.UTF_8)
        try {
            parseApps(body, baseUrl)
        } catch (e: Exception) {
            throw StoreException("Respuesta inesperada de la tienda")
        }
    }

    suspend fun icon(app: StoreApp): ByteArray = withContext(Dispatchers.IO) { get(app.iconUrl) }

    // Descarga genérica (capturas)
    suspend fun bytes(url: String): ByteArray = withContext(Dispatchers.IO) { get(url) }

    private fun get(url: String): ByteArray {
        try {
            val c = URL(url).openConnection() as HttpURLConnection
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

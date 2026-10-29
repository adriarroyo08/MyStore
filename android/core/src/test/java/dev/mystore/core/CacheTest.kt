package dev.mystore.core

import org.junit.Assert.assertEquals
import org.junit.Test

class CacheTest {
    private fun app(id: String, code: Long) = StoreApp(id, id.substringBeforeLast('-'), "phone", id, code, "", 0, "", "", "", "")
    private val day = 86_400_000L

    @Test fun staleApksRules() {
        val now = 100 * day
        val apps = listOf(app("a-phone", 3), app("b-phone", 5))
        val files = listOf(
            "a-phone-3.apk" to now,            // en la tienda, reciente: se queda
            "gone-phone-1.apk" to now,         // ya no está en la tienda
            "b-phone-4.apk" to now,            // instalada una más nueva
            "a-phone-2.apk" to now - 31 * day, // más de 30 días sin usarse
            "b-phone-5.apk.part" to now - 2 * day, // descarga a medias abandonada
        )
        assertEquals(
            listOf("gone-phone-1.apk", "b-phone-4.apk", "a-phone-2.apk", "b-phone-5.apk.part"),
            staleApks(files, apps, mapOf("b-phone" to 5L), now),
        )
    }

    @Test fun parsesCategory() {
        val a = parseApps("""[{"id":"x-phone","package":"x","type":"phone","label":"X","versionCode":1,"category":"Juegos"}]""", "")
        assertEquals("Juegos", a[0].category)
    }

    @Test fun autoOrderPutsOwnPackageLast() {
        val list = listOf(app("dev.mystore-phone", 2), app("z-phone", 2), app("a-phone", 2))
        assertEquals(listOf("z-phone", "a-phone", "dev.mystore-phone"), autoUpdateOrder(list, "dev.mystore").map { it.id })
    }
}

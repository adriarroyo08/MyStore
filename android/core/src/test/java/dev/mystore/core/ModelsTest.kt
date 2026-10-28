package dev.mystore.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ModelsTest {
    @Test fun actionForStates() {
        assertEquals(AppAction.INSTALL, actionFor(5, null))
        assertEquals(AppAction.UPDATE, actionFor(5, 4))
        assertEquals(AppAction.OPEN, actionFor(5, 5))
        assertEquals(AppAction.OPEN, actionFor(5, 6))
    }

    @Test fun summaryIsFirstNonBlankLine() {
        val a = StoreApp("i", "p", "phone", "L", 1, "1", 0, "\n  Hola mundo \nmás", "", "", "")
        assertEquals("Hola mundo", a.summary)
        assertEquals("", a.copy(description = "").summary)
    }

    @Test fun normalizesUrls() {
        assertEquals("https://tienda.com", normalizeStoreUrl("tienda.com/"))
        assertEquals("http://192.168.1.2:8080", normalizeStoreUrl("http://192.168.1.2:8080"))
        assertNull(normalizeStoreUrl("ftp://x"))
        assertNull(normalizeStoreUrl(""))
    }

    @Test fun parsesApps() {
        val json = """[{"id":"a-phone","package":"a","type":"phone","label":"A","versionCode":3,
            "versionName":"1.2","minSdk":30,"size":100,"description":"d","notes":"n","hasIcon":true,
            "updatedAt":"x","iconUrl":"/api/apps/a-phone/icon","apkUrl":"/api/apps/a-phone/apk"}]"""
        val a = parseApps(json, "http://h:8080").single()
        assertEquals("http://h:8080/api/apps/a-phone/icon", a.iconUrl)
        assertEquals("http://h:8080/api/apps/a-phone/apk", a.apkUrl)
        assertEquals("a", a.pkg)
        assertEquals(3L, a.versionCode)
    }

    @Test fun parsesScreenshots() {
        val j = """[{"id":"a-phone","package":"a","type":"phone","label":"A","versionCode":1,"updatedAt":"2026-10-27T10:00:00Z",
            "screenshots":["/api/apps/a-phone/screenshots/0","/api/apps/a-phone/screenshots/1"]},
            {"id":"b","package":"b","type":"phone","label":"B","versionCode":1}]"""
        val (a, b) = parseApps(j, "http://h:8080")
        assertEquals(listOf("http://h:8080/api/apps/a-phone/screenshots/0", "http://h:8080/api/apps/a-phone/screenshots/1"), a.screenshots)
        assertEquals("2026-10-27T10:00:00Z", a.updatedAt)
        assertEquals(emptyList<String>(), b.screenshots)
    }

    @Test fun searchIgnoresCaseAndAccents() {
        val app = StoreApp("c", "dev.cam", "phone", "Cámara", 1, "1", 1, "", "", "", "")
        assertEquals(listOf(app), listOf(app).search("camara"))
        assertEquals(listOf(app), listOf(app).search("DEV.CAM"))
        assertEquals(emptyList<StoreApp>(), listOf(app).search("zzz"))
    }

    @Test fun parsesVersionsAndBuildsInstallableCopy() {
        val j = """[{"id":"a-phone","package":"a","type":"phone","label":"A","versionCode":3,"apkUrl":"/api/apps/a-phone/apk",
            "versions":[{"versionCode":2,"versionName":"1.1","notes":"n","size":7,"updatedAt":"2026-10-01T10:00:00Z",
            "apkUrl":"/api/apps/a-phone/versions/2/apk"}]},
            {"id":"b","package":"b","type":"phone","label":"B","versionCode":1,"versions":null}]"""
        val (a, b) = parseApps(j, "http://h")
        val v = a.versions.single()
        assertEquals(StoreVersion(2, "1.1", "n", 7, "2026-10-01T10:00:00Z", "http://h/api/apps/a-phone/versions/2/apk"), v)
        assertEquals(emptyList<StoreVersion>(), b.versions)
        val old = a.asVersion(v)
        assertEquals(listOf("a-phone", "a", "1.1", "http://h/api/apps/a-phone/versions/2/apk"), listOf(old.id, old.pkg, old.versionName, old.apkUrl))
        assertEquals(listOf(2L, 7L), listOf(old.versionCode, old.size))
    }
}

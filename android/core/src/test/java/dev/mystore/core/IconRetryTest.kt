package dev.mystore.core

import org.junit.Assert.assertEquals
import org.junit.Test

class IconRetryTest {
    private fun app(id: String, v: Long) = StoreApp(id, "p.$id", "phone", id, v, "", 0, "", "", "", "")

    @Test fun keyIncludesVersion() = assertEquals("a:3", iconKey(app("a", 3)))

    @Test fun retriesOnlyFailedStillInCatalog() {
        val apps = listOf(app("a", 1), app("b", 2), app("c", 3))
        // b falló; c falló con una versión vieja (ya no está); z ya no está en la tienda
        val failed = setOf("b:2", "c:2", "z:1")
        assertEquals(listOf("b"), iconsToRetry(failed, apps).map { it.id })
    }

    @Test fun nothingFailedNothingToRetry() = assertEquals(emptyList<StoreApp>(), iconsToRetry(emptySet(), listOf(app("a", 1))))
}

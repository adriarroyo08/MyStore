package dev.mystore.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class UpdatesTest {
    private fun app(id: String, v: Long) = StoreApp(id, "p.$id", "phone", id.uppercase(), v, "", 0, "", "", "", "")

    @Test fun pendingOnlyInstalledAndOlder() {
        val apps = listOf(app("a", 5), app("b", 5), app("c", 5), app("d", 5))
        val installed = mapOf("a" to 4L, "b" to 5L, "c" to null, "d" to 1L)
        assertEquals(listOf("a", "d"), pendingUpdates(apps, installed).map { it.id })
    }

    @Test fun newSetDetection() {
        assertTrue(isNewSet(emptySet(), setOf("a:1")))
        assertTrue(isNewSet(setOf("a:1"), setOf("a:1", "b:2")))
        assertTrue(isNewSet(setOf("a:1"), setOf("a:2")))
        assertFalse(isNewSet(setOf("a:1", "b:2"), setOf("a:1")))
        assertFalse(isNewSet(setOf("a:1"), emptySet()))
    }

    @Test fun textListsLabels() {
        assertEquals("Workstation, PokePet", updatesText(listOf("Workstation", "PokePet")))
        assertEquals("3 apps: A, B, C", updatesText(listOf("A", "B", "C")))
    }
}

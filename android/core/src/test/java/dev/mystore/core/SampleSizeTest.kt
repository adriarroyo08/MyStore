package dev.mystore.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class SampleSizeTest {
    @Test fun bigImageFitsTarget() {
        val s = sampleSize(4096, 4096, 400)
        assertTrue(4096 / s <= 400)
        assertTrue(4096 / s > 200)
    }

    @Test fun smallImageUnchanged() = assertEquals(1, sampleSize(300, 200, 400))

    @Test fun exactMultiple() = assertEquals(4, sampleSize(1600, 900, 400))
}

package dev.mystore.core

import com.sun.net.httpserver.HttpServer
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.net.InetSocketAddress

class StoreClientTest {
    private var server: HttpServer? = null

    @After fun stop() { server?.stop(0) }

    private fun serve(code: Int, body: String): String {
        val s = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        s.createContext("/") { ex ->
            val b = body.toByteArray()
            ex.sendResponseHeaders(code, b.size.toLong())
            ex.responseBody.use { it.write(b) }
        }
        s.start()
        server = s
        return "http://127.0.0.1:${s.address.port}"
    }

    @Test fun clientReportsUnreachableStore() {
        try {
            runBlocking { StoreClient("http://127.0.0.1:1").apps("phone") }
            fail("debía lanzar")
        } catch (e: StoreException) {
            assertTrue(e.message!!.isNotEmpty())
        }
    }

    @Test fun clientReportsHttpError() {
        val url = serve(500, """{"error":"x"}""")
        try {
            runBlocking { StoreClient(url).apps("phone") }
            fail("debía lanzar")
        } catch (e: StoreException) {
            assertEquals("x", e.message)
        }
    }

    @Test fun clientParsesList() {
        val url = serve(200, """[{"id":"a-phone","package":"a","type":"phone","label":"A","versionCode":1,
            "versionName":"1","size":1,"description":"","notes":"","iconUrl":"/i","apkUrl":"/p"}]""")
        assertEquals(1, runBlocking { StoreClient(url).apps("phone") }.size)
    }
}

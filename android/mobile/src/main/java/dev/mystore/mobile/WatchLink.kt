package dev.mystore.mobile

import android.content.Context
import com.google.android.gms.tasks.Tasks
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

// Mensajes a la app MyStore del reloj (mismo applicationId y firma); la recibe WatchListener en :wear
const val INSTALL_PATH = "/mystore/install"

suspend fun watchConnected(context: Context): Boolean = withContext(Dispatchers.IO) {
    runCatching { Tasks.await(Wearable.getNodeClient(context).connectedNodes).isNotEmpty() }.getOrDefault(false)
}

// Devuelve si llegó a algún reloj
suspend fun installOnWatch(context: Context, pkg: String): Boolean = withContext(Dispatchers.IO) {
    runCatching {
        val nodes = Tasks.await(Wearable.getNodeClient(context).connectedNodes)
        val msg = Wearable.getMessageClient(context)
        nodes.count { n -> runCatching { Tasks.await(msg.sendMessage(n.id, INSTALL_PATH, pkg.toByteArray())) }.isSuccess } > 0
    }.getOrDefault(false)
}

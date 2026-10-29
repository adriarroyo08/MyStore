package dev.mystore.wear

import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.WearableListenerService
import dev.mystore.core.UpdateChecks

// "Instalar en el reloj" desde el móvil: descarga e instala en segundo plano; si Android pide
// confirmación, sale una notificación "Toca para instalar"
class WatchListener : WearableListenerService() {
    override fun onMessageReceived(event: MessageEvent) {
        if (event.path == "/mystore/install") UpdateChecks.installRemote(this, String(event.data), "watch")
    }
}

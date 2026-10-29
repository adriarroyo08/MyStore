package dev.mystore.mobile

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewModelScope
import dev.mystore.core.StoreClient
import dev.mystore.core.StoreViewModel
import kotlinx.coroutines.launch

class PhoneViewModel(app: Application) : StoreViewModel(app, "phone") {
    // Filtro de la lista; vive aquí para que la notificación pueda abrir "Actualizaciones"
    var filter by mutableStateOf(Filter.ALL)
    var category by mutableStateOf<String?>(null)
    var sortRecent by mutableStateOf(false)

    // Paquetes con versión de reloj en la tienda y si hay un reloj conectado, para "Instalar en el reloj"
    var watchPkgs by mutableStateOf<Set<String>>(emptySet())
        private set
    var watchOnline by mutableStateOf(false)
        private set

    fun loadWatch() {
        if (storeUrl.isEmpty()) return
        viewModelScope.launch {
            watchOnline = watchConnected(getApplication())
            if (watchOnline) runCatching { watchPkgs = StoreClient(storeUrl).apps("watch").map { it.pkg }.toSet() }
        }
    }

    suspend fun sendToWatch(pkg: String): Boolean = installOnWatch(getApplication(), pkg)
}

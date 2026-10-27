package dev.mystore.mobile

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import dev.mystore.core.StoreViewModel

class PhoneViewModel(app: Application) : StoreViewModel(app, "phone") {
    // Filtro de la lista; vive aquí para que la notificación pueda abrir "Actualizaciones"
    var filter by mutableStateOf(Filter.ALL)
}

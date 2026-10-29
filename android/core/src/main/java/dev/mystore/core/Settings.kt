package dev.mystore.core

import android.content.Context

class StoreSettings(context: Context) {
    private val prefs = context.getSharedPreferences("mystore", Context.MODE_PRIVATE)
    var storeUrl: String?
        get() = prefs.getString("storeUrl", null)
        set(v) = prefs.edit().putString("storeUrl", v).apply()

    // Actualizar solas las apps instaladas desde la tienda (cargando y con WiFi)
    var autoUpdate: Boolean
        get() = prefs.getBoolean("autoUpdate", true)
        set(v) = prefs.edit().putBoolean("autoUpdate", v).apply()
}

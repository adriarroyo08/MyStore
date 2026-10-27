package dev.mystore.core

import android.content.Context

class StoreSettings(context: Context) {
    private val prefs = context.getSharedPreferences("mystore", Context.MODE_PRIVATE)
    var storeUrl: String?
        get() = prefs.getString("storeUrl", null)
        set(v) = prefs.edit().putString("storeUrl", v).apply()
}

package dev.mystore.wear

import android.app.Application
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.wear.compose.material.MaterialTheme
import androidx.activity.result.contract.ActivityResultContracts
import dev.mystore.core.StoreViewModel
import dev.mystore.core.UpdateChecks
import dev.mystore.core.needsNotificationPermission

class WatchViewModel(app: Application) : StoreViewModel(app, "watch", shotTarget = 400)

class MainActivity : ComponentActivity() {
    private val vm: WatchViewModel by viewModels()

    private val notifPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        UpdateChecks.schedule(this, "watch")
        // Solo en debug: `am start ... --ez checkNow true` lanza la comprobación de actualizaciones al momento
        if (applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE != 0 &&
            intent.getBooleanExtra("checkNow", false)) UpdateChecks.checkNow(this, "watch")
        // EXTRA_SHOW_UPDATES: la lista ya marca las actualizaciones, no hace falta nada más
        setContent {
            MaterialTheme { WearApp(vm, onPermission = { startActivity(vm.installer.permissionIntent()) }) }
        }
    }

    // Al volver del instalador o de ajustes se recalcula qué está instalado
    override fun onResume() {
        super.onResume()
        vm.installer.refresh()
        // Se pide el permiso una sola vez, cuando ya hay tienda configurada
        val prefs = getSharedPreferences("mystore_updates", MODE_PRIVATE)
        if (vm.storeUrl.isNotEmpty() && needsNotificationPermission(this) && !prefs.getBoolean("asked", false)) {
            prefs.edit().putBoolean("asked", true).apply()
            notifPermission.launch(android.Manifest.permission.POST_NOTIFICATIONS)
        }
    }
}

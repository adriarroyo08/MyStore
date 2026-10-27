package dev.mystore.mobile

import android.content.Intent
import android.content.pm.ApplicationInfo
import android.os.Bundle
import android.Manifest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.snapshotFlow
import androidx.lifecycle.lifecycleScope
import dev.mystore.core.EXTRA_SHOW_UPDATES
import dev.mystore.core.Screen
import dev.mystore.core.UpdateChecks
import dev.mystore.core.needsNotificationPermission
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier

class MainActivity : ComponentActivity() {
    private val vm: PhoneViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        UpdateChecks.schedule(this, "phone")
        // Solo en debug: `am start ... --ez checkNow true` lanza la comprobación de actualizaciones al momento
        if (applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE != 0 &&
            intent.getBooleanExtra("checkNow", false)) UpdateChecks.checkNow(this, "phone")
        handleIntent(intent)
        askNotificationsOnce()
        setContent {
            MyStoreTheme {
                Surface(Modifier.fillMaxSize()) {
                    StoreScreen(vm, onPermission = { startActivity(vm.installer.permissionIntent()) })
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        handleIntent(intent)
    }

    private val notifPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    // Pide permiso de notificaciones una sola vez, cuando ya hay tienda configurada
    private fun askNotificationsOnce() {
        val prefs = getSharedPreferences("mystore", MODE_PRIVATE)
        val v = vm // crear el ViewModel fuera del snapshot de solo lectura de snapshotFlow
        lifecycleScope.launch {
            snapshotFlow { v.screen != Screen.Setup }.first { it }
            if (needsNotificationPermission(this@MainActivity) && !prefs.getBoolean("notifAsked", false)) {
                prefs.edit().putBoolean("notifAsked", true).apply()
                notifPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }
    }

    // Tocar la notificación de actualizaciones abre la lista filtrada
    private fun handleIntent(i: Intent) {
        if (i.getBooleanExtra(EXTRA_SHOW_UPDATES, false)) {
            vm.filter = Filter.UPDATES
            vm.query = ""
            if (vm.screen != Screen.Setup) vm.screen = Screen.List
        }
        debugNav(i)
    }

    // Solo en debug: abrir pantallas por adb para capturas (--es open settings|<id> --es q <texto> --ei scroll <px>)
    private fun debugNav(i: Intent) {
        if (applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE == 0) return
        i.getStringExtra("open")?.let { vm.screen = if (it == "settings") Screen.Settings else Screen.Detail(it) }
        i.getStringExtra("q")?.let { vm.query = it }
        debugScroll = i.getIntExtra("scroll", 0)
    }

    // Al volver (del instalador o de ajustes) se recalcula qué está instalado
    override fun onResume() {
        super.onResume()
        vm.installer.refresh()
    }
}

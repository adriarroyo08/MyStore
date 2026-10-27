package dev.mystore.wear

import android.app.Application
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.wear.compose.material.MaterialTheme
import dev.mystore.core.StoreViewModel

class WatchViewModel(app: Application) : StoreViewModel(app, "watch", shotTarget = 400)

class MainActivity : ComponentActivity() {
    private val vm: WatchViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme { WearApp(vm, onPermission = { startActivity(vm.installer.permissionIntent()) }) }
        }
    }

    // Al volver del instalador o de ajustes se recalcula qué está instalado
    override fun onResume() {
        super.onResume()
        vm.installer.refresh()
    }
}

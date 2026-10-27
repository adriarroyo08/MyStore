package dev.mystore.mobile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface

class MainActivity : ComponentActivity() {
    private val vm: StoreViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                Surface { StoreScreen(vm, onPermission = { startActivity(vm.installer.permissionIntent()) }) }
            }
        }
    }

    // Al volver (del instalador o de ajustes) se recalcula qué está instalado
    override fun onResume() {
        super.onResume()
        vm.installer.refresh()
    }
}

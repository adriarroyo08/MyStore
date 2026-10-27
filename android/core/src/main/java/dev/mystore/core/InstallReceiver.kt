package dev.mystore.core

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInstaller
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.update

// Estado compartido entre Installer y el receiver (mismo proceso)
internal val installStates = MutableStateFlow<Map<String, InstallState>>(emptyMap())

internal fun setInstallState(id: String, s: InstallState) = installStates.update { it + (id to s) }

class InstallReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val id = intent.getStringExtra(EXTRA_APP_ID) ?: return
        when (val code = intent.getIntExtra(PackageInstaller.EXTRA_STATUS, -999)) {
            PackageInstaller.STATUS_PENDING_USER_ACTION -> {
                @Suppress("DEPRECATION")
                val confirm = intent.getParcelableExtra<Intent>(Intent.EXTRA_INTENT)
                if (confirm == null) {
                    setInstallState(id, InstallState.Failed("El sistema no mostró el instalador"))
                } else {
                    try {
                        context.startActivity(confirm.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
                    } catch (e: Exception) {
                        setInstallState(id, InstallState.Failed("No se pudo abrir el instalador"))
                    }
                }
            }
            // Éxito o cancelado por el usuario: el botón vuelve a calcularse solo
            PackageInstaller.STATUS_SUCCESS, PackageInstaller.STATUS_FAILURE_ABORTED ->
                setInstallState(id, InstallState.Idle)
            else -> {
                val msg = intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE).orEmpty()
                setInstallState(id, InstallState.Failed("No se pudo instalar ($code) $msg".trim()))
            }
        }
    }

    companion object { const val EXTRA_APP_ID = "mystore.appId" }
}

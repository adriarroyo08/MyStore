package dev.mystore.spike

import android.app.Activity
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageInstaller
import android.net.Uri
import android.os.Bundle
import android.provider.Settings
import android.view.Gravity
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import java.net.URL

// Desechable: comprueba si una app de Wear OS puede instalar otra APK sin adb.
class MainActivity : Activity() {
    private lateinit var status: TextView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        status = TextView(this).apply { gravity = Gravity.CENTER; textSize = 12f; text = BuildConfig.APK_URL }
        val button = Button(this).apply { text = "Instalar"; setOnClickListener { install() } }
        setContentView(LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER
            setPadding(24, 24, 24, 24)
            addView(button); addView(status)
        })
        intent.getStringExtra("status")?.let { status.text = it }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        intent.getStringExtra("status")?.let { status.text = it }
    }

    private fun install() {
        if (!packageManager.canRequestPackageInstalls()) {
            status.text = "Falta permiso: abriendo ajustes"
            try {
                startActivity(Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:$packageName")))
            } catch (e: Exception) {
                status.text = "No hay pantalla de permisos: $e"
            }
            return
        }
        status.text = "Descargando..."
        Thread {
            try {
                val installer = packageManager.packageInstaller
                val params = PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL)
                val id = installer.createSession(params)
                installer.openSession(id).use { session ->
                    URL(BuildConfig.APK_URL).openStream().use { input ->
                        session.openWrite("app.apk", 0, -1).use { out -> input.copyTo(out); session.fsync(out) }
                    }
                    val pi = PendingIntent.getBroadcast(this, id, Intent(this, InstallReceiver::class.java),
                        PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE)
                    session.commit(pi.intentSender)
                }
                runOnUiThread { status.text = "Enviado al instalador" }
            } catch (e: Exception) {
                runOnUiThread { status.text = "Error: $e" }
            }
        }.start()
    }
}

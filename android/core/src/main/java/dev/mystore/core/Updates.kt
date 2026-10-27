package dev.mystore.core

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.work.*
import java.util.concurrent.TimeUnit

const val EXTRA_SHOW_UPDATES = "dev.mystore.SHOW_UPDATES"

fun pendingUpdates(apps: List<StoreApp>, installed: Map<String, Long?>): List<StoreApp> =
    apps.filter { actionFor(it.versionCode, installed[it.id]) == AppAction.UPDATE }

// Solo se avisa si aparece algo que no se había notificado
fun isNewSet(previous: Set<String>, current: Set<String>): Boolean = !previous.containsAll(current)

fun updatesText(labels: List<String>): String =
    if (labels.size >= 3) "${labels.size} apps: ${labels.joinToString(", ")}" else labels.joinToString(", ")

fun needsNotificationPermission(context: Context): Boolean =
    Build.VERSION.SDK_INT >= 33 &&
        context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED

class UpdateWorker(private val ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        try {
            val url = StoreSettings(ctx).storeUrl?.takeIf { it.isNotBlank() } ?: return Result.success()
            val apps = StoreClient(url).apps(inputData.getString("type") ?: "phone")
            val installer = Installer(ctx)
            val pending = pendingUpdates(apps, apps.associate { it.id to installer.installedVersion(it.pkg) })
            val nm = ctx.getSystemService(NotificationManager::class.java)
            val prefs = ctx.getSharedPreferences("mystore_updates", Context.MODE_PRIVATE)
            if (pending.isEmpty()) {
                nm.cancel(NOTIF_ID)
                prefs.edit().remove("notified").apply()
                return Result.success()
            }
            val current = pending.map { "${it.id}:${it.versionCode}" }.toSet()
            if (!isNewSet(prefs.getStringSet("notified", emptySet()).orEmpty(), current)) return Result.success()
            if (needsNotificationPermission(ctx)) return Result.success()
            nm.createNotificationChannel(NotificationChannel("updates", "Actualizaciones", NotificationManager.IMPORTANCE_DEFAULT))
            val launch = ctx.packageManager.getLaunchIntentForPackage(ctx.packageName)?.putExtra(EXTRA_SHOW_UPDATES, true)
            val tap = launch?.let { PendingIntent.getActivity(ctx, 0, it, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE) }
            nm.notify(NOTIF_ID, NotificationCompat.Builder(ctx, "updates")
                .setSmallIcon(R.drawable.ic_stat_update)
                .setContentTitle("Actualizaciones disponibles")
                .setContentText(updatesText(pending.map { it.label }))
                .setContentIntent(tap).setAutoCancel(true).build())
            prefs.edit().putStringSet("notified", current).apply()
        } catch (e: Exception) {
            // Red caída u otro fallo: se reintenta en el siguiente periodo
        }
        return Result.success()
    }

    private companion object { const val NOTIF_ID = 1001 }
}

object UpdateChecks {
    private fun data(type: String) = workDataOf("type" to type)

    fun schedule(context: Context, type: String) {
        val req = PeriodicWorkRequestBuilder<UpdateWorker>(6, TimeUnit.HOURS)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .setInputData(data(type)).build()
        WorkManager.getInstance(context).enqueueUniquePeriodicWork("mystore-updates", ExistingPeriodicWorkPolicy.KEEP, req)
    }

    fun checkNow(context: Context, type: String) {
        WorkManager.getInstance(context).enqueue(OneTimeWorkRequestBuilder<UpdateWorker>().setInputData(data(type)).build())
    }
}

import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "dev.mystore.mobile"
    compileSdk = 37
    defaultConfig {
        applicationId = "dev.mystore"
        minSdk = 30
        targetSdk = 35
        versionCode = (findProperty("versionCode") as String? ?: "1").toInt()
        versionName = findProperty("versionName") as String? ?: "1.0"
    }
    // Misma firma en debug y release para que la app se autoactualice.
    // Sin keystore.properties se usa la clave debug por defecto.
    val ksFile = rootProject.file("keystore.properties")
    if (ksFile.exists()) {
        val ks = Properties()
        ksFile.inputStream().use { ks.load(it) }
        signingConfigs.create("store") {
            storeFile = rootProject.file(ks.getProperty("storeFile"))
            storePassword = ks.getProperty("storePassword")
            keyAlias = ks.getProperty("keyAlias")
            keyPassword = ks.getProperty("keyPassword")
        }
        buildTypes.all { signingConfig = signingConfigs.getByName("store") }
    } else {
        buildTypes.all { signingConfig = signingConfigs.getByName("debug") }
    }
    buildFeatures { compose = true }
    lint { abortOnError = false }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(project(":core"))
    implementation(platform("androidx.compose:compose-bom:2026.09.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-core")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.core:core-ktx:1.19.1")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.4")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.11.0")
}

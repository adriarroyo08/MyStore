plugins { id("com.android.application") }

android {
    namespace = "dev.mystore.spike"
    compileSdk = 36
    defaultConfig {
        applicationId = "dev.mystore.spike"
        minSdk = 30
        targetSdk = 35
        versionCode = 1
        versionName = "1.0"
        val url = (project.findProperty("apkUrl") as String?) ?: "http://192.168.18.3:8000/watch-v1.apk"
        buildConfigField("String", "APK_URL", "\"$url\"")
    }
    buildFeatures { buildConfig = true }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

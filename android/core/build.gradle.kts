plugins { id("com.android.library") }

android {
    namespace = "dev.mystore.core"
    compileSdk = 37
    defaultConfig { minSdk = 30 }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation(platform("androidx.compose:compose-bom:2026.09.00"))
    api("androidx.compose.runtime:runtime")
    implementation("androidx.core:core-ktx:1.19.1")
    api("androidx.work:work-runtime-ktx:2.10.0")
    implementation("androidx.collection:collection:1.5.0")
    api("androidx.compose.ui:ui")
    api("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.4")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.11.0")
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20240303")
}

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

// Inject this project's SDK signing parameters without committing them.
val httpDnsFile = providers.environmentVariable("RUI_HTTPDNS_CONFIG").orNull
    ?.let { file(it) } ?: rootProject.file("HTTPDNS.private.json")
val httpDnsValues = if (httpDnsFile.isFile) {
    groovy.json.JsonSlurper().parse(httpDnsFile) as Map<*, *>
} else emptyMap<String, String>()
// Accept both the local minimal config and the framework-neutral r2 attachment.
val httpDnsSDK = (httpDnsValues["httpdns"] as? Map<*, *>) ?: httpDnsValues
val httpDnsAccount = (httpDnsSDK["account_id"] ?: httpDnsSDK["accountId"])?.toString()?.trim().orEmpty()
val httpDnsSecret = ((httpDnsSDK["secret_key"] ?: httpDnsSDK["secretKey"]) as? String)?.trim().orEmpty()
fun javaLiteral(value: String) = groovy.json.JsonOutput.toJson(value)

android {
    namespace = "com.balabalaim.gcloud.tim.push.appdev"
    compileSdk = 37

    defaultConfig {
        buildConfigField("String", "HTTPDNS_ACCOUNT_ID", javaLiteral(httpDnsAccount))
        buildConfigField("String", "HTTPDNS_SECRET_KEY", javaLiteral(httpDnsSecret))
        applicationId = "com.balabalaim.gcloud.tim.push.appdev"
        minSdk = 24
        targetSdk = 36
        versionCode = 15
        versionName = "1.7.8"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        create("release") {
            storeFile = file("balabala.jks")
            storePassword = "balabala"
            keyAlias = "balabala.jks"
            keyPassword = "balabala"
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("release")
            ndk {
                abiFilters += listOf("armeabi-v7a", "arm64-v8a")
            }
            optimization {
                enable = false
            }
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    buildFeatures {
        buildConfig = true
        compose = true
    }
    packaging {
        resources {
            excludes += setOf(
                "META-INF/DEPENDENCIES",
                "META-INF/LICENSE*",
                "META-INF/NOTICE*"
            )
        }
    }
}

dependencies {
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.process)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.appcompat)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.okhttp)
    implementation("com.aliyun.ams:alicloud-android-httpdns:2.7.0")
    implementation(libs.moshi)
    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.coil.compose)
    implementation(libs.tencent.imsdk)
    implementation(libs.tencent.trtc)
    implementation(libs.tencent.timpush)
    testImplementation(libs.junit)
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    androidTestImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
    testImplementation("org.json:json:20240303")
    testImplementation("org.json:json:20180813")
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(libs.androidx.junit)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    debugImplementation(libs.androidx.compose.ui.tooling)
}

val verifyHttpDnsRelease by tasks.registering {
    val configured = (httpDnsAccount.toLongOrNull() ?: 0) > 0 && httpDnsSecret.matches(Regex("[0-9a-fA-F]{32}"))
    doLast {
        check(configured) {
            "Release requires Rui HTTPDNS accountId/secretKey in RUI_HTTPDNS_CONFIG or HTTPDNS.private.json"
        }
    }
}
tasks.matching { it.name == "preReleaseBuild" }.configureEach { dependsOn(verifyHttpDnsRelease) }

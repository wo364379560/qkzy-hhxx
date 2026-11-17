package com.balabalaim.gcloud.tim.push.appdev.data.network

import okhttp3.Authenticator
import okhttp3.Interceptor
import okhttp3.MediaType
import okhttp3.OkHttpClient
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okio.BufferedSink

/** Retry/failover belongs to the caller, where method, session and deadline are known. */
internal object LinLanHttpTransport {
    fun configure(builder: OkHttpClient.Builder): OkHttpClient.Builder = builder
        .retryOnConnectionFailure(false)
        .followRedirects(false)
        .followSslRedirects(false)
        .authenticator(Authenticator.NONE)
        .proxyAuthenticator(Authenticator.NONE)
        .addInterceptor { chain ->
            val original = chain.request()
            val state = CallState()
            val prepared = original.newBuilder().tag(CallState::class.java, state)
            if (original.method !in setOf("GET", "HEAD")) {
                // Also protect bodyless DELETE/OPTIONS against status-code follow-ups.
                val body = original.body ?: ByteArray(0).toRequestBody()
                prepared.method(original.method, OneShotBody(body))
            }
            val response = chain.proceed(prepared.build())
            val headers = state.retryAfter ?: return@addInterceptor response
            // Restore the server's exact Retry-After values for caller backoff/diagnostics.
            response.newBuilder().removeHeader("Retry-After").apply {
                headers.forEach { addHeader("Retry-After", it) }
            }.build()
        }
        .addNetworkInterceptor { chain ->
            val response = chain.proceed(chain.request())
            // OkHttp 4.12 retries 503/Retry-After:0 even with connection retry disabled.
            // Shield it inside its follow-up layer, then restore it outside that layer.
            if (response.code == 503 && response.header("Retry-After")?.matches(Regex("0+")) == true) {
                chain.request().tag(CallState::class.java)!!.retryAfter = response.headers.values("Retry-After")
                response.newBuilder().header("Retry-After", "1").build()
            } else response
        }

    private class CallState { var retryAfter: List<String>? = null }

    private class OneShotBody(private val delegate: RequestBody) : RequestBody() {
        override fun contentType(): MediaType? = delegate.contentType()
        override fun contentLength(): Long = delegate.contentLength()
        override fun isDuplex(): Boolean = delegate.isDuplex()
        override fun isOneShot(): Boolean = true
        override fun writeTo(sink: BufferedSink) = delegate.writeTo(sink)
    }
}

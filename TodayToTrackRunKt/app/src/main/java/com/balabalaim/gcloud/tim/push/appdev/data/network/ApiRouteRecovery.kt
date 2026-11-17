package com.balabalaim.gcloud.tim.push.appdev.data.network

import java.io.IOException
import kotlinx.coroutines.CancellationException
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.Request

/** Shared by real API calls and isolated wire tests; never sends a business request itself. */
internal class ApiRouteRecovery(
    private val currentBaseUrl: () -> String,
    private val recover: suspend (failedOrigin: String, budgetMillis: Long) -> Unit,
    private val prepareFuture: (failedOrigin: String) -> Unit,
    private val nowMillis: () -> Long,
) {
    suspend fun nextReadRequest(
        request: Request,
        attempt: Int,
        error: Throwable,
        deadline: Long,
        sessionIsCurrent: suspend () -> Boolean,
    ): Request? {
        if (!isRouteFailure(error)) return null
        if (!sessionIsCurrent()) throw CancellationException("账号已切换，请忽略旧请求结果")
        val failed = origin(request.url)
        val remaining = deadline - nowMillis()
        if (request.method != "GET" || attempt != 0 || remaining <= 0) {
            // A write/finished read returns its error now. Anonymous bounded probes
            // prepare a future action even when this call spent its whole deadline.
            prepareFuture(failed)
            return null
        }
        recover(failed, minOf(remaining, 8_000))
        if (!sessionIsCurrent()) throw CancellationException("账号已切换，请忽略旧请求结果")
        if (nowMillis() >= deadline) return null
        val next = currentBaseUrl().toHttpUrl()
        if (origin(next) == failed) return null
        return request.newBuilder().url(request.url.newBuilder()
            .scheme(next.scheme).host(next.host).port(next.port).build()).build()
    }

    companion object {
        fun isRouteFailure(error: Throwable): Boolean = when (error) {
            is ApiException -> !error.unauthorized && error.status in setOf(502, 503, 504)
            is IOException -> true
            else -> false
        }
        fun origin(url: HttpUrl): String = url.newBuilder().encodedPath("/")
            .query(null).fragment(null).build().toString().removeSuffix("/")
    }
}

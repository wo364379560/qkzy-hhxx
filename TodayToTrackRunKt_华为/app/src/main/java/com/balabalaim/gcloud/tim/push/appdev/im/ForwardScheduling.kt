package com.balabalaim.gcloud.tim.push.appdev.im

import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import java.util.concurrent.atomic.AtomicInteger

class AccountForwardPacer(private val intervalMS: Long = 500, private val now: () -> Long = { System.nanoTime() / 1_000_000 }, private val wait: suspend (Long) -> Unit = { delay(it) }) {
    private val mutex = Mutex()
    private var next = 0L
    private var pauseUntil = 0L
    suspend fun acquire(active: () -> Boolean) = mutex.withLock {
        while (true) {
            if (!active()) throw CancellationException("转发已停止")
            val remaining = maxOf(next, pauseUntil) - now()
            if (remaining <= 0) break
            wait(minOf(remaining, 100))
        }
        next = now() + intervalMS
    }
    suspend fun backoff(attempt: Int) = mutex.withLock { pauseUntil = maxOf(pauseUntil, now() + minOf(8000L, 1000L shl attempt.coerceIn(0, 3))) }
}

suspend fun <T> runBoundedForwardTargets(targets: List<T>, parallelism: Int, active: () -> Boolean,
                                         process: suspend (T) -> Unit, failed: suspend (T, Throwable) -> Unit) = supervisorScope {
    val cursor = AtomicInteger()
    repeat(minOf(parallelism.coerceAtLeast(1), targets.size)) {
        launch {
            while (active()) {
                val index = cursor.getAndIncrement()
                if (index >= targets.size) break
                val target = targets[index]
                try { process(target) } catch (error: Throwable) {
                    currentCoroutineContext().ensureActive()
                    failed(target, error)
                }
            }
        }
    }
}

class PermissionChanged : Exception("权限已变化，请重新确认")

/** Shared in-flight reads; invalidation fences late replies, including logout. */
class PermissionMemo<T>(private val scope: CoroutineScope, private val ttlMS: Long = 2000,
                         private val now: () -> Long = { System.nanoTime() / 1_000_000 }) {
    private data class Flight<T>(val generation: Long, val epoch: Long, val task: Deferred<T>)
    private val lock = Any()
    private val cache = LinkedHashMap<String, Pair<Long, T>>()
    private val pending = mutableMapOf<String, Flight<T>>()
    private val versions = mutableMapOf<String, Long>()
    private var epoch = 0L

    suspend fun get(key: String, load: suspend () -> T): T {
        val flight = synchronized(lock) {
            cache[key]?.takeIf { it.first > now() }?.let { return it.second }
            pending[key] ?: Flight(versions[key] ?: 0L, epoch, scope.async(start = CoroutineStart.LAZY) { load() }).also { pending[key] = it }
        }
        try {
            val value = flight.task.await()
            synchronized(lock) {
                if (flight.epoch != epoch || flight.generation != (versions[key] ?: 0L)) throw PermissionChanged()
                cache[key] = now() + ttlMS to value
                while (cache.size > 256) cache.remove(cache.keys.first())
            }
            return value
        } finally { synchronized(lock) { if (pending[key] === flight) pending.remove(key) } }
    }

    /** Re-read an invalidated snapshot; a logout/account change must never be retried. */
    suspend fun getCurrent(key: String, isCurrent: () -> Boolean,
                           retry: (Exception) -> Boolean = { it is PermissionChanged },
                           load: suspend () -> T): T {
        val sessionEpoch = synchronized(lock) { epoch }
        fun checkSession() {
            if (!isCurrent() || synchronized(lock) { epoch != sessionEpoch }) {
                throw CancellationException("账号已切换")
            }
        }
        repeat(3) { attempt ->
            currentCoroutineContext().ensureActive()
            checkSession()
            try {
                val value = get(key, load)
                currentCoroutineContext().ensureActive()
                checkSession()
                return value
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                checkSession()
                if (attempt == 2 || !retry(error)) throw error
                delay(100L * (attempt + 1))
            }
        }
        error("Unreachable")
    }
    fun invalidateGroup(groupID: String) = synchronized(lock) {
        (cache.keys + pending.keys + versions.keys).filter { it.endsWith(":$groupID") }.forEach { key -> cache.remove(key); pending.remove(key); versions[key] = (versions[key] ?: 0L) + 1L }
    }
    fun clear() = synchronized(lock) { epoch++; cache.clear(); pending.clear(); versions.clear() }
}

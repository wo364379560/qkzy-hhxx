package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.Context
import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanReliability
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ForwardPayload
import com.balabalaim.gcloud.tim.push.appdev.ui.main.ForwardTarget
import com.tencent.imsdk.v2.V2TIMManager
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

data class ForwardItemResult(val target: String, val name: String, val index: Int, val sourceId: String,
    val status: String = "pending", val error: String = "", val sdkId: String = "")
data class ForwardMetrics(val preparationMS: Long = 0, val permissionMS: Long = 0, val sendMS: Long = 0, val saveMS: Long = 0, val totalMS: Long = 0, val writes: Int = 0)
data class NativeForwardJob(val id: String, val owner: String, val sdkAppId: Int, val items: List<ForwardItemResult>,
    val running: Boolean = false, val resumable: Boolean = true, val metrics: ForwardMetrics = ForwardMetrics())

/** Three target workers share one account pacing budget; each target stays ordered. */
object LinLanForwardJobs {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val mutable = MutableStateFlow<NativeForwardJob?>(null)
    val current = mutable.asStateFlow()
    private val presented = MutableStateFlow<String?>(null)
    val presentedJobId = presented.asStateFlow()
    fun dismissResult(id: String) {
        if (presented.value == id) presented.value = null
    }
    private val pacers = mutableMapOf<String, AccountForwardPacer>()
    private var store: ForwardResultStore? = null
    private var epoch = 0L
    private var execution: Execution? = null
    private data class Execution(val initial: NativeForwardJob, val source: ForwardPayload, val targets: List<ForwardTarget>,
        val sessionGeneration: Long, val rows: MutableList<ForwardItemResult>, @Volatile var stopped: Boolean = false,
        var metrics: ForwardMetrics = ForwardMetrics(), var lastPublished: Long = 0, val initialized: CompletableDeferred<Unit> = CompletableDeferred())
    private fun now() = System.nanoTime() / 1_000_000
    private fun database(context: Context) = store ?: ForwardResultStore(context.applicationContext).also { store = it }
    private fun active(run: Execution) = execution === run && !run.stopped &&
        V2TIMManager.getInstance().loginUser == run.initial.owner && LinLanRuntimeConfig.sdkAppId == run.initial.sdkAppId &&
        LinLanAppContainer.sessionStore().sessionGeneration() == run.sessionGeneration

    fun restore(context: Context, owner: String, appId: Int) {
        val running = execution
        if (running != null && active(running) && mutable.value?.owner == owner && mutable.value?.sdkAppId == appId) return
        if (running == null && mutable.value?.owner == owner && mutable.value?.sdkAppId == appId) return
        running?.stopped = true
        execution = null
        val generation = ++epoch
        mutable.value = null
        presented.value = null
        val db = database(context)
        scope.launch {
            val saved = runCatching { db.restore(owner, appId) }.getOrNull()
            if (generation == epoch && execution == null) mutable.value = saved
        }
    }

    fun start(context: Context, source: ForwardPayload, selected: List<ForwardTarget>): Boolean {
        if (mutable.value?.running == true && execution?.let(::active) == true) return false
        if (source.messages.size !in 1..LinLanReliability.MAX_FORWARD_MESSAGES || selected.isEmpty()) return false
        val preparingAt = now()
        val owner = V2TIMManager.getInstance().loginUser.orEmpty(); if (owner.isBlank()) return false
        val appID = LinLanRuntimeConfig.sdkAppId
        val snapshot = source.copy(messages = source.messages.sortedWith(compareBy({ it.timestamp }, { it.seq })).toList())
        val targets = selected.distinctBy { it.key }.toList()
        val count = if (snapshot.merge) 1 else snapshot.messages.size
        val job = NativeForwardJob(UUID.randomUUID().toString(), owner, appID, targets.flatMap { target ->
            (0 until count).map { index -> ForwardItemResult(target.key, target.title, index,
                if (snapshot.merge) snapshot.messages.joinToString(",") { it.msgID.orEmpty() } else snapshot.messages[index].msgID.orEmpty()) }
        }, running = true)
        execution?.stopped = true
        val run = Execution(job, snapshot, targets, LinLanAppContainer.sessionStore().sessionGeneration(), job.items.toMutableList())
        run.metrics = run.metrics.copy(preparationMS = now() - preparingAt)
        execution = run; ++epoch; mutable.value = job; presented.value = job.id
        val db = database(context)
        scope.launch {
            val began = now()
            try {
                val time = now(); db.create(job); run.metrics = run.metrics.copy(saveMS = now() - time, writes = job.items.size + 1)
                run.initialized.complete(Unit)
                execute(run, db)
            } catch (error: Throwable) {
                run.stopped = true
                run.initialized.completeExceptionally(error)
                run.rows.replaceAll { if (it.status == "pending") it.copy(status = "failed", error = "任务无法可靠保存，未发送") else it }
            } finally {
                runCatching { db.finish(job.id) }
                run.metrics = run.metrics.copy(totalMS = now() - began)
                publish(run, running = false, force = true)
            }
        }
        return true
    }

    fun cancel() {
        val run = execution ?: return; run.stopped = true
        scope.launch {
            runCatching { run.initialized.await() }.onSuccess {
                cancelPending(run, store ?: return@onSuccess)
                publish(run, force = true)
            }
        }
    }
    fun retry() {
        val run = execution ?: return
        if (mutable.value?.running == true || !mutable.value!!.resumable ||
            V2TIMManager.getInstance().loginUser != run.initial.owner || LinLanRuntimeConfig.sdkAppId != run.initial.sdkAppId ||
            LinLanAppContainer.sessionStore().sessionGeneration() != run.sessionGeneration) return
        run.stopped = false; mutable.value = mutable.value!!.copy(running = true)
        scope.launch {
            val began = now()
            try { execute(run, store ?: return@launch) }
            finally { run.metrics = run.metrics.copy(totalMS = now() - began); publish(run, running = false, force = true) }
        }
    }

    private fun publish(run: Execution, running: Boolean = mutable.value?.running == true, force: Boolean = false) {
        if (execution !== run || mutable.value?.id != run.initial.id) return
        val time = now(); if (!force && time - run.lastPublished < 100) return
        run.lastPublished = time
        mutable.value = run.initial.copy(items = run.rows.toList(), running = running, metrics = run.metrics)
    }
    private suspend fun save(run: Execution, db: ForwardResultStore, position: Int, row: ForwardItemResult) {
        // Reserve this row on Main before suspending, so cancellation cannot turn a send into pending.
        run.rows[position] = row
        val time = now()
        try { db.update(run.initial.id, position, row) }
        catch (error: Throwable) {
            run.stopped = true
            run.rows[position] = row.copy(status = if (row.status == "sending") "failed" else "unknown", error = "结果保存未确认，已停止后续发送")
            throw error
        }
        run.metrics = run.metrics.copy(saveMS = run.metrics.saveMS + now() - time, writes = run.metrics.writes + 1)
        publish(run)
    }
    private suspend fun cancelPending(run: Execution, db: ForwardResultStore) {
        run.rows.indices.forEach { index ->
            val row = run.rows[index]
            if (row.status == "pending") runCatching { save(run, db, index, row.copy(status = "cancelled", error = "已停止，未发送")) }
        }
    }
    private fun limited(error: LinLanSendFailure) = error.code in setOf(60018, 10023, 7008, 60011) ||
        error.message.orEmpty().contains("frequency", true) || error.message.orEmpty().contains("频率")

    private suspend fun execute(run: Execution, db: ForwardResultStore) {
        val pacer = pacers.getOrPut("${run.initial.sdkAppId}:${run.initial.owner}") { AccountForwardPacer() }
        for (batch in run.targets.chunked(20)) {
        if (!active(run)) break
        runBoundedForwardTargets(batch, 3, { active(run) }, process = { target ->
            val positions = run.rows.indices.filter { run.rows[it].target == target.key }
            for (position in positions) {
                var row = run.rows[position]
                if (row.status in setOf("success", "cancelled")) continue
                if (row.status == "unknown" || row.status == "interrupted") break
                if (!active(run)) break
                var attempt = 0
                while (active(run)) {
                    var sdkStarted = false
                    try {
                        if (!target.groupId.isNullOrBlank()) {
                            val time = now(); val permission = LinLanImManager.resolveGroupSendPermission(target.groupId)
                            run.metrics = run.metrics.copy(permissionMS = run.metrics.permissionMS + now() - time)
                            if (!permission.allowsSend) throw LinLanSendFailure(permission.blockedHint, notDispatched = true)
                        }
                        pacer.acquire { active(run) }
                        if (!active(run) || run.rows[position].status == "cancelled") break
                        save(run, db, position, row.copy(status = "sending", error = ""))
                        if (!active(run)) { save(run, db, position, row.copy(status = "cancelled", error = "已停止，未发送")); break }
                        val time = now(); sdkStarted = true
                        val sent = if (run.source.merge) LinLanImManager.forwardMessagesMerged(run.source.messages, run.source.mergeTitle, target.receiver, target.groupId)
                            else LinLanImManager.forwardMessage(run.source.messages[row.index], target.receiver, target.groupId)
                        run.metrics = run.metrics.copy(sendMS = run.metrics.sendMS + now() - time)
                        save(run, db, position, row.copy(status = "success", error = "", sdkId = sent.msgID.orEmpty()))
                        break
                    } catch (error: Throwable) {
                        if (run.stopped && run.rows[position].error.contains("保存未确认")) break
                        val known = error as? LinLanSendFailure
                        val unknown = known?.unknown == true || sdkStarted && known == null
                        row = row.copy(status = if (unknown) "unknown" else if (!active(run)) "cancelled" else "failed",
                            error = if (unknown) "SDK结果待确认，不会盲目补发" else error.message ?: "发送失败")
                        runCatching { save(run, db, position, row) }
                        if (known != null && limited(known)) pacer.backoff(attempt)
                        if (!unknown && active(run) && known != null && limited(known) && attempt++ < 2) continue
                        break
                    }
                }
                if (run.rows[position].status != "success") break
            }
        }, failed = { _, _ -> /* A failed target is isolated; individual rows retain their outcome. */ })
        }
        if (!active(run)) cancelPending(run, db)
    }
}

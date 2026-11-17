package com.balabalaim.gcloud.tim.push.appdev.run

import android.Manifest
import android.app.*
import android.content.*
import android.content.pm.PackageManager
import android.location.*
import android.os.*
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.balabalaim.gcloud.tim.push.appdev.MainActivity
import com.balabalaim.gcloud.tim.push.appdev.R
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.json.JSONArray
import org.json.JSONObject
import java.io.File

/** Points are WGS84 from the system GPS, stored locally and drawn without a map SDK. */
data class TrackPoint(val latitude: Double, val longitude: Double, val time: Long, val segment: Int)
data class RunSession(val startedAt: Long = 0, val durationSeconds: Long = 0, val distanceMeters: Double = 0.0,
    val points: List<TrackPoint> = emptyList(), val active: Boolean = false, val paused: Boolean = false,
    val segment: Int = 0, val message: String = "准备出发", val owner: String = "")

object RunState {
    internal val mutable = MutableStateFlow(RunSession())
    val session = mutable.asStateFlow()
}

object RunRecords {
    private fun folder(context: Context, owner: String): File {
        val safe = java.security.MessageDigest.getInstance("SHA-256").digest(owner.toByteArray()).joinToString("") { "%02x".format(it) }
        return File(context.filesDir, "run_records/$safe").apply { mkdirs() }
    }
    fun encode(run: RunSession): String = JSONObject().put("startedAt", run.startedAt)
        .put("durationSeconds", run.durationSeconds).put("distanceMeters", run.distanceMeters)
        .put("segment", run.segment).put("owner", run.owner)
        .put("points", JSONArray().apply { run.points.forEach { put(JSONArray().put(it.latitude).put(it.longitude).put(it.time).put(it.segment)) } }).toString()
    fun decode(raw: String): RunSession {
        val json = JSONObject(raw); val points = json.getJSONArray("points")
        return RunSession(startedAt = json.getLong("startedAt"), durationSeconds = json.getLong("durationSeconds"),
            distanceMeters = json.getDouble("distanceMeters"), owner = json.optString("owner"), segment = json.optInt("segment"),
            points = (0 until points.length()).map { i -> val p = points.getJSONArray(i); TrackPoint(p.getDouble(0), p.getDouble(1), p.getLong(2), p.optInt(3)) })
    }
    @Synchronized fun save(context: Context, run: RunSession, draft: Boolean = false) {
        val file = File(folder(context, run.owner), if (draft) "draft.json" else "${run.startedAt}.json")
        val atomic = android.util.AtomicFile(file)
        val stream = atomic.startWrite()
        try { stream.write(encode(run).toByteArray()); atomic.finishWrite(stream) }
        catch (error: Exception) { atomic.failWrite(stream); throw error }
        if (!draft) android.util.AtomicFile(File(folder(context, run.owner), "draft.json")).delete()
    }
    fun list(context: Context, owner: String, includeDraft: Boolean = true): List<RunSession> = folder(context, owner).listFiles().orEmpty()
        .filter { it.extension == "json" && (includeDraft || it.name != "draft.json") }.mapNotNull { runCatching { decode(it.readText()) }.getOrNull() }.sortedByDescending { it.startedAt }
    fun draft(context: Context, owner: String): RunSession? = runCatching { decode(File(folder(context, owner), "draft.json").readText()) }.getOrNull()
}

class RunTrackingService : Service(), LocationListener {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private lateinit var locations: LocationManager
    private var lastLocation: Location? = null
    private var tick: Job? = null
    private var baseSeconds = 0L
    private var clockStarted = 0L
    private var lastSaved = 0L
    private var saving = false
    override fun onBind(intent: Intent?) = null
    override fun onCreate() { super.onCreate(); locations = getSystemService(LocationManager::class.java) }
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            "start" -> {
                if (RunState.mutable.value.active || saving) return START_NOT_STICKY
                val owner = intent.getStringExtra("owner").orEmpty()
                if (owner.isBlank()) { stopSelf(); return START_NOT_STICKY }
                if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
                    RunState.mutable.value = RunSession(message = "请允许精确定位后再开始"); stopSelf(); return START_NOT_STICKY
                }
                // Preserve an interrupted recording before starting a new one.
                val draft = RunRecords.draft(this, owner)
                if (draft != null) runCatching { RunRecords.save(this, draft) }
                RunState.mutable.value = RunSession(startedAt = System.currentTimeMillis(), active = true, owner = owner, message = "正在等待 GPS 定位，请到室外开阔处")
                val manager = getSystemService(NotificationManager::class.java)
                manager.createNotificationChannel(NotificationChannel("rzh_running", "运动轨迹记录", NotificationManager.IMPORTANCE_LOW))
                startForeground(902, notification())
                baseSeconds = 0; clockStarted = SystemClock.elapsedRealtime(); lastLocation = null
                subscribe()
                tick = scope.launch { while (isActive) {
                    updateClock()
                    if (SystemClock.elapsedRealtime() - lastSaved > 10000) { persist(true); lastSaved = SystemClock.elapsedRealtime() }
                    delay(1000)
                } }
            }
            "pause" -> if (RunState.mutable.value.active && !RunState.mutable.value.paused) {
                updateClock(); baseSeconds = RunState.mutable.value.durationSeconds
                locations.removeUpdates(this); lastLocation = null
                RunState.mutable.value = RunState.mutable.value.copy(paused = true, message = "运动已暂停")
                scope.launch { persist(true) }
            }
            "resume" -> if (RunState.mutable.value.active && RunState.mutable.value.paused) {
                clockStarted = SystemClock.elapsedRealtime()
                RunState.mutable.value = RunState.mutable.value.copy(paused = false, segment = RunState.mutable.value.segment + 1, message = "正在重新定位")
                lastLocation = null; subscribe()
            }
            "stop" -> if (RunState.mutable.value.active) {
                saving = true
                updateClock(); locations.removeUpdates(this)
                RunState.mutable.value = RunState.mutable.value.copy(active = false, paused = false, message = "正在保存运动记录")
                scope.launch {
                    val saved = persist(false)
                    if (saved) {
                        tick?.cancel()
                        RunState.mutable.value = RunState.mutable.value.copy(message = "运动记录已保存")
                        stopForeground(STOP_FOREGROUND_REMOVE); stopSelf()
                    } else {
                        saving = false
                        baseSeconds = RunState.mutable.value.durationSeconds
                        RunState.mutable.value = RunState.mutable.value.copy(active = true, paused = true,
                            message = "保存失败，请检查存储空间后再次结束保存")
                    }
                }
            }
        }
        return START_NOT_STICKY
    }
    private fun updateClock() {
        val current = RunState.mutable.value
        if (current.active && !current.paused) RunState.mutable.value = current.copy(durationSeconds = baseSeconds + (SystemClock.elapsedRealtime() - clockStarted) / 1000)
    }
    private suspend fun persist(draft: Boolean): Boolean {
        val snapshot = RunState.mutable.value
        return withContext(Dispatchers.IO) { runCatching { RunRecords.save(this@RunTrackingService, snapshot, draft) }.isSuccess }
    }
    private fun subscribe() {
        try {
            locations.requestLocationUpdates(LocationManager.GPS_PROVIDER, 2000, 3f, this)
            if (!locations.isProviderEnabled(LocationManager.GPS_PROVIDER)) onProviderDisabled(LocationManager.GPS_PROVIDER)
        } catch (error: Exception) {
            RunState.mutable.value = RunState.mutable.value.copy(message = "定位不可用，请检查定位权限与系统定位开关")
        }
    }
    override fun onLocationChanged(location: Location) {
        val current = RunState.mutable.value
        if (!current.active || current.paused || !location.hasAccuracy()) return
        val age = (SystemClock.elapsedRealtimeNanos() - location.elapsedRealtimeNanos) / 1e9
        val previous = lastLocation
        val elapsed = previous?.let { (location.elapsedRealtimeNanos - it.elapsedRealtimeNanos) / 1e9 } ?: 0.0
        val split = previous != null && elapsed > 30
        val meters = if (previous != null && !split) previous.distanceTo(location).toDouble() else 0.0
        if (!RunRules.accept(location.accuracy, age, previous != null, elapsed, meters)) return
        lastLocation = Location(location)
        val segment = current.segment + if (split) 1 else 0
        RunState.mutable.value = current.copy(distanceMeters = current.distanceMeters + meters, segment = segment,
            points = current.points + TrackPoint(location.latitude, location.longitude, location.time, segment), message = "GPS 信号正常 · 正在记录")
    }
    override fun onProviderDisabled(provider: String) {
        lastLocation = null
        RunState.mutable.value = RunState.mutable.value.copy(segment = RunState.mutable.value.segment + 1, message = "系统定位已关闭，请开启后继续")
    }
    override fun onProviderEnabled(provider: String) { RunState.mutable.value = RunState.mutable.value.copy(message = "定位已开启，正在获取 GPS") }
    private fun notification(): Notification {
        val open = PendingIntent.getActivity(this, 902, Intent(this, MainActivity::class.java), PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
        return NotificationCompat.Builder(this, "rzh_running").setSmallIcon(R.drawable.ruizhihe_app_icon)
            .setContentTitle("睿之禾 · 运动记录中").setContentText("正在记录你的运动轨迹，点击返回应用")
            .setContentIntent(open).setOngoing(true).build()
    }
    override fun onDestroy() {
        locations.removeUpdates(this)
        val current = RunState.mutable.value
        if (current.active) {
            RunState.mutable.value = current.copy(active = false, paused = false, message = "运动已中断，已保留最近的轨迹记录")
        }
        scope.cancel(); super.onDestroy()
    }
}

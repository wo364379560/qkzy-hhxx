package com.balabalaim.gcloud.tim.push.appdev.run

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Locale
import java.util.UUID

data class RunPlan(
    val id: String = UUID.randomUUID().toString(), val day: String,
    val activity: String = "跑步", val unit: String = "公里", val target: Double,
    val note: String = "", val recordId: Long? = null,
)

object PlanRules {
    fun day(time: Long): String = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(java.util.Date(time))
    fun calendar(day: String): Calendar {
        require(Regex("\\d{4}-\\d{2}-\\d{2}").matches(day)) { "日期格式应为 yyyy-MM-dd" }
        val formatter = SimpleDateFormat("yyyy-MM-dd", Locale.US).apply { isLenient = false }
        val parsed = requireNotNull(formatter.parse(day))
        return Calendar.getInstance().apply { time = parsed }
    }
    fun validate(plan: RunPlan) {
        require(plan.id.isNotBlank()); calendar(plan.day)
        require(plan.activity in listOf("跑步", "步行"))
        require(plan.unit in listOf("公里", "分钟"))
        require(plan.target.isFinite() && plan.target > 0 && plan.target <= if (plan.unit == "公里") 200 else 1440) { "请输入有效目标：距离不超过 200 公里，时长不超过 1440 分钟" }
        require(plan.note.length <= 200)
    }
    fun link(plans: List<RunPlan>, plan: RunPlan, record: RunSession): RunPlan {
        require(!record.active && record.points.isNotEmpty() && record.durationSeconds > 0) { "请选择已保存的有效运动记录" }
        require(day(record.startedAt) == plan.day) { "请选择计划当天的运动记录" }
        require(plans.none { it.id != plan.id && it.recordId == record.startedAt }) { "这条运动记录已关联其他计划" }
        return plan.copy(recordId = record.startedAt)
    }
    fun progress(plan: RunPlan, record: RunSession?): Double = if (record == null) 0.0 else
        (if (plan.unit == "公里") record.distanceMeters / 1000 else record.durationSeconds / 60.0) / plan.target
    fun encode(plans: List<RunPlan>): String = JSONArray().apply { plans.forEach { p ->
        validate(p)
        put(JSONObject().put("id", p.id).put("day", p.day).put("activity", p.activity).put("unit", p.unit)
            .put("target", p.target).put("note", p.note).put("record", p.recordId))
    } }.toString()
    fun decode(raw: String): List<RunPlan> {
        val array = JSONArray(raw)
        return (0 until array.length()).map { i ->
            val p = array.getJSONObject(i)
            RunPlan(p.getString("id"), p.getString("day"), p.getString("activity"), p.getString("unit"),
                p.getDouble("target"), p.optString("note"), if (p.has("record") && !p.isNull("record")) p.getLong("record") else null)
                .also { validate(it) }
        }.also { require(it.map { p -> p.id }.distinct().size == it.size) }
    }
}

class RunPlanStore(context: Context, owner: String) {
    private val prefs = context.getSharedPreferences("run_calendar_plans", Context.MODE_PRIVATE)
    private val key = java.security.MessageDigest.getInstance("SHA-256").digest(owner.toByteArray()).joinToString("") { "%02x".format(it) }
    init { require(owner.isNotBlank()) }
    fun load(): List<RunPlan> = PlanRules.decode(prefs.getString(key, "[]") ?: "[]")
    fun save(plans: List<RunPlan>) { check(prefs.edit().putString(key, PlanRules.encode(plans)).commit()) { "保存失败，请检查手机存储空间" } }
}

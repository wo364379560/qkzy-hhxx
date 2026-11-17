package com.balabalaim.gcloud.tim.push.appdev.im

import org.json.JSONArray
import org.json.JSONObject

/** Confirmed recall evidence only. No message text, credentials, or failed requests are persisted. */
internal class RecalledMessageState(
    private val read: (String) -> String? = { null },
    private val write: (String, String) -> Unit = { _, _ -> },
    private val limit: Int = 2048,
) {
    enum class Kind { UNKNOWN, OWN, MODERATOR }
    data class Scope(val key: String, val generation: Long)
    data class Intent(val scope: Scope, val id: String, val serial: Long)
    private data class Evidence(val kind: Kind, val operator: String = "", val sender: String = "", val group: String = "")
    private data class Pending(val intent: Intent, val evidence: Evidence)
    private val lock = Any()
    private var active: Scope? = null
    private var generation = 0L
    private var serial = 0L
    private val records = linkedMapOf<String, Evidence>()
    private val pending = mutableMapOf<String, Pending>()

    fun activate(key: String): Scope = synchronized(lock) {
        active?.takeIf { it.key == key }?.let { return it }
        generation++
        records.clear(); pending.clear()
        val scope = Scope(key, generation)
        active = scope
        runCatching {
            val array = JSONArray(read(key) ?: "[]")
            for (index in maxOf(0, array.length() - limit) until array.length()) {
                val item = array.getJSONObject(index)
                val id = item.optString("id").takeIf { it.isNotBlank() } ?: continue
                val kind = runCatching { Kind.valueOf(item.optString("kind")) }.getOrDefault(Kind.UNKNOWN)
                records[id] = Evidence(kind, item.optString("operator"), item.optString("sender"), item.optString("group"))
            }
        }
        scope
    }

    fun current(): Scope? = synchronized(lock) { active }
    fun deactivate() = synchronized(lock) { generation++; active = null; records.clear(); pending.clear() }
    fun isCurrent(scope: Scope?): Boolean = synchronized(lock) { scope != null && active == scope }

    fun begin(scope: Scope, id: String, kind: Kind, sender: String, group: String, operator: String): Intent = synchronized(lock) {
        check(active == scope) { "账号已切换" }
        require(id.isNotBlank()) { "消息ID无效" }
        check(!pending.containsKey(id)) { "正在撤回，请稍候" }
        val intent = Intent(scope, id, ++serial)
        pending[id] = Pending(intent, Evidence(kind, operator, sender, group))
        intent
    }

    fun cancel(intent: Intent) = synchronized(lock) {
        if (active == intent.scope && pending[intent.id]?.intent == intent) pending.remove(intent.id)
    }

    fun confirm(intent: Intent): Boolean = synchronized(lock) {
        if (active != intent.scope) return false
        val request = pending[intent.id]?.takeIf { it.intent == intent } ?: return false
        pending.remove(intent.id)
        remember(intent.scope, intent.id, request.evidence)
        true
    }

    fun receive(scope: Scope, id: String, operator: String = "", sender: String = "", group: String = ""): Boolean = synchronized(lock) {
        if (active != scope || id.isBlank()) return false
        val request = pending[id]?.evidence
        val actualSender = sender.ifBlank { request?.sender.orEmpty() }
        val actualOperator = operator.ifBlank { request?.operator.orEmpty() }
        val kind = when {
            actualSender.isNotBlank() && actualOperator.isNotBlank() -> if (actualSender == actualOperator) Kind.OWN else Kind.MODERATOR
            else -> request?.kind ?: Kind.UNKNOWN
        }
        remember(scope, id, Evidence(kind, actualOperator, actualSender, group.ifBlank { request?.group.orEmpty() }))
        true
    }

    fun isRevoked(scope: Scope?, id: String, sdkRevoked: Boolean): Boolean = synchronized(lock) {
        sdkRevoked || (scope != null && active == scope && records.containsKey(id))
    }

    fun shouldHide(scope: Scope?, id: String, group: String, sender: String, operator: String, sdkRevoked: Boolean): Boolean = synchronized(lock) {
        if (!isRevoked(scope, id, sdkRevoked)) return false
        val evidence = if (scope != null && active == scope) records[id] ?: pending[id]?.evidence else null
        if (group.ifBlank { evidence?.group.orEmpty() }.isBlank()) return false
        when (evidence?.kind) {
            Kind.MODERATOR -> return true
            Kind.OWN -> return false
            else -> Unit
        }
        val actualSender = sender.ifBlank { evidence?.sender.orEmpty() }
        val actualOperator = evidence?.operator.orEmpty().ifBlank { operator }
        // Missing metadata is not proof of a self-recall. Never expose a moderator tip by guessing.
        return actualSender.isBlank() || actualOperator.isBlank() || actualSender != actualOperator
    }

    private fun remember(scope: Scope, id: String, incoming: Evidence) {
        val previous = records[id]
        val kind = when {
            previous?.kind == Kind.MODERATOR || incoming.kind == Kind.MODERATOR -> Kind.MODERATOR
            incoming.kind != Kind.UNKNOWN -> incoming.kind
            else -> previous?.kind ?: Kind.UNKNOWN
        }
        records.remove(id)
        records[id] = Evidence(kind, incoming.operator.ifBlank { previous?.operator.orEmpty() },
            incoming.sender.ifBlank { previous?.sender.orEmpty() }, incoming.group.ifBlank { previous?.group.orEmpty() })
        while (records.size > limit) records.remove(records.keys.first())
        val array = JSONArray()
        records.forEach { (messageId, value) -> array.put(JSONObject().put("id", messageId).put("kind", value.kind.name)
            .put("operator", value.operator).put("sender", value.sender).put("group", value.group)) }
        write(scope.key, array.toString())
    }
}

package com.balabalaim.gcloud.tim.push.appdev.im

import com.balabalaim.gcloud.tim.push.appdev.LinLanAppContainer
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.data.model.ChatProfile
import com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnerAssignment
import com.tencent.imsdk.v2.V2TIMManager
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

data class FriendOwnerProfile(val owner: String, val generation: Long, val sdkAppId: Int, val profile: ChatProfile)
data class FriendOwnerBrief(val ownerId: Long, val ownerRevision: Long, val isAssigned: Boolean?, val ownerState: String)

/** Application lifetime, one synchronization per session + target. SDK friendship
 * outcomes stay independent from assignment failures and are never re-sent. */
object LinLanFriendOwnership {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private val pending = mutableMapOf<String, Deferred<FriendOwnerAssignment?>>()
    private val profiles = MutableStateFlow<FriendOwnerProfile?>(null)
    val currentProfile = profiles.asStateFlow()
    private val states = MutableStateFlow<Map<String, FriendOwnerAssignment>>(emptyMap())
    val results = states.asStateFlow()
    private val requests = MutableStateFlow<Set<String>>(emptySet())
    val waitingRequests = requests.asStateFlow()
    private val peers = MutableStateFlow<Map<String, FriendOwnerBrief>>(emptyMap())
    val targetProfiles = peers.asStateFlow()
    private fun store() = LinLanAppContainer.sessionStore()
    fun sessionKey() = "${LinLanRuntimeConfig.sdkAppId}:${V2TIMManager.getInstance().loginUser}:${store().sessionGeneration()}"
    fun requestPending(target: String) { requests.value = requests.value + (sessionKey() + ":" + target) }
    fun requestConfirmed(target: String) { requests.value = requests.value - (sessionKey() + ":" + target) }

    fun publishProfile(profile: ChatProfile, generation: Long = store().sessionGeneration(), owner: String = V2TIMManager.getInstance().loginUser.orEmpty()) {
        if (profile.imUserId != owner || owner.isBlank() || generation != store().sessionGeneration() || V2TIMManager.getInstance().loginUser != owner) return
        val previous = profiles.value
        if (previous?.owner == owner && previous.generation == generation && previous.profile.ownerRevision > profile.ownerRevision) return
        profiles.value = FriendOwnerProfile(owner, generation, LinLanRuntimeConfig.sdkAppId, profile)
    }

    fun established(target: String) { scope.launch { synchronize(target) } }

    suspend fun synchronize(target: String): FriendOwnerAssignment? = withContext(Dispatchers.Main.immediate) {
        val owner = V2TIMManager.getInstance().loginUser.orEmpty()
        val generation = store().sessionGeneration()
        val sdk = LinLanRuntimeConfig.sdkAppId
        val captured = com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnershipScope(sdk, owner, generation)
        fun active() = captured.matches(LinLanRuntimeConfig.sdkAppId, V2TIMManager.getInstance().loginUser.orEmpty(), store().sessionGeneration())
        if (!active() || target.isBlank()) return@withContext null
        val key = "${sessionKey()}:$target"
        pending[key]?.let { return@withContext it.await() }
        if (pending.size >= 32) return@withContext FriendOwnerAssignment("pending", "client_queue_full")
        val requestID = UUID.randomUUID().toString()
        val work = scope.async(start = CoroutineStart.LAZY) {
            var result = FriendOwnerAssignment("pending", "synchronization_unavailable", requestID)
            // The local SDK check only avoids unnecessary work. The server always
            // independently proves BOTH directions before changing ownership.
            if (!runCatching { LinLanImManager.areBothWayFriends(target) }.getOrDefault(false) || !active()) return@async null
            requestConfirmed(target)
            for (wait in listOf(0L, 500L, 1500L)) {
                if (wait > 0) delay(wait)
                if (!active()) return@async null
                runCatching { LinLanAppContainer.apiClient().syncFriendOwnership(target, requestID, owner, generation) }.onSuccess { result = it }
                if (!active()) return@async null
                runCatching { LinLanAppContainer.apiClient().fetchProfile() }.onSuccess { if (active()) publishProfile(it,generation,owner) }
                if (!active()) return@async null
                // Refresh the peer without reversing search/discovery rules.
                runCatching { LinLanAppContainer.apiClient().fetchFriendOwnerBrief(target) }.onSuccess { data ->
                    val users=data.optJSONArray("users")
                    if(active() && users!=null) for (i in 0 until users.length()) {
                        val user=users.optJSONObject(i) ?: continue
                        if(user.optString("im_user_id")==target) peers.value=peers.value+(key to FriendOwnerBrief(user.optLong("owner_id"),user.optLong("owner_revision"),if(user.has("is_assigned")) user.optBoolean("is_assigned") else null,user.optString("owner_state","unknown")))
                    }
                }
                if (!active()) return@async null
                states.value = states.value + (key to result)
                if (result.state != "pending") break
            }
            if (active()) result else null
        }
        pending[key] = work
        work.invokeOnCompletion { scope.launch { if (pending[key] === work) pending.remove(key) } }
        work.start(); work.await()
    }
}

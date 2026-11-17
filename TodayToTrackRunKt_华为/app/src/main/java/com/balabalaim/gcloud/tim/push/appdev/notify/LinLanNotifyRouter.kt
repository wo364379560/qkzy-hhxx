package com.balabalaim.gcloud.tim.push.appdev.notify

import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow

/** 点击本地通知后，由导航层消费并打开对应聊天。 */
object LinLanNotifyRouter {
    @Volatile
    var pendingPeerId: String? = null

    @Volatile
    var pendingIsGroup: Boolean = false

    @Volatile
    var pendingTitle: String = ""

    @Volatile
    var pendingIsCall: Boolean = false

    private val _events = MutableSharedFlow<Unit>(extraBufferCapacity = 8)
    val events: SharedFlow<Unit> = _events.asSharedFlow()

    fun offerChat(peerId: String, isGroup: Boolean, title: String) {
        pendingPeerId = peerId
        pendingIsGroup = isGroup
        pendingTitle = title
        pendingIsCall = false
        _events.tryEmit(Unit)
    }

    fun offerCall() {
        pendingIsCall = true
        pendingPeerId = null
        _events.tryEmit(Unit)
    }

    fun consumeChat(): Triple<String, Boolean, String>? {
        val peer = pendingPeerId ?: return null
        val group = pendingIsGroup
        val title = pendingTitle.ifBlank { peer }
        pendingPeerId = null
        return Triple(peer, group, title)
    }

    fun consumeCall(): Boolean {
        if (!pendingIsCall) return false
        pendingIsCall = false
        return true
    }
}

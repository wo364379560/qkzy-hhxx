package com.balabalaim.gcloud.tim.push.appdev.push

import android.content.Context
import android.util.Log
import com.balabalaim.gcloud.tim.push.appdev.core.LinLanRuntimeConfig
import com.balabalaim.gcloud.tim.push.appdev.notify.LinLanNotifyRouter
import com.tencent.qcloud.tim.push.TIMPushCallback
import com.tencent.qcloud.tim.push.TIMPushListener
import com.tencent.qcloud.tim.push.TIMPushManager
import com.tencent.qcloud.tim.push.TIMPushMessage
import org.json.JSONObject

/** TIMPush registration and notification routing. Vendor credentials come from timpush-configs.json. */
object LinLanPushManager {
    private const val TAG = "LinLanPushManager"
    private var registered = false
    private var listenerAdded = false

    private val listener = object : TIMPushListener() {
        override fun onRecvPushMessage(message: TIMPushMessage?) {
            Log.d(TAG, "push received id=${message?.messageID.orEmpty()}")
        }

        override fun onRevokePushMessage(messageId: String?) {
            Log.d(TAG, "push revoked id=${messageId.orEmpty()}")
        }

        override fun onNotificationClicked(ext: String?) {
            routeExt(ext)
        }
    }

    fun registerIfAvailable(context: Context, userId: String) {
        if (registered || userId.isBlank()) return
        val manager = TIMPushManager.getInstance()
        if (!listenerAdded) {
            manager.addPushListener(listener)
            listenerAdded = true
        }
        manager.disablePostNotificationInForeground(true)
        // IM is already logged in, so a null appKey intentionally reuses that login state.
        manager.registerPush(
            context.applicationContext,
            LinLanRuntimeConfig.sdkAppId,
            null,
            object : TIMPushCallback<Any?>() {
                override fun onSuccess(data: Any?) {
                    registered = true
                    manager.getRegistrationID(
                        object : TIMPushCallback<Any?>() {
                            override fun onSuccess(value: Any?) {
                                Log.i(TAG, "TIMPush registered")
                            }

                            override fun onError(code: Int, desc: String?, data: Any?) {
                                Log.w(TAG, "getRegistrationID failed($code): $desc")
                            }
                        },
                    )
                }

                override fun onError(code: Int, desc: String?, data: Any?) {
                    registered = false
                    Log.w(TAG, "TIMPush register failed($code): $desc")
                }
            },
        )
    }

    fun unregister() {
        val manager = TIMPushManager.getInstance()
        if (registered) {
            manager.unRegisterPush(
                object : TIMPushCallback<Any?>() {
                    override fun onSuccess(data: Any?) = Unit

                    override fun onError(code: Int, desc: String?, data: Any?) {
                        Log.w(TAG, "TIMPush unregister failed($code): $desc")
                    }
                },
            )
        }
        registered = false
    }

    private fun routeExt(raw: String?) {
        val ext = raw?.trim().orEmpty()
        if (ext.isEmpty()) return
        runCatching {
            val json = JSONObject(ext)
            val type = json.optInt("conversationType", json.optInt("conversation_type", 0))
            val groupId = json.optString("groupID").ifBlank { json.optString("group_id") }
            val userId = json.optString("userID").ifBlank { json.optString("user_id") }
            val generic = json.optString("conversationID").ifBlank { json.optString("conversation_id") }
            val isGroup = type == 2 || groupId.isNotBlank() || generic.startsWith("group_")
            val peerId = when {
                groupId.isNotBlank() -> groupId
                userId.isNotBlank() -> userId
                else -> generic.removePrefix("group_").removePrefix("c2c_")
            }
            if (peerId.isNotBlank()) {
                LinLanNotifyRouter.offerChat(
                    peerId = peerId,
                    isGroup = isGroup,
                    title = json.optString("title").ifBlank { peerId },
                )
            }
        }.onFailure { Log.w(TAG, "invalid push ext", it) }
    }
}

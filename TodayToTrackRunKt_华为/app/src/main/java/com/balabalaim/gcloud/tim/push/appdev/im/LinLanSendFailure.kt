package com.balabalaim.gcloud.tim.push.appdev.im

class LinLanSendFailure(message: String, val code: Int = 0, val notDispatched: Boolean = false) : Exception(message) {
    val unknown: Boolean get() = !notDispatched && (code == 0 || code in setOf(2800, 2801, 2805, 60008))
}

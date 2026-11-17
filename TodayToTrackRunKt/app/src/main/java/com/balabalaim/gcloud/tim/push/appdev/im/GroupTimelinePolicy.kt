package com.balabalaim.gcloud.tim.push.appdev.im

internal fun hideUnjoinedConversation(
    groupId: String, joined: Set<String>, normalLoaded: Boolean, communityLoaded: Boolean, isCommunity: Boolean,
): Boolean {
    if (groupId.isBlank() || groupId in joined || groupId.substringBefore("@TOPIC#") in joined) return false
    // Community groups can appear in either SDK result.
    return normalLoaded && (!isCommunity || communityLoaded)
}

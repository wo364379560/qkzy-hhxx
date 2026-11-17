package com.balabalaim.gcloud.tim.push.appdev.im

import org.junit.Assert.*
import org.junit.Test

class GroupTimelinePolicyTest {
    @Test fun offlineRemovalAndReadmission() {
        assertTrue(hideUnjoinedConversation("g", emptySet(), true, true, false))
        assertFalse(hideUnjoinedConversation("g", setOf("g"), true, true, false))
        assertFalse(hideUnjoinedConversation("", emptySet(), true, true, false))
    }
    @Test fun failedMembershipQueriesAndJoinedTopics() {
        assertFalse(hideUnjoinedConversation("g", emptySet(), false, false, false))
        assertFalse(hideUnjoinedConversation("@TGS#_a", emptySet(), true, false, true))
        assertFalse(hideUnjoinedConversation("a@TOPIC#_b", setOf("a"), true, true, true))
    }
}

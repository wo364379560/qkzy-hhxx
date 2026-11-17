package com.balabalaim.gcloud.tim.push.appdev.im

import com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnerProtocol
import com.balabalaim.gcloud.tim.push.appdev.data.model.FriendOwnershipScope
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class FriendOwnershipProtocolTest {
    @Test fun oldProfileDoesNotInventAssignment() {
        val p = FriendOwnerProtocol.profile(JSONObject("""{"im_user_id":"old","wanan_id":"old-id","nickname":"old"}"""))
        assertEquals("old-id", p.chuanluId); assertNull(p.isAssigned); assertEquals(0L, p.ownerRevision); assertEquals("unknown", p.ownerState)
    }
    @Test fun extendedProfileAndInvalidOwnerRetainState() {
        val p = FriendOwnerProtocol.profile(JSONObject("""{"owner_id":81,"owner_revision":22,"is_assigned":false,"owner_state":"unavailable","owner_reason_code":"owner_inactive"}"""))
        assertEquals(81L, p.ownerId); assertEquals(22L, p.ownerRevision); assertEquals(false, p.isAssigned); assertEquals("unavailable", p.ownerState)
    }
    @Test fun legacyHttpSuccessIsPending() {
        val p = FriendOwnerProtocol.assignment(JSONObject("""{"user":{"im_user_id":"peer"}}"""), "stable")
        assertEquals("pending", p.state); assertEquals("stable", p.requestId)
    }
    @Test fun accountSameUserReloginAndSdkSwitchInvalidateOldWork() {
        val scope = FriendOwnershipScope(7, "A", 12)
        assertTrue(scope.matches(7, "A", 12)); assertFalse(scope.matches(7, "B", 12))
        assertFalse(scope.matches(7, "A", 13)); assertFalse(scope.matches(8, "A", 12))
        assertFalse(FriendOwnershipScope(7, "", 12).matches(7, "", 12))
    }
}

package com.balabalaim.gcloud.tim.push.appdev.im

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LinLanGroupPolicyTest {
    @Test
    fun policyDefaultsFailSafeForPrivacySensitiveControls() {
        assertTrue(LinLanGroupPolicy.isMemberProfileAccessEnabled(emptyMap()))
        assertFalse(LinLanGroupPolicy.isMuteAdminAlsoEnabled(emptyMap()))
        assertFalse(LinLanGroupPolicy.isVoiceCallEnabled(emptyMap()))
        assertFalse(LinLanGroupPolicy.isVideoCallEnabled(emptyMap()))
    }

    @Test
    fun policyParsesBackendCompatibleBooleanValues() {
        assertTrue(
            LinLanGroupPolicy.isMemberProfileAccessEnabled(
                mapOf(LinLanGroupPolicy.MEMBER_PROFILE_ACCESS_KEY to "yes"),
            ),
        )
        assertFalse(
            LinLanGroupPolicy.isMemberProfileAccessEnabled(
                mapOf(LinLanGroupPolicy.MEMBER_PROFILE_ACCESS_KEY to "0"),
            ),
        )
    }
}

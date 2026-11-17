package com.balabalaim.gcloud.tim.push.appdev.ui.common

import org.junit.Assert.*
import org.junit.Test
import com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermissionPolicy.Action.*
import com.balabalaim.gcloud.tim.push.appdev.ui.common.FeaturePermissionPolicy.Choice.*

class FeaturePermissionPolicyTest {
    private val choices = mutableMapOf<FeaturePermission, FeaturePermissionPolicy.Choice>()
    private val requested = mutableMapOf<String, Boolean>()
    private fun fresh() = FeaturePermissionPolicy({ choices[it] ?: UNKNOWN }, { k, v -> choices[k] = v },
        { requested[it] == true }, { k, v -> requested[k] = v })
    private val mic = listOf(FeaturePermission.MICROPHONE)
    private val both = listOf(FeaturePermission.MICROPHONE, FeaturePermission.CAMERA)
    private val micPermission = FeaturePermission.MICROPHONE.systemPermissions
    private fun decision(policy: FeaturePermissionPolicy, features: List<FeaturePermission>, granted: Set<String> = emptySet()) =
        policy.action(features, features.flatMap { it.systemPermissions }, granted)

    @Test fun allowedPickersRunOnSubsequentTapsAndAfterReconstruction() {
        for (feature in listOf(FeaturePermission.IMAGE_PICK, FeaturePermission.VIDEO_PICK, FeaturePermission.FILE_PICK, FeaturePermission.AUDIO_PICK)) {
            val features = listOf(feature); val p = fresh()
            assertEquals(EXPLAIN, decision(p, features))
            p.choose(features, ALLOWED)
            repeat(3) { assertEquals(RUN, decision(fresh(), features)) }
        }
    }
    @Test fun rejectedPickersNeverRunOrReopenExplanationUntilExplicitReset() {
        for (feature in listOf(FeaturePermission.IMAGE_PICK, FeaturePermission.VIDEO_PICK, FeaturePermission.FILE_PICK)) {
            val features = listOf(feature); val p = fresh(); p.choose(features, DENIED)
            repeat(3) { assertEquals(BLOCKED_FEATURE, decision(fresh(), features)) }
            p.reset(feature); assertEquals(EXPLAIN, decision(fresh(), features))
        }
    }
    @Test fun denyingCameraLeavesPreviouslyAllowedVoiceWorking() {
        val p = fresh(); p.choose(mic, ALLOWED)
        assertEquals(EXPLAIN, decision(p, both, micPermission.toSet()))
        p.choose(both, DENIED)
        assertEquals(RUN, decision(p, mic, micPermission.toSet()))
        assertEquals(BLOCKED_FEATURE, decision(p, both, micPermission.toSet()))
    }
    @Test fun deniedMicrophoneBlocksBothVoiceAndVideoEvenIfAndroidGrantedIt() {
        val p = fresh(); p.choose(mic, DENIED)
        assertEquals(BLOCKED_FEATURE, decision(p, mic, micPermission.toSet()))
        assertEquals(BLOCKED_FEATURE, decision(p, both, both.flatMap { it.systemPermissions }.toSet()))
    }
    @Test fun appApprovalCannotBypassMissingSystemGrant() {
        val p = fresh(); p.choose(mic, ALLOWED)
        assertEquals(REQUEST_SYSTEM, decision(p, mic))
        p.requested(micPermission)
        assertEquals(BLOCKED_SYSTEM, decision(fresh(), mic))
        assertEquals(RUN, decision(fresh(), mic, micPermission.toSet()))
    }
    @Test fun runtimeRequestIsNotAutomaticallyRepeatedAfterProcessDeathOrRevocation() {
        val p = fresh(); p.choose(mic, ALLOWED); p.requested(micPermission)
        assertEquals(BLOCKED_SYSTEM, decision(fresh(), mic))
        assertEquals(RUN, decision(fresh(), mic, micPermission.toSet()))
        assertEquals(BLOCKED_SYSTEM, decision(fresh(), mic))
        p.reset(FeaturePermission.MICROPHONE); p.choose(mic, ALLOWED)
        assertEquals(REQUEST_SYSTEM, decision(fresh(), mic))
    }
    @Test fun cameraAndMicrophoneAreIndependentlyRequiredForVideo() {
        val p = fresh(); p.choose(both, ALLOWED)
        assertEquals(REQUEST_SYSTEM, decision(p, both, micPermission.toSet()))
        p.requested(FeaturePermission.CAMERA.systemPermissions)
        assertEquals(BLOCKED_SYSTEM, decision(p, both, micPermission.toSet()))
        assertEquals(RUN, decision(p, both, both.flatMap { it.systemPermissions }.toSet()))
        assertEquals(RUN, decision(p, mic, micPermission.toSet()))
    }
    @Test fun anExistingAndroidGrantStillGetsOnePurposeExplanation() {
        val p = fresh()
        assertEquals(EXPLAIN, decision(p, mic, micPermission.toSet()))
        p.choose(mic, ALLOWED)
        assertEquals(RUN, decision(p, mic, micPermission.toSet()))
    }
}

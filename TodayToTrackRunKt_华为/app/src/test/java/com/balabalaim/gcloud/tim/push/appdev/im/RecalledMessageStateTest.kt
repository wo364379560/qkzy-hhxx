package com.balabalaim.gcloud.tim.push.appdev.im

import org.junit.Assert.*
import org.junit.Test

class RecalledMessageStateTest {
    private fun RecalledMessageState.hidden(scope: RecalledMessageState.Scope?, id: String = "m", sender: String = "member", actor: String = "", sdk: Boolean = false, group: String = "g") =
        shouldHide(scope, id, group, sender, actor, sdk)

    @Test fun missingOperatorNeverCreatesAModeratorRecallTip() {
        val state = RecalledMessageState(); val owner = state.activate("app:member")
        assertTrue(state.hidden(owner, sdk = true))
        assertTrue(state.hidden(owner, sender = "", sdk = true))
        assertFalse(state.hidden(owner, actor = "member", sdk = true))
        assertTrue(state.hidden(owner, actor = "admin", sdk = true))
    }
    @Test fun unknownCallbackHidesAStaleOriginalBeforeSDKObjectsChange() {
        val state = RecalledMessageState(); val owner = state.activate("app:member")
        assertTrue(state.receive(owner, "m"))
        assertTrue(state.isRevoked(owner, "m", false)); assertTrue(state.hidden(owner))
    }
    @Test fun detailedCallbackRetainsItsOperatorWhenHistoryOmitsIt() {
        val state = RecalledMessageState(); val owner = state.activate("app:member")
        state.receive(owner, "m", operator = "admin")
        state.receive(owner, "m", sender = "member", group = "g")
        assertTrue(state.hidden(owner, actor = "member"))
    }
    @Test fun aMembersOwnMessageIsHiddenWhenAModeratorRecalledIt() {
        val state = RecalledMessageState(); val owner = state.activate("app:member")
        state.receive(owner, "m", operator = "admin", sender = "member", group = "g")
        assertTrue(state.hidden(owner, sender = "member"))
    }
    @Test fun rememberedGroupSurvivesASDKSnapshotWithoutConversationMetadata() {
        val state = RecalledMessageState(); val owner = state.activate("app:admin")
        state.confirm(state.begin(owner, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "admin"))
        assertTrue(state.hidden(owner, group = "", sender = ""))
    }
    @Test fun failedRestRequestDoesNotRemoveTheMessage() {
        val state = RecalledMessageState(); val owner = state.activate("app:admin")
        val request = state.begin(owner, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "admin")
        assertFalse(state.hidden(owner)); state.cancel(request)
        assertFalse(state.isRevoked(owner, "m", false)); assertFalse(state.hidden(owner))
    }
    @Test fun restSuccessStaysSilentWithoutAnySDKCallback() {
        val state = RecalledMessageState(); val owner = state.activate("app:admin")
        val request = state.begin(owner, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "admin")
        assertTrue(state.confirm(request)); assertTrue(state.hidden(owner))
        state.receive(owner, "m", operator = "member", sender = "member", group = "g")
        assertTrue("A late legacy snapshot cannot undo a confirmed moderator recall", state.hidden(owner))
    }
    @Test fun explicitOwnRecallKeepsItsTipEvenWithALegacyCallback() {
        val state = RecalledMessageState(); val owner = state.activate("app:member")
        val request = state.begin(owner, "m", RecalledMessageState.Kind.OWN, "member", "g", "member")
        state.receive(owner, "m"); assertFalse(state.hidden(owner)); assertTrue(state.confirm(request))
        assertFalse(state.hidden(owner)); assertTrue(state.isRevoked(owner, "m", false))
    }
    @Test fun sdkSelfRevokeFromAnotherDeviceIsNotConfusedWithModeration() {
        val state = RecalledMessageState(); val owner = state.activate("app:viewer")
        state.receive(owner, "m", operator = "member")
        assertFalse(state.hidden(owner, sender = "member"))
    }
    @Test fun privateRecallKeepsTheExistingHintBehavior() {
        val state = RecalledMessageState(); val owner = state.activate("app:viewer")
        state.receive(owner, "m", operator = "admin")
        assertFalse(state.hidden(owner, group = ""))
    }
    @Test fun callbackIsAuthoritativeEvenIfTheRequestLaterReportsFailure() {
        val state = RecalledMessageState(); val owner = state.activate("app:admin")
        val request = state.begin(owner, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "admin")
        state.receive(owner, "m"); state.cancel(request)
        assertTrue(state.hidden(owner))
    }
    @Test fun restartAndAccountSwitchPreserveOnlyTheCorrectNamespace() {
        val disk = mutableMapOf<String, String>()
        fun store() = RecalledMessageState({ disk[it] }, { key, raw -> disk[key] = raw })
        val first = store(); val a = first.activate("app:a")
        first.confirm(first.begin(a, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "a"))
        first.confirm(first.begin(a, "own", RecalledMessageState.Kind.OWN, "a", "g", "a"))
        val second = store(); val b = second.activate("app:b")
        assertFalse(second.hidden(b)); val again = second.activate("app:a")
        assertTrue(second.hidden(again)); assertFalse(second.hidden(again, id = "own", sender = "a"))
        assertFalse(second.receive(b, "late", operator = "admin"))
        val otherApp = second.activate("other-app:a"); assertFalse(second.hidden(otherApp))
    }
    @Test fun staleAcknowledgementCannotAffectANewSession() {
        val state = RecalledMessageState(); val old = state.activate("app:a")
        val request = state.begin(old, "m", RecalledMessageState.Kind.MODERATOR, "member", "g", "a")
        state.deactivate(); val next = state.activate("app:a")
        assertFalse(state.confirm(request)); assertFalse(state.receive(old, "m")); assertFalse(state.hidden(next))
    }
    @Test fun evidenceStorageIsBoundedAndMalformedStorageDoesNotBreakChat() {
        var raw = "broken"; val state = RecalledMessageState({ raw }, { _, value -> raw = value }, limit = 2)
        val owner = state.activate("app:a")
        repeat(3) { state.receive(owner, "m$it") }
        assertFalse(state.isRevoked(owner, "m0", false)); assertTrue(state.isRevoked(owner, "m2", false))
        val restarted = RecalledMessageState({ raw }); val scope = restarted.activate("app:a")
        assertTrue(restarted.isRevoked(scope, "m2", false))
    }
}

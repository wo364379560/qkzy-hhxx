package com.balabalaim.gcloud.tim.push.appdev.ui.auth

import org.junit.Assert.*
import org.junit.Test

class PrivacyConsentTest {
    @Test fun uncheckedAlwaysBlocksEvenAfterPreviouslyAccepting() {
        for (persisted in listOf(false, true)) for (dialog in listOf(false, true)) {
            assertFalse(canSubmitPrivacyConsent(persisted, dialog, false))
        }
    }
    @Test fun checkboxAndPrivacyConsentAreBothRequired() {
        assertFalse(canSubmitPrivacyConsent(false, false, true))
        assertTrue(canSubmitPrivacyConsent(true, false, true))
        assertTrue(canSubmitPrivacyConsent(false, true, true))
        assertTrue(canSubmitPrivacyConsent(true, true, true))
    }
}

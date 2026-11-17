package com.balabalaim.gcloud.tim.push.appdev.run

import org.junit.Assert.*
import org.junit.Test

class RunRulesTest {
    @Test fun ignoresStaleAndInaccurateFixesEvenForFirstPoint() {
        assertFalse(RunRules.accept(80f, 0.0, false, 0.0, 0.0))
        assertFalse(RunRules.accept(5f, 20.0, false, 0.0, 0.0))
        assertFalse(RunRules.accept(Float.NaN, 0.0, false, 0.0, 0.0))
        assertTrue(RunRules.accept(5f, 1.0, false, 0.0, 0.0))
    }
    @Test fun rejectsGpsJitterAndTeleportationWithoutRejectingRunning() {
        assertFalse(RunRules.accept(5f, 0.0, true, 2.0, 1.0))
        assertFalse(RunRules.accept(5f, 0.0, true, 2.0, 100.0))
        assertFalse(RunRules.accept(5f, 0.0, true, 0.0, 5.0))
        assertTrue(RunRules.accept(5f, 0.0, true, 2.0, 6.0))
    }
    @Test fun recoversAfterGpsGapAsNewSegment() {
        assertTrue(RunRules.accept(5f, 0.0, true, 60.0, 500.0))
        assertFalse(RunRules.accept(5f, 25.0, true, 60.0, 500.0))
    }
}

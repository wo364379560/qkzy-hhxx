package com.balabalaim.gcloud.tim.push.appdev.run

import org.junit.Assert.*
import org.junit.Test

class RunPlansTest {
    private val plan = RunPlan(id = "p1", day = "2026-09-28", target = 3.0, note = "晚饭后跑步")
    private fun record() = RunSession(startedAt = PlanRules.calendar(plan.day).timeInMillis + 3600000,
        durationSeconds = 1800, distanceMeters = 3200.0, points = listOf(TrackPoint(30.0, 120.0, 0, 0)))
    @Test fun savedPlansRoundTripIncludingLinkAndNotes() {
        val linked = PlanRules.link(listOf(plan), plan, record())
        assertEquals(listOf(linked), PlanRules.decode(PlanRules.encode(listOf(linked))))
        assertEquals(emptyList<RunPlan>(), PlanRules.decode("[]"))
    }
    @Test fun validatesCalendarAndTargets() {
        assertTrue(runCatching { PlanRules.validate(plan.copy(day = "2026-02-29")) }.isFailure)
        PlanRules.validate(plan.copy(day = "2028-02-29"))
        for (target in listOf(0.0, -1.0, Double.NaN, Double.POSITIVE_INFINITY, 201.0)) {
            assertTrue(runCatching { PlanRules.validate(plan.copy(target = target)) }.isFailure)
        }
        PlanRules.validate(plan.copy(unit = "分钟", target = 60.0))
    }
    @Test fun cannotLinkAnotherDayActiveOrAlreadyUsedRecord() {
        val run = record()
        assertTrue(runCatching { PlanRules.link(emptyList(), plan.copy(day = "2026-09-29"), run) }.isFailure)
        assertTrue(runCatching { PlanRules.link(emptyList(), plan, run.copy(active = true)) }.isFailure)
        assertTrue(runCatching { PlanRules.link(listOf(plan.copy(id = "other", recordId = run.startedAt)), plan, run) }.isFailure)
        assertTrue(runCatching { PlanRules.link(emptyList(), plan, run.copy(points = emptyList())) }.isFailure)
    }
    @Test fun completionUsesActualDistanceOrTime() {
        assertTrue(PlanRules.progress(plan, record()) >= 1)
        assertTrue(PlanRules.progress(plan.copy(target = 5.0), record()) < 1)
        assertEquals(1.0, PlanRules.progress(plan.copy(unit = "分钟", target = 30.0), record()), .0001)
        assertEquals(0.0, PlanRules.progress(plan, null), .0001)
    }
}

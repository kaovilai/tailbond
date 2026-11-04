package dev.tailbond.app

import dev.tailbond.app.report.Redactor
import dev.tailbond.app.report.Report
import dev.tailbond.app.report.ReportText
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/** Report a problem: the port against the Go reference's vectors (agent/internal/report). */
class ReportTest {
    private val v: JsonObject by lazy {
        val f = listOf("../../agent/internal/report/testdata/vectors.json", "../agent/internal/report/testdata/vectors.json")
            .map { File(it) }.first { it.exists() }
        Json.parseToJsonElement(f.readText()).jsonObject
    }

    private fun JsonObject.s(k: String) = this[k]!!.jsonPrimitive.content

    @Test fun redactsLikeGo() {
        val cases = v["redact"]!!.jsonArray
        assertTrue(cases.size >= 20)
        for (c in cases) {
            val o = c.jsonObject
            assertEquals(o.s("name"), o.s("out"), Redactor.redact(o.s("in")))
        }
    }

    @Test fun escapesLikeGo() {
        for (c in v["escape"]!!.jsonArray) {
            val o = c.jsonObject
            assertEquals(o.s("out"), Report.escape(o.s("in")))
        }
    }

    @Test fun reportsLikeGo() {
        val cases = v["reports"]!!.jsonArray
        assertTrue(cases.size >= 4)
        for (c in cases) {
            val o = c.jsonObject
            val r = o["report"]!!.jsonObject
            val rep = Report(r.s("component"), r.s("version"), r.s("device"), r.s("what"), r.s("steps"), r.s("logs"), r.s("crash"))
            assertEquals(o.s("name") + " full", o.s("full"), rep.full())
            assertEquals(o.s("name") + " url", o.s("url"), rep.issueUrl())
            assertTrue(rep.issueUrl().length <= ReportText.MAX_URL)
        }
    }

    @Test fun sameWords() {
        val t = v["texts"]!!.jsonObject
        assertEquals(t.s("notice"), ReportText.NOTICE)
        assertEquals(t.s("confirm"), ReportText.CONFIRM)
        assertEquals(t.s("security_note"), ReportText.SECURITY_NOTE)
        assertEquals(t.s("security_url"), ReportText.SECURITY_URL)
        assertEquals(t.s("repo"), ReportText.REPO)
        assertEquals(t.s("max_url"), ReportText.MAX_URL.toString())
    }

    @Test fun fileName() {
        assertEquals("tailbond-report-20261104-065124.txt", Report.fileName("2026-11-04T06:51:24Z"))
    }
}

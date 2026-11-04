package dev.tailbond.app

import android.app.Application
import dev.tailbond.app.report.CrashCapture

/** Process start: crash capture first (Report a problem, docs/reporting.md). */
class TailbondApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        CrashCapture.install(this)
    }
}

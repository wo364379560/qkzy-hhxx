package com.balabalaim.gcloud.tim.push.appdev.im

import android.content.ContentValues
import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

/** All database work runs on IO. Sending boundaries update one row durably. */
internal class ForwardResultStore(private val context: Context) : SQLiteOpenHelper(context, "rui-forward-results-v2.db", null, 1) {
    init { setWriteAheadLoggingEnabled(true) }
    override fun onConfigure(db: SQLiteDatabase) { db.execSQL("PRAGMA synchronous=FULL") }
    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL("CREATE TABLE jobs(id TEXT PRIMARY KEY,owner TEXT NOT NULL,sdk INTEGER NOT NULL,created INTEGER NOT NULL,running INTEGER NOT NULL)")
        db.execSQL("CREATE INDEX jobs_owner ON jobs(owner,sdk,created)")
        db.execSQL("CREATE TABLE items(job TEXT NOT NULL,pos INTEGER NOT NULL,target TEXT NOT NULL,name TEXT NOT NULL,msg_index INTEGER NOT NULL,source TEXT NOT NULL,status TEXT NOT NULL,error TEXT NOT NULL,sdk_id TEXT NOT NULL,PRIMARY KEY(job,pos))")
    }
    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) = Unit

    suspend fun create(job: NativeForwardJob) = withContext(Dispatchers.IO) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            db.insertOrThrow("jobs", null, ContentValues().apply { put("id", job.id); put("owner", job.owner); put("sdk", job.sdkAppId); put("created", System.currentTimeMillis()); put("running", 1) })
            job.items.forEachIndexed { index, row -> db.insertOrThrow("items", null, values(job.id, index, row)) }
            db.setTransactionSuccessful()
        } finally { db.endTransaction() }
    }
    suspend fun update(job: String, position: Int, row: ForwardItemResult) = withContext(Dispatchers.IO) {
        val value = ContentValues().apply { put("status", row.status); put("error", row.error); put("sdk_id", row.sdkId) }
        check(writableDatabase.update("items", value, "job=? AND pos=?", arrayOf(job, position.toString())) == 1) { "转发结果保存未确认" }
    }
    suspend fun finish(job: String) = withContext(Dispatchers.IO) { writableDatabase.execSQL("UPDATE jobs SET running=0 WHERE id=?", arrayOf(job)) }

    suspend fun restore(owner: String, appID: Int): NativeForwardJob? = withContext(Dispatchers.IO) {
        val db = readableDatabase
        val id = db.rawQuery("SELECT id FROM jobs WHERE owner=? AND sdk=? ORDER BY created DESC LIMIT 1", arrayOf(owner, appID.toString())).use { if (it.moveToFirst()) it.getString(0) else null }
        if (id == null) {
            val raw = context.getSharedPreferences("rui-forward-results", Context.MODE_PRIVATE).getString("$appID:$owner", null) ?: return@withContext null
            val legacy = runCatching {
                val objectValue = JSONObject(raw); val rows = objectValue.getJSONArray("items")
                NativeForwardJob(objectValue.getString("id"), owner, appID, (0 until rows.length()).map { i ->
                    val row = rows.getJSONObject(i)
                    ForwardItemResult(row.getString("target"), row.getString("name"), row.getInt("index"), row.optString("source"), row.getString("status"), row.optString("error"), row.optString("sdk_id"))
                })
            }.getOrNull() ?: return@withContext null
            create(legacy); finish(legacy.id)
            return@withContext recovered(legacy)
        }
        val rows = mutableListOf<ForwardItemResult>()
        db.rawQuery("SELECT target,name,msg_index,source,status,error,sdk_id FROM items WHERE job=? ORDER BY pos", arrayOf(id)).use { cursor ->
            while (cursor.moveToNext()) rows += ForwardItemResult(cursor.getString(0), cursor.getString(1), cursor.getInt(2), cursor.getString(3), cursor.getString(4), cursor.getString(5), cursor.getString(6))
        }
        recovered(NativeForwardJob(id, owner, appID, rows))
    }
    private fun recovered(job: NativeForwardJob) = job.copy(running = false, resumable = false, items = job.items.map { row ->
        when (row.status) {
            "sending" -> row.copy(status = "unknown", error = "进程已中断，请核对接收端；不会自动重发")
            "pending" -> row.copy(status = "interrupted", error = "进程已中断，尚未发送；不自动续发")
            else -> row
        }
    })
    private fun values(job: String, position: Int, row: ForwardItemResult) = ContentValues().apply {
        put("job", job); put("pos", position); put("target", row.target); put("name", row.name); put("msg_index", row.index)
        put("source", row.sourceId); put("status", row.status); put("error", row.error); put("sdk_id", row.sdkId)
    }
}

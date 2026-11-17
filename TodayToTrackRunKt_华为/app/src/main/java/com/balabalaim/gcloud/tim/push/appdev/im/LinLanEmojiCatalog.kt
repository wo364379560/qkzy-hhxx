package com.balabalaim.gcloud.tim.push.appdev.im

import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.input.OffsetMapping
import androidx.compose.ui.text.input.TransformedText
import androidx.compose.ui.text.input.VisualTransformation

/**
 * Tencent default pack — same tags as ShangLan / chixin_TUIKit_Flutter atomic-x.
 * Assets: assets/emoji/emoji_0.png … emoji_61.png
 * 输入框用 Unicode 展示，发送仍保留 [TUIEmoji_*] 标签。
 */
object LinLanEmojiCatalog {
    data class Item(val tag: String, val assetPath: String, val glyph: String)

    private val tagsAndGlyphs: List<Pair<String, String>> = listOf(
        "[TUIEmoji_Smile]" to "😊", "[TUIEmoji_Expect]" to "🤩", "[TUIEmoji_Blink]" to "😉",
        "[TUIEmoji_Guffaw]" to "😂", "[TUIEmoji_KindSmile]" to "🙂", "[TUIEmoji_Haha]" to "😄",
        "[TUIEmoji_Cheerful]" to "😁", "[TUIEmoji_Speechless]" to "😐", "[TUIEmoji_Amazed]" to "😲",
        "[TUIEmoji_Sorrow]" to "😢", "[TUIEmoji_Complacent]" to "😌", "[TUIEmoji_Silly]" to "🤪",
        "[TUIEmoji_Lustful]" to "😏", "[TUIEmoji_Giggle]" to "😆", "[TUIEmoji_Kiss]" to "😘",
        "[TUIEmoji_Wail]" to "😭", "[TUIEmoji_TearsLaugh]" to "🤣", "[TUIEmoji_Trapped]" to "😖",
        "[TUIEmoji_Mask]" to "😷", "[TUIEmoji_Fear]" to "😨", "[TUIEmoji_BareTeeth]" to "😬",
        "[TUIEmoji_FlareUp]" to "😡", "[TUIEmoji_Yawn]" to "🥱", "[TUIEmoji_Tact]" to "🧐",
        "[TUIEmoji_Stareyes]" to "🌟", "[TUIEmoji_ShutUp]" to "🤐", "[TUIEmoji_Sigh]" to "😮",
        "[TUIEmoji_Hehe]" to "🤭", "[TUIEmoji_Silent]" to "🤫", "[TUIEmoji_Surprised]" to "😯",
        "[TUIEmoji_Askance]" to "😒", "[TUIEmoji_Ok]" to "👌", "[TUIEmoji_Shit]" to "💩",
        "[TUIEmoji_Monster]" to "👾", "[TUIEmoji_Daemon]" to "😈", "[TUIEmoji_Rage]" to "🤬",
        "[TUIEmoji_Fool]" to "🤡", "[TUIEmoji_Pig]" to "🐷", "[TUIEmoji_Cow]" to "🐮",
        "[TUIEmoji_Ai]" to "🤖", "[TUIEmoji_Skull]" to "💀", "[TUIEmoji_Bombs]" to "💣",
        "[TUIEmoji_Coffee]" to "☕", "[TUIEmoji_Cake]" to "🎂", "[TUIEmoji_Beer]" to "🍺",
        "[TUIEmoji_Flower]" to "🌸", "[TUIEmoji_Watermelon]" to "🍉", "[TUIEmoji_Rich]" to "💰",
        "[TUIEmoji_Heart]" to "❤️", "[TUIEmoji_Moon]" to "🌙", "[TUIEmoji_Sun]" to "☀️",
        "[TUIEmoji_Star]" to "⭐", "[TUIEmoji_RedPacket]" to "🧧", "[TUIEmoji_Celebrate]" to "🎉",
        "[TUIEmoji_Bless]" to "🙏", "[TUIEmoji_Fortune]" to "🀄", "[TUIEmoji_Convinced]" to "👍",
        "[TUIEmoji_Prohibit]" to "🚫", "[TUIEmoji_666]" to "6️⃣", "[TUIEmoji_857]" to "8️⃣",
        "[TUIEmoji_Knife]" to "🔪", "[TUIEmoji_Like]" to "👍",
    )

    val items: List<Item> = tagsAndGlyphs.mapIndexed { index, (tag, glyph) ->
        Item(tag, "file:///android_asset/emoji/emoji_$index.png", glyph)
    }

    private val tagToAsset: Map<String, String> = items.associate { it.tag to it.assetPath }
    private val tagToGlyph: Map<String, String> = items.associate { it.tag to it.glyph }

    private val TAG_REGEX = Regex("""\[TUIEmoji_\w{2,}]""")

    fun assetForTag(tag: String): String? = tagToAsset[tag]

    fun glyphForTag(tag: String): String? = tagToGlyph[tag]

    fun previewText(raw: String): String =
        TAG_REGEX.replace(raw) { m -> tagToGlyph[m.value] ?: "[表情]" }

    fun splitWithEmojis(raw: String): List<Pair<Boolean, String>> {
        if (raw.isEmpty()) return emptyList()
        val out = mutableListOf<Pair<Boolean, String>>()
        var last = 0
        for (m in TAG_REGEX.findAll(raw)) {
            if (m.range.first > last) {
                out.add(false to raw.substring(last, m.range.first))
            }
            out.add(true to m.value)
            last = m.range.last + 1
        }
        if (last < raw.length) out.add(false to raw.substring(last))
        return out
    }

    /**
     * 删除时若碰到 [TUIEmoji_*] 标签的任意一部分，整段删掉，避免留下 `[TUIEmoji_xxx`。
     */
    fun coerceAfterEdit(before: String, after: String): String {
        if (after.length >= before.length) return after
        var prefix = 0
        while (prefix < after.length && prefix < before.length && after[prefix] == before[prefix]) {
            prefix++
        }
        var suffix = 0
        while (
            suffix < after.length - prefix &&
            suffix < before.length - prefix &&
            after[after.length - 1 - suffix] == before[before.length - 1 - suffix]
        ) {
            suffix++
        }
        var removeStart = prefix
        var removeEnd = before.length - suffix
        if (removeStart >= removeEnd) return after
        for (m in TAG_REGEX.findAll(before)) {
            val tagStart = m.range.first
            val tagEnd = m.range.last + 1
            if (removeStart < tagEnd && removeEnd > tagStart) {
                removeStart = minOf(removeStart, tagStart)
                removeEnd = maxOf(removeEnd, tagEnd)
            }
        }
        return before.removeRange(removeStart, removeEnd)
    }

    /** 输入框：把 [TUIEmoji_*] 显示成真正表情字符。 */
    val inputVisualTransformation: VisualTransformation = VisualTransformation { text ->
        val original = text.text
        if (original.isEmpty()) {
            return@VisualTransformation TransformedText(text, OffsetMapping.Identity)
        }
        val out = StringBuilder()
        val origToTrans = IntArray(original.length + 1)
        // Index i = transformed offset → original offset (size = transformedLen + 1)
        val transToOrig = ArrayList<Int>()
        var i = 0
        var t = 0
        while (i < original.length) {
            val m = TAG_REGEX.find(original, i)
            if (m != null && m.range.first == i) {
                val glyph = tagToGlyph[m.value] ?: m.value
                val tagStart = m.range.first
                val tagEnd = m.range.last + 1
                for (k in tagStart until tagEnd) {
                    origToTrans[k] = t
                }
                out.append(glyph)
                // Caret on/inside glyph → tag start; caret after glyph → tag end.
                repeat(glyph.length) { transToOrig.add(tagStart) }
                t += glyph.length
                i = tagEnd
            } else {
                origToTrans[i] = t
                out.append(original[i])
                transToOrig.add(i)
                t++
                i++
            }
        }
        origToTrans[original.length] = t
        transToOrig.add(original.length)
        val mapping = object : OffsetMapping {
            override fun originalToTransformed(offset: Int): Int {
                val clamped = offset.coerceIn(0, original.length)
                return origToTrans[clamped]
            }

            override fun transformedToOriginal(offset: Int): Int {
                val clamped = offset.coerceIn(0, transToOrig.lastIndex)
                return transToOrig[clamped]
            }
        }
        TransformedText(AnnotatedString(out.toString()), mapping)
    }
}

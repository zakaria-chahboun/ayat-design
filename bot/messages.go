package bot

import "fmt"

// ─── Selection confirmation ───────────────────────────────────────────────────

// GetSelectionMessage returns the header line shown after a valid verse request.
// e.g. "اخترتم سورة العلق، الآية (9)" or "اخترتم سورة العلق، الآيات (4-9)"
func GetSelectionMessage(surahName string, startAyah, endAyah int) string {
	if startAyah == endAyah {
		return fmt.Sprintf("اخترتم سورة %s، الآية (%d)", surahName, startAyah)
	}
	return fmt.Sprintf("اخترتم سورة %s، الآيات (%d-%d)", surahName, startAyah, endAyah)
}

// ─── Text output ──────────────────────────────────────────────────────────────

// GetTextHeader returns the caption / header sent with a text response.
// e.g. "سورة العلق، الآية (9)"
func GetTextHeader(surahName string, startAyah, endAyah int) string {
	if startAyah == endAyah {
		return fmt.Sprintf("سورة %s، الآية (%d)", surahName, startAyah)
	}
	return fmt.Sprintf("سورة %s، الآيات (%d-%d)", surahName, startAyah, endAyah)
}

// GetTextMessage returns the full text message: header + 2 blank lines + verses in a code block.
func GetTextMessage(surahName string, startAyah, endAyah int, versesText string) string {
	header := GetTextHeader(surahName, startAyah, endAyah)
	return fmt.Sprintf("%s\n\n```\n%s\n```", header, versesText)
}

// ─── Image flow ───────────────────────────────────────────────────────────────

func GetChooseStyleMessage() string {
	return "اختر نوع التصميم:"
}

// GetImageWaitingMessage returns the message edited into the chat while the image renders.
func GetImageWaitingMessage(selectionMsg string) string {
	return selectionMsg + "\n\nيرجى الانتظار قليلاً حتى تجهز الصورة بإذن الله 🕐"
}

// GetImageCaption returns the caption attached to the finished image.
func GetImageCaption(surahName string, startAyah, endAyah int) string {
	return GetTextHeader(surahName, startAyah, endAyah)
}

// ─── Video flow ───────────────────────────────────────────────────────────────

func GetChooseReciterMessage() string {
	return "اختر القارئ المفضل:"
}

// GetVideoVerseExceededMessage returns the edited message when the user picks
// more verses than the video limit allows.
func GetVideoVerseExceededMessage(selectionMsg string, limit int) string {
	return fmt.Sprintf(
		"%s\n\nعذراً، لا يمكن اختيار أكثر من %d آيات في المقطع، يرجى تقليل عدد الآيات.",
		selectionMsg, limit,
	)
}

// GetVideoWaitingMessage returns the edited message confirming the video job is queued.
func GetVideoWaitingMessage(selectionMsg string) string {
	return selectionMsg + "\n\nسيتم إعلامكم حالما يجهز المقطع إن شاء الله، يرجى الانتظار 🎬"
}

// ─── Validation / error messages ─────────────────────────────────────────────

// GetInvalidFormatMessage is sent when the user's input doesn't match the expected pattern.
func GetInvalidFormatMessage() string {
	return "⚠️ صيغة غير صحيحة. يرجى استخدام:\n[اسم السورة] [الآية أو من-إلى]\n\nمثال:\nالبقرة 2-3\nالفاتحة ١-٣"
}

// GetSurahNotFoundMessage wraps the error returned by the surah lookup.
func GetSurahNotFoundMessage(err error) string {
	return "⚠️ " + err.Error()
}

// GetVerseRangeErrorMessage is sent when fetching verses fails.
func GetVerseRangeErrorMessage(err error) string {
	return "⚠️ " + err.Error()
}

// GetTextLimitExceededMessage is sent when the user asks for more verses than the text limit.
func GetTextLimitExceededMessage(limit int) string {
	return fmt.Sprintf("⚠️ يمكن عرض %d آية كحد أقصى كنص. يرجى تقليل عدد الآيات.", limit)
}

// GetImageLimitExceededMessage is sent when the user asks for more verses than the image limit.
func GetImageLimitExceededMessage(limit int) string {
	return fmt.Sprintf("⚠️ يمكن توليد صورة لـ %d آيات كحد أقصى. يرجى تقليل عدد الآيات.", limit)
}

// GetExpiredRequestMessage is sent as a callback toast when the session has expired.
func GetExpiredRequestMessage() string {
	return "انتهت صلاحية الطلب، يرجى إرساله مجدداً."
}

// GetAlreadyBusyMessage is sent when the user sends a new request while a
// previous job is still in the queue.
func GetAlreadyBusyMessage() string {
	return "⏳ طلبك السابق لا يزال قيد المعالجة، يرجى الانتظار حتى ينتهي."
}

// GetTextWaitingMessage is the edited message shown while a text job is queued.
func GetTextWaitingMessage(selectionMsg string) string {
	return selectionMsg + "\n\nجارٍ تجهيز النص..."
}

// ─── Queue messages ───────────────────────────────────────────────────────────

// GetQueueFullMessage is sent when the job queue is at capacity.
func GetQueueFullMessage() string {
	return "⚠️ الطابور ممتلئ حالياً، يرجى المحاولة بعد قليل."
}

// GetImageQueuedMessage is sent as a toast confirming the image job position.
func GetImageQueuedMessage(position int) string {
	return fmt.Sprintf("✅ تم إضافة طلبك إلى الطابور. موضعك: %d", position)
}

// GetVideoQueuedMessage is sent confirming the video job position.
func GetVideoQueuedMessage(position int) string {
	return fmt.Sprintf("✅ تم إضافة طلب الفيديو إلى الطابور. موضعك: %d", position)
}

// ─── Welcome ──────────────────────────────────────────────────────────────────

// GetStartMessage returns the welcome message sent on /start.
func GetStartMessage() string {
	return "مرحباً بك في روبوت آيات وأجر 📖\n\n" +
		"أرسل طلبك بالصيغة التالية:\n" +
		"[اسم السورة] [رقم الآية أو من-إلى]\n\n" +
		"مثال:\n" +
		"البقرة 2-3\n" +
		"آل عمران 7\n" +
		"الفاتحة ١-٣\n" +
		"يوسف ٤\n\n" +
		"> 📝 ملاحظة: الفيديو قد يأخذ وقتا بسبب الضغط على الخادم، لكن الصور والنصوص ستكون دائماً جاهزة بفضل الله"
}

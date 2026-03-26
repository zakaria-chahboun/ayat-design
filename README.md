# AyatDesignBot

Quran verse images and videos bot on Telegram.

<img src="logo.png" width="300" alt="Logo">

## The Story

I wanted a service that gives me Quran verses, but none of them let me show multiple verses on a single image. I had to take the verses and manually combine them in an image editor every time. So I decided to build my own.

I made it public through a Telegram bot because it's the easiest way for anyone to use it. Then I added video support too.

This project is completely free to use. Consider it Sadaqa Jariya from me and my family.

May Allah forgive my father (Said) and my sister (Aicha) and accept this work as charity on their behalf. Ameen.

Enjoy, and please make dua for me.

## Try it now

https://t.me/AyatDesignBot

## How to use

1. Open the bot
2. Send your request in this format: `[Surah Name] [Verse or From-To]`
3. Choose a design style
4. Choose output type (text or image or video)
5. For video, choose a reciter

You can use English numerals (123) or Hindi numerals (١٢٣).

Examples:
```
البقرة 1
الفاتحة 1-3
يوسف 4-6
البقرة ١
الفاتحة ١-٣
يوسف ٤-٦
```

## Examples

**Images:**

<img src="examples/alhaj.jpg" width="300" alt="Al-Hajj">
<img src="examples/alqalam.jpg" width="300" alt="An-Qalam">

**Videos:**

<video src="examples/alkawtar.mp4" width="300" controls></video>
<video src="examples/almulk.mp4" width="300" controls></video>

## Tech Stack

Go + FFmpeg

FFmpeg handles video generation. Go manages everything else.

Why Go and FFmpeg instead of Chrome headless or Python?

- No Docker needed, just install FFmpeg and run
- Go compiles to a single binary, deploy anywhere
- FFmpeg is fast and handles video encoding efficiently
- Go is faster than Python or JavaScript for this kind of work
- No browser overhead, keeps things lightweight

## Configuration

Edit `config.json` to customize the bot:

### Cache
```json
"cache": {
  "audio": true
}
```
When enabled, audio files are downloaded once and cached locally in `cache/audio/`. Reduces bandwidth and speeds up subsequent requests.

### Styles
Define background images and text colors for verse designs.

### Reciters
Configure Quran reciters. Audio is fetched from [everyayah.com](https://everyayah.com/data/).

### Bypass Limit
Set `BYPASS_KEYWORD` environment variable to enable bypassing verse limits for all output types (text: 30, image: 10, video: 3).
When a user includes this keyword in their request the verse limit is bypassed, example:

```
BYPASS_KEYWORD=مميز
```

Then the prompt will be like:
```
النور 1-7 مميز
```


## Creator
تم بفضل الله وبحمده وتيسيره
Zakaria Chahboun

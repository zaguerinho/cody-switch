package assembler

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/zaguerinho/claude-switch/video-tutorial/embed"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

// AssembleParams holds all inputs for assembly.
type AssembleParams struct {
	Manifest     *manifest.Manifest
	SynthResults []tts.SynthResult
	SourceDir    string
	OutputPath   string
	Slug         string
	Config       *config.Config
}

// tutorialData is the JSON payload embedded in the HTML for the browser player.
type tutorialData struct {
	Slug          string            `json:"slug"`
	Title         string            `json:"title"`
	Words         []SyncWord        `json:"words"`
	WordToCue     []string          `json:"wordToCue"`
	CueMap        map[string]manifest.Cue `json:"cueMap"`
	Chapters      []SyncChapter     `json:"chapters"`
	TotalDuration float64           `json:"totalDuration"`
	AudioSegments []AudioSegment    `json:"audioSegments"`
}

// Assemble builds the final self-contained tutorial HTML file.
func Assemble(p AssembleParams) error {
	fmt.Fprintf(os.Stderr, "Assembling tutorial: %q\n", p.Manifest.Title)

	// Read embedded assets.
	css := embed.MustReadAsset("template/player.css")
	syncJs := embed.MustReadAsset("browser/sync.js")
	rendererJs := embed.MustReadAsset("browser/renderer.js")
	chatbotJs := embed.MustReadAsset("browser/chatbot.js")

	// Strip export lines from inlined JS (same as Node.js version).
	syncJs = strings.Replace(syncJs, "export default SyncEngine;", "", 1)
	rendererJs = strings.Replace(rendererJs, "export default SceneRenderer;", "", 1)
	chatbotJs = strings.Replace(chatbotJs, "export default Chatbot;", "", 1)

	// Segment gap in seconds (Config.SegmentGapMS -> seconds).
	segmentGapSec := float64(p.Config.SegmentGapMS) / 1000.0

	// Build components.
	fileTabsHTML, codeBlocksHTML, err := BuildCodeViewer(p.Manifest, p.SourceDir)
	if err != nil {
		return err
	}
	chapterListHTML := BuildChapterList(p.Manifest)
	syncData, err := BuildSyncData(p.Manifest, p.SynthResults, segmentGapSec)
	if err != nil {
		return err
	}
	cueMap := BuildCueMap(p.Manifest)
	audioSegments, err := BuildAudioSegments(p.Manifest, p.SynthResults)
	if err != nil {
		return err
	}

	// Count segments.
	segCount := 0
	for _, ch := range p.Manifest.Chapters {
		segCount += len(ch.Segments)
	}

	// Build tutorial data JSON.
	td := tutorialData{
		Slug:          p.Slug,
		Title:         p.Manifest.Title,
		Words:         syncData.Words,
		WordToCue:     syncData.WordToCue,
		CueMap:        cueMap,
		Chapters:      syncData.Chapters,
		TotalDuration: syncData.TotalDuration,
		AudioSegments: audioSegments,
	}
	dataJSON, err := json.Marshal(td)
	if err != nil {
		return fmt.Errorf("assembler: marshal tutorial data: %w", err)
	}

	// Format time strings.
	totalMins := int(math.Ceil(syncData.TotalDuration / 60))
	durationMins := int(math.Floor(syncData.TotalDuration / 60))
	durationSecs := int(math.Floor(math.Mod(syncData.TotalDuration, 60)))

	// Safe title for HTML context.
	escapedTitle := html.EscapeString(p.Manifest.Title)

	// First chapter title (for topbar).
	firstChapterTitle := ""
	if len(p.Manifest.Chapters) > 0 {
		firstChapterTitle = p.Manifest.Chapters[0].Title
	}

	// Build the final HTML. This matches the structure from assemble.js exactly.
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>`)
	b.WriteString(escapedTitle)
	b.WriteString(` — Tutorial</title>
  <style>`)
	b.WriteString(css)
	b.WriteString(`</style>
</head>
<body>

<div class="player-layout">

  <aside class="sidebar">
    <div class="sidebar-header">
      <h1>`)
	b.WriteString(escapedTitle)
	b.WriteString(`</h1>
      <div class="tutorial-meta">`)
	fmt.Fprintf(&b, "%d chapters &middot; %d min", len(p.Manifest.Chapters), totalMins)
	b.WriteString(`</div>
    </div>
    <ol class="chapter-list">
`)
	b.WriteString(chapterListHTML)
	b.WriteString(`
    </ol>
    <div class="sidebar-footer">
      <div class="sidebar-progress">
        <span>0 of `)
	fmt.Fprintf(&b, "%d", len(p.Manifest.Chapters))
	b.WriteString(`</span>
        <div class="sidebar-progress-bar">
          <div class="sidebar-progress-fill" style="width: 0%"></div>
        </div>
      </div>
    </div>
  </aside>

  <header class="topbar">
    <span class="topbar-chapter-badge">Chapter 1</span>
    <span class="topbar-title">`)
	b.WriteString(firstChapterTitle)
	b.WriteString(`</span>
    <div class="topbar-actions">
      <button class="btn-icon btn-chatbot-toggle" aria-label="Ask about this code">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
      </button>
    </div>
    <div class="topbar-progress">
      <div class="topbar-progress-fill" style="width: 0%"></div>
    </div>
  </header>

  <main class="code-viewer">
    <div class="file-tabs">
`)
	b.WriteString(fileTabsHTML)
	b.WriteString(`
    </div>
    <div class="code-container">
`)
	b.WriteString(codeBlocksHTML)
	b.WriteString(`
    </div>
    <div class="transcript-bar">
      <div class="transcript-text"></div>
    </div>
  </main>

  <footer class="controls">
    <button class="btn-play" aria-label="Play">
      <svg viewBox="0 0 24 24" fill="currentColor">
        <polygon points="5,3 19,12 5,21"/>
      </svg>
    </button>
    <span class="time-display">0:00 / `)
	fmt.Fprintf(&b, "%d:%02d", durationMins, durationSecs)
	b.WriteString(`</span>
    <div class="scrub-bar">
      <div class="scrub-track">
        <div class="scrub-fill" style="width: 0%">
          <span class="scrub-thumb"></span>
        </div>
      </div>
    </div>
    <div class="control-group">
      <button class="btn-speed">1x</button>
      <button class="btn-control" aria-label="Volume">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
          <path d="M19.07 4.93a10 10 0 0 1 0 14.14"/>
          <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
        </svg>
      </button>
      <div class="volume-slider">
        <div class="volume-fill"></div>
      </div>
    </div>
  </footer>

</div>

<div class="chatbot-drawer">
  <div class="chatbot-header">
    <h2>Ask about this code</h2>
    <button class="btn-icon btn-chatbot-close" aria-label="Close">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="18" y1="6" x2="6" y2="18"/>
        <line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </button>
  </div>
  <div class="chatbot-messages">
    <div class="chatbot-empty">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <p>Pause the tutorial and ask anything about the code you're seeing.</p>
    </div>
  </div>
  <div class="chatbot-input">
    <input type="text" placeholder="Ask a question..." aria-label="Chat input">
    <button aria-label="Send">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="22" y1="2" x2="11" y2="13"/>
        <polygon points="22 2 15 22 11 13 2 9 22 2"/>
      </svg>
    </button>
  </div>
</div>

<script id="tutorial-data" type="application/json">`)
	b.Write(dataJSON)
	b.WriteString(`</script>

<script type="module">
// --- Inline sync engine (no external imports in self-contained file) ---
`)
	b.WriteString(syncJs)
	b.WriteString(`

// --- Inline renderer ---
`)
	b.WriteString(rendererJs)
	b.WriteString(`

// --- Inline chatbot ---
`)
	b.WriteString(chatbotJs)
	b.WriteString(`

// --- Load tutorial data ---
const data = JSON.parse(document.getElementById('tutorial-data').textContent);

// --- Audio engine: sequence base64 segments ---
// AudioEngine: uses <audio> element for playback so preservesPitch works.
// Web Audio's AudioBufferSourceNode does NOT support preservesPitch —
// only <audio>/<video> elements do. So we decode + concatenate segments
// into a single WAV blob, then play it via <audio>.
const AudioEngine = (() => {
  const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  const audio = document.createElement('audio');
  audio.preservesPitch = true;
  if ('mozPreservesPitch' in audio) audio.mozPreservesPitch = true;
  if ('webkitPreservesPitch' in audio) audio.webkitPreservesPitch = true;
  let totalDuration = 0;

  function writeStr(view, off, s) {
    for (let i = 0; i < s.length; i++) view.setUint8(off + i, s.charCodeAt(i));
  }

  // Encode an AudioBuffer to a WAV Blob.
  function bufferToWav(buf) {
    const numCh = buf.numberOfChannels, sr = buf.sampleRate;
    const blockAlign = numCh * 2, dataLen = buf.length * blockAlign;
    const ab = new ArrayBuffer(44 + dataLen);
    const v = new DataView(ab);
    writeStr(v, 0, 'RIFF');  v.setUint32(4, 36 + dataLen, true);
    writeStr(v, 8, 'WAVE');  writeStr(v, 12, 'fmt ');
    v.setUint32(16, 16, true); v.setUint16(20, 1, true);
    v.setUint16(22, numCh, true); v.setUint32(24, sr, true);
    v.setUint32(28, sr * blockAlign, true); v.setUint16(32, blockAlign, true);
    v.setUint16(34, 16, true); writeStr(v, 36, 'data');
    v.setUint32(40, dataLen, true);
    let off = 44;
    for (let i = 0; i < buf.length; i++) {
      for (let ch = 0; ch < numCh; ch++) {
        const s = Math.max(-1, Math.min(1, buf.getChannelData(ch)[i]));
        v.setInt16(off, s * 0x7FFF | 0, true); off += 2;
      }
    }
    return new Blob([ab], { type: 'audio/wav' });
  }

  async function init(audioSegments) {
    const buffers = [];
    const sampleRate = audioCtx.sampleRate;
    const GAP_SECONDS = 0.3;
    const gapSamples = Math.floor(GAP_SECONDS * sampleRate);

    for (const seg of audioSegments) {
      const binary = atob(seg.audioBase64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      buffers.push(await audioCtx.decodeAudioData(bytes.buffer.slice(0)));
    }

    let totalSamples = 0;
    for (let i = 0; i < buffers.length; i++) {
      totalSamples += buffers[i].length;
      if (i < buffers.length - 1) totalSamples += gapSamples;
    }

    const channels = buffers.length > 0 ? buffers[0].numberOfChannels : 1;
    const master = audioCtx.createBuffer(channels, totalSamples, sampleRate);
    let wp = 0;
    for (let i = 0; i < buffers.length; i++) {
      for (let ch = 0; ch < channels; ch++) {
        const src = buffers[i].numberOfChannels > ch ? buffers[i].getChannelData(ch) : buffers[i].getChannelData(0);
        master.getChannelData(ch).set(src, wp);
      }
      wp += buffers[i].length;
      if (i < buffers.length - 1) wp += gapSamples;
    }

    // Convert to WAV blob → <audio> src (enables preservesPitch)
    audio.src = URL.createObjectURL(bufferToWav(master));
    await new Promise((resolve, reject) => {
      audio.oncanplaythrough = resolve;
      audio.onerror = reject;
      audio.load();
    });
    totalDuration = audio.duration;

    audio.onended = () => {
      document.dispatchEvent(new CustomEvent('playback:ended', {}));
      document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: false } }));
    };
  }

  return {
    init,
    play()    { audio.play().catch(() => {}); },
    pause()   { audio.pause(); },
    seek(t)   { audio.currentTime = Math.max(0, Math.min(t, totalDuration)); },
    setRate(r){ audio.playbackRate = r; },
    getTime() { return audio.currentTime || 0; },
    get isPlaying() { return !audio.paused && !audio.ended; },
    get duration()  { return totalDuration; }
  };
})();

// --- Initialize ---
SceneRenderer.init();
SceneRenderer.buildTranscript(data.words, data.wordToCue);

SyncEngine.init({
  wordTimestamps: data.words,
  cueMap: data.cueMap,
  wordToCue: data.wordToCue,
  chapters: data.chapters,
  totalDuration: data.totalDuration
});

// Decode audio segments
await AudioEngine.init(data.audioSegments);

// Override SyncEngine play/pause to use real audio
const origPlay = SyncEngine.play.bind(SyncEngine);
const origPause = SyncEngine.pause.bind(SyncEngine);

// Tick loop driven by real audio time
let tickRaf;
function audioTick() {
  if (AudioEngine.isPlaying) {
    SyncEngine.seek(AudioEngine.getTime());
    tickRaf = requestAnimationFrame(audioTick);
  }
}

document.querySelector('.btn-play').addEventListener('click', () => {
  if (AudioEngine.isPlaying) {
    AudioEngine.pause();
    SyncEngine.pause();
    cancelAnimationFrame(tickRaf);
  } else {
    AudioEngine.play();
    tickRaf = requestAnimationFrame(audioTick);
    document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: true } }));
  }
});

const scrubTrack = document.querySelector('.scrub-track');
scrubTrack.addEventListener('click', (e) => {
  const rect = scrubTrack.getBoundingClientRect();
  const pct = ((e.clientX - rect.left) / rect.width) * 100;
  const time = (pct / 100) * data.totalDuration;
  AudioEngine.seek(time);
  SyncEngine.seek(time);
  if (!AudioEngine.isPlaying) {
    AudioEngine.play();
    tickRaf = requestAnimationFrame(audioTick);
    document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: true } }));
  }
});

const SPEEDS = [0.75, 1, 1.25, 1.5, 2];
let speedIdx = 1;
document.querySelector('.btn-speed').addEventListener('click', () => {
  speedIdx = (speedIdx + 1) % SPEEDS.length;
  AudioEngine.setRate(SPEEDS[speedIdx]);
  document.dispatchEvent(new CustomEvent('playback:rate', { detail: { rate: SPEEDS[speedIdx] } }));
});

document.querySelectorAll('.chapter-item').forEach((item, idx) => {
  item.addEventListener('click', () => {
    const ch = data.chapters[idx];
    if (ch && ch.startWord < data.words.length) {
      const time = data.words[ch.startWord].start;
      AudioEngine.seek(time);
      SyncEngine.seek(time);
      if (!AudioEngine.isPlaying) {
        AudioEngine.play();
        tickRaf = requestAnimationFrame(audioTick);
        document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: true } }));
      }
    }
  });
});

// Chatbot: lazy-initialize on first open to avoid triggering API/SSO on page load.
let chatbotReady = false;
document.querySelector('.btn-chatbot-toggle').addEventListener('click', async () => {
  if (document.body.classList.contains('chatbot-open')) {
    document.body.classList.remove('chatbot-open');
    SyncEngine.resume();
  } else {
    if (!chatbotReady) {
      await Chatbot.init({
        syncEngine: SyncEngine,
        chapters: data.chapters,
        words: data.words,
        cueMap: data.cueMap
      });
      chatbotReady = true;
    }
    document.body.classList.add('chatbot-open');
    SyncEngine.saveAndPause();
    AudioEngine.pause();
    cancelAnimationFrame(tickRaf);
  }
});

document.querySelector('.btn-chatbot-close').addEventListener('click', () => {
  document.body.classList.remove('chatbot-open');
  SyncEngine.resume();
});

// Bridge: chatbot module requests pause (e.g. queue-until-pause flush)
document.addEventListener('chatbot:pause-requested', () => {
  AudioEngine.pause();
  cancelAnimationFrame(tickRaf);
  SyncEngine.saveAndPause();
  document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: false } }));
});

document.addEventListener('keydown', (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  switch (e.key) {
    case ' ':
      e.preventDefault();
      document.querySelector('.btn-play').click();
      break;
    case 'ArrowRight':
      e.preventDefault();
      AudioEngine.seek(AudioEngine.getTime() + 5);
      SyncEngine.seek(AudioEngine.getTime());
      break;
    case 'ArrowLeft':
      e.preventDefault();
      AudioEngine.seek(Math.max(0, AudioEngine.getTime() - 5));
      SyncEngine.seek(AudioEngine.getTime());
      break;
  }
});

// File tabs switching
document.querySelectorAll('.file-tab').forEach(tab => {
  tab.addEventListener('click', () => {
    const file = tab.dataset.file;
    document.querySelectorAll('.file-tab').forEach(t => t.classList.remove('file-tab--active'));
    tab.classList.add('file-tab--active');
    document.querySelectorAll('.code-block').forEach(b => {
      b.style.display = b.dataset.file === file ? '' : 'none';
    });
  });
});


// --- Progress persistence (localStorage) ---
const STORAGE_KEY = 'tutorial-progress-' + (data.slug || 'default');

function saveProgress() {
  const state = {
    time: AudioEngine.getTime(),
    chapter: SyncEngine.activeChapter,
    timestamp: Date.now()
  };
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch(e) {}
}

// Save progress every 3 seconds during playback
let progressInterval = null;
document.addEventListener('playback:state', (e) => {
  if (e.detail.playing) {
    progressInterval = setInterval(saveProgress, 3000);
  } else {
    clearInterval(progressInterval);
    saveProgress();
  }
});

// Restore progress on load
try {
  const saved = JSON.parse(localStorage.getItem(STORAGE_KEY));
  if (saved && saved.time > 1 && saved.time < data.totalDuration - 1) {
    // Show resume banner
    const banner = document.createElement('div');
    banner.style.cssText = 'position:fixed;bottom:80px;left:50%;transform:translateX(-50%);background:#1a1a2e;color:#e2e8f0;padding:10px 20px;border-radius:8px;font-size:13px;z-index:200;display:flex;align-items:center;gap:12px;box-shadow:0 4px 20px rgba(0,0,0,0.3);';
    const mins = Math.floor(saved.time / 60);
    const secs = Math.floor(saved.time % 60).toString().padStart(2, '0');
    banner.innerHTML = 'Resume from ' + mins + ':' + secs + '? ' +
      '<button style="background:#6c63ff;color:white;border:none;padding:6px 14px;border-radius:6px;cursor:pointer;font-size:13px;" id="resume-yes">Resume</button>' +
      '<button style="background:transparent;color:#a0aec0;border:1px solid #a0aec0;padding:6px 14px;border-radius:6px;cursor:pointer;font-size:13px;" id="resume-no">Start over</button>';
    document.body.appendChild(banner);
    document.getElementById('resume-yes').onclick = () => {
      AudioEngine.seek(saved.time);
      SyncEngine.seek(saved.time);
      AudioEngine.play();
      tickRaf = requestAnimationFrame(audioTick);
      document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: true } }));
      banner.remove();
    };
    document.getElementById('resume-no').onclick = () => {
      localStorage.removeItem(STORAGE_KEY);
      banner.remove();
    };
    setTimeout(() => { if (banner.parentNode) banner.remove(); }, 10000);
  }
} catch(e) {}
</script>

</body>
</html>`)

	htmlContent := b.String()

	// Ensure output directory exists.
	outputDir := filepath.Dir(p.OutputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("assembler: create output dir: %w", err)
	}

	if err := os.WriteFile(p.OutputPath, []byte(htmlContent), 0o644); err != nil {
		return fmt.Errorf("assembler: write output: %w", err)
	}

	sizeMB := float64(len(htmlContent)) / 1024 / 1024
	fmt.Fprintf(os.Stderr, "Tutorial assembled: %s\n", p.OutputPath)
	fmt.Fprintf(os.Stderr, "  %d chapters, %d segments\n", len(p.Manifest.Chapters), segCount)
	fmt.Fprintf(os.Stderr, "  %.1f MB total\n", sizeMB)

	return nil
}

/**
 * HTML Assembler — build-time module.
 *
 * Takes:
 *   - template.html (the player shell)
 *   - player.css
 *   - manifest.json (scene manifest from generate-manifest.js)
 *   - Per-segment { audioBase64, wordTimestamps } (from elevenlabs.js)
 *   - Source files referenced by the manifest
 *   - Browser modules (sync.js, renderer.js)
 *
 * Produces: a single self-contained tutorial.html file.
 */

const fs = require('fs');
const path = require('path');

/**
 * Simple syntax tokenizer for common languages.
 * Returns HTML with <span class="tk-*"> tokens.
 * Supports JS/TS, Python, Go, PHP, Ruby, and generic code.
 */
function tokenizeLine(line) {
  // Tokenize into segments first, then render to HTML at the end.
  // This avoids regex passes clobbering each other's HTML attributes.
  const tokens = []; // [{ type: 'plain'|'comment'|'string'|'keyword'|'function'|'number', text }]

  const KW_SET = new Set([
    'const','let','var','function','return','if','else','for','while','do',
    'switch','case','break','continue','class','import','export','from',
    'default','async','await','try','catch','finally','throw','new','typeof',
    'instanceof','in','of','true','false','null','undefined','this',
    'def','self','elif','pass','raise','with','as','yield','lambda',
    'func','go','defer','chan','select','range','package','type','struct',
    'interface','nil','None','True','False','not','and','or','is','print',
    'extends','implements','static','public','private','protected','final',
    'abstract','super','void','int','float','double','bool','string','char'
  ]);

  let i = 0;
  let buf = '';

  function flushBuf() {
    if (buf.length > 0) {
      tokens.push({ type: 'plain', text: buf });
      buf = '';
    }
  }

  while (i < line.length) {
    const ch = line[i];

    // Line comments: // or # (Python/Ruby/Shell — only if not inside a word)
    if (ch === '/' && line[i + 1] === '/') {
      flushBuf();
      tokens.push({ type: 'comment', text: line.substring(i) });
      i = line.length;
      continue;
    }
    if (ch === '#' && (i === 0 || /\s/.test(line[i - 1]))) {
      flushBuf();
      tokens.push({ type: 'comment', text: line.substring(i) });
      i = line.length;
      continue;
    }

    // Block comment opening: /* ... (treat rest of line as comment)
    if (ch === '/' && line[i + 1] === '*') {
      flushBuf();
      tokens.push({ type: 'comment', text: line.substring(i) });
      i = line.length;
      continue;
    }

    // Block comment closing or continuation: * or */
    // Lines starting with * or */ inside block comments
    if ((ch === '*' && i === line.trimStart().indexOf('*') + (line.length - line.trimStart().length)) &&
        line.trimStart().startsWith('*')) {
      flushBuf();
      tokens.push({ type: 'comment', text: line.substring(i) });
      i = line.length;
      continue;
    }

    // Template literals with ${} interpolation
    if (ch === '`') {
      flushBuf();
      let j = i + 1;
      let depth = 0;
      while (j < line.length) {
        if (line[j] === '\\') { j += 2; continue; }
        if (line[j] === '$' && line[j + 1] === '{') { depth++; j += 2; continue; }
        if (line[j] === '}' && depth > 0) { depth--; j++; continue; }
        if (line[j] === '`' && depth === 0) { j++; break; }
        j++;
      }
      tokens.push({ type: 'string', text: line.substring(i, j) });
      i = j;
      continue;
    }

    // String literals (single and double quotes)
    if (ch === '"' || ch === "'") {
      flushBuf();
      const quote = ch;
      // Check for triple-quoted strings (Python)
      if (line[i + 1] === quote && line[i + 2] === quote) {
        let j = i + 3;
        const triple = quote + quote + quote;
        const endIdx = line.indexOf(triple, j);
        j = endIdx >= 0 ? endIdx + 3 : line.length;
        tokens.push({ type: 'string', text: line.substring(i, j) });
        i = j;
        continue;
      }
      let j = i + 1;
      while (j < line.length && line[j] !== quote) {
        if (line[j] === '\\') j++;
        j++;
      }
      j = Math.min(j + 1, line.length);
      tokens.push({ type: 'string', text: line.substring(i, j) });
      i = j;
      continue;
    }

    // Numbers (including hex 0x, binary 0b, octal 0o)
    if (/\d/.test(ch) && (i === 0 || !/\w/.test(line[i - 1]))) {
      flushBuf();
      let j = i;
      if (ch === '0' && (line[j + 1] === 'x' || line[j + 1] === 'X')) {
        j += 2;
        while (j < line.length && /[\da-fA-F_]/.test(line[j])) j++;
      } else if (ch === '0' && (line[j + 1] === 'b' || line[j + 1] === 'B')) {
        j += 2;
        while (j < line.length && /[01_]/.test(line[j])) j++;
      } else if (ch === '0' && (line[j + 1] === 'o' || line[j + 1] === 'O')) {
        j += 2;
        while (j < line.length && /[0-7_]/.test(line[j])) j++;
      } else {
        while (j < line.length && /[\d._eE]/.test(line[j])) j++;
      }
      tokens.push({ type: 'number', text: line.substring(i, j) });
      i = j;
      continue;
    }

    // Decorators (@something in Python/Java/TS)
    if (ch === '@' && /[a-zA-Z_]/.test(line[i + 1] || '')) {
      flushBuf();
      let j = i + 1;
      while (j < line.length && /[\w.]/.test(line[j])) j++;
      tokens.push({ type: 'function', text: line.substring(i, j) });
      i = j;
      continue;
    }

    // Identifiers (keywords and function names)
    if (/[a-zA-Z_]/.test(ch)) {
      flushBuf();
      let j = i;
      while (j < line.length && /\w/.test(line[j])) j++;
      const word = line.substring(i, j);
      // Check if followed by ( → function call
      let afterWord = j;
      while (afterWord < line.length && line[afterWord] === ' ') afterWord++;
      if (line[afterWord] === '(') {
        tokens.push({ type: 'function', text: word });
      } else if (KW_SET.has(word)) {
        tokens.push({ type: 'keyword', text: word });
      } else {
        tokens.push({ type: 'plain', text: word });
      }
      i = j;
      continue;
    }

    buf += ch;
    i++;
  }
  flushBuf();

  return renderTokens(tokens);
}

function renderTokens(tokens) {
  const classMap = {
    comment: 'tk-comment', string: 'tk-string', keyword: 'tk-keyword',
    function: 'tk-function', number: 'tk-number'
  };
  return tokens.map(t => {
    const escaped = t.text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
    const cls = classMap[t.type];
    return cls ? `<span class="${cls}">${escaped}</span>` : escaped;
  }).join('');
}

/**
 * Build code viewer HTML from source files referenced in the manifest.
 *
 * @param {Object} manifest - Scene manifest
 * @param {string} baseDir - Base directory for resolving file paths
 * @returns {{ codeHtml: string, fileTabsHtml: string }}
 */
function buildCodeViewer(manifest, baseDir) {
  // Collect unique files referenced in cues
  const files = new Set();
  for (const ch of manifest.chapters) {
    for (const seg of ch.segments) {
      if (seg.cue && seg.cue.file) {
        files.add(seg.cue.file);
      }
    }
  }

  const fileList = [...files];
  const fileTabsHtml = fileList.map((f, i) => {
    const active = i === 0 ? ' file-tab--active' : '';
    const name = path.basename(f);
    return `      <button class="file-tab${active}" data-file="${f}">
        <svg class="file-tab-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
        </svg>
        ${name}
      </button>`;
  }).join('\n');

  // Build code lines for each file
  const codeBlocksHtml = fileList.map((f, fileIdx) => {
    const filePath = path.resolve(baseDir, f);
    if (!fs.existsSync(filePath)) {
      throw new Error(`Assembler: source file not found: ${filePath} (referenced as "${f}")`);
    }
    const content = fs.readFileSync(filePath, 'utf-8');
    const lines = content.split('\n');
    const display = fileIdx === 0 ? '' : ' style="display:none"';

    const linesHtml = lines.map((line, i) => {
      const num = i + 1;
      const tokenized = tokenizeLine(line);
      return `      <div class="code-line" data-line="${num}">
        <span class="line-number">${num}</span>
        <span class="line-content">${tokenized || ''}</span>
      </div>`;
    }).join('\n');

    return `    <div class="code-block" data-file="${f}"${display}>
${linesHtml}
    </div>`;
  }).join('\n');

  return { fileTabsHtml, codeBlocksHtml };
}

/**
 * Build sidebar chapter list HTML from manifest.
 */
function buildChapterList(manifest) {
  return manifest.chapters.map((ch, i) => {
    return `      <li class="chapter-item" data-chapter="${ch.id}">
        <span class="chapter-status">
          <span class="status-dot"></span>
        </span>
        <div class="chapter-info">
          <div class="chapter-title">${ch.title}</div>
        </div>
      </li>`;
  }).join('\n');
}

/**
 * Build the global word timestamps array and cue map from per-segment synthesis results.
 *
 * Offsets each segment's timestamps so they're globally monotonic.
 *
 * @param {Object} manifest - Scene manifest
 * @param {Array} synthResults - [{ id, audioBase64, wordTimestamps, duration }]
 * @returns {{ words: Array, wordToCue: Array, chapters: Array, totalDuration: number }}
 */
function buildSyncData(manifest, synthResults) {
  const words = [];
  const wordToCue = [];
  const chapters = [];
  let timeOffset = 0;

  // Map segment id → synth result
  const resultMap = {};
  for (const r of synthResults) {
    resultMap[r.id] = r;
  }

  for (const ch of manifest.chapters) {
    chapters.push({
      id: ch.id,
      title: ch.title,
      startWord: words.length
    });

    for (const seg of ch.segments) {
      const result = resultMap[seg.id];
      if (!result) {
        throw new Error(`Assembler: no synthesis result for segment "${seg.id}"`);
      }

      const segStartWord = words.length;
      for (const w of result.wordTimestamps) {
        words.push({
          word: w.word,
          start: w.start + timeOffset,
          end: w.end + timeOffset
        });
        wordToCue.push(seg.id);
      }

      timeOffset += result.duration + 0.3; // 300ms gap between segments
    }
  }

  return {
    words,
    wordToCue,
    chapters,
    totalDuration: timeOffset
  };
}

/**
 * Build the cue map from the manifest.
 *
 * @param {Object} manifest
 * @returns {Object} { segId: { type, file, lines, scroll_to } }
 */
function buildCueMap(manifest) {
  const map = {};
  for (const ch of manifest.chapters) {
    for (const seg of ch.segments) {
      map[seg.id] = seg.cue;
    }
  }
  return map;
}

/**
 * Concatenate segment audio base64 strings.
 * In a real implementation, we'd decode, concatenate PCM, and re-encode.
 * For now, we store them as an array for the player to sequence.
 *
 * @param {Object} manifest
 * @param {Array} synthResults
 * @returns {Array<{ id: string, audioBase64: string, duration: number }>}
 */
function buildAudioSegments(manifest, synthResults) {
  const resultMap = {};
  for (const r of synthResults) {
    resultMap[r.id] = r;
  }

  const segments = [];
  for (const ch of manifest.chapters) {
    for (const seg of ch.segments) {
      const result = resultMap[seg.id];
      if (!result) {
        throw new Error(`Assembler: no audio for segment "${seg.id}"`);
      }
      segments.push({
        id: seg.id,
        audioBase64: result.audioBase64,
        duration: result.duration
      });
    }
  }
  return segments;
}

/**
 * Assemble the final self-contained tutorial.html.
 *
 * @param {Object} params
 * @param {string} params.templateDir - Path to template/ directory
 * @param {string} params.browserDir - Path to browser/ directory
 * @param {Object} params.manifest - Scene manifest
 * @param {Array}  params.synthResults - Per-segment synthesis results
 * @param {string} params.sourceDir - Base directory for source files
 * @param {string} params.outputPath - Where to write the final HTML
 */
function assemble({ templateDir, browserDir, manifest, synthResults, sourceDir, outputPath }) {
  process.stderr.write(`Assembling tutorial: "${manifest.title}"\n`);

  // Read template files
  const css = fs.readFileSync(path.join(templateDir, 'player.css'), 'utf-8');
  const syncJs = fs.readFileSync(path.join(browserDir, 'sync.js'), 'utf-8');
  const rendererJs = fs.readFileSync(path.join(browserDir, 'renderer.js'), 'utf-8');
  const chatbotJs = fs.readFileSync(path.join(browserDir, 'chatbot.js'), 'utf-8');

  // Build components
  const { fileTabsHtml, codeBlocksHtml } = buildCodeViewer(manifest, sourceDir);
  const chapterListHtml = buildChapterList(manifest);
  const syncData = buildSyncData(manifest, synthResults);
  const cueMap = buildCueMap(manifest);
  const audioSegments = buildAudioSegments(manifest, synthResults);

  // Count segments
  const segCount = manifest.chapters.reduce((n, ch) => n + ch.segments.length, 0);

  // Build the final HTML
  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${manifest.title} — Tutorial</title>
  <style>${css}</style>
</head>
<body>

<div class="player-layout">

  <aside class="sidebar">
    <div class="sidebar-header">
      <h1>${manifest.title}</h1>
      <div class="tutorial-meta">${manifest.chapters.length} chapters &middot; ${Math.ceil(syncData.totalDuration / 60)} min</div>
    </div>
    <ol class="chapter-list">
${chapterListHtml}
    </ol>
    <div class="sidebar-footer">
      <div class="sidebar-progress">
        <span>0 of ${manifest.chapters.length}</span>
        <div class="sidebar-progress-bar">
          <div class="sidebar-progress-fill" style="width: 0%"></div>
        </div>
      </div>
    </div>
  </aside>

  <header class="topbar">
    <span class="topbar-chapter-badge">Chapter 1</span>
    <span class="topbar-title">${manifest.chapters[0].title}</span>
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
${fileTabsHtml}
    </div>
    <div class="code-container">
${codeBlocksHtml}
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
    <span class="time-display">0:00 / ${Math.floor(syncData.totalDuration / 60)}:${Math.floor(syncData.totalDuration % 60).toString().padStart(2, '0')}</span>
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

<script id="tutorial-data" type="application/json">${JSON.stringify({
    slug: path.basename(path.dirname(outputPath)),
    title: manifest.title,
    words: syncData.words,
    wordToCue: syncData.wordToCue,
    cueMap,
    chapters: syncData.chapters,
    totalDuration: syncData.totalDuration,
    audioSegments: audioSegments.map(s => ({ id: s.id, duration: s.duration, audioBase64: s.audioBase64 }))
  })}</script>

<script type="module">
// --- Inline sync engine (no external imports in self-contained file) ---
${syncJs.replace('export default SyncEngine;', '')}

// --- Inline renderer ---
${rendererJs.replace('export default SceneRenderer;', '')}

// --- Inline chatbot ---
${chatbotJs.replace('export default Chatbot;', '')}

// --- Load tutorial data ---
const data = JSON.parse(document.getElementById('tutorial-data').textContent);

// --- Audio engine: sequence base64 segments ---
const AudioEngine = (() => {
  const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  let masterBuffer = null;  // Single concatenated AudioBuffer
  let source = null;
  let startOffset = 0;      // Where in the buffer we are (seconds)
  let playing = false;
  let ctxTimeAtPlay = 0;    // audioCtx.currentTime when play started
  let rate = 1;
  let totalDuration = 0;

  // Decode all segments and concatenate into a single AudioBuffer
  async function init(audioSegments) {
    const buffers = [];
    const sampleRate = audioCtx.sampleRate;
    const GAP_SECONDS = 0.3;
    const gapSamples = Math.floor(GAP_SECONDS * sampleRate);

    // Decode each segment
    for (const seg of audioSegments) {
      const binary = atob(seg.audioBase64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      const buf = await audioCtx.decodeAudioData(bytes.buffer.slice(0));
      buffers.push(buf);
    }

    // Calculate total length with gaps
    let totalSamples = 0;
    for (let i = 0; i < buffers.length; i++) {
      totalSamples += buffers[i].length;
      if (i < buffers.length - 1) totalSamples += gapSamples;
    }

    // Create single buffer (use first buffer's channel count, or 1)
    const channels = buffers.length > 0 ? buffers[0].numberOfChannels : 1;
    masterBuffer = audioCtx.createBuffer(channels, totalSamples, sampleRate);

    // Copy segment data into master buffer
    let writePos = 0;
    for (let i = 0; i < buffers.length; i++) {
      const buf = buffers[i];
      for (let ch = 0; ch < channels; ch++) {
        const src = buf.numberOfChannels > ch ? buf.getChannelData(ch) : buf.getChannelData(0);
        masterBuffer.getChannelData(ch).set(src, writePos);
      }
      writePos += buf.length;
      if (i < buffers.length - 1) {
        // Gap is already zeros (buffer initialized to 0)
        writePos += gapSamples;
      }
    }

    totalDuration = masterBuffer.duration;
  }

  function getTime() {
    if (!playing || !masterBuffer) return startOffset;
    return Math.min(startOffset + (audioCtx.currentTime - ctxTimeAtPlay) * rate, totalDuration);
  }

  function playFrom(time) {
    stop();
    if (!masterBuffer) return;
    startOffset = Math.max(0, Math.min(time, totalDuration));
    source = audioCtx.createBufferSource();
    source.buffer = masterBuffer;
    source.playbackRate.value = rate;
    source.connect(audioCtx.destination);
    source.onended = () => {
      if (playing) {
        playing = false;
        startOffset = totalDuration;
        document.dispatchEvent(new CustomEvent('playback:ended', {}));
        document.dispatchEvent(new CustomEvent('playback:state', { detail: { playing: false } }));
      }
    };
    source.start(0, startOffset);
    ctxTimeAtPlay = audioCtx.currentTime;
    playing = true;
  }

  function stop() {
    if (source) {
      try { source.onended = null; source.stop(); } catch(e) {}
      source = null;
    }
  }

  return {
    init,
    play() { playFrom(startOffset); },
    pause() { startOffset = getTime(); stop(); playing = false; },
    seek(t) { startOffset = t; if (playing) playFrom(t); },
    setRate(r) {
      rate = r;
      if (source) source.playbackRate.value = r;
    },
    getTime,
    get isPlaying() { return playing; },
    get duration() { return totalDuration; }
  };
})();

// --- Initialize ---
SceneRenderer.init();
SceneRenderer.buildTranscript(data.words);

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

const SPEEDS = [0.5, 1, 1.5, 2];
let speedIdx = 1;
document.querySelector('.btn-speed').addEventListener('click', () => {
  speedIdx = (speedIdx + 1) % SPEEDS.length;
  SyncEngine.setRate(SPEEDS[speedIdx]);
  AudioEngine.setRate(SPEEDS[speedIdx]);
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

document.querySelector('.btn-chatbot-toggle').addEventListener('click', () => {
  if (document.body.classList.contains('chatbot-open')) {
    document.body.classList.remove('chatbot-open');
    SyncEngine.resume();
  } else {
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

// Initialize chatbot
Chatbot.init({
  syncEngine: SyncEngine,
  chapters: data.chapters,
  words: data.words,
  cueMap: data.cueMap
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
</html>`;

  // Write output
  const outputDir = path.dirname(outputPath);
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
  fs.writeFileSync(outputPath, html, 'utf-8');
  process.stderr.write(`Tutorial assembled: ${outputPath}\n`);
  process.stderr.write(`  ${manifest.chapters.length} chapters, ${segCount} segments\n`);
  process.stderr.write(`  ${(Buffer.byteLength(html) / 1024 / 1024).toFixed(1)} MB total\n`);

  return outputPath;
}

module.exports = {
  assemble,
  buildSyncData,
  buildCueMap,
  buildCodeViewer,
  tokenizeLine
};

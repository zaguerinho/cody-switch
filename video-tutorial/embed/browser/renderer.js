/**
 * SceneRenderer — reacts to sync engine events and updates the DOM.
 *
 * Listens for:
 *   'scene:change'    → scroll code viewer, highlight lines
 *   'chapter:change'  → update sidebar active chapter, topbar title
 *   'progress:tick'   → update progress bars, time display
 *   'word:change'     → highlight active word in transcript
 *   'playback:state'  → toggle play/pause button icon
 *   'playback:rate'   → update speed button text
 *
 * No dependencies. Plain DOM manipulation.
 */

const SceneRenderer = (() => {
  // --- DOM element cache ---
  let els = {};

  // --- State ---
  let currentHighlightLines = [];
  let activeFile = null;

  // --- Helpers ---
  function formatTime(seconds) {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  function clearHighlights() {
    currentHighlightLines.forEach(el => {
      el.classList.remove('code-line--highlighted');
    });
    currentHighlightLines = [];
  }

  function switchFile(file) {
    if (!file || file === activeFile) return;
    activeFile = file;

    // Switch file tabs
    const tabs = document.querySelectorAll('.file-tab');
    tabs.forEach(t => t.classList.toggle('file-tab--active', t.dataset.file === file));

    // Switch code blocks
    const blocks = els.codeContainer.querySelectorAll('.code-block');
    blocks.forEach(b => {
      b.style.display = b.dataset.file === file ? '' : 'none';
    });
  }

  function getActiveCodeLines() {
    // Scope to active file's code-block if multi-file, otherwise all code-lines
    const block = activeFile
      ? els.codeContainer.querySelector(`.code-block[data-file="${activeFile}"]`)
      : null;
    const container = block || els.codeContainer;
    return container.querySelectorAll('.code-line');
  }

  function highlightLines(startLine, endLine) {
    clearHighlights();
    const codeLines = getActiveCodeLines();
    for (let i = startLine - 1; i < endLine && i < codeLines.length; i++) {
      if (i >= 0) {
        codeLines[i].classList.add('code-line--highlighted');
        currentHighlightLines.push(codeLines[i]);
      }
    }
  }

  function scrollToLine(lineNum) {
    const codeLines = getActiveCodeLines();
    const targetIdx = lineNum - 1;
    if (targetIdx >= 0 && targetIdx < codeLines.length) {
      const target = codeLines[targetIdx];
      const container = els.codeContainer;
      const targetTop = target.offsetTop - container.offsetTop;
      const scrollTarget = targetTop - container.clientHeight / 4;
      container.scrollTo({
        top: Math.max(0, scrollTarget),
        behavior: 'smooth'
      });
    }
  }

  // --- Event handlers ---

  function onSceneChange(e) {
    const { file, lines, scroll_to } = e.detail;
    if (file) switchFile(file);
    if (lines && lines.length === 2) {
      highlightLines(lines[0], lines[1]);
    }
    if (scroll_to) {
      scrollToLine(scroll_to);
    }
  }

  function onChapterChange(e) {
    const { chapterId, title, index } = e.detail;

    // Update topbar
    if (els.topbarTitle) {
      els.topbarTitle.textContent = title;
    }
    if (els.topbarBadge) {
      els.topbarBadge.textContent = `Chapter ${index + 1}`;
    }

    // Update sidebar active state
    const items = els.chapterList.querySelectorAll('.chapter-item');
    items.forEach((item, i) => {
      item.classList.toggle('chapter-item--active', i === index);
    });
  }

  function onProgressTick(e) {
    const { pct, elapsed, total } = e.detail;

    // Update time display
    if (els.timeDisplay) {
      els.timeDisplay.textContent = `${formatTime(elapsed)} / ${formatTime(total)}`;
    }

    // Update scrub bar
    if (els.scrubFill) {
      els.scrubFill.style.width = `${pct}%`;
    }

    // Update topbar progress
    if (els.topbarProgress) {
      els.topbarProgress.style.width = `${pct}%`;
    }
  }

  function onWordChange(e) {
    const { wordIdx, word, segmentId } = e.detail;
    if (!els._transcriptText) return;

    // Switch visible segment if needed
    if (segmentId && segmentId !== els._activeTranscriptSeg) {
      const prev = els._transcriptText.querySelector(`[data-segment="${els._activeTranscriptSeg}"]`);
      if (prev) prev.style.display = 'none';
      const next = els._transcriptText.querySelector(`[data-segment="${segmentId}"]`);
      if (next) next.style.display = '';
      els._activeTranscriptSeg = segmentId;
    }

    // Highlight active word within the visible segment
    const activeSeg = els._transcriptText.querySelector(`[data-segment="${els._activeTranscriptSeg}"]`);
    if (activeSeg) {
      const spans = activeSeg.querySelectorAll('[data-word-idx]');
      spans.forEach(el => {
        el.classList.toggle('word--active',
          parseInt(el.dataset.wordIdx) === wordIdx);
      });
    }
  }

  function onPlaybackState(e) {
    const { playing } = e.detail;
    if (els.playBtn) {
      els.playBtn.innerHTML = playing
        ? '<svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg>'
        : '<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5,3 19,12 5,21"/></svg>';
      els.playBtn.setAttribute('aria-label', playing ? 'Pause' : 'Play');
    }
  }

  function onPlaybackRate(e) {
    const { rate } = e.detail;
    if (els.speedBtn) {
      els.speedBtn.textContent = `${rate}x`;
    }
  }

  // --- Public API ---
  return {
    /**
     * Initialize the renderer by caching DOM references and binding events.
     * Call after DOM is ready.
     */
    init() {
      els = {
        codeContainer: document.querySelector('.code-container'),
        chapterList: document.querySelector('.chapter-list'),
        topbarTitle: document.querySelector('.topbar-title'),
        topbarBadge: document.querySelector('.topbar-chapter-badge'),
        topbarProgress: document.querySelector('.topbar-progress-fill'),
        timeDisplay: document.querySelector('.time-display'),
        scrubFill: document.querySelector('.scrub-fill'),
        playBtn: document.querySelector('.btn-play'),
        speedBtn: document.querySelector('.btn-speed'),
        transcript: document.querySelector('.transcript-bar')
      };

      // Bind sync engine events
      document.addEventListener('scene:change', onSceneChange);
      document.addEventListener('chapter:change', onChapterChange);
      document.addEventListener('progress:tick', onProgressTick);
      document.addEventListener('word:change', onWordChange);
      document.addEventListener('playback:state', onPlaybackState);
      document.addEventListener('playback:rate', onPlaybackRate);
    },

    /**
     * Build transcript HTML from word timestamps grouped by segment.
     * Only the active segment's words are visible at a time.
     * @param {Array} words - [{ word, start, end }]
     * @param {Array} wordToCue - segment ID for each word index
     */
    buildTranscript(words, wordToCue) {
      if (!els.transcript) return;
      const textEl = els.transcript.querySelector('.transcript-text');
      if (!textEl) return;

      // Group words by segment ID
      const segments = {};
      const segOrder = [];
      words.forEach((w, i) => {
        const segId = wordToCue ? wordToCue[i] : 'all';
        if (!segments[segId]) {
          segments[segId] = [];
          segOrder.push(segId);
        }
        segments[segId].push({ word: w.word, idx: i });
      });

      // Build one <p> per segment, hidden by default
      const html = segOrder.map(segId => {
        const spans = segments[segId].map(w =>
          `<span data-word-idx="${w.idx}">${w.word}</span>`
        ).join(' ');
        return `<p class="transcript-segment" data-segment="${segId}" style="display:none">${spans}</p>`;
      }).join('');

      textEl.innerHTML = html;
      // Show the first segment
      if (segOrder.length > 0) {
        const first = textEl.querySelector(`[data-segment="${segOrder[0]}"]`);
        if (first) first.style.display = '';
      }

      els._transcriptText = textEl;
      els._activeTranscriptSeg = segOrder[0] || null;
    },

    /**
     * Mark chapters as complete up to (but not including) the given index.
     * @param {number} currentIndex - index of current chapter
     */
    markChaptersComplete(currentIndex) {
      const items = els.chapterList.querySelectorAll('.chapter-item');
      items.forEach((item, i) => {
        if (i < currentIndex) {
          item.classList.add('chapter-item--complete');
          // Replace status dot with checkmark
          const status = item.querySelector('.chapter-status');
          if (status && !status.querySelector('svg')) {
            status.innerHTML = `
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="8" cy="8" r="7"/>
                <path d="M5 8l2 2 4-4"/>
              </svg>`;
          }
        }
      });
    },

    // Expose for external use
    clearHighlights,
    highlightLines,
    scrollToLine
  };
})();

export default SceneRenderer;

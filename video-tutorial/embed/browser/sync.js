/**
 * SyncEngine — drives the tutorial from the audio clock.
 *
 * Single source of truth: the current playback time.
 * Binary-searches a flat word-timestamp array to find the active word,
 * maps it to a cue via cueMap, and fires DOM events when state changes.
 *
 * Events dispatched on `document`:
 *   'scene:change'   → detail: { cueId, type, file, lines, scroll_to }
 *   'chapter:change'  → detail: { chapterId, title, index }
 *   'progress:tick'   → detail: { pct: 0-100, elapsed, total }
 *   'word:change'      → detail: { wordIdx, word }
 *
 * Usage:
 *   SyncEngine.init({ wordTimestamps, cueMap, chapters, totalDuration });
 *   SyncEngine.seek(timeInSeconds);
 *   SyncEngine.play() / .pause() / .toggle();
 */

const SyncEngine = (() => {
  // --- State ---
  let words = [];        // [{ word, start, end }]
  let cueMap = {};       // { cueId: { type, file, lines, scroll_to } }
  let wordToCue = [];    // wordIndex → cueId
  let chapters = [];     // [{ id, title, startWord }]
  let totalDuration = 0;

  let currentTime = 0;
  let playing = false;
  let playbackRate = 1;
  let lastFrameTime = 0;
  let rafId = null;

  // Track last-fired state to avoid duplicate events
  let activeCueId = null;
  let activeChapterId = null;
  let activeWordIdx = -1;

  // Saved state for pause/resume (chatbot)
  let savedState = null;

  // --- Binary search: find rightmost word where start <= time ---
  function bsearch(time) {
    let lo = 0;
    let hi = words.length - 1;
    let result = -1;
    while (lo <= hi) {
      const mid = (lo + hi) >>> 1;
      if (words[mid].start <= time) {
        result = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }
    return result;
  }

  // --- Resolve which cue a word index belongs to ---
  function resolveCue(wordIdx) {
    if (wordIdx < 0 || !wordToCue[wordIdx]) return null;
    return wordToCue[wordIdx];
  }

  // --- Resolve which chapter a word index belongs to ---
  function resolveChapter(wordIdx) {
    if (wordIdx < 0) return null;
    for (let i = chapters.length - 1; i >= 0; i--) {
      if (wordIdx >= chapters[i].startWord) {
        return { ...chapters[i], index: i };
      }
    }
    return null;
  }

  // --- Fire a CustomEvent on document ---
  function emit(name, detail) {
    document.dispatchEvent(new CustomEvent(name, { detail }));
  }

  // --- Core tick: evaluate current time and fire events ---
  function tick() {
    const wordIdx = bsearch(currentTime);

    // Word change
    if (wordIdx !== activeWordIdx) {
      activeWordIdx = wordIdx;
      if (wordIdx >= 0) {
        const segmentId = wordToCue[wordIdx] || null;
        emit('word:change', { wordIdx, word: words[wordIdx].word, segmentId });
      }
    }

    // Cue change
    const cueId = resolveCue(wordIdx);
    if (cueId !== activeCueId) {
      activeCueId = cueId;
      if (cueId && cueMap[cueId]) {
        emit('scene:change', { cueId, ...cueMap[cueId] });
      }
    }

    // Chapter change
    const chapter = resolveChapter(wordIdx);
    const chapId = chapter ? chapter.id : null;
    if (chapId !== activeChapterId) {
      activeChapterId = chapId;
      if (chapter) {
        emit('chapter:change', {
          chapterId: chapter.id,
          title: chapter.title,
          index: chapter.index
        });
      }
    }

    // Progress
    const pct = totalDuration > 0
      ? Math.min(100, (currentTime / totalDuration) * 100)
      : 0;
    emit('progress:tick', {
      pct,
      elapsed: currentTime,
      total: totalDuration
    });
  }

  // --- Animation frame loop (simulated playback without real audio) ---
  function frame(timestamp) {
    if (!playing) return;
    if (lastFrameTime > 0) {
      const delta = (timestamp - lastFrameTime) / 1000;
      currentTime += delta * playbackRate;
      if (currentTime >= totalDuration) {
        currentTime = totalDuration;
        playing = false;
        emit('playback:ended', {});
      }
    }
    lastFrameTime = timestamp;
    tick();
    if (playing) {
      rafId = requestAnimationFrame(frame);
    }
  }

  // --- Public API ---
  return {
    /**
     * Initialize the sync engine with baked tutorial data.
     * @param {Object} config
     * @param {Array}  config.wordTimestamps - [{ word, start, end }]
     * @param {Object} config.cueMap - { cueId: { type, file, lines, scroll_to } }
     * @param {Array}  config.wordToCue - wordIndex → cueId mapping array
     * @param {Array}  config.chapters - [{ id, title, startWord }]
     * @param {number} config.totalDuration - total duration in seconds
     */
    init({ wordTimestamps, cueMap: cm, wordToCue: wtc, chapters: ch, totalDuration: td }) {
      words = wordTimestamps || [];
      cueMap = cm || {};
      wordToCue = wtc || [];
      chapters = ch || [];
      totalDuration = td || 0;

      // Reset state
      currentTime = 0;
      playing = false;
      activeCueId = null;
      activeChapterId = null;
      activeWordIdx = -1;

      // Fire initial tick
      tick();
    },

    play() {
      if (playing) return;
      if (currentTime >= totalDuration) currentTime = 0;
      playing = true;
      lastFrameTime = 0;
      rafId = requestAnimationFrame(frame);
      emit('playback:state', { playing: true });
    },

    pause() {
      playing = false;
      if (rafId) cancelAnimationFrame(rafId);
      rafId = null;
      emit('playback:state', { playing: false });
    },

    toggle() {
      if (playing) this.pause();
      else this.play();
    },

    /**
     * Seek to a specific time in seconds.
     */
    seek(time) {
      currentTime = Math.max(0, Math.min(time, totalDuration));
      tick();
    },

    /**
     * Seek to a percentage (0-100).
     */
    seekPct(pct) {
      this.seek((pct / 100) * totalDuration);
    },

    /**
     * Seek to the start of a chapter by ID.
     */
    seekChapter(chapterId) {
      const ch = chapters.find(c => c.id === chapterId);
      if (!ch) return;
      const wordIdx = ch.startWord;
      if (wordIdx >= 0 && wordIdx < words.length) {
        this.seek(words[wordIdx].start);
      }
    },

    /**
     * Set playback speed multiplier.
     */
    setRate(rate) {
      playbackRate = rate;
      emit('playback:rate', { rate });
    },

    /**
     * Save state and pause (for chatbot drawer).
     * Returns the saved state object.
     */
    saveAndPause() {
      savedState = {
        time: currentTime,
        wasPlaying: playing,
        chapterId: activeChapterId,
        wordIdx: activeWordIdx
      };
      this.pause();
      return savedState;
    },

    /**
     * Resume from saved state (after chatbot closes).
     */
    resume() {
      if (!savedState) return;
      this.seek(savedState.time);
      if (savedState.wasPlaying) this.play();
      savedState = null;
    },

    // --- Getters ---
    get currentTime() { return currentTime; },
    get isPlaying() { return playing; },
    get duration() { return totalDuration; },
    get rate() { return playbackRate; },
    get activeChapter() { return activeChapterId; },
    get activeCue() { return activeCueId; }
  };
})();

export default SyncEngine;

/**
 * Chatbot — browser-side module.
 *
 * Manages the chat drawer UI: pauses tutorial on open, sends questions
 * to Claude via a local server (preferred) or the Anthropic API (fallback),
 * streams responses, and resumes playback on close.
 *
 * Local server mode: requires `video-tutorial serve` running on localhost.
 * No API key needed — uses existing Claude Code authentication.
 *
 * API mode (fallback): requires an Anthropic API key in localStorage.
 *
 * Queue-until-pause: questions asked during playback are held until
 * the next segment boundary, then dispatched automatically.
 *
 * Usage:
 *   Chatbot.init({ syncEngine, chapters, words, cueMap, getApiKey });
 *   // Drawer open/close is handled internally via DOM events.
 */

const Chatbot = (() => {
  let syncEngine = null;
  let chapters = [];
  let words = [];
  let cueMap = {};
  let getApiKey = () => null;
  let messages = []; // Chat history for this session

  // Serve mode detection
  let serveMode = false;
  const SERVE_URL = 'http://localhost:19191';
  let pollTimer = null;

  // Queue-until-pause state
  let pendingQuestion = null;
  let isPlaybackActive = false;

  // DOM refs
  let els = {};

  // --- Server detection ---

  async function detectServeMode() {
    try {
      const resp = await fetch(`${SERVE_URL}/health`, {
        signal: AbortSignal.timeout(1000)
      });
      serveMode = resp.ok;
    } catch {
      serveMode = false;
    }
    return serveMode;
  }

  // --- Context helpers ---

  function getCurrentContext() {
    const chapterId = syncEngine.activeChapter;
    const chapter = chapters.find(c => c.id === chapterId);
    const cueId = syncEngine.activeCue;
    const cue = cueId ? cueMap[cueId] : null;

    // Get the narration text around current position
    let narrationText = '';
    if (chapter) {
      const startIdx = chapter.startWord;
      const nextChapter = chapters.find(c => c.startWord > startIdx);
      const endIdx = nextChapter ? nextChapter.startWord : words.length;
      narrationText = words.slice(startIdx, endIdx).map(w => w.word).join(' ');
    }

    // Get the code snippet for the current cue
    let codeSnippet = '';
    if (cue && cue.file) {
      const codeBlock = document.querySelector(`.code-block[data-file="${cue.file}"]`);
      if (codeBlock && cue.lines) {
        const lines = codeBlock.querySelectorAll('.code-line');
        const [start, end] = cue.lines;
        const snippetLines = [];
        for (let i = start - 1; i < end && i < lines.length; i++) {
          if (i >= 0) {
            const content = lines[i].querySelector('.line-content');
            snippetLines.push(`${i + 1}: ${content ? content.textContent : ''}`);
          }
        }
        codeSnippet = snippetLines.join('\n');
      }
    }

    return {
      chapterTitle: chapter ? chapter.title : 'Unknown',
      narration: narrationText,
      file: cue ? cue.file : null,
      lines: cue ? cue.lines : null,
      codeSnippet
    };
  }

  // --- Streaming: dispatcher ---

  async function streamResponse(question, context) {
    if (serveMode) {
      return streamViaLocalServer(question, context);
    }

    const apiKey = getApiKey();
    if (!apiKey) {
      showSetupPrompt();
      return;
    }
    return streamViaAnthropicApi(question, context, apiKey);
  }

  // --- Streaming: local server (preferred) ---

  async function streamViaLocalServer(question, context) {
    addMessage('user', question);
    const msgEl = addMessage('assistant', '');
    msgEl.innerHTML = '<span class="chatbot-typing"><span></span><span></span><span></span></span>';
    let fullText = '';

    try {
      const response = await fetch(`${SERVE_URL}/ask`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          question,
          chapterTitle: context.chapterTitle,
          narration: context.narration,
          file: context.file,
          lines: context.lines,
          codeSnippet: context.codeSnippet,
          tutorialTitle: document.title.replace(' — Tutorial', ''),
          history: messages
            .slice(0, -2) // exclude the just-added user+assistant placeholders
            .filter(m => m.text)
            .map(m => ({ role: m.role, text: m.text }))
        })
      });

      if (!response.ok) {
        const errText = await response.text();
        throw new Error(`Server error ${response.status}: ${errText.substring(0, 200)}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          try {
            const event = JSON.parse(line.slice(6));
            if (event.text) {
              fullText += event.text;
              msgEl.textContent = fullText;
              scrollToBottom();
            }
            if (event.done) break;
            if (event.error) {
              msgEl.textContent = `Error: ${event.error}`;
              return;
            }
          } catch (e) {
            // Skip unparseable lines
          }
        }
      }

      // Update message record
      const lastMsg = messages[messages.length - 1];
      if (lastMsg && lastMsg.role === 'assistant') {
        lastMsg.text = fullText;
      }
    } catch (err) {
      msgEl.textContent = `Error: ${err.message}`;
    }
  }

  // --- Streaming: Anthropic API (fallback) ---

  function buildSystemPrompt(context) {
    return `You are a helpful coding tutor embedded in a narrated code tutorial. The user has paused the tutorial to ask a question.

Current chapter: "${context.chapterTitle}"
${context.file ? `Current file: ${context.file} (lines ${context.lines ? context.lines.join('-') : 'unknown'})` : ''}

Narration at this point:
${context.narration}

${context.codeSnippet ? `Code being discussed:\n\`\`\`\n${context.codeSnippet}\n\`\`\`` : ''}

Keep answers concise and focused on the code being shown. Use code examples when helpful. If the question is unrelated to the tutorial, gently redirect.`;
  }

  async function streamViaAnthropicApi(question, context, apiKey) {
    addMessage('user', question);
    const msgEl = addMessage('assistant', '');
    msgEl.innerHTML = '<span class="chatbot-typing"><span></span><span></span><span></span></span>';
    let fullText = '';

    try {
      const response = await fetch('https://api.anthropic.com/v1/messages', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
          'anthropic-dangerous-direct-browser-access': 'true'
        },
        body: JSON.stringify({
          model: 'claude-sonnet-4-5-20250514',
          max_tokens: 1024,
          stream: true,
          system: buildSystemPrompt(context),
          messages: messages.map(m => ({
            role: m.role,
            content: m.text
          })).filter(m => m.content)
        })
      });

      if (!response.ok) {
        const errorText = await response.text();
        if (response.status === 401) {
          msgEl.textContent = 'Invalid API key. Click the key icon below to update it.';
          return;
        }
        throw new Error(`API error ${response.status}: ${errorText.substring(0, 200)}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const data = line.slice(6);
          if (data === '[DONE]') continue;

          try {
            const event = JSON.parse(data);
            if (event.type === 'content_block_delta' && event.delta?.text) {
              fullText += event.delta.text;
              msgEl.textContent = fullText;
              scrollToBottom();
            }
          } catch (e) {
            // Skip unparseable lines
          }
        }
      }

      // Update message record
      const lastMsg = messages[messages.length - 1];
      if (lastMsg && lastMsg.role === 'assistant') {
        lastMsg.text = fullText;
      }
    } catch (err) {
      msgEl.textContent = `Error: ${err.message}`;
    }
  }

  // --- Queue-until-pause ---

  function flushPendingQuestion() {
    if (!pendingQuestion) return;
    const { question, context } = pendingQuestion;
    pendingQuestion = null;
    hideQueueIndicator();

    // Pause playback via bridge event (AudioEngine is in assembler scope)
    document.dispatchEvent(new CustomEvent('chatbot:pause-requested'));

    // Open drawer
    document.body.classList.add('chatbot-open');

    streamResponse(question, context);
  }

  function showQueueIndicator(text) {
    hideQueueIndicator();
    const indicator = document.createElement('div');
    indicator.className = 'chatbot-queue-indicator';
    indicator.innerHTML = `
      <span>Question queued — sending at next pause</span>
      <button class="chatbot-queue-cancel" aria-label="Cancel">&times;</button>`;
    indicator.querySelector('.chatbot-queue-cancel').addEventListener('click', () => {
      pendingQuestion = null;
      hideQueueIndicator();
    });
    els.drawer.querySelector('.chatbot-input').insertAdjacentElement('beforebegin', indicator);
  }

  function hideQueueIndicator() {
    const existing = els.drawer.querySelector('.chatbot-queue-indicator');
    if (existing) existing.remove();
  }

  // --- DOM manipulation ---

  function addMessage(role, text) {
    // Remove empty state
    const empty = els.messagesContainer.querySelector('.chatbot-empty');
    if (empty) empty.remove();

    messages.push({ role, text });

    const msgEl = document.createElement('div');
    msgEl.className = `chatbot-message chatbot-message--${role}`;
    msgEl.textContent = text;
    els.messagesContainer.appendChild(msgEl);
    scrollToBottom();
    return msgEl;
  }

  function scrollToBottom() {
    els.messagesContainer.scrollTop = els.messagesContainer.scrollHeight;
  }

  function showToast(message) {
    const toast = document.createElement('div');
    toast.className = 'chatbot-toast';
    toast.textContent = message;
    document.body.appendChild(toast);
    // Trigger reflow then animate in
    toast.offsetHeight;
    toast.classList.add('chatbot-toast--visible');
    setTimeout(() => {
      toast.classList.remove('chatbot-toast--visible');
      setTimeout(() => toast.remove(), 300);
    }, 2500);
  }

  function showSetupPrompt() {
    const empty = els.messagesContainer.querySelector('.chatbot-empty');
    if (empty) empty.remove();

    // Remove any existing prompt
    const existing = els.messagesContainer.querySelector('.chatbot-setup');
    if (existing) existing.remove();

    const cmd = 'video-tutorial serve';

    const setup = document.createElement('div');
    setup.className = 'chatbot-setup';
    setup.innerHTML = `
      <div class="chatbot-setup-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
      </div>
      <p class="chatbot-setup-title">Interactive Q&A</p>
      <p class="chatbot-setup-desc">Ask questions about the code while watching.<br>Run this command in your terminal to start:</p>
      <div class="chatbot-cmd-row">
        <code class="chatbot-cmd-text">${cmd}</code>
        <button class="chatbot-cmd-copy" aria-label="Copy command">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
          <span>Copy</span>
        </button>
      </div>
      <div class="chatbot-launch-status" style="display:none;"></div>
      <button class="chatbot-launch-btn">
        I started the server
      </button>
      <details class="chatbot-setup-alt">
        <summary>Or use an API key instead</summary>
        <div style="display:flex;gap:8px;margin-top:10px;">
          <input type="password" placeholder="sk-ant-..."
                 class="chatbot-api-key-input">
          <button class="chatbot-api-key-save">Save</button>
        </div>
        <p class="chatbot-setup-hint">Stored in localStorage. Sent only to api.anthropic.com.</p>
      </details>`;
    els.messagesContainer.appendChild(setup);

    // --- Copy button ---
    const copyBtn = setup.querySelector('.chatbot-cmd-copy');
    const copyLabel = copyBtn.querySelector('span');
    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(cmd);
      } catch {
        // file:// fallback: select the text so user can Cmd+C
        const range = document.createRange();
        range.selectNodeContents(setup.querySelector('.chatbot-cmd-text'));
        const sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
      }
      copyLabel.textContent = 'Copied!';
      showToast('Copied! Paste in your terminal.');
      setTimeout(() => { copyLabel.textContent = 'Copy'; }, 2000);
    });

    // --- "I started the server" button: start polling ---
    const launchBtn = setup.querySelector('.chatbot-launch-btn');
    const statusEl = setup.querySelector('.chatbot-launch-status');

    launchBtn.addEventListener('click', () => {
      statusEl.style.display = 'block';
      statusEl.className = 'chatbot-launch-status chatbot-launch-waiting';
      statusEl.innerHTML = `
        <span class="chatbot-typing"><span></span><span></span><span></span></span>
        Looking for server...`;
      launchBtn.disabled = true;
      launchBtn.textContent = 'Connecting...';
      startPolling(setup);
    });

    // --- API key fallback ---
    const keyInput = setup.querySelector('.chatbot-api-key-input');
    const keySaveBtn = setup.querySelector('.chatbot-api-key-save');

    keySaveBtn.addEventListener('click', () => {
      const key = keyInput.value.trim();
      if (key) {
        localStorage.setItem('tutorial-anthropic-key', key);
        stopPolling();
        setup.remove();
        addMessage('assistant', 'API key saved. You can now ask questions about the code!');
      }
    });

    keyInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') keySaveBtn.click();
    });
  }

  function startPolling(setupEl) {
    stopPolling();
    let attempts = 0;
    const maxAttempts = 60; // ~2 minutes at 2s interval

    pollTimer = setInterval(async () => {
      attempts++;
      const found = await detectServeMode();

      if (found) {
        stopPolling();
        setupEl.remove();
        addMessage('assistant', 'Server connected! Ask anything about the code you\'re seeing.');
        els.input.focus();
        return;
      }

      if (attempts >= maxAttempts) {
        stopPolling();
        const statusEl = setupEl.querySelector('.chatbot-launch-status');
        if (statusEl) {
          statusEl.className = 'chatbot-launch-status chatbot-launch-error';
          statusEl.textContent = 'Server not detected. Make sure video-tutorial is installed and on your PATH.';
        }
        const btn = setupEl.querySelector('.chatbot-launch-btn');
        if (btn) {
          btn.disabled = false;
          btn.innerHTML = `
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M1 4v6h6"/><path d="M23 20v-6h-6"/>
              <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15"/>
            </svg>
            Retry`;
        }
      }
    }, 2000);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  // --- Public API ---
  return {
    /**
     * Initialize the chatbot.
     * @param {Object} config
     * @param {Object} config.syncEngine - SyncEngine instance
     * @param {Array}  config.chapters - Chapter definitions
     * @param {Array}  config.words - Word timestamp array
     * @param {Object} config.cueMap - Cue definitions
     * @param {Function} [config.getApiKey] - Returns API key string or null
     */
    async init(config) {
      syncEngine = config.syncEngine;
      chapters = config.chapters || [];
      words = config.words || [];
      cueMap = config.cueMap || {};
      getApiKey = config.getApiKey || (() => {
        return localStorage.getItem('tutorial-anthropic-key') || null;
      });

      els = {
        drawer: document.querySelector('.chatbot-drawer'),
        messagesContainer: document.querySelector('.chatbot-messages'),
        input: document.querySelector('.chatbot-input input'),
        sendBtn: document.querySelector('.chatbot-input button')
      };

      // Detect local server availability
      await detectServeMode();

      // Update empty state based on mode
      if (serveMode) {
        const emptyP = els.messagesContainer.querySelector('.chatbot-empty p');
        if (emptyP) {
          emptyP.textContent = 'Ask anything about the code you\'re seeing. Powered by Claude Code.';
        }
      } else if (!getApiKey()) {
        // No server and no API key — show launch prompt
        showSetupPrompt();
      }

      // Send on button click
      els.sendBtn.addEventListener('click', () => {
        this.send();
      });

      // Send on Enter
      els.input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          this.send();
        }
      });

      // Queue-until-pause: track playback state
      document.addEventListener('playback:state', (e) => {
        isPlaybackActive = e.detail.playing;
      });

      // Queue-until-pause: flush on segment boundary
      document.addEventListener('scene:change', () => {
        if (pendingQuestion && isPlaybackActive) {
          flushPendingQuestion();
        }
      });

      // Inject chatbot message styles
      if (!document.querySelector('#chatbot-styles')) {
        const style = document.createElement('style');
        style.id = 'chatbot-styles';
        style.textContent = `
          .chatbot-message {
            padding: 10px 14px;
            border-radius: 12px;
            font-size: 13px;
            line-height: 1.5;
            max-width: 90%;
            white-space: pre-wrap;
            word-wrap: break-word;
          }
          .chatbot-message--user {
            background: #6c63ff;
            color: white;
            align-self: flex-end;
            border-bottom-right-radius: 4px;
          }
          .chatbot-message--assistant {
            background: #f0f4f8;
            color: #2d3436;
            align-self: flex-start;
            border-bottom-left-radius: 4px;
          }
          .chatbot-api-prompt {
            padding: 20px;
            text-align: center;
            color: #4a5568;
            font-size: 13px;
          }
          .chatbot-queue-indicator {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 8px 14px;
            background: #2d1f5e;
            color: #c4b5fd;
            font-size: 12px;
            border-top: 1px solid #3d2d7e;
          }
          .chatbot-queue-cancel {
            background: none;
            border: none;
            color: #c4b5fd;
            font-size: 18px;
            cursor: pointer;
            padding: 0 4px;
            line-height: 1;
          }
          .chatbot-queue-cancel:hover {
            color: white;
          }
          .chatbot-toast {
            position: fixed;
            bottom: 90px;
            left: 50%;
            transform: translateX(-50%) translateY(20px);
            background: #1a1a2e;
            color: #e2e8f0;
            padding: 10px 20px;
            border-radius: 8px;
            font-size: 13px;
            z-index: 300;
            box-shadow: 0 4px 20px rgba(0,0,0,0.3);
            opacity: 0;
            transition: opacity 0.25s, transform 0.25s;
            pointer-events: none;
          }
          .chatbot-toast--visible {
            opacity: 1;
            transform: translateX(-50%) translateY(0);
          }
          .chatbot-setup {
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 28px 20px;
            text-align: center;
          }
          .chatbot-setup-icon {
            color: #6c63ff;
            margin-bottom: 12px;
          }
          .chatbot-setup-title {
            font-size: 16px;
            font-weight: 600;
            color: #2d3436;
            margin: 0 0 6px;
          }
          .chatbot-setup-desc {
            font-size: 13px;
            color: #718096;
            margin: 0 0 20px;
            line-height: 1.5;
          }
          .chatbot-cmd-row {
            display: flex;
            align-items: center;
            gap: 0;
            background: #1a1a2e;
            border-radius: 8px;
            overflow: hidden;
            margin-bottom: 16px;
            width: 100%;
          }
          .chatbot-cmd-text {
            flex: 1;
            padding: 10px 14px;
            color: #a5d6ff;
            font-family: monospace;
            font-size: 13px;
            user-select: all;
            text-align: left;
          }
          .chatbot-cmd-copy {
            display: flex;
            align-items: center;
            gap: 5px;
            padding: 10px 14px;
            background: rgba(255,255,255,0.08);
            border: none;
            border-left: 1px solid rgba(255,255,255,0.1);
            color: #a0aec0;
            font-size: 12px;
            cursor: pointer;
            white-space: nowrap;
            transition: background 0.15s, color 0.15s;
          }
          .chatbot-cmd-copy:hover {
            background: rgba(255,255,255,0.15);
            color: white;
          }
          .chatbot-launch-btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 8px 20px;
            background: transparent;
            color: #6c63ff;
            border: 1.5px solid #6c63ff;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.15s, color 0.15s;
          }
          .chatbot-launch-btn:hover:not(:disabled) {
            background: #6c63ff;
            color: white;
          }
          .chatbot-launch-btn:disabled {
            border-color: #a0aec0;
            color: #a0aec0;
            cursor: default;
            background: transparent;
          }
          .chatbot-launch-status {
            margin-top: 16px;
            font-size: 12px;
            line-height: 1.6;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 6px;
          }
          .chatbot-launch-waiting {
            color: #6c63ff;
          }
          .chatbot-launch-error {
            color: #e53e3e;
          }
          .chatbot-launch-cmd {
            display: block;
            margin: 4px 0;
            padding: 8px 14px;
            background: #1a1a2e;
            color: #a5d6ff;
            border-radius: 6px;
            font-family: monospace;
            font-size: 13px;
            user-select: all;
          }
          .chatbot-setup-alt {
            margin-top: 20px;
            width: 100%;
            font-size: 12px;
            color: #a0aec0;
          }
          .chatbot-setup-alt summary {
            cursor: pointer;
            user-select: none;
          }
          .chatbot-setup-alt .chatbot-api-key-input {
            flex: 1;
            padding: 7px 10px;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            font-size: 12px;
          }
          .chatbot-setup-alt .chatbot-api-key-save {
            padding: 7px 14px;
            background: #6c63ff;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 12px;
          }
          .chatbot-setup-hint {
            margin-top: 6px;
            font-size: 11px;
            color: #a0aec0;
          }
          .chatbot-typing {
            display: inline-flex;
            gap: 4px;
            align-items: center;
            height: 20px;
          }
          .chatbot-typing span {
            width: 7px;
            height: 7px;
            border-radius: 50%;
            background: #a0aec0;
            animation: chatbot-bounce 1.4s ease-in-out infinite;
          }
          .chatbot-typing span:nth-child(2) {
            animation-delay: 0.2s;
          }
          .chatbot-typing span:nth-child(3) {
            animation-delay: 0.4s;
          }
          @keyframes chatbot-bounce {
            0%, 80%, 100% { transform: scale(0.6); opacity: 0.4; }
            40% { transform: scale(1); opacity: 1; }
          }
        `;
        document.head.appendChild(style);
      }
    },

    /**
     * Send the current input text as a question.
     * If playback is active, queues until next segment boundary.
     */
    send() {
      const text = els.input.value.trim();
      if (!text) return;
      els.input.value = '';

      const context = getCurrentContext();

      if (isPlaybackActive && !document.body.classList.contains('chatbot-open')) {
        // Queue the question — will be sent at next segment boundary
        pendingQuestion = { question: text, context };
        showQueueIndicator(text);
      } else {
        // Playback is paused or drawer is open — send immediately
        streamResponse(text, context);
      }
    },

    /**
     * Clear chat history and reset the drawer.
     */
    clear() {
      messages = [];
      pendingQuestion = null;
      hideQueueIndicator();
      const modeHint = serveMode
        ? 'Ask anything about the code you\'re seeing. Powered by Claude Code.'
        : 'Pause the tutorial and ask anything about the code you\'re seeing.';
      els.messagesContainer.innerHTML = `
        <div class="chatbot-empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p>${modeHint}</p>
        </div>`;
    }
  };
})();

export default Chatbot;

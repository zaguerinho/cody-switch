// agent-hub dashboard SPA

const $ = (sel) => document.querySelector(sel);
const app = $('#app');
let currentView = 'home';
let currentRoom = null;
let refreshTimer = null;
let activityFilter = 'all';

// --- API ---
async function api(path) {
  const resp = await fetch('/api/v1' + path);
  const data = await resp.json();
  if (!data.ok) throw new Error(data.error || 'API error');
  return data.data;
}

// --- Navigation ---
function navigate(view, param) {
  currentView = view;
  currentRoom = param || null;
  clearInterval(refreshTimer);
  render();
  refreshTimer = setInterval(render, 5000);
}

// --- Render dispatcher ---
async function render() {
  try {
    switch (currentView) {
      case 'home': await renderHome(); break;
      case 'room': await renderRoom(currentRoom); break;
      case 'activity': await renderActivity(); break;
    }
    $('#server-status').innerHTML = '<span class="refresh-dot"></span>Connected';
  } catch (e) {
    $('#server-status').innerHTML = 'Disconnected';
    console.error(e);
  }
}

// --- Home: Room Cards ---
async function renderHome() {
  const sessions = await api('/rooms?all=true') || [];

  if (!sessions.length) {
    app.innerHTML = `
      <div class="page-header"><h1>Rooms</h1></div>
      <div class="empty">
        <h2>No rooms yet</h2>
        <p>Create one with: <code>agent-hub room create &lt;name&gt;</code></p>
      </div>`;
    return;
  }

  const cards = sessions.map(s => {
    const ago = timeAgo(s.last_activity);
    const archived = s.archived ? '<span class="card-badge badge-archived">archived</span>' : '';
    return `
      <div class="card" onclick="navigate('room', '${esc(s.name)}')">
        <div class="card-header">
          <span class="card-title">${esc(s.name)}</span>
          ${archived}
        </div>
        ${s.description ? `<div class="card-desc">${esc(s.description)}</div>` : ''}
        <div class="card-stats">
          <span>${s.agent_count || 0} agents</span>
          <span>${s.message_count || 0} messages</span>
          ${ago ? `<span>${ago}</span>` : ''}
        </div>
      </div>`;
  }).join('');

  app.innerHTML = `
    <div class="page-header"><h1>Rooms</h1></div>
    <div class="cards">${cards}</div>`;
}

// --- Room Detail ---
async function renderRoom(name) {
  const [info, agents, messages, status, docs, assess] = await Promise.all([
    api(`/rooms/${name}`),
    api(`/rooms/${name}/agents`),
    api(`/rooms/${name}/messages`),
    api(`/rooms/${name}/status`),
    api(`/rooms/${name}/docs`).catch(() => []),
    api(`/rooms/${name}/assess`).catch(() => null),
  ]);

  // Build cursor map for read markers
  const cursorMap = {};
  if (agents && agents.length) {
    for (const a of agents) {
      try {
        const c = await api(`/rooms/${name}/messages/check?as=${a.alias}`);
        const lastRead = (c.unread === 0 && messages.length)
          ? messages[messages.length - 1].id
          : (messages.length - c.unread > 0 ? messages[messages.length - 1 - c.unread].id : 0);
        cursorMap[a.alias] = lastRead;
      } catch { /* ignore */ }
    }
  }

  // Sidebar: Agents
  const agentItems = (agents || []).map(a => `
    <div class="agent-item">
      <span class="agent-dot"></span>
      <span>${esc(a.alias)}</span>
      ${a.role ? `<span class="agent-role">${esc(a.role)}</span>` : ''}
    </div>`).join('') || '<div style="color:var(--text-muted);font-size:13px">No agents</div>';

  // Sidebar: Phase + Progress (from assess)
  const phaseLabels = {0:'Discovery',1:'Setup',2:'Investigation',3:'Evidence',4:'Review',5:'Sign-off'};
  const phase = assess?.phase ?? 0;
  const phaseLabel = phaseLabels[phase] || 'Unknown';
  const phaseColors = {0:'var(--purple)',1:'var(--blue)',2:'var(--orange)',3:'var(--orange)',4:'var(--cyan)',5:'var(--green)'};

  // Dimensions from assess
  const dims = assess?.dimensions || {};
  const dimEntries = Object.entries(dims);
  const dimCount = dimEntries.length || 1;
  const dimScores = dimEntries.map(([,d]) => d.status === 'GREEN' ? 1 : d.status === 'YELLOW' ? 0.5 : 0);
  const dimProgress = dimEntries.length ? Math.round((dimScores.reduce((a,b) => a+b, 0) / dimCount) * 100) : 0;
  // Overall progress: phase weight (40%) + dimension progress (60%)
  const phaseProgress = Math.round((phase / 5) * 100);
  const overallProgress = Math.round(phaseProgress * 0.4 + dimProgress * 0.6);

  const dimStatusColors = {'RED':'var(--red)','YELLOW':'var(--orange)','GREEN':'var(--green)'};

  const dimensionHTML = dimEntries.length ? dimEntries.map(([k, d]) => `
    <div class="dim-row">
      <span class="dim-dot" style="background:${dimStatusColors[d.status] || 'var(--text-muted)'}"></span>
      <span class="dim-name">${esc(k.replace('dim-',''))}</span>
      <span class="dim-status" style="color:${dimStatusColors[d.status] || 'var(--text-muted)'}">${esc(d.status)}</span>
      <span class="dim-owner">${d.owner ? esc(d.owner) : '—'}</span>
    </div>`).join('') : '<div style="color:var(--text-muted);font-size:12px">No dimensions tracked yet</div>';

  // Open RFCs
  const openRFCs = assess?.open_rfcs || [];
  const rfcHTML = openRFCs.length ? openRFCs.map(id =>
    `<span class="rfc-badge">RFC #${id}</span>`).join(' ') : '';

  // Advance readiness
  let advanceHTML = '';
  if (assess?.ready_to_advance) {
    advanceHTML = assess?.human_approval_needed
      ? '<div class="advance-ready human">Awaiting human approval</div>'
      : '<div class="advance-ready">Ready to advance</div>';
  } else if (assess?.advance_blocker) {
    advanceHTML = `<div class="advance-blocked">${esc(assess.advance_blocker)}</div>`;
  }

  // Warnings from assess
  const warnings = assess?.warnings || [];
  const warningsHTML = warnings.map(w =>
    `<div class="warning-item">${esc(w)}</div>`).join('');

  // Other status entries (non-phase, non-dim, non-rfc)
  const otherStatus = (status?.entries || []).filter(e =>
    e.key !== 'phase' && !e.key.startsWith('dim-') && !e.key.startsWith('rfc-')
  );
  const otherHTML = otherStatus.map(e => `
    <div class="status-entry">
      <div><span class="status-key">${esc(e.key)}:</span> ${esc(e.value)}</div>
      <div class="status-by">by ${esc(e.updated_by)} ${timeAgo(e.updated_at)}</div>
    </div>`).join('');

  // Messages with read markers
  let msgHTML = '';
  const sortedMsgs = [...(messages || [])];
  for (const msg of sortedMsgs) {
    // Insert read markers before this message
    for (const [alias, lastRead] of Object.entries(cursorMap)) {
      if (lastRead === msg.id - 1 && lastRead > 0 && lastRead < sortedMsgs[sortedMsgs.length - 1].id) {
        msgHTML += `<div class="read-marker">${esc(alias)} read up to here</div>`;
      }
    }
    msgHTML += renderMessage(msg);
  }

  if (!sortedMsgs.length) {
    msgHTML = '<div class="empty"><p>No messages yet</p></div>';
  }

  // Sidebar: Docs
  const docItems = (docs || []).map(d => `
    <div class="doc-item" onclick="showDoc('${esc(name)}', '${esc(d.name)}')">
      <span class="doc-icon">&#128196;</span>
      <span>${esc(d.name)}</span>
    </div>`).join('') || '<div style="color:var(--text-muted);font-size:13px">No docs</div>';

  app.innerHTML = `
    <a href="#" onclick="navigate('home')" class="back-link">Back to rooms</a>
    <div class="room-header">
      <h1>${esc(info.name)}</h1>
      ${info.description ? `<p>${esc(info.description)}</p>` : ''}
    </div>
    <div class="room-layout">
      <div class="sidebar">
        <div class="sidebar-section">
          <div class="phase-header">
            <span class="phase-badge" style="background:${phaseColors[phase]}">Phase ${phase}</span>
            <span class="phase-label">${phaseLabel}</span>
          </div>
          <div class="progress-bar-wrap">
            <div class="progress-bar" style="width:${overallProgress}%;background:${overallProgress >= 80 ? 'var(--green)' : overallProgress >= 40 ? 'var(--orange)' : 'var(--red)'}"></div>
          </div>
          <div class="progress-label">${overallProgress}% complete</div>
          ${advanceHTML}
        </div>
        <div class="sidebar-section">
          <h3>Dimensions</h3>
          ${dimensionHTML}
        </div>
        ${warningsHTML ? `<div class="sidebar-section warnings-section"><h3>Warnings</h3>${warningsHTML}</div>` : ''}
        ${rfcHTML ? `<div class="sidebar-section"><h3>Open RFCs</h3><div class="rfc-list">${rfcHTML}</div></div>` : ''}
        ${otherHTML ? `<div class="sidebar-section"><h3>Status</h3>${otherHTML}</div>` : ''}
        <div class="sidebar-section">
          <h3>Agents (${(agents || []).length})</h3>
          ${agentItems}
        </div>
        <div class="sidebar-section">
          <h3>Docs</h3>
          ${docItems}
        </div>
      </div>
      <div class="messages" id="room-main">${msgHTML}</div>
    </div>`;
}

function renderMessage(msg) {
  return `
    <div class="message type-${esc(msg.type)}">
      <div class="msg-header">
        <span class="msg-badge ${esc(msg.type)}">${esc(msg.type)}</span>
        <span class="msg-from">${esc(msg.from)}</span>
        <span class="msg-id">#${msg.id}</span>
        <span class="msg-time">${formatTime(msg.timestamp)}</span>
      </div>
      ${msg.subject ? `<div class="msg-subject">${esc(msg.subject)}</div>` : ''}
      <div class="msg-body">${renderMarkdown(msg.body)}</div>
    </div>`;
}

// Lightweight markdown renderer (no external deps)
function renderMarkdown(text) {
  if (!text) return '';
  let html = esc(text);

  // Code blocks (``` ... ```)
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre class="md-code-block"><code>$2</code></pre>');

  // Inline code (`...`)
  html = html.replace(/`([^`]+)`/g, '<code class="md-code">$1</code>');

  // Bold (**...**)
  html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');

  // Italic (*...*)
  html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>');

  // Headers (## ... at start of line)
  html = html.replace(/^### (.+)$/gm, '<div class="md-h3">$1</div>');
  html = html.replace(/^## (.+)$/gm, '<div class="md-h2">$1</div>');

  // Numbered lists (1. 2. 3.)
  html = html.replace(/^(\d+)\.\s+(.+)$/gm, '<div class="md-li"><span class="md-li-num">$1.</span> $2</div>');

  // Bullet lists (- ...)
  html = html.replace(/^- (.+)$/gm, '<div class="md-li"><span class="md-li-bullet">-</span> $1</div>');

  // Paragraphs: double newline = paragraph break
  html = html.replace(/\n\n+/g, '</p><p>');

  // Single newlines = line break
  html = html.replace(/\n/g, '<br>');

  return '<p>' + html + '</p>';
}

// --- Activity Feed ---
async function renderActivity() {
  const sessions = await api('/rooms') || [];
  let allMessages = [];

  for (const s of sessions) {
    try {
      const msgs = await api(`/rooms/${s.name}/messages`);
      if (msgs) {
        for (const m of msgs) {
          allMessages.push({ ...m, room: s.name });
        }
      }
    } catch { /* skip */ }
  }

  // Sort by timestamp descending
  allMessages.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));

  // Apply filter
  if (activityFilter !== 'all') {
    allMessages = allMessages.filter(m => m.type === activityFilter);
  }

  const types = ['all', 'question', 'answer', 'rfc', 'note', 'status-update'];
  const filterHTML = types.map(t =>
    `<button class="filter-btn ${activityFilter === t ? 'active' : ''}" onclick="setActivityFilter('${t}')">${t}</button>`
  ).join('');

  const items = allMessages.slice(0, 100).map(m => `
    <div class="activity-item">
      <span class="activity-room" onclick="navigate('room', '${esc(m.room)}')">${esc(m.room)}</span>
      <span class="msg-badge ${esc(m.type)}">${esc(m.type)}</span>
      <span class="msg-from">${esc(m.from)}</span>
      <span style="flex:1;color:var(--text-muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
        ${m.subject ? esc(m.subject) : esc(m.body.slice(0, 80))}
      </span>
      <span class="msg-time">${timeAgo(m.timestamp)}</span>
    </div>`).join('');

  app.innerHTML = `
    <div class="page-header"><h1>Activity Feed</h1></div>
    <div class="filters">${filterHTML}</div>
    ${items || '<div class="empty"><p>No activity</p></div>'}`;
}

function setActivityFilter(f) {
  activityFilter = f;
  renderActivity();
}

// --- Helpers ---
function esc(s) {
  if (s == null) return '';
  const div = document.createElement('div');
  div.textContent = String(s);
  return div.innerHTML;
}

function timeAgo(ts) {
  if (!ts) return '';
  const diff = Date.now() - new Date(ts).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function formatTime(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  return d.toLocaleString('en-US', {
    month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
    hour12: false
  });
}

// --- Doc viewer ---
async function showDoc(room, docName) {
  try {
    const data = await api(`/rooms/${room}/docs/${docName}`);
    const main = document.getElementById('room-main');
    if (main) {
      main.innerHTML = `
        <div class="doc-viewer">
          <div class="doc-viewer-header">
            <h2>${esc(docName)}</h2>
            <button class="filter-btn" onclick="navigate('room', '${esc(room)}')">Back to messages</button>
          </div>
          <pre class="doc-content">${esc(data.content)}</pre>
        </div>`;
    }
  } catch (e) {
    console.error('Failed to load doc:', e);
  }
}

// --- Boot ---
navigate('home');

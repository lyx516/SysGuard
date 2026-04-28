package ui

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>SysGuard 监控看板</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f7f9;
      --ink: #171717;
      --muted: #60646c;
      --line: #d9dde3;
      --panel: #ffffff;
      --panel-soft: #f1f4f4;
      --good: #12805c;
      --warn: #9a6700;
      --bad: #c62828;
      --accent: #007a7a;
      --accent-2: #5d6b2f;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--ink);
      font: 15px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      letter-spacing: 0;
    }
    header {
      display: flex;
      justify-content: space-between;
      gap: 18px;
      padding: 22px 28px;
      border-bottom: 1px solid var(--line);
      background: var(--panel);
      position: sticky;
      top: 0;
      z-index: 4;
    }
    h1, h2, h3, p { margin: 0; }
    h1 { font-size: 24px; font-weight: 760; }
    h2 { font-size: 17px; margin-bottom: 12px; }
    h3 { font-size: 15px; margin-bottom: 4px; }
    button, select, input {
      border-radius: 6px;
      border: 1px solid var(--line);
      background: #fff;
      color: var(--ink);
      padding: 9px 10px;
      font: inherit;
    }
    button {
      border-color: #0a6f6f;
      background: var(--accent);
      color: #fff;
      cursor: pointer;
      font-weight: 650;
    }
    button.secondary {
      background: #fff;
      color: var(--accent);
    }
    main {
      display: grid;
      grid-template-columns: 230px 1fr;
      gap: 18px;
      padding: 18px 28px 36px;
    }
    nav {
      position: sticky;
      top: 94px;
      display: grid;
      gap: 8px;
      align-self: start;
    }
    nav button {
      width: 100%;
      text-align: left;
      background: #fff;
      color: var(--ink);
      border-color: var(--line);
    }
    nav button.active {
      color: #fff;
      background: var(--accent);
      border-color: var(--accent);
    }
    .status-line { color: var(--muted); margin-top: 4px; }
    .actions { display: flex; flex-wrap: wrap; gap: 9px; align-items: center; justify-content: flex-end; }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 16px;
      align-items: start;
    }
    section {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 16px;
      min-width: 0;
    }
    .view { display: none; }
    .view.active { display: grid; }
    .span-3 { grid-column: span 3; }
    .span-4 { grid-column: span 4; }
    .span-5 { grid-column: span 5; }
    .span-6 { grid-column: span 6; }
    .span-7 { grid-column: span 7; }
    .span-8 { grid-column: span 8; }
    .span-12 { grid-column: span 12; }
    .metric {
      display: flex;
      align-items: end;
      justify-content: space-between;
      gap: 10px;
      min-height: 82px;
    }
    .metric strong { font-size: 30px; line-height: 1; }
    .muted { color: var(--muted); }
    .pill {
      display: inline-flex;
      align-items: center;
      border-radius: 6px;
      border: 1px solid var(--line);
      padding: 2px 7px;
      font-size: 12px;
      font-weight: 700;
      white-space: nowrap;
    }
    .healthy, .completed { color: var(--good); background: #e9f7f1; border-color: #b8e2d1; }
    .running, .warning, .degraded { color: var(--warn); background: #fff7df; border-color: #ead589; }
    .error, .down, .failed { color: var(--bad); background: #fdebec; border-color: #f1b9bd; }
    .standby, .info, .started { color: #4a4f58; background: #f0f2f5; border-color: #d7dbe1; }
    .toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 9px;
      margin-bottom: 12px;
      align-items: center;
    }
    .toolbar input { min-width: 240px; flex: 1; }
    .list { display: grid; gap: 10px; }
    .row {
      display: grid;
      gap: 8px;
      border-top: 1px solid var(--line);
      padding-top: 10px;
      cursor: pointer;
    }
    .row:first-child { border-top: 0; padding-top: 0; }
    .row:hover { background: var(--panel-soft); outline: 8px solid var(--panel-soft); }
    .row-head {
      display: flex;
      gap: 10px;
      justify-content: space-between;
      align-items: start;
    }
    .agent-row {
      grid-template-columns: 130px 1fr auto;
      align-items: center;
    }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th, td {
      border-top: 1px solid var(--line);
      padding: 9px 7px;
      text-align: left;
      vertical-align: top;
      overflow-wrap: anywhere;
    }
    th { color: var(--muted); font-size: 12px; font-weight: 740; }
    tr.clickable { cursor: pointer; }
    tr.clickable:hover { background: var(--panel-soft); }
    .bar {
      width: 100%;
      height: 10px;
      background: #eef0f3;
      border-radius: 6px;
      overflow: hidden;
      margin-top: 8px;
    }
    .bar span { display: block; height: 100%; background: var(--accent); }
    .pre {
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      background: #f4f5f6;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 10px;
      max-height: 360px;
      overflow: auto;
    }
    .timeline-row {
      display: grid;
      grid-template-columns: 90px 82px 1fr;
      gap: 9px;
      border-top: 1px solid var(--line);
      padding-top: 9px;
      min-width: 0;
    }
    .timeline-row:first-child { border-top: 0; padding-top: 0; }
    .drawer {
      position: fixed;
      right: 0;
      top: 0;
      width: min(720px, 100vw);
      height: 100vh;
      background: var(--panel);
      border-left: 1px solid var(--line);
      box-shadow: -12px 0 28px rgba(0,0,0,.12);
      transform: translateX(105%);
      transition: transform .18s ease;
      z-index: 10;
      padding: 18px;
      overflow: auto;
    }
    .drawer.open { transform: translateX(0); }
    .drawer-header {
      display: flex;
      justify-content: space-between;
      gap: 14px;
      align-items: start;
      margin-bottom: 14px;
    }
    .drawer h2 { margin-bottom: 4px; }
    .overlay {
      position: fixed;
      inset: 0;
      background: rgba(0,0,0,.18);
      opacity: 0;
      pointer-events: none;
      transition: opacity .18s ease;
      z-index: 9;
    }
    .overlay.open { opacity: 1; pointer-events: auto; }
    .tabs { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
    .tabs button {
      background: #fff;
      color: var(--accent);
    }
    .tabs button.active {
      background: var(--accent-2);
      border-color: var(--accent-2);
      color: #fff;
    }
    @media (max-width: 1040px) {
      header { flex-direction: column; align-items: flex-start; }
      main { grid-template-columns: 1fr; }
      nav { position: static; display: flex; flex-wrap: wrap; }
      nav button { width: auto; }
      .span-3, .span-4, .span-5, .span-6, .span-7, .span-8 { grid-column: span 12; }
      .agent-row, .timeline-row { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <h1>SysGuard 监控看板</h1>
      <p class="status-line" id="connection">连接中，等待 A2UI 实时数据。</p>
    </div>
    <div class="actions">
      <button id="check">立即巡检</button>
      <button id="pause" class="secondary">暂停实时刷新</button>
      <button id="open-a2ui" class="secondary">查看 A2UI 数据</button>
    </div>
  </header>

  <main>
    <nav>
      <button class="nav active" data-view="overview">总览</button>
      <button class="nav" data-view="agents-view">Agent</button>
      <button class="nav" data-view="tools-view">工具调用</button>
      <button class="nav" data-view="docs-view">历史文档</button>
      <button class="nav" data-view="logs-view">日志</button>
      <button class="nav" data-view="history-view">修复历史</button>
      <button class="nav" data-view="runs-view">运行记录</button>
    </nav>

    <div>
      <div id="overview" class="grid view active">
        <section class="span-3 metric"><div><p class="muted">健康分</p><strong id="score">--</strong></div><span id="health-pill" class="pill standby">等待</span></section>
        <section class="span-3 metric"><div><p class="muted">Agent</p><strong id="agent-count">--</strong></div><span class="pill info">运行态</span></section>
        <section class="span-3 metric"><div><p class="muted">工具调用</p><strong id="tool-count">--</strong></div><span id="tool-errors" class="pill standby">-- 错误</span></section>
        <section class="span-3 metric"><div><p class="muted">运行记录</p><strong id="run-count">--</strong></div><span id="run-failures" class="pill completed">-- 失败</span></section>

        <section class="span-4">
          <h2>系统占用</h2>
          <div id="metrics"></div>
        </section>
        <section class="span-8">
          <h2>运行脉络</h2>
          <div id="timeline" class="list"></div>
        </section>
      </div>

      <div id="agents-view" class="grid view">
        <section class="span-12">
          <h2>Agent 运行过程</h2>
          <div id="agents" class="list"></div>
        </section>
      </div>

      <div id="tools-view" class="grid view">
        <section class="span-12">
          <h2>工具调用历史</h2>
          <div class="toolbar">
            <input id="tool-search" placeholder="搜索工具、状态、摘要">
            <select id="tool-status"><option value="">全部状态</option><option value="completed">completed</option><option value="started">started</option><option value="error">error</option></select>
          </div>
          <table>
            <thead><tr><th>工具/节点</th><th>状态</th><th>开始时间</th><th>耗时</th><th>摘要</th></tr></thead>
            <tbody id="tools"></tbody>
          </table>
        </section>
      </div>

      <div id="docs-view" class="grid view">
        <section class="span-12">
          <h2>历史文档与技能库</h2>
          <div class="toolbar">
            <input id="doc-search" placeholder="搜索 SOP、技能、命令、路径">
            <select id="doc-kind"><option value="">全部文档</option><option value="sop">SOP</option><option value="skill">技能</option></select>
          </div>
          <div id="documents" class="list"></div>
        </section>
      </div>

      <div id="logs-view" class="grid view">
        <section class="span-12">
          <h2>日志统计</h2>
          <div class="toolbar">
            <input id="log-search" placeholder="搜索日志内容">
            <select id="log-level"><option value="">全部级别</option><option value="info">info</option><option value="warning">warning</option><option value="error">error</option></select>
          </div>
          <div id="logs" class="list"></div>
        </section>
      </div>

      <div id="history-view" class="grid view">
        <section class="span-12">
          <h2>问题解决历史</h2>
          <div class="toolbar">
            <input id="history-search" placeholder="搜索问题、方案、命令步骤">
            <select id="history-status"><option value="">全部结果</option><option value="true">成功</option><option value="false">失败</option></select>
          </div>
          <div id="history" class="list"></div>
        </section>
      </div>

      <div id="runs-view" class="grid view">
        <section class="span-12">
          <h2>Graph 运行记录</h2>
          <div class="toolbar">
            <input id="run-search" placeholder="搜索 run id、分支、异常、结论">
            <select id="run-status"><option value="">全部状态</option><option value="running">running</option><option value="completed">completed</option><option value="failed">failed</option></select>
          </div>
          <table>
            <thead><tr><th>Run</th><th>状态</th><th>分支</th><th>触发</th><th>健康分</th><th>异常</th></tr></thead>
            <tbody id="runs"></tbody>
          </table>
        </section>
      </div>
    </div>
  </main>

  <div id="overlay" class="overlay"></div>
  <aside id="drawer" class="drawer" aria-live="polite">
    <div class="drawer-header">
      <div><h2 id="drawer-title">详情</h2><p id="drawer-subtitle" class="muted"></p></div>
      <button id="drawer-close" class="secondary">关闭</button>
    </div>
    <div id="drawer-body"></div>
  </aside>

  <script>
    const $ = (id) => document.getElementById(id);
    let state = null;
    let paused = false;

    const escapeHTML = (value) => String(value ?? '').replace(/[&<>"']/g, (char) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
    const fmtTime = (value) => {
      if (!value || value === '0001-01-01T00:00:00Z') return '刚刚';
      const date = new Date(value);
      return Number.isNaN(date.getTime()) ? '刚刚' : date.toLocaleString();
    };
    const pill = (status) => '<span class="pill ' + escapeHTML(status || 'info') + '">' + escapeHTML(status || 'info') + '</span>';
    const pct = (value) => Math.max(0, Math.min(100, Number(value || 0)));
    const match = (text, query) => String(text || '').toLowerCase().includes(String(query || '').toLowerCase());
    const pretty = (value) => escapeHTML(JSON.stringify(value, null, 2));

    function render(snapshot) {
      state = snapshot;
      $('score').textContent = Number(snapshot.system.health_score || 0).toFixed(1);
      $('health-pill').textContent = snapshot.system.is_healthy ? 'healthy' : 'degraded';
      $('health-pill').className = 'pill ' + (snapshot.system.is_healthy ? 'healthy' : 'degraded');
      $('agent-count').textContent = (snapshot.agents || []).length;
      $('tool-count').textContent = snapshot.tools.total;
      $('tool-errors').textContent = snapshot.tools.errors + ' 错误';
      $('tool-errors').className = 'pill ' + (snapshot.tools.errors ? 'error' : 'completed');
      $('run-count').textContent = snapshot.runs.total;
      $('run-failures').textContent = snapshot.runs.failed + ' 失败';
      $('run-failures').className = 'pill ' + (snapshot.runs.failed ? 'error' : 'completed');

      renderMetrics(snapshot);
      renderTimeline(snapshot.timeline || []);
      renderAgents(snapshot.agents || []);
      renderTools();
      renderDocuments();
      renderLogs();
      renderHistory();
      renderRuns();
      $('connection').textContent = 'A2UI 实时数据已连接，最近更新 ' + fmtTime(snapshot.generated_at) + '。';
    }

    function renderMetrics(snapshot) {
      const metrics = snapshot.system.collected || {};
      $('metrics').innerHTML = Object.keys(metrics).map((key) => {
        const item = metrics[key];
        return '<div style="margin-bottom:14px"><strong>' + escapeHTML(item.label) + '</strong><span class="muted" style="float:right">' + Number(item.value).toFixed(1) + escapeHTML(item.unit) + '</span><div class="bar"><span style="width:' + pct(item.value) + '%"></span></div></div>';
      }).join('') || '<p class="muted">暂无系统占用数据。</p>';
      $('metrics').innerHTML += '<div class="pre">受管服务: ' + snapshot.system.managed_services + '\\n巡检周期: ' + escapeHTML(snapshot.system.config.check_interval) + '\\nTrace: ' + escapeHTML(snapshot.system.config.trace_log) + '</div>';
    }

    function renderTimeline(timeline) {
      $('timeline').innerHTML = timeline.map((event) =>
        '<div class="timeline-row row" data-detail="timeline" data-id="' + escapeHTML(event.message) + '"><span class="muted">' + fmtTime(event.time) + '</span><span>' + pill(event.level) + '</span><span>' + escapeHTML(event.source + ': ' + event.message) + '</span></div>'
      ).join('') || '<p class="muted">暂无运行事件。</p>';
    }

    function renderAgents(agents) {
      $('agents').innerHTML = agents.map((agent) =>
        '<div class="row agent-row" data-detail="agent" data-id="' + escapeHTML(agent.name) + '"><div><h3>' + escapeHTML(agent.name) + '</h3><p class="muted">' + escapeHTML(agent.role) + '</p></div><div class="muted">' + escapeHTML(agent.last_event || '等待下一次事件') + '</div><div>' + pill(agent.status) + '<p class="muted">' + agent.runs + ' 次 / ' + agent.errors + ' 错误</p></div></div>'
      ).join('');
    }

    function renderTools() {
      if (!state) return;
      const query = $('tool-search').value;
      const status = $('tool-status').value;
      const tools = (state.tools.recent || []).filter((tool) => (!status || tool.status === status) && match([tool.name, tool.status, tool.summary, tool.error].join(' '), query));
      $('tools').innerHTML = tools.map((tool) =>
        '<tr class="clickable" data-detail="tool" data-id="' + escapeHTML(tool.id) + '"><td>' + escapeHTML(tool.name) + '</td><td>' + pill(tool.status) + '</td><td>' + fmtTime(tool.started_at) + '</td><td>' + tool.duration_millis + 'ms</td><td>' + escapeHTML(tool.summary) + '</td></tr>'
      ).join('') || '<tr><td colspan="5" class="muted">暂无匹配的工具调用。</td></tr>';
    }

    function renderDocuments() {
      if (!state) return;
      const query = $('doc-search').value;
      const kind = $('doc-kind').value;
      const docs = (state.documents.items || []).filter((doc) => (!kind || doc.kind === kind) && match([doc.title, doc.path, doc.preview, (doc.commands || []).join(' ')].join(' '), query));
      $('documents').innerHTML = docs.map((doc) =>
        '<div class="row" data-detail="doc" data-id="' + escapeHTML(doc.id) + '"><div class="row-head"><div><h3>' + escapeHTML(doc.title) + '</h3><p class="muted">' + escapeHTML(doc.path) + '</p></div>' + pill(doc.kind) + '</div><p>' + escapeHTML(doc.preview) + '</p><p class="muted">' + (doc.commands || []).length + ' 条可见命令</p></div>'
      ).join('') || '<p class="muted">暂无匹配的历史文档。</p>';
    }

    function renderLogs() {
      if (!state) return;
      const query = $('log-search').value;
      const level = $('log-level').value;
      const logs = (state.logs.recent || []).filter((log) => (!level || log.level === level) && match(log.message, query));
      $('logs').innerHTML = '<p>总日志 ' + state.logs.total + '，错误 ' + state.logs.errors + '，警告 ' + state.logs.warnings + '</p>' +
        logs.map((log, index) => '<div class="row pre" data-detail="log" data-id="' + index + '">' + pill(log.level) + ' ' + escapeHTML(log.message) + '</div>').join('');
    }

    function renderHistory() {
      if (!state) return;
      const query = $('history-search').value;
      const wanted = $('history-status').value;
      const records = (state.history.recent || []).filter((record) => (wanted === '' || String(record.success) === wanted) && match([record.description, record.solution, (record.steps || []).join(' ')].join(' '), query));
      $('history').innerHTML = records.map((record) =>
        '<div class="row" data-detail="history" data-id="' + escapeHTML(record.id) + '"><div class="row-head"><div><h3>' + escapeHTML(record.solution) + '</h3><p class="muted">' + escapeHTML(record.description) + '</p></div>' + pill(record.success ? 'completed' : 'error') + '</div><p class="muted">' + fmtTime(record.timestamp) + ' / ' + (record.steps || []).length + ' 步</p></div>'
      ).join('') || '<p class="muted">暂无匹配的修复历史。</p>';
    }

    function renderRuns() {
      if (!state) return;
      const query = $('run-search').value;
      const status = $('run-status').value;
      const runs = (state.runs.recent || []).filter((run) => (!status || run.status === status) && match([run.run_id, run.status, run.branch, run.trigger, run.anomaly, run.agent_final, run.agent_error].join(' '), query));
      $('runs').innerHTML = runs.map((run) =>
        '<tr class="clickable" data-detail="run" data-id="' + escapeHTML(run.run_id) + '"><td>' + escapeHTML(run.run_id) + '<p class="muted">' + fmtTime(run.started_at) + '</p></td><td>' + pill(run.status) + '</td><td>' + escapeHTML(run.branch) + '</td><td>' + escapeHTML(run.trigger) + '</td><td>' + Number(run.health_score || 0).toFixed(1) + '</td><td>' + escapeHTML(run.anomaly || '无') + '</td></tr>'
      ).join('') || '<tr><td colspan="6" class="muted">暂无匹配的 graph 运行。</td></tr>';
    }

    function openDrawer(title, subtitle, html) {
      $('drawer-title').textContent = title;
      $('drawer-subtitle').textContent = subtitle || '';
      $('drawer-body').innerHTML = html;
      $('drawer').classList.add('open');
      $('overlay').classList.add('open');
    }

    function closeDrawer() {
      $('drawer').classList.remove('open');
      $('overlay').classList.remove('open');
    }

    function detail(kind, id) {
      if (!state) return;
      if (kind === 'agent') {
        const agent = (state.agents || []).find((item) => item.name === id);
        const related = (state.tools.recent || []).filter((tool) => tool.name.startsWith(agent.name + '.'));
        openDrawer(agent.name, agent.role, '<p>' + pill(agent.status) + ' 运行 ' + agent.runs + ' 次，错误 ' + agent.errors + ' 次</p><h3>最近事件</h3><div class="pre">' + escapeHTML(agent.last_event || '暂无') + '</div><h3>关联工具调用</h3><div class="pre">' + pretty(related) + '</div>');
      }
      if (kind === 'tool') {
        const tool = (state.tools.recent || []).find((item) => item.id === id);
        openDrawer(tool.name, tool.id, '<p>' + pill(tool.status) + ' 耗时 ' + tool.duration_millis + 'ms</p><h3>摘要</h3><div class="pre">' + escapeHTML(tool.summary) + '</div><h3>Payload Data</h3><div class="pre">' + pretty(tool.data || {}) + '</div><h3>Trace 事件</h3><div class="pre">' + pretty(tool.events || []) + '</div>');
      }
      if (kind === 'doc') {
        const doc = (state.documents.items || []).find((item) => item.id === id);
        openDrawer(doc.title, doc.path, '<p>' + pill(doc.kind) + '</p><h3>摘要</h3><div class="pre">' + escapeHTML(doc.preview) + '</div><h3>命令</h3><div class="pre">' + escapeHTML((doc.commands || []).join('\\n') || '无命令块') + '</div>');
      }
      if (kind === 'history') {
        const record = (state.history.recent || []).find((item) => item.id === id);
        openDrawer(record.solution, record.description, '<p>' + pill(record.success ? 'completed' : 'error') + ' ' + fmtTime(record.timestamp) + '</p><h3>执行步骤</h3><div class="pre">' + escapeHTML((record.steps || []).join('\\n') || '无步骤') + '</div>');
      }
      if (kind === 'run') {
        const run = (state.runs.recent || []).find((item) => item.run_id === id);
        openDrawer(run.run_id, fmtTime(run.started_at), '<p>' + pill(run.status) + ' 分支 ' + escapeHTML(run.branch) + ' / 触发 ' + escapeHTML(run.trigger) + '</p><h3>异常</h3><div class="pre">' + escapeHTML(run.anomaly || '无异常') + '</div><h3>Agent 输出</h3><div class="pre">' + escapeHTML(run.agent_final || run.agent_error || '无') + '</div><h3>工具与验证</h3><div class="pre">' + escapeHTML('tools: ' + (run.tools || []).join(', ') + '\\nverification: ' + (run.verification || '无') + '\\nhistory_written: ' + run.history_written) + '</div>');
      }
      if (kind === 'log') {
        const log = (state.logs.recent || [])[Number(id)];
        openDrawer('日志详情', log.level, '<div class="pre">' + escapeHTML(log.message) + '</div>');
      }
      if (kind === 'timeline') {
        openDrawer('时间线事件', '', '<div class="pre">' + escapeHTML(id) + '</div>');
      }
    }

    async function loadSnapshot() {
      const response = await fetch('/api/snapshot');
      render(await response.json());
    }

    async function openA2UI() {
      const response = await fetch('/a2ui/render');
      const data = await response.json();
      openDrawer('A2UI 数据模型', '/a2ui/render', '<div class="pre">' + pretty(data) + '</div>');
    }

    document.body.addEventListener('click', (event) => {
      const nav = event.target.closest('.nav');
      if (nav) {
        document.querySelectorAll('.nav').forEach((item) => item.classList.remove('active'));
        document.querySelectorAll('.view').forEach((item) => item.classList.remove('active'));
        nav.classList.add('active');
        $(nav.dataset.view).classList.add('active');
      }
      const target = event.target.closest('[data-detail]');
      if (target) detail(target.dataset.detail, target.dataset.id);
    });

    ['tool-search','tool-status'].forEach((id) => $(id).addEventListener('input', renderTools));
    ['doc-search','doc-kind'].forEach((id) => $(id).addEventListener('input', renderDocuments));
    ['log-search','log-level'].forEach((id) => $(id).addEventListener('input', renderLogs));
    ['history-search','history-status'].forEach((id) => $(id).addEventListener('input', renderHistory));
    ['run-search','run-status'].forEach((id) => $(id).addEventListener('input', renderRuns));
    $('drawer-close').addEventListener('click', closeDrawer);
    $('overlay').addEventListener('click', closeDrawer);
    $('open-a2ui').addEventListener('click', openA2UI);
    $('pause').addEventListener('click', () => {
      paused = !paused;
      $('pause').textContent = paused ? '恢复实时刷新' : '暂停实时刷新';
    });
    $('check').addEventListener('click', async () => {
      const response = await fetch('/api/check', { method: 'POST' });
      render(await response.json());
    });

    loadSnapshot();
    const events = new EventSource('/api/stream');
    events.addEventListener('a2ui', (event) => {
      if (paused) return;
      const message = JSON.parse(event.data);
      render(message.payload.model);
    });
    events.onerror = () => {
      $('connection').textContent = 'A2UI 实时数据连接中断，正在重连。';
    };
  </script>
</body>
</html>`

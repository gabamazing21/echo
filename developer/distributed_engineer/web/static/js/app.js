// Consensus — progressive enhancement. The app renders fully server-side;
// this script wires the interactive bits to the JSON API.

function flash(msg) {
  const f = document.getElementById('flash');
  if (!f) return;
  f.textContent = msg || 'saved';
  f.classList.add('show');
  setTimeout(() => f.classList.remove('show'), 900);
}

async function post(url, data) {
  const body = new URLSearchParams(data || {});
  const r = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });
  if (!r.ok) throw new Error('request failed: ' + r.status);
  return r.json();
}

document.addEventListener('click', async (e) => {
  const t = e.target;

  // commit / un-commit a task
  const commit = t.closest('[data-task]');
  if (commit) {
    const id = commit.getAttribute('data-task');
    try {
      await post(`/api/task/${id}/toggle`);
      location.reload();
    } catch (err) { console.error(err); }
    return;
  }

  // step accordion (roadmap)
  const toggle = t.closest('[data-toggle]');
  if (toggle) {
    toggle.closest('.step').classList.toggle('open');
    return;
  }

  // grade a theory quiz
  const grade = t.closest('[data-grade]');
  if (grade) {
    const id = grade.getAttribute('data-grade');
    const picked = document.querySelector(`input[name="q${id}"]:checked`);
    if (!picked) { flash('pick an answer first'); return; }
    try {
      await post(`/api/quiz/${id}/grade`, { answer: picked.value });
      location.reload();
    } catch (err) { console.error(err); }
    return;
  }

  // reveal/hide a code challenge's approach
  const reveal = t.closest('[data-reveal]');
  if (reveal) {
    const id = reveal.getAttribute('data-reveal');
    try {
      await post(`/api/quiz/${id}/reveal`);
      location.reload();
    } catch (err) { console.error(err); }
    return;
  }

  // resource filter chips
  if (t.hasAttribute('data-rs')) {
    document.querySelectorAll('[data-rs]').forEach((c) => c.classList.toggle('on', c === t));
    filterResources();
    return;
  }
  if (t.hasAttribute('data-free')) {
    t.classList.toggle('on');
    filterResources();
    return;
  }

  // run the code checker
  const run = t.closest('[data-run]');
  if (run) {
    const id = run.getAttribute('data-run');
    const panel = run.closest('[data-challenge]');
    const code = panel.querySelector('[data-code]').value;
    const status = panel.querySelector('[data-status]');
    const out = panel.querySelector('[data-out]');
    status.className = 'run-status run';
    status.textContent = 'running go test…';
    run.disabled = true;
    try {
      const r = await post(`/api/checker/${id}/run`, { code });
      out.hidden = false;
      out.textContent = r.output || (r.passed ? 'ok' : 'no output');
      out.className = 'checker-out ' + (r.passed ? 'pass' : 'fail');
      status.className = 'run-status ' + (r.passed ? 'pass' : 'fail');
      status.textContent = r.passed ? `✓ all checks passed (${r.ms}ms)` : (r.timedOut ? '⏱ timed out' : '✗ checks failed');
    } catch (err) {
      status.className = 'run-status fail';
      status.textContent = 'error running checker';
      console.error(err);
    } finally {
      run.disabled = false;
    }
    return;
  }

  // ask Claude for a review
  const review = t.closest('[data-review]');
  if (review) {
    const id = review.getAttribute('data-review');
    const panel = review.closest('[data-challenge]');
    const code = panel.querySelector('[data-code]').value;
    const box = panel.querySelector('[data-review-out]');
    box.hidden = false;
    box.innerHTML = '<p>Asking Claude for a review…</p>';
    review.disabled = true;
    try {
      const r = await post(`/api/checker/${id}/review`, { code });
      box.innerHTML = renderMarkdown(r.review || 'No review returned.');
    } catch (err) {
      box.innerHTML = '<p>Review failed.</p>';
      console.error(err);
    } finally {
      review.disabled = false;
    }
    return;
  }

  // interview: reveal model answer (server renders it, then reload)
  const ivReveal = t.closest('[data-iq-reveal]');
  if (ivReveal) {
    const slug = ivReveal.getAttribute('data-iq-reveal');
    try { await post(`/api/interview/${slug}/reveal`); location.reload(); }
    catch (err) { console.error(err); }
    return;
  }

  // interview: self-assessment (got it / needs review)
  const ivOutcome = t.closest('[data-iq-outcome]');
  if (ivOutcome) {
    const slug = ivOutcome.getAttribute('data-iq-slug');
    const cur = ivOutcome.getAttribute('data-iq-outcome');
    const card = ivOutcome.closest('[data-iq]');
    const already = (cur === 'got_it' && ivOutcome.classList.contains('on-got')) ||
                    (cur === 'review' && ivOutcome.classList.contains('on-review'));
    const next = already ? 'unset' : cur;
    try {
      await post(`/api/interview/${slug}/outcome`, { outcome: next });
      card.querySelectorAll('[data-iq-outcome]').forEach((b) => b.classList.remove('on-got', 'on-review'));
      if (next === 'got_it') ivOutcome.classList.add('on-got');
      if (next === 'review') ivOutcome.classList.add('on-review');
      flash();
    } catch (err) { console.error(err); }
    return;
  }

  // reset all progress
  if (t.id === 'reset') {
    if (confirm('Reset ALL progress? This clears every commit, note and quiz result.')) {
      try { await post('/api/reset'); location.reload(); } catch (err) { console.error(err); }
    }
    return;
  }
});

document.addEventListener('change', async (e) => {
  const t = e.target;

  // task status select
  if (t.matches('[data-status]')) {
    const id = t.getAttribute('data-status');
    try { await post(`/api/task/${id}/status`, { status: t.value }); flash(); location.reload(); }
    catch (err) { console.error(err); }
    return;
  }

  // code quiz self-grade checkbox
  if (t.matches('[data-self]')) {
    const id = t.getAttribute('data-self');
    try { await post(`/api/quiz/${id}/self`, { done: t.checked }); flash(); }
    catch (err) { console.error(err); }
    return;
  }
});

// notes autosave (debounced)
const noteTimers = {};
document.addEventListener('input', (e) => {
  const t = e.target;
  if (t.matches('[data-note]')) {
    const id = t.getAttribute('data-note');
    clearTimeout(noteTimers[id]);
    noteTimers[id] = setTimeout(async () => {
      try { await post(`/api/task/${id}/notes`, { notes: t.value }); flash(); }
      catch (err) { console.error(err); }
    }, 500);
  }
  if (t.id === 'resSearch') filterResources();
});

// ----- resources client-side filter -----
function filterResources() {
  const list = document.getElementById('resList');
  if (!list) return;
  const activeChip = document.querySelector('[data-rs].on');
  const step = activeChip ? activeChip.getAttribute('data-rs') : 'all';
  const freeOnly = document.querySelector('[data-free]')?.classList.contains('on');
  const q = (document.getElementById('resSearch')?.value || '').toLowerCase();

  let shown = 0;
  list.querySelectorAll('.res[data-step]').forEach((row) => {
    const matchStep = step === 'all' || row.getAttribute('data-step') === step;
    const matchFree = !freeOnly || row.getAttribute('data-free') === 'true';
    const matchText = !q || row.getAttribute('data-text').toLowerCase().includes(q);
    const visible = matchStep && matchFree && matchText;
    row.style.display = visible ? '' : 'none';
    if (visible) shown++;
  });
  const empty = document.getElementById('resEmpty');
  if (empty) empty.style.display = shown === 0 ? '' : 'none';
}

// Tab key inserts a tab in code editors instead of leaving the field.
document.addEventListener('keydown', (e) => {
  if (e.key === 'Tab' && e.target.matches('[data-code]')) {
    e.preventDefault();
    const ta = e.target;
    const s = ta.selectionStart, en = ta.selectionEnd;
    ta.value = ta.value.slice(0, s) + '\t' + ta.value.slice(en);
    ta.selectionStart = ta.selectionEnd = s + 1;
  }
});

// Minimal, safe markdown → HTML for Claude reviews (escape first, then format).
function renderMarkdown(md) {
  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  const lines = esc(md).split('\n');
  let html = '', inList = false, inCode = false;
  for (let line of lines) {
    if (line.trim().startsWith('```')) {
      if (inCode) { html += '</pre>'; inCode = false; }
      else { html += '<pre class="checker-out">'; inCode = true; }
      continue;
    }
    if (inCode) { html += line + '\n'; continue; }
    line = line.replace(/\*\*(.+?)\*\*/g, '<b>$1</b>').replace(/`(.+?)`/g, '<code>$1</code>');
    const h = line.match(/^(#{1,4})\s+(.*)/);
    if (h) { if (inList) { html += '</ul>'; inList = false; } html += `<h3>${h[2]}</h3>`; continue; }
    const li = line.match(/^[-*]\s+(.*)/);
    if (li) { if (!inList) { html += '<ul>'; inList = true; } html += `<li>${li[1]}</li>`; continue; }
    if (inList) { html += '</ul>'; inList = false; }
    if (line.trim()) html += `<p>${line}</p>`;
  }
  if (inList) html += '</ul>';
  if (inCode) html += '</pre>';
  return html;
}

const API = '/api/v1';
let pollTimer = null;
let activeScanId = null;

const $ = (sel) => document.querySelector(sel);

async function fetchJSON(url, opts = {}) {
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

function formatDate(iso) {
  if (!iso) return '—';
  return new Date(iso).toLocaleString();
}

function statusClass(status) {
  return `status status-${status}`;
}

function renderProfiles(profiles) {
  const el = $('#profiles-list');
  if (!profiles.length) {
    el.innerHTML = '<p class="empty">No scan profiles configured. Add profiles to your YAML config and start the server with --config.</p>';
    return;
  }
  el.innerHTML = profiles.map((p) => `
    <div class="profile-card">
      <h3>${escapeHTML(p.name)}</h3>
      <p>Provider: <strong>${escapeHTML(p.provider)}</strong></p>
      <p>${escapeHTML(p.state_source)}</p>
      ${p.schedule ? `<p class="schedule">⏱ ${escapeHTML(p.schedule)}</p>` : ''}
      <button class="btn btn-primary btn-sm" style="margin-top:0.75rem" onclick="runProfile('${escapeHTML(p.name)}')">Run Now</button>
    </div>
  `).join('');
}

function renderScans(scans) {
  const tbody = $('#scans-tbody');
  $('#scan-count').textContent = `${scans.length} scan${scans.length !== 1 ? 's' : ''}`;

  if (!scans.length) {
    tbody.innerHTML = '<tr><td colspan="7" class="empty">No scans yet. Trigger one from a profile or click New Scan.</td></tr>';
    return;
  }

  tbody.innerHTML = scans.map((s) => {
    const driftClass = s.total_drifts > 0 ? 'has-drift' : 'no-drift';
    return `
      <tr>
        <td><span class="${statusClass(s.status)}">${s.status}</span></td>
        <td>${escapeHTML(s.provider)}</td>
        <td title="${escapeHTML(s.state_source)}">${truncate(s.state_source, 40)}</td>
        <td>${escapeHTML(s.profile_name || '—')}</td>
        <td><span class="drift-count ${driftClass}">${s.status === 'completed' ? s.total_drifts : '—'}</span></td>
        <td>${formatDate(s.created_at)}</td>
        <td><button class="link-btn" onclick="viewScan('${s.id}')">View</button></td>
      </tr>
    `;
  }).join('');

  const inProgress = scans.some((s) => s.status === 'pending' || s.status === 'running');
  if (inProgress) startPolling();
  else stopPolling();
}

async function loadProfiles() {
  try {
    const profiles = await fetchJSON(`${API}/profiles`);
    renderProfiles(profiles);
  } catch (e) {
    $('#profiles-list').innerHTML = `<p class="empty">Failed to load profiles: ${escapeHTML(e.message)}</p>`;
  }
}

async function loadScans() {
  try {
    const scans = await fetchJSON(`${API}/scans?summary=true&limit=50`);
    renderScans(scans);
  } catch (e) {
    $('#scans-tbody').innerHTML = `<tr><td colspan="7" class="empty">Failed to load scans: ${escapeHTML(e.message)}</td></tr>`;
  }
}

async function viewScan(id) {
  activeScanId = id;
  $('#detail-panel').classList.remove('hidden');
  await refreshDetail(id);
}

async function refreshDetail(id) {
  try {
    const scan = await fetchJSON(`${API}/scans/${id}`);
    renderDetail(scan);
    if (scan.status === 'pending' || scan.status === 'running') {
      startPolling();
    }
  } catch (e) {
    $('#scan-meta').innerHTML = `<p class="empty">${escapeHTML(e.message)}</p>`;
  }
}

function renderDetail(scan) {
  $('#scan-meta').innerHTML = `
    <dl><dt>Scan ID</dt><dd>${escapeHTML(scan.id)}</dd></dl>
    <dl><dt>Status</dt><dd><span class="${statusClass(scan.status)}">${scan.status}</span></dd></dl>
    <dl><dt>Provider</dt><dd>${escapeHTML(scan.provider)}</dd></dl>
    <dl><dt>State</dt><dd>${escapeHTML(scan.state_source)}</dd></dl>
    <dl><dt>Created</dt><dd>${formatDate(scan.created_at)}</dd></dl>
    <dl><dt>Completed</dt><dd>${formatDate(scan.completed_at)}</dd></dl>
  `;

  if (scan.error) {
    $('#scan-meta').innerHTML += `<dl><dt>Error</dt><dd style="color:var(--danger)">${escapeHTML(scan.error)}</dd></dl>`;
  }

  const summaryEl = $('#summary-cards');
  const driftEl = $('#drift-list');

  if (!scan.report) {
    summaryEl.innerHTML = '';
    driftEl.innerHTML = `<p class="empty">${scan.status === 'running' || scan.status === 'pending' ? 'Scan in progress…' : 'No report available.'}</p>`;
    return;
  }

  const s = scan.report.summary;
  summaryEl.innerHTML = `
    <div class="summary-card"><div class="value">${s.total_drifts}</div><div class="label">Total Drifts</div></div>
    <div class="summary-card"><div class="value">${s.missing_in_cloud}</div><div class="label">Missing in Cloud</div></div>
    <div class="summary-card"><div class="value">${s.missing_in_state}</div><div class="label">Missing in State</div></div>
    <div class="summary-card"><div class="value">${s.attribute_drifts}</div><div class="label">Attribute</div></div>
    <div class="summary-card"><div class="value">${s.tag_drifts}</div><div class="label">Tag</div></div>
  `;

  if (!scan.report.drifts?.length) {
    driftEl.innerHTML = '<p class="empty">No drift detected. Infrastructure matches Terraform state.</p>';
    return;
  }

  driftEl.innerHTML = scan.report.drifts.map((d) => {
    const changes = [];
    (d.attribute_changes || []).forEach((c) => {
      changes.push(`<li><strong>${escapeHTML(c.attribute)}</strong>: expected ${escapeHTML(String(c.expected))} → actual ${escapeHTML(String(c.actual))}</li>`);
    });
    (d.tag_changes || []).forEach((t) => {
      changes.push(`<li>Tag <strong>${escapeHTML(t.key)}</strong> (${t.change}): ${escapeHTML(t.expected || '')} → ${escapeHTML(t.actual || '')}</li>`);
    });
    return `
      <div class="drift-item">
        <div class="drift-type">${escapeHTML(d.type)}</div>
        <h4>${escapeHTML(d.resource_type)} — ${escapeHTML(d.resource_id)}</h4>
        <p>${escapeHTML(d.message || '')}</p>
        ${changes.length ? `<ul class="drift-changes">${changes.join('')}</ul>` : ''}
      </div>
    `;
  }).join('');
}

async function runProfile(name) {
  try {
    await fetchJSON(`${API}/scans/profile/${encodeURIComponent(name)}`, { method: 'POST' });
    await loadScans();
  } catch (e) {
    alert(`Failed to start scan: ${e.message}`);
  }
}

function startPolling() {
  if (pollTimer) return;
  pollTimer = setInterval(async () => {
    await loadScans();
    if (activeScanId) await refreshDetail(activeScanId);
  }, 3000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

function escapeHTML(str) {
  if (!str) return '';
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function truncate(str, len) {
  if (!str) return '';
  return str.length > len ? str.slice(0, len) + '…' : str;
}

$('#refresh-btn').addEventListener('click', () => { loadScans(); loadProfiles(); });
$('#new-scan-btn').addEventListener('click', () => $('#scan-dialog').showModal());
$('#cancel-scan').addEventListener('click', () => $('#scan-dialog').close());
$('#close-detail').addEventListener('click', () => {
  $('#detail-panel').classList.add('hidden');
  activeScanId = null;
});

$('#scan-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const body = {
    provider: fd.get('provider'),
    state_path: fd.get('state_path'),
    regions: fd.get('regions') ? String(fd.get('regions')).split(',').map((s) => s.trim()).filter(Boolean) : [],
    resource_types: fd.get('resource_types') ? String(fd.get('resource_types')).split(',').map((s) => s.trim()).filter(Boolean) : [],
  };
  try {
    await fetchJSON(`${API}/scans`, { method: 'POST', body: JSON.stringify(body) });
    $('#scan-dialog').close();
    e.target.reset();
    await loadScans();
  } catch (err) {
    alert(`Failed to start scan: ${err.message}`);
  }
});

loadProfiles();
loadScans();

let devices = [];
let selected = null;
let currentFilter = 'all';
let offlineAfterMs = 10 * 60 * 1000;
let pairingActive = false;
let map;
let marker;
let historyMap;
let route;
let playMarker;
let history = [];

const $ = (selector) => document.querySelector(selector);
const api = async (url, options = {}) => {
  const response = await fetch(url, options);
  if (!response.ok) throw new Error(await response.text());
  return response.json();
};

function initMap(id) {
  const value = L.map(id).setView([47.5, 19.05], 13);
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap contributors',
  }).addTo(value);
  return value;
}

map = initMap('map');
historyMap = initMap('historyMap');

function isOnline(device) {
  return Boolean(device.last_seen) && Date.now() - new Date(device.last_seen).getTime() < offlineAfterMs;
}

function trackingStatus(device) {
  if (!isOnline(device)) return 'Offline';
  if (device.privacy) return 'Privacy';
  if (!device.last_position?.valid) return 'No position';
  return (device.last_position.speed_kmh || 0) > 1 ? 'Moving' : 'Stopped';
}

function direction(degrees) {
  const value = Number(degrees || 0);
  const names = ['N', 'NE', 'E', 'SE', 'S', 'SW', 'W', 'NW'];
  return `${Math.round(value)}° ${names[Math.round(value / 45) % 8]}`;
}

document.querySelectorAll('nav button').forEach((button) => {
  button.onclick = () => {
    document.querySelectorAll('nav button,.tab').forEach((element) => element.classList.remove('active'));
    button.classList.add('active');
    $('#' + button.dataset.tab).classList.add('active');
    setTimeout(() => {
      map.invalidateSize();
      historyMap.invalidateSize();
    }, 50);
    if (button.dataset.tab === 'alarms') loadAlarms();
  };
});

document.querySelectorAll('.filters button').forEach((button) => {
  button.onclick = () => {
    currentFilter = button.dataset.filter;
    document.querySelectorAll('.filters button').forEach((item) => item.classList.toggle('active', item === button));
    renderDevices();
  };
});

$('#pair').onclick = async () => {
  await api(pairingActive ? 'api/pairing/stop' : 'api/pairing/start', { method: 'POST' });
  await refreshStatus();
};
$('#loadHistory').onclick = loadHistory;
$('#loadStats').onclick = loadStats;
$('#playback').oninput = (event) => showPlayback(+event.target.value);

async function refreshStatus() {
  try {
    const status = await api('api/status');
    const until = status.pairing_until ? new Date(status.pairing_until) : null;
    pairingActive = Boolean(until && until.getTime() > Date.now());
    offlineAfterMs = Math.max(1000, Number(status.offline_after_seconds || 600) * 1000);
    $('#status').textContent = `v${status.version} · ${status.learned_cells} learned cells${pairingActive ? ' · pairing open until ' + until.toLocaleTimeString() : ''}`;
    $('#pair').textContent = pairingActive ? 'Stop pairing' : 'Pair new tracker (10 min)';
  } catch (error) {
    $('#status').textContent = 'Gateway unavailable';
  }
}

async function refresh() {
  try {
    devices = await api('api/devices');
    renderDevices();
    if (selected) {
      const device = devices.find((item) => item.key === selected);
      if (device) showDevice(device);
    }
  } catch (error) {
    // Status line handles transient App/API failures.
  }
}

function renderDevices() {
  const active = devices.filter(isOnline).length;
  $('#countAll').textContent = devices.length;
  $('#countActive').textContent = active;
  $('#countInactive').textContent = devices.length - active;

  const visible = devices.filter((device) => {
    if (currentFilter === 'active') return isOnline(device);
    if (currentFilter === 'inactive') return !isOnline(device);
    return true;
  });

  $('#devices').innerHTML = visible.map((device) => {
    const online = isOnline(device);
    const status = trackingStatus(device);
    const battery = Number.isFinite(device.battery) && device.battery > 0 ? `${Math.round(device.battery)}%` : '—';
    return `<div class="device ${device.key === selected ? 'active' : ''}" data-key="${device.key}">
      <div class="row"><strong>${esc(device.name)}</strong><button class="edit" data-edit="${device.key}" title="Tracker settings">⋯</button></div>
      <small><span class="dot ${online ? 'on' : ''}"></span>${esc(status)} · ${esc(device.protocol || 'unknown')} · ${battery}${device.privacy ? ' <span class="privacy">· privacy</span>' : ''}</small>
    </div>`;
  }).join('');

  document.querySelectorAll('.device').forEach((element) => {
    element.onclick = (event) => {
      if (event.target.dataset.edit) return;
      selected = element.dataset.key;
      renderDevices();
      showDevice(devices.find((device) => device.key === selected));
    };
  });
  document.querySelectorAll('.edit').forEach((element) => {
    element.onclick = (event) => {
      event.stopPropagation();
      editDevice(devices.find((device) => device.key === element.dataset.edit));
    };
  });
}

function showDevice(device) {
  if (!device?.last_position || device.privacy) {
    $('#empty').style.display = 'block';
    return;
  }
  const position = device.last_position;
  if (!position.valid) {
    $('#empty').style.display = 'block';
    return;
  }

  $('#empty').style.display = 'none';
  const latlng = [position.latitude, position.longitude];
  if (!marker) marker = L.marker(latlng).addTo(map);
  else marker.setLatLng(latlng);

  const accuracy = position.accuracy ? `<br>Accuracy ±${Math.round(position.accuracy)} m` : '';
  marker.bindPopup(
    `<b>${esc(device.name)}</b><br>${esc(trackingStatus(device))}` +
    `<br>Position: ${esc(position.source)}` + accuracy +
    `<br>Speed: ${(position.speed_kmh || 0).toFixed(1)} km/h` +
    `<br>Direction: ${direction(position.heading)}` +
    `<br>Battery: ${Math.round(device.battery || 0)}%` +
    `<br>Last update: ${new Date(position.received_at).toLocaleString()}`
  ).openPopup();
  map.setView(latlng, Math.max(map.getZoom(), 15));
}

function editDevice(device) {
  if (!device) return;
  selected = device.key;
  $('#fName').value = device.name || '';
  $('#fModel').value = device.model || '';
  $('#fSim').value = device.sim || '';
  $('#fIccid').value = device.iccid || '';
  $('#fPlate').value = device.plate || '';
  $('#fIcon').value = device.icon || '';
  $('#fOverspeed').value = device.overspeed_kmh || '';
  $('#fFilterLBS').checked = Boolean(device.filter_lbs);
  $('#fPrivacy').checked = Boolean(device.privacy);
  $('#editDialog').showModal();
}

$('#saveDevice').onclick = async (event) => {
  event.preventDefault();
  const overspeed = Number($('#fOverspeed').value || 0);
  await api(`api/devices/${selected}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: $('#fName').value,
      model: $('#fModel').value,
      sim: $('#fSim').value,
      iccid: $('#fIccid').value,
      plate: $('#fPlate').value,
      icon: $('#fIcon').value,
      overspeed_kmh: Number.isFinite(overspeed) ? Math.max(0, Math.min(400, overspeed)) : 0,
      filter_lbs: $('#fFilterLBS').checked,
      privacy: $('#fPrivacy').checked,
    }),
  });
  $('#editDialog').close();
  await refresh();
};

async function loadHistory() {
  if (!selected) return;
  history = await api(`api/devices/${selected}/positions?hours=${$('#historyPeriod').value}`);
  if (route) historyMap.removeLayer(route);
  if (playMarker) historyMap.removeLayer(playMarker);
  route = null;
  playMarker = null;
  const points = history.filter((position) => position.valid).map((position) => [position.latitude, position.longitude]);
  if (!points.length) return;
  route = L.polyline(points, { weight: 4 }).addTo(historyMap);
  historyMap.fitBounds(route.getBounds(), { padding: [20, 20] });
  $('#playback').max = points.length - 1;
  $('#playback').value = points.length - 1;
  showPlayback(points.length - 1);
}

function showPlayback(index) {
  const position = history.filter((item) => item.valid)[index];
  if (!position) return;
  const latlng = [position.latitude, position.longitude];
  if (!playMarker) playMarker = L.marker(latlng).addTo(historyMap);
  else playMarker.setLatLng(latlng);
  playMarker.bindPopup(
    `${new Date(position.received_at).toLocaleString()} · ${esc(position.source)} · ${(position.speed_kmh || 0).toFixed(1)} km/h`
  ).openPopup();
}

async function loadStats() {
  if (!selected) return;
  const stats = await api(`api/devices/${selected}/stats?hours=${$('#statsPeriod').value}`);
  $('#statsCards').innerHTML = `
    <div class="stat"><b>${stats.distance_km.toFixed(1)} km</b><span>Distance</span></div>
    <div class="stat"><b>${stats.max_speed_kmh.toFixed(1)} km/h</b><span>Maximum speed</span></div>
    <div class="stat"><b>${stats.points}</b><span>Position points</span></div>`;
}

async function loadAlarms() {
  if (!selected) {
    $('#alarmList').innerHTML = 'Select a tracker first.';
    return;
  }
  const alarms = await api(`api/devices/${selected}/alarms?hours=4320`);
  $('#alarmList').innerHTML = alarms.length
    ? alarms.reverse().map((alarm) => `<div class="alarm"><b>${esc(alarm.alarm)}</b><br><small>${new Date(alarm.received_at).toLocaleString()} · ${esc(alarm.source)}</small></div>`).join('')
    : 'No alarms in retained history.';
}

function esc(value) {
  return String(value ?? '').replace(/[&<>\"]/g, (character) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '\"': '&quot;',
  }[character]));
}

refreshStatus().then(refresh);
setInterval(() => {
  refreshStatus();
  refresh();
}, 5000);

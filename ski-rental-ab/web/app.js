'use strict';
const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];
const field = (form, name) => form.elements.namedItem(name);
const money = cents => new Intl.NumberFormat('de-AT', {style: 'currency', currency: 'EUR'}).format(cents / 100);
const day = value => new Intl.DateTimeFormat('de-AT', {day: '2-digit', month: '2-digit', year: 'numeric'}).format(new Date(value + 'T12:00:00'));
const personName = p => p ? `${p.firstName} ${p.lastName}` : 'Unbekannt';
const statusNames = {reserved: 'Reserviert', active: 'Ausgeliehen', returned: 'Zurückgegeben', cancelled: 'Storniert'};
let state, config, quoteVersion = 0, currentQuote = null, previewURL = null;
function node(tag, className, text) {
 const n = document.createElement(tag); if (className) n.className = className;
 if (text !== undefined) n.textContent = String(text); return n;
}
function action(label, name, id, className = 'small-button') {
 const b = node('button', className, label); b.type = 'button'; b.dataset.action = name;
 if (id !== undefined) b.dataset.id = String(id); return b;
}
function option(value, text) {const o = node('option', '', text); o.value = String(value); return o;}
function cell(content) {const td = node('td'); td.append(content instanceof Node ? content : document.createTextNode(String(content))); return td;}
function nameCell(title, subtitle) {const d = node('div'); d.append(node('strong', 'row-title', title), node('small', 'muted row-subtitle', subtitle || '')); return d;}
function emptyRow(body, columns, message) {const tr = node('tr'), td = node('td', 'empty-state', message); td.colSpan = columns; tr.append(td); body.append(tr);}
async function api(path, options = {}) {
 const opts = {...options}; if (opts.body && !(opts.body instanceof FormData)) opts.headers = {'Content-Type': 'application/json', ...opts.headers};
 let response; try {response = await fetch(path, opts);} catch {throw new Error('Server nicht erreichbar. Bitte prüfen Sie, ob der Go-Server läuft.');}
 if (response.status === 204) return null;
 const data = await response.json().catch(() => null);
 if (!response.ok) {const error = new Error(data?.error || `Anfrage fehlgeschlagen (${response.status}).`); error.status = response.status; throw error;}
 return data;
}
function notice(message, error = false) {const n = $('#notice'); n.textContent = message; n.className = error ? 'notice error' : 'notice'; n.hidden = false;}
function formError(form, message) {const e = $('.form-error', form); e.textContent = message; e.hidden = !message;}
function showDialog(id) {const d = $(id); formError($('form', d), ''); d.showModal();}
function closeDialog(dialog) {dialog.close();}
async function refresh() {state = await api('/api/state'); render();}
function showTab(tab) {
 const valid = ['dashboard', 'items', 'people', 'rentals', 'prices']; if (!valid.includes(tab)) tab = 'dashboard';
 const labels = {dashboard: ['Dashboard', 'Alle wichtigen Zahlen. Alle nächsten Schritte.'], items: ['Ausrüstung', 'Bestand pflegen und heutige Verfügbarkeit prüfen.'], people: ['Personen', 'Kontaktdaten und Ausweise an einem Ort.'], rentals: ['Reservierungen & Ausleihen', 'Vom geplanten Termin bis zur Rückgabe.'], prices: ['Preise', 'Einfache Tagespreise. Flexible Mietdauer.']};
 $$('.view').forEach(v => {v.hidden = v.id !== tab;});
 $$('.nav').forEach(b => {b.classList.toggle('active', b.dataset.tab === tab); b.setAttribute('aria-current', b.dataset.tab === tab ? 'page' : 'false');});
 $('#page-title').textContent = labels[tab][0]; $('#page-subtitle').textContent = labels[tab][1];
 if (location.hash !== '#' + tab) history.replaceState(null, '', '#' + tab);
}
function blocks(r, start, end) {if (!['reserved', 'active'].includes(r.status)) return false; const until = r.status === 'active' && r.endDate < state.today ? state.today : r.endDate; return r.startDate <= end && start <= until;}
function availability(id, start, end) {return !state.rentals.some(r => r.itemId === id && blocks(r, start, end));}
function render() {
 $('#brand-title').textContent = config.title; document.title = config.title + ' · Rental Desk';
 $('#today').textContent = day(state.today); $('#size-heading').textContent = config.sizeLabel; $('#size-label').textContent = config.sizeLabel;
 $('#available-label').textContent = `${config.itemLabel} ohne heutige Belegung`;
 for (const key of ['available', 'rented', 'total', 'people']) $('#metric-' + key).textContent = state.dashboard[key];
 $('#reserved-today').textContent = state.dashboard.reservedToday; $('#overdue').textContent = state.dashboard.overdue;
 renderItems(); renderPeople(); renderRentals(); renderPrices();
}
function renderItems() {
 const body = $('#item-rows'); body.replaceChildren(); const query = $('#item-search').value.toLocaleLowerCase();
 state.items.filter(i => `${i.name} ${i.category} ${i.size}`.toLocaleLowerCase().includes(query)).forEach(i => {
  const tr = node('tr'), active = state.rentals.some(r => r.itemId === i.id && r.status === 'active'), free = availability(i.id, state.today, state.today);
  const badge = node('span', `badge ${active ? 'active' : free ? 'available' : 'reserved'}`, active ? 'Ausgeliehen' : free ? 'Verfügbar' : 'Reserviert');
  const controls = node('div', 'row-actions'); controls.append(action('Bearbeiten', 'edit-item', i.id), action('Löschen', 'delete-item', i.id, 'small-button danger-text'));
  tr.append(cell(nameCell(i.name, i.details)), cell(i.category), cell(i.size || '—'), cell(badge), cell(controls)); body.append(tr);
 }); if (!body.children.length) emptyRow(body, 5, 'Keine Ausrüstung gefunden. Fügen Sie einen Eintrag hinzu.');
}
function renderPeople() {
 const body = $('#person-rows'); body.replaceChildren(); const query = $('#person-search').value.toLocaleLowerCase();
 state.people.filter(p => `${personName(p)} ${p.email}`.toLocaleLowerCase().includes(query)).forEach(p => {
  const tr = node('tr'), docs = node('div'); docs.append(node('span', 'badge ' + (p.hasDocument ? 'available' : 'neutral'), p.hasDocument ? 'Bild vorhanden' : 'Kein Bild'));
  docs.append(node('small', 'muted row-subtitle', [p.documentType, p.documentNumber].filter(Boolean).join(' · ') || 'Keine Ausweisdaten'));
  const controls = node('div', 'row-actions'); controls.append(action('Bearbeiten', 'edit-person', p.id), action('Ausweisbild', 'document', p.id), action('Löschen', 'delete-person', p.id, 'small-button danger-text'));
  tr.append(cell(nameCell(personName(p), `Person #${p.id}`)), cell(nameCell(p.email || '—', p.phone)), cell(docs), cell(controls)); body.append(tr);
 }); if (!body.children.length) emptyRow(body, 4, 'Keine Personen gefunden. Fügen Sie eine Person hinzu.');
}
function rentalRow(r) {
 const tr = node('tr'), i = state.items.find(i => i.id === r.itemId), p = state.people.find(p => p.id === r.personId), late = r.endDate < state.today && ['active', 'reserved'].includes(r.status);
 const status = node('div'); status.append(node('span', 'badge ' + r.status, statusNames[r.status])); if (late) status.append(node('small', 'late row-subtitle', r.status === 'active' ? 'Rückgabe überfällig' : 'Termin abgelaufen'));
 const controls = node('div', 'row-actions');
 if (r.status === 'reserved') {if (r.startDate <= state.today && r.endDate >= state.today) controls.append(action('Ausgeben', 'checkout', r.id)); controls.append(action('Stornieren', 'cancel', r.id, 'small-button danger-text'));}
 if (r.status === 'active') controls.append(action('Zurücknehmen', 'return', r.id));
 if (!controls.children.length) controls.append(node('span', 'muted', 'Abgeschlossen'));
 const cost = nameCell(money(r.quote.totalCents), `${r.quote.days} Tag(e) × ${money(r.quote.dailyCents)} · ${r.quote.rule}`);
 tr.append(cell('#' + String(r.id).padStart(3, '0')), cell(nameCell(i?.name || '—', personName(p))), cell(nameCell(day(r.startDate) + ' –', day(r.endDate))), cell(status), cell(cost), cell(controls)); return tr;
}
function renderRentals() {
 const body = $('#rental-rows'), dashboard = $('#dashboard-rows'); body.replaceChildren(); dashboard.replaceChildren();
 const filter = $('#rental-filter').value, sorted = [...state.rentals].reverse();
 sorted.filter(r => filter === 'all' || r.status === filter).forEach(r => body.append(rentalRow(r)));
 sorted.filter(r => ['reserved', 'active'].includes(r.status)).slice(0, 6).forEach(r => dashboard.append(rentalRow(r)));
 if (!body.children.length) emptyRow(body, 6, 'Keine Vorgänge in dieser Auswahl.'); if (!dashboard.children.length) emptyRow(dashboard, 6, 'Alles erledigt. Keine offenen Vorgänge.');
}
function renderPrices() {
 const cards = $('#price-cards'); cards.replaceChildren();
 state.categories.forEach(category => {
  const rate = state.rates.find(r => r.category === category), card = node('article', 'panel price-card'); card.append(node('p', 'eyebrow', 'TARIF'), node('h2', '', category));
  if (rate) {card.append(node('strong', 'price', money(rate.dailyCents)), node('p', 'muted', 'pro Kalendertag'));
   const list = node('div', 'tier-list'); if (!rate.tiers.length) list.append(node('p', 'muted', 'Keine Preisstaffeln.'));
   rate.tiers.forEach(t => {const row = node('div', 'tier-summary'); row.append(node('span', '', `Ab ${t.minDays} Tagen`), node('strong', '', `${money(t.dailyCents)} / Tag`)); list.append(row);}); card.append(list);
  } else card.append(node('p', 'muted', 'Noch kein Tarif hinterlegt.'));
  card.append(action('Tarif bearbeiten', 'edit-price', category, 'button secondary')); cards.append(card);
 });
}
function editForm(selector, values) {const f = $(selector); f.reset(); for (const [key, value] of Object.entries(values)) {const e = field(f, key); if (e) e.value = value;} return f;}
function openItem(id) {
 const item = state.items.find(i => i.id === id), f = $('#item-form'); field(f, 'category').replaceChildren(...state.categories.map(c => option(c, c)));
 editForm('#item-form', item || {id: ''}); $('#item-dialog-title').textContent = item ? 'Ausrüstung bearbeiten' : 'Ausrüstung hinzufügen'; showDialog('#item-dialog');
}
function openPerson(id) {const p = state.people.find(p => p.id === id); editForm('#person-form', p || {id: ''}); $('#person-dialog-title').textContent = p ? 'Person bearbeiten' : 'Person hinzufügen'; showDialog('#person-dialog');}
function rentalItems() {
 const f = $('#rental-form'), select = field(f, 'itemId'), selected = select.value, start = field(f, 'startDate').value, end = field(f, 'endDate').value;
 select.replaceChildren(option('', 'Bitte Ausrüstung wählen'));
 state.items.forEach(i => {const available = start && end && start <= end && availability(i.id, start, end); const o = option(i.id, `${i.name} · ${i.category} · ${i.size}${available ? '' : ' — belegt / Zeitraum prüfen'}`); o.disabled = !available; select.append(o);});
 const old = [...select.options].find(o => o.value === selected && !o.disabled); select.value = old ? selected : '';
}
async function updateQuote() {
 const version = ++quoteVersion, f = $('#rental-form'); currentQuote = null; $('#reserve-submit').disabled = true; formError(f, '');
 const itemId = field(f, 'itemId').value, startDate = field(f, 'startDate').value, endDate = field(f, 'endDate').value;
 if (!itemId || !startDate || !endDate) {$('#quote').textContent = 'Bitte Ausrüstung und gültigen Zeitraum wählen.'; return;}
 $('#quote').textContent = 'Preis wird berechnet …';
 try {const q = await api('/api/quote?' + new URLSearchParams({itemId, startDate, endDate})); if (version !== quoteVersion) return;
  currentQuote = q; $('#quote').replaceChildren(node('span', 'muted', `${q.days} Tag(e) × ${money(q.dailyCents)} · ${q.rule}`), node('strong', '', money(q.totalCents))); $('#reserve-submit').disabled = false;
 } catch (e) {if (version === quoteVersion) $('#quote').textContent = e.message;}
}
function openRental() {
 if (!state.items.length || !state.people.length) {notice('Bitte zuerst Ausrüstung und mindestens eine Person anlegen.', true); return;}
 const f = editForm('#rental-form', {startDate: state.today, endDate: state.today});
 field(f, 'personId').replaceChildren(option('', 'Bitte Person wählen'), ...state.people.map(p => option(p.id, personName(p))));
 field(f, 'startDate').min = state.today; field(f, 'endDate').min = state.today; rentalItems(); showDialog('#rental-dialog'); updateQuote();
}
function cents(value) {const s = value.trim().replace(',', '.'); if (!/^\d+(\.\d{1,2})?$/.test(s)) throw new Error('Preis bitte mit maximal zwei Nachkommastellen eingeben.'); const parts = s.split('.'), amount = Number(parts[0]) * 100 + Number((parts[1] || '').padEnd(2, '0')); if (!Number.isSafeInteger(amount) || amount < 1 || amount > 100000000) throw new Error('Tagespreis muss zwischen 0,01 und 1.000.000 EUR liegen.'); return amount;}
function addTier(tier) {
 const row = node('div', 'tier-row'); const minimum = node('label', '', 'Ab Tagen'), price = node('label', '', 'EUR / Tag');
 const m = node('input'); m.name = 'minDays'; m.type = 'number'; m.min = '2'; m.max = '3660000'; m.required = true; m.value = tier?.minDays || '';
 const p = node('input'); p.name = 'tierDaily'; p.inputMode = 'decimal'; p.required = true; p.value = tier ? (tier.dailyCents / 100).toFixed(2).replace('.', ',') : '';
 minimum.append(m); price.append(p); const remove = node('button', 'small-button danger-text', 'Entfernen'); remove.type = 'button'; remove.addEventListener('click', () => row.remove()); row.append(minimum, price, remove); $('#tier-rows').append(row);
}
function openPrice(category) {const rate = state.rates.find(r => r.category === category); editForm('#price-form', {category, daily: rate ? (rate.dailyCents / 100).toFixed(2).replace('.', ',') : ''}); $('#price-dialog-title').textContent = 'Tarif · ' + category; $('#tier-rows').replaceChildren(); (rate?.tiers || []).forEach(addTier); showDialog('#price-dialog');}
function revokePreview() {if (previewURL) {URL.revokeObjectURL(previewURL); previewURL = null;}}
function documentPreview(src) {const img = $('#document-preview'); img.hidden = !src; $('#document-empty').hidden = !!src; if (src) img.src = src; else img.removeAttribute('src');}
function openDocument(id) {
 const p = state.people.find(p => p.id === id); editForm('#document-form', {personId: id}); $('#document-person').textContent = personName(p); revokePreview();
 documentPreview(p.hasDocument ? `/api/people/${id}/document?v=${Date.now()}` : ''); $('#delete-document').disabled = !p.hasDocument; showDialog('#document-dialog');
}
async function submit(form, work) {
 const button = $('[type="submit"]', form); if (button.disabled) return; button.disabled = true; formError(form, '');
 try {await work();} catch (e) {formError(form, e.message);} finally {button.disabled = form.id === 'rental-form' ? !currentQuote : false;}
}
$('#item-form').addEventListener('submit', e => {e.preventDefault(); const f = e.currentTarget; submit(f, async () => {const values = Object.fromEntries(new FormData(f)), id = values.id; delete values.id; await api('/api/items' + (id ? '/' + id : ''), {method: id ? 'PUT' : 'POST', body: JSON.stringify(values)}); closeDialog($('#item-dialog')); await refresh(); notice('Ausrüstung gespeichert.');});});
$('#person-form').addEventListener('submit', e => {e.preventDefault(); const f = e.currentTarget; submit(f, async () => {const values = Object.fromEntries(new FormData(f)), id = values.id; delete values.id; await api('/api/people' + (id ? '/' + id : ''), {method: id ? 'PUT' : 'POST', body: JSON.stringify(values)}); closeDialog($('#person-dialog')); await refresh(); notice('Person gespeichert. Ausweisbild kann jetzt hinzugefügt werden.');});});
$('#rental-form').addEventListener('change', () => {rentalItems(); updateQuote();});
$('#rental-form').addEventListener('submit', e => {e.preventDefault(); const f = e.currentTarget; submit(f, async () => {const values = Object.fromEntries(new FormData(f)); values.itemId = Number(values.itemId); values.personId = Number(values.personId); values.expectedTotalCents = currentQuote.totalCents;
 try {await api('/api/rentals', {method: 'POST', body: JSON.stringify(values)});}
 catch (error) {if (error.status === 409) {await refresh(); rentalItems(); await updateQuote();} throw error;} closeDialog($('#rental-dialog')); await refresh(); notice('Reservierung gespeichert. Zur Ausgabe den Vorgang öffnen.');});});
$('#price-form').addEventListener('submit', e => {e.preventDefault(); const f = e.currentTarget; submit(f, async () => {const tiers = $$('.tier-row').map(row => ({minDays: Number($('[name="minDays"]', row).value), dailyCents: cents($('[name="tierDaily"]', row).value)})); await api('/api/rates', {method: 'PUT', body: JSON.stringify({category: field(f, 'category').value, dailyCents: cents(field(f, 'daily').value), tiers})}); closeDialog($('#price-dialog')); await refresh(); notice('Tarif gespeichert. Bestehende Vorgänge behalten ihren Preis.');});});
$('#add-tier').addEventListener('click', () => addTier());
$('#document-form [name="document"]').addEventListener('change', e => {
 const file = e.target.files[0]; formError($('#document-form'), ''); revokePreview(); if (!file) return;
 if (file.size > 5 * 1024 * 1024 || file.size === 0 || (file.type && !['image/jpeg', 'image/png'].includes(file.type))) {formError($('#document-form'), 'Bitte ein JPG/PNG zwischen 1 Byte und 5 MiB auswählen.'); e.target.value = ''; return;}
 previewURL = URL.createObjectURL(file); documentPreview(previewURL);
});
$('#document-form').addEventListener('submit', e => {e.preventDefault(); const f = e.currentTarget; submit(f, async () => {const file = field(f, 'document').files[0]; if (!file) throw new Error('Bitte zuerst eine Bilddatei auswählen.'); const data = new FormData(); data.append('document', file); await api('/api/people/' + field(f, 'personId').value + '/document', {method: 'POST', body: data}); closeDialog($('#document-dialog')); await refresh(); notice('Ausweisbild gespeichert.');});});
$('#delete-document').addEventListener('click', async () => {if (!confirm('Gespeichertes Ausweisbild löschen?')) return; try {await api('/api/people/' + field($('#document-form'), 'personId').value + '/document', {method: 'DELETE'}); closeDialog($('#document-dialog')); await refresh(); notice('Ausweisbild gelöscht.');} catch (e) {formError($('#document-form'), e.message);}});
$('#document-dialog').addEventListener('close', revokePreview);
$('#document-preview').addEventListener('error', () => {$('#document-preview').hidden = true; $('#document-empty').hidden = false; $('#document-empty').textContent = 'Bild kann nicht angezeigt werden. Bitte ein gültiges JPG/PNG wählen.';});
$('#item-search').addEventListener('input', renderItems); $('#person-search').addEventListener('input', renderPeople); $('#rental-filter').addEventListener('change', renderRentals);
$('#refresh').addEventListener('click', async () => {try {await refresh(); notice('Daten aktualisiert.');} catch (e) {notice(e.message, true);}});
window.addEventListener('hashchange', () => showTab(location.hash.slice(1)));
document.addEventListener('click', async e => {
 const close = e.target.closest('[data-close]'); if (close) {closeDialog(close.closest('dialog')); return;}
 const tab = e.target.closest('[data-tab]'); if (tab) {showTab(tab.dataset.tab); return;}
 const b = e.target.closest('[data-action]'); if (!b || !state || b.disabled) return; const id = Number(b.dataset.id), a = b.dataset.action;
 try {
  if (a === 'new-item' || a === 'edit-item') openItem(id);
  else if (a === 'new-person' || a === 'edit-person') openPerson(id);
  else if (a === 'new-rental') openRental();
  else if (a === 'document') openDocument(id);
  else if (a === 'edit-price') openPrice(b.dataset.id);
  else if (a === 'delete-item' || a === 'delete-person') {if (!confirm('Eintrag löschen? Einträge mit Verleihhistorie bleiben geschützt.')) return; b.disabled = true; await api(`/api/${a === 'delete-item' ? 'items' : 'people'}/${id}`, {method: 'DELETE'}); await refresh(); notice('Eintrag gelöscht.');}
  else if (['checkout', 'return', 'cancel'].includes(a)) {if (a === 'cancel' && !confirm('Reservierung stornieren?')) return; b.disabled = true; await api(`/api/rentals/${id}/${a}`, {method: 'POST'}); await refresh(); notice({checkout: 'Ausrüstung ausgegeben.', return: 'Rückgabe gespeichert. Ausrüstung ist wieder frei.', cancel: 'Reservierung storniert.'}[a]);}
 } catch (error) {notice(error.message, true); b.disabled = false;}
});
(async () => {try {[config, state] = await Promise.all([api('/api/config'), api('/api/state')]); render(); showTab(location.hash.slice(1));} catch (e) {notice(e.message, true);}})();

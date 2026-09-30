// Dispatch flow on the web: technicians accept/decline assigned jobs, the owner sees it.
// Seeds via the API (2 technicians, 4 jobs; technician tokens come from the OTP stored in
// worker_otps since Twilio is off), then screenshots Dashboard + Jobs and reassigns a job
// through the edit form.
//   node .claude/skills/run-routewise/drive.mjs flow .claude/skills/run-routewise/flows/dispatch.mjs --auth --lang he
import { execFileSync } from 'node:child_process';

const API = (process.env.RW_API_URL || 'http://localhost:18080') + '/api/v1';

async function call(method, path, token, body) {
  const res = await fetch(API + path, {
    method,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}: ${text}`);
  return text ? JSON.parse(text) : null;
}

// Technician login without SMS: request an OTP, read it from the DB, verify it.
async function workerToken(companyCode, phone) {
  await fetch(`${API}/workers/request-otp`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ company_code: companyCode, phone }),
  });
  const otp = execFileSync('docker', ['exec', 'routewise-run-db', 'psql', '-U', 'routewise', '-tAc',
    `select otp_code from worker_otps where phone='${phone}' order by id desc limit 1`]).toString().trim();
  const r = await call('POST', '/worker/verify-otp', null, { company_code: companyCode, phone, code: otp });
  return r.token;
}

export default async ({ page, shot, user, web }) => {
  const owner = user.token;
  await page.goto(web, { waitUntil: 'domcontentloaded' });
  const lang = (await page.evaluate(() => localStorage.getItem('language'))) || 'en';
  const p = `dispatch-${lang}-`;
  const suffix = String(Date.now()).slice(-6);

  const avi = await call('POST', '/workers', owner, { name: 'Avi Levi', phone: `+97250${suffix}1`, is_active: true });
  const beni = await call('POST', '/workers', owner, { name: 'Beni Cohen', phone: `+97250${suffix}2`, is_active: true });
  const cust = await call('POST', '/customers', owner, { name: 'Dana Cohen', phone: '+972502223333', address: 'Herzl 1, Haifa' });
  const job = (title, technician_id, day) => call('POST', '/jobs', owner, {
    customer_id: cust.id, technician_id, title, scheduled_at: `2026-10-0${day}T09:00:00Z`, price: 350,
  });
  const j1 = await job('AC repair', avi.id, 1);        // stays pending
  const j2 = await job('Filter change', beni.id, 2);   // Beni declines
  await job('New unit install', null, 3);              // never assigned
  const j4 = await job('Yearly maintenance', avi.id, 4); // Avi accepts

  const code = user.organization.company_code;
  const tAvi = await workerToken(code, avi.phone);
  const tBeni = await workerToken(code, beni.phone);
  await call('POST', `/jobs/${j2.id}/decline`, tBeni, { reason: lang === 'he' ? 'חולה היום' : 'Sick today' });
  await call('POST', `/jobs/${j4.id}/accept`, tAvi);

  await page.goto(`${web}/dashboard`, { waitUntil: 'networkidle' });
  await page.getByTestId('dispatch-banner').waitFor({ timeout: 10000 });
  await shot(`${p}1-dashboard.png`);

  await page.goto(`${web}/jobs`, { waitUntil: 'networkidle' });
  await page.getByText('AC repair').waitFor();
  await shot(`${p}2-jobs.png`);

  await page.goto(`${web}/jobs?filter=unassigned`, { waitUntil: 'networkidle' });
  await page.getByText('Filter change').waitFor();
  await shot(`${p}3-unassigned.png`);

  // Reassign "AC repair" from Avi to Beni through the edit form.
  await page.goto(`${web}/jobs`, { waitUntil: 'networkidle' });
  const row = page.locator('li', { hasText: 'AC repair' });
  await row.getByRole('button', { name: /Edit|עריכה/ }).click();
  await page.locator('select[name="technician_id"]').selectOption(String(beni.id));
  await shot(`${p}4-edit-modal.png`);
  await page.locator('form button[type="submit"]').click();
  // The modal closes only after the update + reassign calls finished and the list reloaded.
  await page.locator('select[name="technician_id"]').waitFor({ state: 'detached', timeout: 10000 });

  const after = await call('GET', `/jobs/${j1.id}`, owner);
  if (after.worker_id !== beni.id || after.assignment_status !== 'pending') {
    throw new Error(`reassign via edit form failed: worker_id=${after.worker_id} status=${after.assignment_status}`);
  }
  const beniJobs = await call('GET', '/jobs', tBeni);
  console.error(`reassigned via edit form → Beni now sees: ${beniJobs.map((j) => j.title).join(', ')}`);
  await shot(`${p}5-after-reassign.png`);
};

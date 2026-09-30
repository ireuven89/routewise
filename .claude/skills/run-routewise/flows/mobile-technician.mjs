// Technician side in the Expo app (web target): new-job badge → accept → start → complete,
// and decline with a reason. Seeds via the API as the --auth owner; the technician logs in by
// having their token put into storage (the navigator polls it), since SMS is off.
//   S=.claude/skills/run-routewise/stack.sh; $S up mobile
//   RW_WEB_URL=http://localhost:8190 node .claude/skills/run-routewise/drive.mjs \
//     flow .claude/skills/run-routewise/flows/mobile-technician.mjs --auth --lang he --viewport 390x844
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

export default async ({ page, shot, user, web }) => {
  const owner = user.token;
  const suffix = String(Date.now()).slice(-6);
  const tech = await call('POST', '/workers', owner, { name: 'Avi Levi', phone: `+97250${suffix}1`, is_active: true });
  const cust = await call('POST', '/customers', owner, { name: 'Dana Cohen', phone: '+972502223333', address: 'Herzl 1, Haifa' });
  const job = (title, day) => call('POST', '/jobs', owner, {
    customer_id: cust.id, technician_id: tech.id, title, scheduled_at: `2026-10-0${day}T09:00:00Z`,
  });
  const jAccept = await job('AC repair', 1);
  const jDecline = await job('Filter change', 2);

  // Technician token via the OTP stored in the DB.
  const code = user.organization.company_code;
  await fetch(`${API}/workers/request-otp`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ company_code: code, phone: tech.phone }),
  });
  const otp = execFileSync('docker', ['exec', 'routewise-run-db', 'psql', '-U', 'routewise', '-tAc',
    `select otp_code from worker_otps where phone='${tech.phone}' order by id desc limit 1`]).toString().trim();
  const login = await call('POST', '/worker/verify-otp', null, { company_code: code, phone: tech.phone, code: otp });

  page.on('dialog', (d) => d.accept());
  await page.goto(web, { waitUntil: 'networkidle' });
  const lang = (await page.evaluate(() => localStorage.getItem('language'))) || 'he';
  const p = `m-tech-${lang}-`;
  await page.evaluate(([t, w]) => {
    localStorage.setItem('mode', 'technician');
    localStorage.setItem('token', t);
    localStorage.setItem('worker', JSON.stringify(w));
  }, [login.token, login.worker]);
  await page.reload({ waitUntil: 'networkidle' });

  await page.getByTestId(`new-${jAccept.id}`).waitFor({ timeout: 15000 });
  await shot(`${p}1-list-new.png`);

  // Accept → Start → Complete
  await page.getByText('AC repair').click();
  await page.getByTestId('accept-job').waitFor({ timeout: 10000 });
  await shot(`${p}2-offer.png`);
  await page.getByTestId('accept-job').click();
  await page.getByTestId('start-job').waitFor({ timeout: 10000 });
  await shot(`${p}3-accepted.png`);
  await page.getByTestId('start-job').click();
  await page.getByTestId('complete-job').waitFor({ timeout: 10000 });
  await page.getByTestId('complete-job').click();
  await page.getByTestId('complete-job').waitFor({ state: 'detached', timeout: 10000 });
  const done = await call('GET', `/jobs/${jAccept.id}`, owner);
  if (done.status !== 'completed' || done.assignment_status !== 'accepted') {
    throw new Error(`accept flow ended as status=${done.status} assignment=${done.assignment_status}`);
  }
  await shot(`${p}4-completed.png`);

  // Decline with a reason → job leaves the technician's list
  // Reload to the job list: the navigator keeps no browser history, so page.goBack() would leave the app.
  await page.goto(web);
  await page.getByText('Filter change').click();
  await page.getByTestId('decline-job').click();
  await page.getByTestId('decline-reason').fill(lang === 'he' ? 'לא זמין מחר' : 'Not available tomorrow');
  await shot(`${p}5-decline.png`);
  await page.getByTestId('confirm-decline').click();
  // The detail screen shows the title twice (header + body), so wait for every copy to go.
  await page.waitForFunction(() => !document.body.innerText.includes('Filter change'), null, { timeout: 10000 });
  await shot(`${p}6-after-decline.png`);

  const declined = await call('GET', `/jobs/${jDecline.id}`, owner);
  if (declined.worker_id !== null || !declined.decline_reason) {
    throw new Error(`decline not recorded: worker_id=${declined.worker_id} reason=${declined.decline_reason}`);
  }
  console.error(`accepted+completed #${jAccept.id}; declined #${jDecline.id} ("${declined.decline_reason}", by ${declined.declined_by_name})`);
};

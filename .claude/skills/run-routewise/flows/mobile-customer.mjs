// End-to-end customer flow in the Expo app (web target): role → location → providers → post job
// → provider bids via API → award → job created.
//   S=.claude/skills/run-routewise/stack.sh; $S up mobile; $S seed-provider plumbing > $TMPDIR/routewise-run/provider.json
//   RW_WEB_URL=http://localhost:8190 node .claude/skills/run-routewise/drive.mjs \
//     flow .claude/skills/run-routewise/flows/mobile-customer.mjs --lang he --viewport 390x844 --geo 32.0809,34.7806
import fs from 'node:fs';
import path from 'node:path';

const API = process.env.RW_API_URL || 'http://localhost:18080';
const providerFile = path.join(process.env.TMPDIR || '/tmp', 'routewise-run', 'provider.json');

async function call(method, p, token, body) {
  const res = await fetch(API + p, {
    method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`${method} ${p} -> ${res.status}: ${await res.text()}`);
  return res.json();
}

export default async ({ page, shot, web }) => {
  const provider = JSON.parse(fs.readFileSync(providerFile, 'utf8'));
  page.on('dialog', (d) => d.accept()); // award confirm uses window.confirm on web

  await page.goto(web, { waitUntil: 'networkidle' });
  const lang = await page.evaluate(() => localStorage.getItem('language'));
  const p = `m-${lang || 'he'}-`;
  await page.getByTestId('role-customer').click();
  await page.getByTestId('post-job').waitFor();
  await shot(`${p}1-find-service.png`);

  await page.getByTestId('service-plumbing').click();
  await page.getByTestId('use-location').click();
  await page.getByText(/✓/).waitFor({ timeout: 10000 });
  await page.getByTestId('description-input').fill(lang === 'en' ? 'Kitchen sink is leaking' : 'הכיור במטבח דולף');
  await page.getByTestId('time-timeTomorrow').click();
  await shot(`${p}2-find-service-filled.png`);

  await page.getByTestId('search-providers').click();
  await page.getByTestId(/^provider-/).first().waitFor({ timeout: 10000 });
  await shot(`${p}3-providers.png`);

  await page.getByTestId('post-job-from-results').click();
  await page.getByTestId('name-input').fill(lang === 'en' ? 'Dana Cohen' : 'דנה כהן');
  await page.getByTestId('phone-input').fill('0502222222');
  await shot(`${p}4-post-job.png`);
  await page.getByTestId('submit-request').click();

  await page.getByTestId('request-status').waitFor({ timeout: 10000 });
  await shot(`${p}5-tracking-no-bids.png`);

  // Provider side (API): find the new lead and bid on it.
  const leads = await call('GET', '/api/v1/leads', provider.token);
  const list = Array.isArray(leads) ? leads : (leads.leads || leads.requests || []);
  const lead = list.map((l) => l.request || l).sort((a, b) => b.id - a.id)[0];
  if (!lead) throw new Error(`provider got no lead: ${JSON.stringify(leads).slice(0, 300)}`);
  await call('PUT', `/api/v1/leads/${lead.id}/bid`, provider.token, {
    price: 450, eta_minutes: 90, message: lang === 'en' ? 'Can come tomorrow morning' : 'אפשר להגיע מחר בבוקר',
  });

  await page.getByTestId('refresh').click();
  const award = page.getByTestId(/^award-/).first();
  await award.waitFor({ timeout: 10000 });
  await shot(`${p}6-tracking-bid.png`);

  await award.click();
  await page.getByText(/✓ /).first().waitFor({ timeout: 10000 });
  await shot(`${p}7-awarded.png`);

  const jobs = await call('GET', '/api/v1/jobs', provider.token);
  const jobList = Array.isArray(jobs) ? jobs : (jobs.jobs || []);
  if (!jobList.length) throw new Error('award did not create a job for the provider');
  console.error(`provider now has ${jobList.length} job(s); lead #${lead.id} awarded`);

  await page.getByTestId('header-back').click();
  await page.getByTestId('open-my-requests').click();
  await page.getByTestId(/^my-request-/).first().waitFor({ timeout: 10000 });
  await shot(`${p}8-my-requests.png`);
};

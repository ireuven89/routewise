// Example flow: add a customer through the UI, then confirm it via the API.
// node .claude/skills/run-routewise/drive.mjs flow .claude/skills/run-routewise/flows/add-customer.mjs --auth
export default async ({ page, shot, api, web }) => {
  await page.goto(`${web}/customers`, { waitUntil: 'networkidle' });
  await page.getByRole('button', { name: /Add Customer/i }).first().click();
  await page.fill('input[name="name"]', 'Dana Cohen');
  await page.fill('input[name="phone"]', '+972501111111');
  await page.fill('input[name="address"]', 'Herzl 1, Tel Aviv'); // plain input: no Google Maps key locally
  await shot('customer-modal.png');
  await page.click('form button[type="submit"]');
  await page.getByText('Dana Cohen').first().waitFor({ timeout: 10000 });
  await shot('customers-list.png');

  const customers = await api('GET', '/api/v1/customers');
  if (!customers.some((c) => c.name === 'Dana Cohen')) throw new Error('customer not persisted');
  console.error(`api: ${customers.length} customer(s) persisted`);
};

// Prices are in shekels regardless of UI language.
export const formatMoney = (amount) => {
  if (amount == null || amount === '') return '';
  const n = Number(amount);
  return `₪${Number.isInteger(n) ? n : n.toFixed(2)}`;
};

export const formatDateTime = (value, locale) => {
  if (!value) return '';
  return new Date(value).toLocaleString(locale, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

// Preferred-time chips on FindService → ISO timestamp (null = flexible).
export const PREFERRED_TIMES = ['timeFlexible', 'timeToday', 'timeTomorrow', 'timeThisWeek'];

export const preferredTimeToISO = (option, now = new Date()) => {
  const d = new Date(now);
  switch (option) {
    case 'timeToday':
      d.setHours(d.getHours() + 2, 0, 0, 0);
      return d.toISOString();
    case 'timeTomorrow':
      d.setDate(d.getDate() + 1);
      d.setHours(9, 0, 0, 0);
      return d.toISOString();
    case 'timeThisWeek':
      d.setDate(d.getDate() + 3);
      d.setHours(9, 0, 0, 0);
      return d.toISOString();
    default:
      return null;
  }
};

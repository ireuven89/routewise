import { formatMoney, formatDateTime, preferredTimeToISO } from '../format';

describe('formatMoney', () => {
  it('returns an empty string for null, undefined and empty string', () => {
    expect(formatMoney(null)).toBe('');
    expect(formatMoney(undefined)).toBe('');
    expect(formatMoney('')).toBe('');
  });

  it('formats an integer amount with no decimals', () => {
    expect(formatMoney(20)).toBe('₪20');
    expect(formatMoney('30')).toBe('₪30');
    expect(formatMoney(0)).toBe('₪0');
  });

  it('formats a decimal amount with 2 decimal places', () => {
    expect(formatMoney(15.5)).toBe('₪15.50');
    expect(formatMoney(10.99)).toBe('₪10.99');
  });
});

describe('formatDateTime', () => {
  it('returns an empty string for a falsy value', () => {
    expect(formatDateTime(null, 'en-US')).toBe('');
    expect(formatDateTime('', 'en-US')).toBe('');
    expect(formatDateTime(undefined, 'en-US')).toBe('');
  });

  it('formats a date using the given locale and the month/day/hour/minute options', () => {
    const value = '2024-03-15T10:30:00Z';
    const expected = new Date(value).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });

    expect(formatDateTime(value, 'en-US')).toBe(expected);
  });
});

describe('preferredTimeToISO', () => {
  // Fixed local time (not UTC) so getHours()/getDate() below don't depend on the runner's timezone.
  const now = new Date(2024, 0, 10, 10, 0, 0, 0); // Wed Jan 10 2024, 10:00 local

  it('returns null for timeFlexible (or any unrecognized option)', () => {
    expect(preferredTimeToISO('timeFlexible', now)).toBeNull();
    expect(preferredTimeToISO('somethingElse', now)).toBeNull();
    expect(preferredTimeToISO(undefined, now)).toBeNull();
  });

  it('timeToday: same day, 2 hours from now, minutes/seconds zeroed', () => {
    const result = preferredTimeToISO('timeToday', now);
    const d = new Date(result);
    expect(d.getDate()).toBe(now.getDate());
    expect(d.getHours()).toBe(12);
    expect(d.getMinutes()).toBe(0);
    expect(d.getSeconds()).toBe(0);
  });

  it('timeTomorrow: next day at 9am', () => {
    const result = preferredTimeToISO('timeTomorrow', now);
    const d = new Date(result);
    expect(d.getDate()).toBe(now.getDate() + 1);
    expect(d.getHours()).toBe(9);
    expect(d.getMinutes()).toBe(0);
  });

  it('timeThisWeek: 3 days from now at 9am', () => {
    const result = preferredTimeToISO('timeThisWeek', now);
    const d = new Date(result);
    expect(d.getDate()).toBe(now.getDate() + 3);
    expect(d.getHours()).toBe(9);
    expect(d.getMinutes()).toBe(0);
  });
});

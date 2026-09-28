import { formatPhone } from '../phone';

describe('formatPhone', () => {
  it('strips a single leading 0 and prepends the country code', () => {
    expect(formatPhone('+972', '0501234567')).toBe('+972501234567');
  });

  it('leaves a number with no leading 0 unchanged aside from the prefix', () => {
    expect(formatPhone('+972', '501234567')).toBe('+972501234567');
  });

  it('prepends whichever country code is given', () => {
    expect(formatPhone('+1', '05551234567')).toBe('+15551234567');
  });
});

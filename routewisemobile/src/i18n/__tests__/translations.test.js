import en from '../translations/en';
import he from '../translations/he';

const keyPaths = (obj, prefix = '') =>
  Object.entries(obj).flatMap(([k, v]) =>
    v && typeof v === 'object' ? keyPaths(v, `${prefix}${k}.`) : [`${prefix}${k}`]
  );

describe('translations', () => {
  it('he and en define exactly the same keys', () => {
    expect(keyPaths(he).sort()).toEqual(keyPaths(en).sort());
  });

  it('every value is a non-empty string', () => {
    for (const dict of [en, he]) {
      for (const path of keyPaths(dict)) {
        const value = path.split('.').reduce((o, k) => o[k], dict);
        expect(typeof value).toBe('string');
        expect(value.trim()).not.toBe('');
      }
    }
  });

  it('uses the same {{params}} in both languages', () => {
    for (const path of keyPaths(en)) {
      const get = (d) => path.split('.').reduce((o, k) => o[k], d);
      const params = (s) => (s.match(/{{\w+}}/g) || []).sort();
      expect([path, params(get(he))]).toEqual([path, params(get(en))]);
    }
  });
});

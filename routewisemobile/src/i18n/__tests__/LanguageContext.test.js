import React from 'react';
import { Text, TouchableOpacity } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {
  translate, LanguageProvider, useLanguage, useRTL, LANGUAGE_KEY, DEFAULT_LANGUAGE,
} from '../LanguageContext';

afterEach(async () => {
  await AsyncStorage.clear();
});

describe('translate', () => {
  it('resolves a nested key', () => {
    expect(translate('en', 'common.error')).toBe('Error');
    expect(translate('he', 'common.error')).toBe('שגיאה');
  });

  it('returns the key itself when the key is missing', () => {
    expect(translate('en', 'nope.missing')).toBe('nope.missing');
  });

  it('returns the key itself when the resolved value is not a string (e.g. a namespace object)', () => {
    expect(translate('en', 'common')).toBe('common');
  });

  it('returns the key itself for an unknown language', () => {
    expect(translate('fr', 'common.error')).toBe('common.error');
  });

  it('interpolates a single {{param}}', () => {
    expect(translate('en', 'requestTracking.awardedTo', { name: 'Acme Plumbing' }))
      .toBe('You selected Acme Plumbing for this job.');
  });

  it('interpolates multiple different params', () => {
    expect(translate('en', 'requestTracking.awardConfirmBody', { name: 'Acme', price: '₪100' }))
      .toBe('Acme will get the job for ₪100. Other offers will be declined.');
  });

  it('interpolates the same param repeated more than once', () => {
    jest.isolateModules(() => {
      jest.doMock('../translations/en', () => ({ greeting: 'Hi {{name}}, bye {{name}}!' }));
      jest.doMock('../translations/he', () => ({ greeting: 'שלום {{name}}, להתראות {{name}}!' }));
      // eslint-disable-next-line global-require
      const { translate: isolatedTranslate } = require('../LanguageContext');
      expect(isolatedTranslate('en', 'greeting', { name: 'Dana' })).toBe('Hi Dana, bye Dana!');
    });
  });
});

const Consumer = () => {
  const { language, t, toggleLanguage } = useLanguage();
  const rtl = useRTL();
  return (
    <>
      <Text testID="lang">{language}</Text>
      <Text testID="translated">{t('common.error')}</Text>
      <Text testID="rtl-dump">{JSON.stringify(rtl)}</Text>
      <TouchableOpacity testID="toggle" onPress={toggleLanguage}><Text>toggle</Text></TouchableOpacity>
    </>
  );
};

describe('LanguageProvider', () => {
  it('defaults to Hebrew when no initialLanguage and nothing saved', async () => {
    render(<LanguageProvider><Consumer /></LanguageProvider>);

    expect(await screen.findByTestId('lang')).toHaveTextContent(DEFAULT_LANGUAGE);
    expect(screen.getByTestId('translated')).toHaveTextContent('שגיאה');
  });

  it('honors initialLanguage and skips the AsyncStorage read', () => {
    render(<LanguageProvider initialLanguage="en"><Consumer /></LanguageProvider>);

    expect(screen.getByTestId('lang')).toHaveTextContent('en');
    expect(screen.getByTestId('translated')).toHaveTextContent('Error');
  });

  it('loads a previously saved language from AsyncStorage when no initialLanguage is given', async () => {
    await AsyncStorage.setItem(LANGUAGE_KEY, 'en');

    render(<LanguageProvider><Consumer /></LanguageProvider>);

    await waitFor(() => expect(screen.getByTestId('lang')).toHaveTextContent('en'));
    expect(screen.getByTestId('translated')).toHaveTextContent('Error');
  });

  it('toggleLanguage flips the language and persists the choice', async () => {
    render(<LanguageProvider initialLanguage="en"><Consumer /></LanguageProvider>);

    fireEvent.press(screen.getByTestId('toggle'));

    expect(screen.getByTestId('lang')).toHaveTextContent('he');
    await waitFor(async () => expect(await AsyncStorage.getItem(LANGUAGE_KEY)).toBe('he'));
  });
});

describe('useRTL', () => {
  it('returns RTL layout values for Hebrew', () => {
    render(<LanguageProvider initialLanguage="he"><Consumer /></LanguageProvider>);

    const rtl = JSON.parse(screen.getByTestId('rtl-dump').props.children);
    expect(rtl).toEqual({
      isRTL: true,
      row: { flexDirection: 'row-reverse' },
      text: { textAlign: 'right', writingDirection: 'rtl' },
      alignStart: { alignItems: 'flex-end' },
      arrowBack: '→',
    });
  });

  it('returns LTR layout values for English', () => {
    render(<LanguageProvider initialLanguage="en"><Consumer /></LanguageProvider>);

    const rtl = JSON.parse(screen.getByTestId('rtl-dump').props.children);
    expect(rtl).toEqual({
      isRTL: false,
      row: { flexDirection: 'row' },
      text: { textAlign: 'left', writingDirection: 'ltr' },
      alignStart: { alignItems: 'flex-start' },
      arrowBack: '←',
    });
  });
});

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { ModeContext } from '../../../navigation/ModeContext';
import RoleSelectScreen from '../RoleSelectScreen';

const renderWithProviders = ({ initialLanguage, setMode = jest.fn() } = {}) => {
  render(
    <LanguageProvider initialLanguage={initialLanguage}>
      <ModeContext.Provider value={{ mode: null, setMode }}>
        <RoleSelectScreen />
      </ModeContext.Provider>
    </LanguageProvider>
  );
  return { setMode };
};

describe('RoleSelectScreen', () => {
  it('renders in Hebrew by default (no initialLanguage given)', () => {
    renderWithProviders();

    expect(screen.getByText('שירותי בית, בפשטות')).toBeTruthy();
    expect(screen.getByText('אני צריך/ה שירות')).toBeTruthy();
    expect(screen.getByText('אני טכנאי/ת')).toBeTruthy();
  });

  it('renders in English when initialLanguage="en"', () => {
    renderWithProviders({ initialLanguage: 'en' });

    expect(screen.getByText('Home services, simplified')).toBeTruthy();
    expect(screen.getByText('I need a service')).toBeTruthy();
  });

  it('pressing the customer role card calls setMode("customer")', () => {
    const { setMode } = renderWithProviders({ initialLanguage: 'en' });

    fireEvent.press(screen.getByTestId('role-customer'));

    expect(setMode).toHaveBeenCalledWith('customer');
  });

  it('pressing the technician role card calls setMode("technician")', () => {
    const { setMode } = renderWithProviders({ initialLanguage: 'en' });

    fireEvent.press(screen.getByTestId('role-technician'));

    expect(setMode).toHaveBeenCalledWith('technician');
  });
});

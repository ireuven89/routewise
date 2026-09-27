import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { LanguageProvider } from '../../i18n/LanguageContext';
import { storage } from '../../services/api';
import AppNavigator from '../AppNavigator';

jest.mock('../../services/api', () => ({
  storage: {
    getMode: jest.fn(),
    setMode: jest.fn(),
  },
}));

afterEach(() => {
  jest.clearAllMocks();
});

describe('AppNavigator', () => {
  it('shows RoleSelect when no mode is saved', async () => {
    storage.getMode.mockResolvedValue(null);

    render(<LanguageProvider initialLanguage="en"><AppNavigator /></LanguageProvider>);

    expect(await screen.findByTestId('role-customer')).toBeTruthy();
  });

  it('shows FindService when the saved mode is "customer"', async () => {
    storage.getMode.mockResolvedValue('customer');

    render(<LanguageProvider initialLanguage="en"><AppNavigator /></LanguageProvider>);

    await waitFor(() => expect(screen.getByTestId('use-location')).toBeTruthy());
  });
});

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { publicApi, storage } from '../../../services/api';
import MyRequestsScreen from '../MyRequestsScreen';

jest.mock('../../../services/api', () => ({
  publicApi: { getRequest: jest.fn() },
  storage: { getMyRequests: jest.fn() },
}));

// useFocusEffect just needs to run its callback on mount for these tests.
jest.mock('@react-navigation/native', () => ({
  useFocusEffect: (cb) => require('react').useEffect(cb, []),
}));

const renderScreen = (navigation = {}) => {
  const nav = { navigate: jest.fn(), goBack: jest.fn(), reset: jest.fn(), ...navigation };
  render(
    <LanguageProvider initialLanguage="en">
      <MyRequestsScreen navigation={nav} />
    </LanguageProvider>
  );
  return { navigation: nav };
};

afterEach(() => {
  jest.clearAllMocks();
});

describe('MyRequestsScreen', () => {
  it('lists saved requests with their live status and offer count', async () => {
    storage.getMyRequests.mockResolvedValue([
      { token: 't1', id: 1, service_type: 'hvac', address: 'Herzl 1', created_at: '2024-01-01T00:00:00Z' },
    ]);
    publicApi.getRequest.mockResolvedValue({
      request: { status: 'open' },
      bids: [{ id: 1 }, { id: 2 }],
    });

    renderScreen();

    const row = await screen.findByTestId('my-request-1');
    expect(row).toHaveTextContent(/Waiting for offers/);
    expect(row).toHaveTextContent(/Offers received: 2/);
  });

  it('marks a request as unavailable when the live fetch fails', async () => {
    storage.getMyRequests.mockResolvedValue([
      { token: 't1', id: 1, service_type: 'hvac', address: 'Herzl 1', created_at: '2024-01-01T00:00:00Z' },
    ]);
    publicApi.getRequest.mockRejectedValue(new Error('network error'));

    renderScreen();

    const row = await screen.findByTestId('my-request-1');
    expect(row).toHaveTextContent(/Unavailable/);
  });

  it('shows an empty state with a "new request" action when there are no saved requests', async () => {
    storage.getMyRequests.mockResolvedValue([]);

    renderScreen();

    await waitFor(() => expect(screen.getByText("You haven't posted any requests yet.")).toBeTruthy());
    expect(screen.getByText('New request')).toBeTruthy();
  });
});

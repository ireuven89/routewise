import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { publicApi } from '../../../services/api';
import { confirm, notify } from '../../../utils/confirm';
import RequestTrackingScreen, { POLL_INTERVAL_MS } from '../RequestTrackingScreen';

jest.mock('../../../services/api', () => ({
  publicApi: { getRequest: jest.fn(), awardBid: jest.fn() },
}));

jest.mock('../../../utils/confirm', () => ({
  confirm: jest.fn(),
  notify: jest.fn(),
}));

const openRequestWithBid = {
  request: { status: 'open', service_type: 'plumbing', address: 'Herzl 1' },
  bids: [{ id: 1, status: 'submitted', organization_name: 'Acme', price: 100 }],
};

const renderScreen = (navigation = {}) => {
  const nav = { navigate: jest.fn(), goBack: jest.fn(), reset: jest.fn(), ...navigation };
  render(
    <LanguageProvider initialLanguage="en">
      <RequestTrackingScreen navigation={nav} route={{ params: { token: 'tok-abc' } }} />
    </LanguageProvider>
  );
  return { navigation: nav };
};

afterEach(() => {
  jest.clearAllMocks();
  jest.useRealTimers();
});

describe('RequestTrackingScreen', () => {
  it('renders the status badge and the list of bids', async () => {
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    renderScreen();

    expect(await screen.findByTestId('request-status')).toHaveTextContent('Waiting for offers');
    expect(screen.getByTestId('bid-1')).toBeTruthy();
    expect(screen.getByText('Acme')).toBeTruthy();
    expect(screen.getByText('₪100')).toBeTruthy();
  });

  it('shows an award button only for a submitted bid on an open request', async () => {
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    renderScreen();

    expect(await screen.findByTestId('award-1')).toBeTruthy();
  });

  it('hides the award button for a bid that is not submitted', async () => {
    publicApi.getRequest.mockResolvedValue({
      request: { status: 'open', service_type: 'plumbing', address: 'Herzl 1' },
      bids: [{ id: 2, status: 'awarded', organization_name: 'Acme', price: 100 }],
    });
    renderScreen();

    await screen.findByTestId('bid-2');
    expect(screen.queryByTestId('award-2')).toBeNull();
  });

  it('hides the award button once the request itself is no longer open', async () => {
    publicApi.getRequest.mockResolvedValue({
      request: { status: 'awarded', service_type: 'plumbing', address: 'Herzl 1' },
      bids: [{ id: 3, status: 'submitted', organization_name: 'Acme', price: 100 }],
    });
    renderScreen();

    await screen.findByTestId('bid-3');
    expect(screen.queryByTestId('award-3')).toBeNull();
  });

  it('awards the bid after confirming, then reloads the request', async () => {
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    confirm.mockResolvedValue(true);
    publicApi.awardBid.mockResolvedValue({});
    renderScreen();

    await screen.findByTestId('award-1');
    fireEvent.press(screen.getByTestId('award-1'));

    await waitFor(() => expect(publicApi.awardBid).toHaveBeenCalledWith('tok-abc', 1));
    expect(confirm).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Choose this provider?',
      message: 'Acme will get the job for ₪100. Other offers will be declined.',
    }));
    await waitFor(() => expect(publicApi.getRequest).toHaveBeenCalledTimes(2));
  });

  it('does not award when the confirmation is cancelled', async () => {
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    confirm.mockResolvedValue(false);
    renderScreen();

    await screen.findByTestId('award-1');
    fireEvent.press(screen.getByTestId('award-1'));

    await waitFor(() => expect(confirm).toHaveBeenCalled());
    expect(publicApi.awardBid).not.toHaveBeenCalled();
  });

  it('notifies an error when awarding fails', async () => {
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    confirm.mockResolvedValue(true);
    publicApi.awardBid.mockRejectedValue(new Error('boom'));
    renderScreen();

    await screen.findByTestId('award-1');
    fireEvent.press(screen.getByTestId('award-1'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'Could not select this provider. Please try again.'));
  });

  it('shows a not-found state when the request cannot be loaded', async () => {
    publicApi.getRequest.mockRejectedValue(new Error('404'));
    renderScreen();

    expect(await screen.findByText('This request could not be found.')).toBeTruthy();
  });

  it('polls every POLL_INTERVAL_MS while the request is open, and stops once it is no longer open', async () => {
    jest.useFakeTimers();
    publicApi.getRequest.mockResolvedValue(openRequestWithBid);
    renderScreen();

    await act(async () => { await Promise.resolve(); });
    expect(publicApi.getRequest).toHaveBeenCalledTimes(1);

    publicApi.getRequest.mockResolvedValue({
      request: { status: 'awarded', service_type: 'plumbing', address: 'Herzl 1' },
      bids: [{ id: 1, status: 'awarded', organization_name: 'Acme', price: 100 }],
    });

    await act(async () => {
      jest.advanceTimersByTime(POLL_INTERVAL_MS);
      await Promise.resolve();
    });
    expect(publicApi.getRequest).toHaveBeenCalledTimes(2);

    await act(async () => {
      jest.advanceTimersByTime(POLL_INTERVAL_MS * 3);
      await Promise.resolve();
    });
    // Status is now 'awarded' (no longer open) — polling must have stopped.
    expect(publicApi.getRequest).toHaveBeenCalledTimes(2);
  });
});

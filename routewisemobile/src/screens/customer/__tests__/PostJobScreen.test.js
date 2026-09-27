import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { publicApi, storage } from '../../../services/api';
import { notify } from '../../../utils/confirm';
import PostJobScreen from '../PostJobScreen';

jest.mock('../../../services/api', () => ({
  publicApi: { createRequest: jest.fn() },
  storage: {
    getCustomerProfile: jest.fn(),
    addMyRequest: jest.fn(),
    saveCustomerProfile: jest.fn(),
  },
}));

jest.mock('../../../utils/confirm', () => ({
  confirm: jest.fn(),
  notify: jest.fn(),
}));

const job = { service_type: 'plumbing', address: 'Herzl 1, Tel Aviv', preferred_time: null };

const renderScreen = (routeParams = { job }, navigation = {}) => {
  const nav = { navigate: jest.fn(), goBack: jest.fn(), reset: jest.fn(), ...navigation };
  render(
    <LanguageProvider initialLanguage="en">
      <PostJobScreen navigation={nav} route={{ params: routeParams }} />
    </LanguageProvider>
  );
  return { navigation: nav };
};

beforeEach(() => {
  storage.getCustomerProfile.mockResolvedValue(null);
});

afterEach(() => {
  jest.clearAllMocks();
});

describe('PostJobScreen', () => {
  it('notifies when name and phone are empty and does not submit', async () => {
    const { navigation } = renderScreen();

    fireEvent.press(screen.getByTestId('submit-request'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'Please enter your name and phone number.'));
    expect(publicApi.createRequest).not.toHaveBeenCalled();
    expect(navigation.reset).not.toHaveBeenCalled();
  });

  it('prefills name/phone from the saved customer profile', async () => {
    storage.getCustomerProfile.mockResolvedValue({ name: 'Dana', countryCode: '+1', phoneNumber: '5551234567' });

    renderScreen();

    await waitFor(() => expect(screen.getByTestId('name-input').props.value).toBe('Dana'));
    expect(screen.getByTestId('phone-input').props.value).toBe('5551234567');
  });

  it('submits with a formatted phone, strips a null preferred_time, saves the token/profile, then resets navigation', async () => {
    publicApi.createRequest.mockResolvedValue({ id: 7, access_token: 'tok-abc' });
    const { navigation } = renderScreen();

    fireEvent.changeText(screen.getByTestId('name-input'), 'Dana Cohen');
    fireEvent.changeText(screen.getByTestId('phone-input'), '0501234567');
    fireEvent.press(screen.getByTestId('submit-request'));

    await waitFor(() => expect(navigation.reset).toHaveBeenCalled());

    expect(publicApi.createRequest).toHaveBeenCalledWith({
      service_type: 'plumbing',
      address: 'Herzl 1, Tel Aviv',
      customer_name: 'Dana Cohen',
      customer_phone: '+972501234567',
    });
    expect(storage.addMyRequest).toHaveBeenCalledWith(expect.objectContaining({
      token: 'tok-abc',
      id: 7,
      service_type: 'plumbing',
      address: 'Herzl 1, Tel Aviv',
      created_at: expect.any(String),
    }));
    expect(storage.saveCustomerProfile).toHaveBeenCalledWith({
      name: 'Dana Cohen',
      countryCode: '+972',
      phoneNumber: '0501234567',
    });
    expect(navigation.reset).toHaveBeenCalledWith({
      index: 1,
      routes: [{ name: 'FindService' }, { name: 'RequestTracking', params: { token: 'tok-abc' } }],
    });
  });

  it('keeps preferred_time in the payload when the job has one', async () => {
    publicApi.createRequest.mockResolvedValue({ id: 8, access_token: 'tok-def' });
    renderScreen({ job: { ...job, preferred_time: '2024-01-11T09:00:00.000Z' } });

    fireEvent.changeText(screen.getByTestId('name-input'), 'Dana');
    fireEvent.changeText(screen.getByTestId('phone-input'), '0501234567');
    fireEvent.press(screen.getByTestId('submit-request'));

    await waitFor(() => expect(publicApi.createRequest).toHaveBeenCalled());

    expect(publicApi.createRequest).toHaveBeenCalledWith(expect.objectContaining({
      preferred_time: '2024-01-11T09:00:00.000Z',
    }));
  });

  it('notifies a request error and does not reset navigation when createRequest fails', async () => {
    publicApi.createRequest.mockRejectedValue(new Error('network down'));
    const { navigation } = renderScreen();

    fireEvent.changeText(screen.getByTestId('name-input'), 'Dana');
    fireEvent.changeText(screen.getByTestId('phone-input'), '0501234567');
    fireEvent.press(screen.getByTestId('submit-request'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'Failed to send request. Please try again.'));
    expect(navigation.reset).not.toHaveBeenCalled();
  });
});

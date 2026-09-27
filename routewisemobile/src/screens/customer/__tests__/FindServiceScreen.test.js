import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { ModeContext } from '../../../navigation/ModeContext';
import { notify } from '../../../utils/confirm';
import { getCurrentLocation, geocodeAddress } from '../../../utils/location';
import FindServiceScreen from '../FindServiceScreen';

jest.mock('../../../utils/confirm', () => ({
  confirm: jest.fn(),
  notify: jest.fn(),
}));

jest.mock('../../../utils/location', () => {
  const actual = jest.requireActual('../../../utils/location');
  return {
    ...actual,
    getCurrentLocation: jest.fn(),
    geocodeAddress: jest.fn(),
  };
});

const renderScreen = (navigation = {}, setMode = jest.fn()) => {
  const nav = { navigate: jest.fn(), goBack: jest.fn(), reset: jest.fn(), ...navigation };
  render(
    <LanguageProvider initialLanguage="en">
      <ModeContext.Provider value={{ mode: 'customer', setMode }}>
        <FindServiceScreen navigation={nav} />
      </ModeContext.Provider>
    </LanguageProvider>
  );
  return { navigation: nav };
};

afterEach(() => {
  jest.clearAllMocks();
});

describe('FindServiceScreen', () => {
  it('"use my location" fills in the address field', async () => {
    getCurrentLocation.mockResolvedValue({ latitude: 32.08, longitude: 34.78, address: 'Herzl 1, Tel Aviv' });
    renderScreen();

    fireEvent.press(screen.getByTestId('use-location'));

    await waitFor(() => expect(screen.getByTestId('address-input').props.value).toBe('Herzl 1, Tel Aviv'));
  });

  it('posting a job with a typed address geocodes it and navigates with a job param', async () => {
    geocodeAddress.mockResolvedValue({ latitude: 1.5, longitude: 2.5, address: 'Typed Address 1' });
    const { navigation } = renderScreen();

    fireEvent.changeText(screen.getByTestId('address-input'), 'Typed Address 1');
    fireEvent.press(screen.getByTestId('post-job'));

    await waitFor(() => expect(navigation.navigate).toHaveBeenCalled());

    expect(geocodeAddress).toHaveBeenCalledWith('Typed Address 1');
    expect(navigation.navigate).toHaveBeenCalledWith('PostJob', {
      job: {
        service_type: 'hvac',
        description: '',
        latitude: 1.5,
        longitude: 2.5,
        address: 'Typed Address 1',
        preferred_time: null,
      },
    });
  });

  it('notifies addressRequired and does not navigate when the address is empty', async () => {
    const { navigation } = renderScreen();

    fireEvent.press(screen.getByTestId('post-job'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'Please enter an address or use your location.'));
    expect(navigation.navigate).not.toHaveBeenCalled();
    expect(geocodeAddress).not.toHaveBeenCalled();
  });

  it('selecting a service type chip changes the service_type sent on submit', async () => {
    geocodeAddress.mockResolvedValue({ latitude: 1, longitude: 2, address: 'Some Address' });
    const { navigation } = renderScreen();

    fireEvent.changeText(screen.getByTestId('address-input'), 'Some Address');
    fireEvent.press(screen.getByTestId('service-plumbing'));
    fireEvent.press(screen.getByTestId('post-job'));

    await waitFor(() => expect(navigation.navigate).toHaveBeenCalled());

    expect(navigation.navigate).toHaveBeenCalledWith('PostJob', expect.objectContaining({
      job: expect.objectContaining({ service_type: 'plumbing' }),
    }));
  });
});

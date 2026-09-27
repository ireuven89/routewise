import * as Location from 'expo-location';
import {
  getCurrentLocation, geocodeAddress, LocationPermissionError, AddressNotFoundError,
} from '../location';

afterEach(() => {
  jest.clearAllMocks();
});

describe('getCurrentLocation', () => {
  it('resolves latitude/longitude/address on success (reverse geocode available)', async () => {
    const result = await getCurrentLocation();

    expect(result.latitude).toBe(32.08);
    expect(result.longitude).toBe(34.78);
    expect(result.address).toBe('Herzl 1, Tel Aviv');
  });

  it('throws LocationPermissionError when permission is denied', async () => {
    Location.requestForegroundPermissionsAsync.mockResolvedValueOnce({ status: 'denied' });

    await expect(getCurrentLocation()).rejects.toBeInstanceOf(LocationPermissionError);
  });

  it('falls back to a "lat, lng" string when reverse geocoding throws', async () => {
    Location.reverseGeocodeAsync.mockRejectedValueOnce(new Error('not supported on web'));

    const result = await getCurrentLocation();

    expect(result.address).toBe('32.08000, 34.78000');
  });

  it('falls back to a "lat, lng" string when reverse geocoding returns no results', async () => {
    Location.reverseGeocodeAsync.mockResolvedValueOnce([]);

    const result = await getCurrentLocation();

    expect(result.address).toBe('32.08000, 34.78000');
  });
});

describe('geocodeAddress', () => {
  it('resolves latitude/longitude from the first result, keeping the given address string', async () => {
    const result = await geocodeAddress('1 Herzl St, Tel Aviv');

    expect(result).toEqual({ latitude: 32.08, longitude: 34.78, address: '1 Herzl St, Tel Aviv' });
  });

  it('throws AddressNotFoundError when there are no results', async () => {
    Location.geocodeAsync.mockResolvedValueOnce([]);

    await expect(geocodeAddress('nowhere')).rejects.toBeInstanceOf(AddressNotFoundError);
  });

  it('throws AddressNotFoundError when the geocoder itself throws', async () => {
    Location.geocodeAsync.mockRejectedValueOnce(new Error('boom'));

    await expect(geocodeAddress('nowhere')).rejects.toBeInstanceOf(AddressNotFoundError);
  });
});

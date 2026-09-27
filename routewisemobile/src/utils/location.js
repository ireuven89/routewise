import * as Location from 'expo-location';

export class LocationPermissionError extends Error {}
export class AddressNotFoundError extends Error {}

const formatAddress = (a) =>
  [a.street && `${a.street}${a.streetNumber ? ` ${a.streetNumber}` : ''}`, a.city]
    .filter(Boolean)
    .join(', ');

// Device GPS → {latitude, longitude, address}. Reverse geocoding is best-effort
// (it isn't available on web), so address may fall back to coordinates.
export const getCurrentLocation = async () => {
  const { status } = await Location.requestForegroundPermissionsAsync();
  if (status !== 'granted') throw new LocationPermissionError();

  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  const { latitude, longitude } = pos.coords;

  let address = '';
  try {
    const [first] = await Location.reverseGeocodeAsync({ latitude, longitude });
    if (first) address = formatAddress(first);
  } catch {
    // not supported on this platform
  }
  return { latitude, longitude, address: address || `${latitude.toFixed(5)}, ${longitude.toFixed(5)}` };
};

// Typed address → {latitude, longitude, address} via the platform geocoder (no Google key needed on iOS/Android).
export const geocodeAddress = async (address) => {
  let results = [];
  try {
    results = await Location.geocodeAsync(address);
  } catch {
    results = [];
  }
  if (!results.length) throw new AddressNotFoundError();
  return { latitude: results[0].latitude, longitude: results[0].longitude, address };
};

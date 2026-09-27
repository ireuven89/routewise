/* global jest */
jest.mock('@react-native-async-storage/async-storage', () =>
  require('@react-native-async-storage/async-storage/jest/async-storage-mock')
);

jest.mock('expo-location', () => ({
  requestForegroundPermissionsAsync: jest.fn(async () => ({ status: 'granted' })),
  getCurrentPositionAsync: jest.fn(async () => ({ coords: { latitude: 32.08, longitude: 34.78 } })),
  reverseGeocodeAsync: jest.fn(async () => [{ street: 'Herzl', streetNumber: '1', city: 'Tel Aviv' }]),
  geocodeAsync: jest.fn(async () => [{ latitude: 32.08, longitude: 34.78 }]),
  Accuracy: { Balanced: 3 },
}));

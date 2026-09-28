import AsyncStorage from '@react-native-async-storage/async-storage';
import api, { publicApi, storage } from '../api';

afterEach(() => {
  jest.restoreAllMocks();
  jest.clearAllMocks();
});

afterEach(async () => {
  await AsyncStorage.clear();
});

describe('publicApi', () => {
  describe('searchProviders', () => {
    it('GETs /public/providers with lat/lng/service_type params and returns the providers array', async () => {
      const getSpy = jest.spyOn(api, 'get').mockResolvedValue({ data: { providers: [{ id: 1 }] } });

      const result = await publicApi.searchProviders(32.08, 34.78, 'hvac');

      expect(getSpy).toHaveBeenCalledWith('/public/providers', {
        params: { lat: 32.08, lng: 34.78, service_type: 'hvac' },
      });
      expect(result).toEqual([{ id: 1 }]);
    });

    it('returns an empty array when the response has no providers field', async () => {
      jest.spyOn(api, 'get').mockResolvedValue({ data: {} });

      const result = await publicApi.searchProviders(1, 2, 'plumbing');

      expect(result).toEqual([]);
    });
  });

  describe('createRequest', () => {
    it('POSTs the payload to /public/service-requests and returns the response body', async () => {
      const responseData = { id: 5, access_token: 'tok-123', status: 'open', tracking_url: 'https://x' };
      const postSpy = jest.spyOn(api, 'post').mockResolvedValue({ data: responseData });
      const payload = { service_type: 'hvac', customer_name: 'Dana', customer_phone: '+972501234567' };

      const result = await publicApi.createRequest(payload);

      expect(postSpy).toHaveBeenCalledWith('/public/service-requests', payload);
      expect(result).toEqual(responseData);
    });
  });

  describe('getRequest', () => {
    it('GETs /public/service-requests/:token and returns {request, bids}', async () => {
      const responseData = { request: { id: 5, status: 'open' }, bids: [] };
      const getSpy = jest.spyOn(api, 'get').mockResolvedValue({ data: responseData });

      const result = await publicApi.getRequest('tok-123');

      expect(getSpy).toHaveBeenCalledWith('/public/service-requests/tok-123');
      expect(result).toEqual(responseData);
    });
  });

  describe('awardBid', () => {
    it('POSTs {bid_id} to /public/service-requests/:token/award', async () => {
      const responseData = { request: { status: 'awarded' }, bids: [] };
      const postSpy = jest.spyOn(api, 'post').mockResolvedValue({ data: responseData });

      const result = await publicApi.awardBid('tok-123', 42);

      expect(postSpy).toHaveBeenCalledWith('/public/service-requests/tok-123/award', { bid_id: 42 });
      expect(result).toEqual(responseData);
    });
  });
});

describe('storage', () => {
  describe('mode', () => {
    it('getMode returns null when nothing was saved', async () => {
      expect(await storage.getMode()).toBeNull();
    });

    it('setMode saves the mode and getMode reads it back', async () => {
      await storage.setMode('customer');
      expect(await storage.getMode()).toBe('customer');
    });

    it('setMode(null) removes the saved mode', async () => {
      await storage.setMode('technician');
      await storage.setMode(null);
      expect(await storage.getMode()).toBeNull();
    });
  });

  describe('myRequests', () => {
    it('getMyRequests returns [] when nothing was saved', async () => {
      expect(await storage.getMyRequests()).toEqual([]);
    });

    it('addMyRequest prepends a new entry (newest first)', async () => {
      await storage.addMyRequest({ token: 't1', id: 1 });
      const next = await storage.addMyRequest({ token: 't2', id: 2 });

      expect(next).toEqual([{ token: 't2', id: 2 }, { token: 't1', id: 1 }]);
      expect(await storage.getMyRequests()).toEqual(next);
    });

    it('addMyRequest dedupes by token, replacing the old entry at the front', async () => {
      await storage.addMyRequest({ token: 't1', id: 1, status: 'open' });
      await storage.addMyRequest({ token: 't2', id: 2, status: 'open' });
      const next = await storage.addMyRequest({ token: 't1', id: 1, status: 'awarded' });

      expect(next).toEqual([
        { token: 't1', id: 1, status: 'awarded' },
        { token: 't2', id: 2, status: 'open' },
      ]);
    });
  });

  describe('customerProfile', () => {
    it('getCustomerProfile returns null when nothing was saved', async () => {
      expect(await storage.getCustomerProfile()).toBeNull();
    });

    it('saveCustomerProfile round-trips through getCustomerProfile', async () => {
      const profile = { name: 'Dana', countryCode: '+972', phoneNumber: '501234567' };
      await storage.saveCustomerProfile(profile);
      expect(await storage.getCustomerProfile()).toEqual(profile);
    });
  });

  describe('clearAll', () => {
    it('removes token and worker but keeps myRequests, customerProfile and language', async () => {
      await storage.saveToken('tok');
      await storage.saveWorker({ id: 1, name: 'Worker' });
      await storage.addMyRequest({ token: 't1', id: 1 });
      await storage.saveCustomerProfile({ name: 'Dana' });
      await AsyncStorage.setItem('language', 'en');

      await storage.clearAll();

      expect(await storage.getToken()).toBeNull();
      expect(await storage.getWorker()).toBeNull();
      expect(await storage.getMyRequests()).toEqual([{ token: 't1', id: 1 }]);
      expect(await storage.getCustomerProfile()).toEqual({ name: 'Dana' });
      expect(await AsyncStorage.getItem('language')).toBe('en');
    });
  });
});

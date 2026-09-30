import AsyncStorage from '@react-native-async-storage/async-storage';
import api, { jobs, publicApi, storage } from '../api';

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

describe('jobs (technician accept/decline)', () => {
  it('accept POSTs /jobs/:id/accept and returns the body', async () => {
    const postSpy = jest.spyOn(api, 'post').mockResolvedValue({ data: { message: 'Job accepted' } });

    const result = await jobs.accept(42);

    expect(postSpy).toHaveBeenCalledWith('/jobs/42/accept');
    expect(result).toEqual({ message: 'Job accepted' });
  });

  it('decline POSTs the reason to /jobs/:id/decline', async () => {
    const postSpy = jest.spyOn(api, 'post').mockResolvedValue({ data: { message: 'Job declined' } });

    const result = await jobs.decline(42, 'Too far away');

    expect(postSpy).toHaveBeenCalledWith('/jobs/42/decline', { reason: 'Too far away' });
    expect(result).toEqual({ message: 'Job declined' });
  });

  it.each([undefined, null, ''])('decline sends an empty reason when given %p', async (reason) => {
    const postSpy = jest.spyOn(api, 'post').mockResolvedValue({ data: {} });

    await jobs.decline(7, reason);

    expect(postSpy).toHaveBeenCalledWith('/jobs/7/decline', { reason: '' });
  });

  it('propagates server errors (e.g. 409 invalid assignment state)', async () => {
    const err = Object.assign(new Error('Request failed with status code 409'), {
      response: { status: 409, data: { error: 'job is not in a state that allows this' } },
    });
    jest.spyOn(api, 'post').mockRejectedValue(err);

    await expect(jobs.accept(1)).rejects.toBe(err);
  });

  it('updateStatus PATCHes /jobs/:id/status', async () => {
    const patchSpy = jest.spyOn(api, 'patch').mockResolvedValue({ data: { message: 'ok' } });

    await jobs.updateStatus(3, 'in_progress');

    expect(patchSpy).toHaveBeenCalledWith('/jobs/3/status', { status: 'in_progress' });
  });

  it('getMyJobs GETs /jobs (the server scopes it to the technician)', async () => {
    const getSpy = jest.spyOn(api, 'get').mockResolvedValue({ data: [{ id: 1 }] });

    expect(await jobs.getMyJobs()).toEqual([{ id: 1 }]);
    expect(getSpy).toHaveBeenCalledWith('/jobs');
  });
});

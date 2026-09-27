// RouteWiseMobile/src/services/api.js
// API Service for RouteWise Backend

import axios from 'axios';
import AsyncStorage from '@react-native-async-storage/async-storage';

// EXPO_PUBLIC_API_URL overrides (e.g. http://localhost:18080/api/v1 for the run-routewise stack).
export const API_URL = process.env.EXPO_PUBLIC_API_URL || (__DEV__
    ? 'http://192.168.1.191:8080/api/v1'
    : 'https://api.routewisehq.com/api/v1');

// Create axios instance
const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 10000,
});

// Request interceptor - Add auth token
api.interceptors.request.use(
  async (config) => {
    const token = await AsyncStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor - Handle errors
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Token expired - logout
      await AsyncStorage.removeItem('token');
      await AsyncStorage.removeItem('worker');
    }
    return Promise.reject(error);
  }
);

// Auth API
export const auth = {
  requestOTP: async (companyCode, phone) => {
    const response = await api.post('/workers/request-otp', {
      company_code: companyCode,
      phone: phone,
    });
    return response.data;
  },

  verifyOTP: async (companyCode, phone, code) => {
    const response = await api.post('/worker/verify-otp', {
      company_code: companyCode,
      phone: phone,
      code: code,
    });
    return response.data;
  },
};

// Jobs API (backend uses /jobs endpoint)
export const jobs = {
  // Get all jobs assigned to the current worker
  getMyJobs: async () => {
    const response = await api.get('/jobs');
    return response.data;
  },

  // Get job details by ID
  getJobDetails: async (jobId) => {
    const response = await api.get(`/jobs/${jobId}`);
    return response.data;
  },

  // Update job status
  updateStatus: async (jobId, status) => {
    const response = await api.patch(`/jobs/${jobId}/status`, { status });
    return response.data;
  },

  // Get job files
  getFiles: async (jobId) => {
    const response = await api.get(`/projects/${jobId}/files`);
    return response.data;
  },
};

// Legacy alias for backward compatibility
export const projects = jobs;

// Files API
export const files = {
  upload: async (projectId, formData) => {
    const response = await api.post(`/projects/${projectId}/files`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 30000, // 30 seconds for file upload
    });
    return response.data;
  },
  
  getById: async (fileId) => {
    const response = await api.get(`/files/${fileId}`);
    return response.data;
  },
};

// Public customer API (no auth) — same endpoints as the web /find-service flow
// (frontend/src/api/client.js providersAPI + serviceRequestsAPI).
export const publicApi = {
  searchProviders: async (lat, lng, serviceType) => {
    const response = await api.get('/public/providers', {
      params: { lat, lng, service_type: serviceType },
    });
    return response.data.providers || [];
  },

  // data: {service_type, description, customer_name, customer_phone, latitude, longitude, address, preferred_time}
  // returns {id, access_token, status, tracking_url}
  createRequest: async (data) => {
    const response = await api.post('/public/service-requests', data);
    return response.data;
  },

  // returns {request, bids}
  getRequest: async (token) => {
    const response = await api.get(`/public/service-requests/${token}`);
    return response.data;
  },

  awardBid: async (token, bidId) => {
    const response = await api.post(`/public/service-requests/${token}/award`, { bid_id: bidId });
    return response.data;
  },
};

const MODE_KEY = 'mode';
const MY_REQUESTS_KEY = 'myRequests';
const CUSTOMER_PROFILE_KEY = 'customerProfile';

// Helper functions
export const storage = {
  // 'customer' | 'technician' | null (null = show role selection)
  getMode: async () => AsyncStorage.getItem(MODE_KEY),

  setMode: async (mode) => {
    if (mode) await AsyncStorage.setItem(MODE_KEY, mode);
    else await AsyncStorage.removeItem(MODE_KEY);
  },

  // Customers have no account: each posted request's access_token is kept on the device.
  getMyRequests: async () => {
    const raw = await AsyncStorage.getItem(MY_REQUESTS_KEY);
    return raw ? JSON.parse(raw) : [];
  },

  addMyRequest: async (entry) => {
    const existing = await storage.getMyRequests();
    const next = [entry, ...existing.filter((r) => r.token !== entry.token)];
    await AsyncStorage.setItem(MY_REQUESTS_KEY, JSON.stringify(next));
    return next;
  },

  getCustomerProfile: async () => {
    const raw = await AsyncStorage.getItem(CUSTOMER_PROFILE_KEY);
    return raw ? JSON.parse(raw) : null;
  },

  saveCustomerProfile: async (profile) => {
    await AsyncStorage.setItem(CUSTOMER_PROFILE_KEY, JSON.stringify(profile));
  },

  saveToken: async (token) => {
    await AsyncStorage.setItem('token', token);
  },
  
  getToken: async () => {
    return await AsyncStorage.getItem('token');
  },
  
  saveWorker: async (worker) => {
    await AsyncStorage.setItem('worker', JSON.stringify(worker));
  },
  
  getWorker: async () => {
    const worker = await AsyncStorage.getItem('worker');
    return worker ? JSON.parse(worker) : null;
  },
  
  // Worker logout. Leaves the customer's saved requests, profile and language alone.
  clearAll: async () => {
    await AsyncStorage.multiRemove(['token', 'worker']);
  },
};

export default api;

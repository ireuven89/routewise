// RouteWiseMobile/src/services/api.js
// API Service for RouteWise Backend

import axios from 'axios';
import AsyncStorage from '@react-native-async-storage/async-storage';

const API_URL = __DEV__
    ? 'http://10.100.102.6:8080/api/v1'
    : 'https://api.routewisehq.com/api/v1';

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
    const response = await api.post('/workers/verify-otp', {
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

// Helper functions
export const storage = {
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
  
  clearAll: async () => {
    await AsyncStorage.multiRemove(['token', 'worker']);
  },
};

export default api;

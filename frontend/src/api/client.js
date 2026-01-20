import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const apiClient = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Add auth token to requests
apiClient.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// Handle 401 errors (token expired)
apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

// Auth API
export const authAPI = {
    register: (data) => apiClient.post('/api/v1/register', data),
    login: (data) => apiClient.post('/api/v1/login', data),
    getCurrentUser: () => apiClient.get('/api/v1/me'),
};

// Jobs API
export const jobsAPI = {
    getAll: (params) => apiClient.get('/api/v1/jobs', { params }),
    getById: (id) => apiClient.get(`/api/v1/jobs/${id}`),
    create: (data) => apiClient.post('/api/v1/jobs', data),
    update: (id, data) => apiClient.put(`/api/v1/jobs/${id}`, data),
    delete: (id) => apiClient.delete(`/api/v1/jobs/${id}`),
    assignTechnician: (id, technicianId) =>
        apiClient.patch(`/api/v1/jobs/${id}/assign`, { technician_id: technicianId }),
    updateStatus: (id, status) => {
        console.log('🔍 Calling updateStatus API:', { id, status });
        return apiClient.patch(`/api/v1/jobs/${id}/status`, { status });
    },
};

// Customers API
export const customersAPI = {
    getAll: (search) => apiClient.get('/api/v1/customers', { params: { search } }),
    getById: (id) => apiClient.get(`/api/v1/customers/${id}`),
    create: (data) => apiClient.post('/api/v1/customers', data),
    update: (id, data) => apiClient.put(`/api/v1/customers/${id}`, data),
    delete: (id) => apiClient.delete(`/api/v1/customers/${id}`),
};

// Technicians API
export const techniciansAPI = {
    getAll: (activeOnly) => apiClient.get('/api/v1/technicians', {
        params: { active_only: activeOnly }
    }),
    getById: (id) => apiClient.get(`/api/v1/technicians/${id}`),
    create: (data) => apiClient.post('/api/v1/technicians', data),
    update: (id, data) => apiClient.put(`/api/v1/technicians/${id}`, data),
    delete: (id) => apiClient.delete(`/api/v1/technicians/${id}`),
};

export default apiClient;
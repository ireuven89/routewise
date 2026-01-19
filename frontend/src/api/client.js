import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

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
    register: (data) => apiClient.post('/register', data),
    login: (data) => apiClient.post('/login', data),
    getCurrentUser: () => apiClient.get('/me'),
};

// Jobs API
export const jobsAPI = {
    getAll: (params) => apiClient.get('/jobs', { params }),
    getById: (id) => apiClient.get(`/jobs/${id}`),  // ✅ Parentheses
    create: (data) => apiClient.post('/jobs', data),
    update: (id, data) => apiClient.put(`/jobs/${id}`, data),  // ✅ Parentheses
    delete: (id) => apiClient.delete(`/jobs/${id}`),  // ✅ Parentheses
    assignTechnician: (id, technicianId) =>
        apiClient.patch(`/jobs/${id}/assign`, { technician_id: technicianId }),  // ✅ Parentheses
    updateStatus: (id, status) => {
        console.log('🔍 Calling updateStatus API:', { id, status });
        return apiClient.patch(`/jobs/${id}/status`, { status });  // ✅ Simplified
    },
};

// Customers API
export const customersAPI = {
    getAll: (search) => apiClient.get('/customers', { params: { search } }),
    getById: (id) => apiClient.get(`/customers/${id}`),  // ✅ Parentheses
    create: (data) => apiClient.post('/customers', data),
    update: (id, data) => apiClient.put(`/customers/${id}`, data),  // ✅ Parentheses
    delete: (id) => apiClient.delete(`/customers/${id}`),  // ✅ Parentheses
};

// Technicians API
export const techniciansAPI = {
    getAll: (activeOnly) => apiClient.get('/technicians', {
        params: { active_only: activeOnly }
    }),
    getById: (id) => apiClient.get(`/technicians/${id}`),  // ✅ Parentheses
    create: (data) => apiClient.post('/technicians', data),
    update: (id, data) => apiClient.put(`/technicians/${id}`, data),  // ✅ Parentheses
    delete: (id) => apiClient.delete(`/technicians/${id}`),  // ✅ Parentheses
};

export default apiClient;
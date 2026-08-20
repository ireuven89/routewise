import React, { createContext, useState, useContext, useEffect } from 'react';
import { authAPI } from '../api/client';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [organization, setOrganization] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        // Check if user is already logged in
        const token = localStorage.getItem('token');
        const savedUser = localStorage.getItem('user');
        const savedOrganization = localStorage.getItem('organization');

        if (token && savedUser) {
            setUser(JSON.parse(savedUser));
        }
        if (token && savedOrganization) {
            setOrganization(JSON.parse(savedOrganization));
        }

        setLoading(false);
    }, []);

    const login = async (email, password) => {
        try {
            const response = await authAPI.login({ email, password });
            const { token, user, organization } = response.data;

            localStorage.setItem('token', token);
            localStorage.setItem('user', JSON.stringify(user));
            setUser(user);
            if (organization) {
                localStorage.setItem('organization', JSON.stringify(organization));
                setOrganization(organization);
            }

            return { success: true };
        } catch (error) {
            return {
                success: false,
                error: error.response?.data?.error || 'Login failed'
            };
        }
    };

    const register = async (data) => {
        try {
            const response = await authAPI.register(data);
            const { token, user, organization } = response.data;

            localStorage.setItem('token', token);
            localStorage.setItem('user', JSON.stringify(user));
            setUser(user);
            if (organization) {
                localStorage.setItem('organization', JSON.stringify(organization));
                setOrganization(organization);
            }

            return { success: true };
        } catch (error) {
            return {
                success: false,
                error: error.response?.data?.error || 'Registration failed'
            };
        }
    };

    const logout = () => {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('organization');
        setUser(null);
        setOrganization(null);
    };

    const updateOrganization = (patch) => {
        setOrganization((prev) => {
            const next = { ...prev, ...patch };
            localStorage.setItem('organization', JSON.stringify(next));
            return next;
        });
    };

    return (
        <AuthContext.Provider value={{ user, organization, login, register, logout, updateOrganization, loading }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within AuthProvider');
    }
    return context;
};
import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { FaCalendarAlt } from 'react-icons/fa';
import { authAPI } from "../api/client";
import { formatPhone } from "../utils/phone";

const COUNTRY_CODES = [
    { code: '+1', country: 'US/Canada', flag: '🇺🇸' },
    { code: '+972', country: 'Israel', flag: '🇮🇱' },
    { code: '+44', country: 'UK', flag: '🇬🇧' },
    { code: '+61', country: 'Australia', flag: '🇦🇺' },
    { code: '+91', country: 'India', flag: '🇮🇳' },
    { code: '+49', country: 'Germany', flag: '🇩🇪' },
    { code: '+33', country: 'France', flag: '🇫🇷' },
    { code: '+52', country: 'Mexico', flag: '🇲🇽' },
];

const Register = () => {
    const navigate = useNavigate();
    const [formData, setFormData] = useState({
        name: '',
        countryCode: '+1',
        phoneNumber: '',
        companyName: '',
        email: '',
        password: '',
        confirmPassword: '',
    });
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleChange = (e) => {
        setFormData({
            ...formData,
            [e.target.name]: e.target.value,
        });
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');

        if (formData.password !== formData.confirmPassword) {
            setError('Passwords do not match');
            return;
        }

        if (formData.password.length < 6) {
            setError('Password must be at least 6 characters');
            return;
        }

        setLoading(true);

        try {
            const response = await authAPI.register({
                name: formData.name,
                phone: formatPhone(formData.countryCode, formData.phoneNumber),
                company_name: formData.companyName,
                email: formData.email,
                password: formData.password,
            });
            const data = response.data;

            localStorage.setItem('token', data.token);
            localStorage.setItem('user', JSON.stringify(data.user));

            window.location.href = '/dashboard';

        } catch (err) {
            setError(err.response?.data?.error || err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden" style={{ background: 'linear-gradient(135deg, #1e3a5f 0%, #1a3350 50%, #152a42 100%)' }}>
            {/* Background Pattern */}
            <div className="absolute inset-0 opacity-10">
                <div className="absolute top-0 left-0 w-96 h-96 bg-white rounded-full blur-3xl transform -translate-x-1/2 -translate-y-1/2"></div>
                <div className="absolute bottom-0 right-0 w-96 h-96 rounded-full blur-3xl transform translate-x-1/2 translate-y-1/2" style={{ backgroundColor: '#ff6b35' }}></div>
            </div>

            {/* Register Card */}
            <div className="w-full max-w-md relative z-10">
                {/* Logo */}
                <div className="flex items-center justify-center space-x-3 mb-8">
                    <div className="bg-white p-3 rounded-xl shadow-lg">
                        <FaCalendarAlt className="text-3xl" style={{ color: '#1e3a5f' }} />
                    </div>
                    <span className="text-white text-4xl font-bold">RouteWise</span>
                </div>

                {/* Card */}
                <div className="bg-white rounded-2xl shadow-2xl p-8">
                    {/* Header */}
                    <div className="mb-8 text-center">
                        <h2 className="text-3xl font-bold text-gray-900 mb-2">Create your account</h2>
                        <p className="text-gray-600">Get started with RouteWise</p>
                    </div>

                    {/* Error Message */}
                    {error && (
                        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
                            <p className="text-red-600 text-sm">{error}</p>
                        </div>
                    )}

                    {/* Register Form */}
                    <form onSubmit={handleSubmit} className="space-y-5">
                        {/* Company Name */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Company Name
                            </label>
                            <input
                                type="text"
                                name="companyName"
                                value={formData.companyName}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                style={{ focusRingColor: '#ff6b35' }}
                                placeholder="ABC HVAC Services"
                            />
                        </div>

                        {/* Your Name */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Your Name
                            </label>
                            <input
                                type="text"
                                name="name"
                                value={formData.name}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder="John Smith"
                            />
                        </div>

                        {/* Phone Number with Country Code */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Phone Number
                            </label>
                            <div className="flex gap-2">
                                {/* Country Code Dropdown */}
                                <select
                                    name="countryCode"
                                    value={formData.countryCode}
                                    onChange={handleChange}
                                    className="px-3 py-3 border border-gray-300 rounded-lg bg-gray-50 focus:ring-2 focus:border-transparent transition-all cursor-pointer"
                                    style={{ minWidth: '100px' }}
                                >
                                    {COUNTRY_CODES.map((item) => (
                                        <option key={item.code} value={item.code}>
                                            {item.flag} {item.code}
                                        </option>
                                    ))}
                                </select>

                                {/* Phone Input */}
                                <input
                                    type="tel"
                                    name="phoneNumber"
                                    value={formData.phoneNumber}
                                    onChange={handleChange}
                                    required
                                    className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                    placeholder="501234567"
                                />
                            </div>
                            <p className="text-xs text-gray-500 mt-1">
                                Full number: {formatPhone(formData.countryCode, formData.phoneNumber)}
                            </p>
                        </div>

                        {/* Email */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Email Address
                            </label>
                            <input
                                type="email"
                                name="email"
                                value={formData.email}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder="you@company.com"
                            />
                        </div>

                        {/* Password */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Password
                            </label>
                            <input
                                type="password"
                                name="password"
                                value={formData.password}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder="At least 6 characters"
                            />
                        </div>

                        {/* Confirm Password */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                Confirm Password
                            </label>
                            <input
                                type="password"
                                name="confirmPassword"
                                value={formData.confirmPassword}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder="Confirm your password"
                            />
                        </div>

                        {/* Submit */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full text-white py-3 rounded-lg font-semibold hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg transform hover:-translate-y-0.5 transition-all"
                            style={{ backgroundColor: '#ff6b35' }}
                        >
                            {loading ? 'Creating account...' : 'Create account'}
                        </button>
                    </form>

                    {/* Divider */}
                    <div className="mt-8 mb-6">
                        <div className="relative">
                            <div className="absolute inset-0 flex items-center">
                                <div className="w-full border-t border-gray-300"></div>
                            </div>
                            <div className="relative flex justify-center text-sm">
                                <span className="px-4 bg-white text-gray-500">Already have an account?</span>
                            </div>
                        </div>
                    </div>

                    {/* Sign In Link */}
                    <Link
                        to="/login"
                        className="block w-full text-center bg-gray-50 border border-gray-300 text-gray-700 py-3 rounded-lg font-semibold hover:bg-gray-100 hover:border-gray-400 transition-all"
                    >
                        Sign in
                    </Link>
                </div>

                {/* Footer */}
                <p className="text-center text-sm mt-6" style={{ color: 'rgba(255,255,255,0.7)' }}>
                    © 2026 RouteWise. HVAC scheduling software.
                </p>
            </div>
        </div>
    );
};

export default Register;
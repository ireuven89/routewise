import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { FaCalendarAlt, FaGlobe } from 'react-icons/fa';
import { authAPI } from "../api/client";
import { useLanguage } from '../context/LanguageContext';

const Login = () => {
    const navigate = useNavigate();
    const { t, language, toggleLanguage } = useLanguage();
    const [formData, setFormData] = useState({
        email: '',
        password: '',
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
        setLoading(true);

        try {
            const response = await authAPI.login(formData);
            const data = response.data;

            localStorage.setItem('token', data.token);
            localStorage.setItem('user', JSON.stringify(data.user));
            if (data.organization) {
                localStorage.setItem('organization', JSON.stringify(data.organization));
            }

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

            {/* Login Card */}
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

                    {/* Language toggle */}
                    <div className="flex justify-end mb-4">
                        <button
                            type="button"
                            onClick={toggleLanguage}
                            className="flex items-center gap-1.5 text-xs font-medium text-gray-500 hover:text-gray-700 transition-colors"
                        >
                            <FaGlobe className="w-3.5 h-3.5" />
                            {language === 'he' ? 'English' : 'עברית'}
                        </button>
                    </div>

                    {/* Header */}
                    <div className="mb-8 text-center">
                        <h2 className="text-3xl font-bold text-gray-900 mb-2">{t('auth.welcomeBack')}</h2>
                        <p className="text-gray-600">{t('auth.signInTo')}</p>
                    </div>

                    {/* Error Message */}
                    {error && (
                        <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
                            <p className="text-red-600 text-sm">{error}</p>
                        </div>
                    )}

                    {/* Login Form */}
                    <form onSubmit={handleSubmit} className="space-y-5">
                        {/* Email */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                {t('auth.email')}
                            </label>
                            <input
                                type="email"
                                name="email"
                                value={formData.email}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder={t('auth.emailPlaceholder')}
                            />
                        </div>

                        {/* Password */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                {t('auth.password')}
                            </label>
                            <input
                                type="password"
                                name="password"
                                value={formData.password}
                                onChange={handleChange}
                                required
                                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:border-transparent transition-all"
                                placeholder={t('auth.passwordPlaceholder')}
                            />
                        </div>

                        {/* Submit */}
                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full text-white py-3 rounded-lg font-semibold hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg transform hover:-translate-y-0.5 transition-all"
                            style={{ backgroundColor: '#ff6b35' }}
                        >
                            {loading ? t('auth.signingIn') : t('auth.signIn')}
                        </button>
                    </form>

                    {/* Divider */}
                    <div className="mt-8 mb-6">
                        <div className="relative">
                            <div className="absolute inset-0 flex items-center">
                                <div className="w-full border-t border-gray-300"></div>
                            </div>
                            <div className="relative flex justify-center text-sm">
                                <span className="px-4 bg-white text-gray-500">{t('auth.dontHaveAccount')}</span>
                            </div>
                        </div>
                    </div>

                    {/* Register Link */}
                    <Link
                        to="/register"
                        className="block w-full text-center bg-gray-50 border border-gray-300 text-gray-700 py-3 rounded-lg font-semibold hover:bg-gray-100 hover:border-gray-400 transition-all"
                    >
                        {t('auth.createAccountBtn')}
                    </Link>
                </div>

                {/* Footer */}
                <p className="text-center text-sm mt-6" style={{ color: 'rgba(255,255,255,0.7)' }}>
                    {t('footer.copyright')}
                </p>
            </div>
        </div>
    );
};

export default Login;

import React, { useState, useEffect, useCallback } from 'react';
import { FaCheckCircle, FaPhone } from 'react-icons/fa';
import axios from 'axios';
import { serviceCallsAPI } from '../api/client';
import { useLanguage } from '../context/LanguageContext';
import GooglePlacesAutocomplete from './GooglePlacesAutocomplete';
import { loadGoogleMapsScript } from '../utils/googleMaps';

const ServiceCallModal = ({ workers, onSuccess, onClose }) => {
    const { t } = useLanguage();

    const [formData, setFormData] = useState({
        customerName: '',
        phone: '',
        email: '',
        address: '',
        latitude: null,
        longitude: null,
        google_place_id: '',
        formatted_address: '',
        address_components: null,
        title: '',
        description: '',
        scheduledDate: '',
        priority: 'medium',
        technicianId: '',
    });
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');
    const [successData, setSuccessData] = useState(null);
    const [googleMapsKey, setGoogleMapsKey] = useState('');
    const [googleMapsLoaded, setGoogleMapsLoaded] = useState(false);

    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.key === 'Escape') onClose();
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onClose]);

    useEffect(() => {
        const loadGoogleMaps = async () => {
            try {
                const token = localStorage.getItem('token');
                const response = await axios.get(`${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/config/google-maps`, {
                    headers: { Authorization: `Bearer ${token}` },
                });
                if (response.data.enabled && response.data.api_key) {
                    setGoogleMapsKey(response.data.api_key);
                    await loadGoogleMapsScript(response.data.api_key);
                    setGoogleMapsLoaded(true);
                }
            } catch {
                setGoogleMapsLoaded(false);
            }
        };
        loadGoogleMaps();
    }, []);

    const handleChange = (e) => {
        const { name, value } = e.target;
        setFormData((prev) => ({ ...prev, [name]: value }));
    };

    const handleAddressSelect = useCallback((addressData) => {
        setFormData((prev) => ({
            ...prev,
            address: addressData.address || addressData.formattedAddress || '',
            latitude: addressData.latitude || null,
            longitude: addressData.longitude || null,
            google_place_id: addressData.placeId || '',
            formatted_address: addressData.formattedAddress || '',
            address_components: addressData.components || null,
        }));
    }, []);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setSubmitting(true);
        setError('');

        try {
            const payload = {
                customer: {
                    name: formData.customerName,
                    phone: formData.phone,
                    email: formData.email,
                    address: formData.address,
                    latitude: formData.latitude,
                    longitude: formData.longitude,
                    google_place_id: formData.google_place_id,
                    formatted_address: formData.formatted_address,
                    address_components: formData.address_components,
                },
                job: {
                    title: formData.title,
                    description: formData.description,
                    scheduled_date: new Date(formData.scheduledDate).toISOString(),
                    priority: formData.priority,
                    status: 'scheduled',
                },
                technician_id: formData.technicianId ? Number(formData.technicianId) : null,
            };

            const res = await serviceCallsAPI.create(payload);
            setSuccessData(res.data);
        } catch (err) {
            setError(err.response?.data?.error || t('serviceCall.error'));
        } finally {
            setSubmitting(false);
        }
    };

    const activeWorkers = (workers || []).filter((w) => w.is_active);

    // ── Success State ────────────────────────────────────────────────────────
    if (successData) {
        return (
            <div
                className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50"
                onClick={onClose}
            >
                <div
                    className="bg-white rounded-lg max-w-md w-full"
                    onClick={(e) => e.stopPropagation()}
                >
                    <div className="px-6 py-10 text-center">
                        <div className="w-16 h-16 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-4">
                            <FaCheckCircle className="w-8 h-8 text-emerald-500" />
                        </div>
                        <h3 className="text-xl font-semibold text-gray-900 mb-2">
                            {t('serviceCall.successTitle')}
                        </h3>
                        <p className="text-gray-600 mb-1">
                            {t('serviceCall.successMessage', { jobId: successData.job_id })}
                        </p>
                        <p className="text-sm text-gray-500">
                            {successData.customer_created
                                ? t('serviceCall.customerNew')
                                : t('serviceCall.customerExisting')}
                        </p>
                        <button
                            onClick={() => onSuccess(successData)}
                            className="mt-6 px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium"
                        >
                            {t('serviceCall.done')}
                        </button>
                    </div>
                </div>
            </div>
        );
    }

    // ── Form State ───────────────────────────────────────────────────────────
    return (
        <div
            className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50"
            onClick={onClose}
        >
            <div
                className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="px-6 py-4 border-b border-gray-200 flex items-center gap-3">
                    <div className="w-9 h-9 bg-orange-100 rounded-lg flex items-center justify-center">
                        <FaPhone className="w-4 h-4 text-orange-600" />
                    </div>
                    <h2 className="text-xl font-semibold text-gray-900">
                        {t('serviceCall.title')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    {/* Error banner */}
                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md text-sm">
                            {error}
                        </div>
                    )}

                    {/* ── Customer Information ────────────────────────────── */}
                    <div>
                        <h3 className="text-sm font-semibold text-gray-900 mb-3 uppercase tracking-wide">
                            {t('serviceCall.customerInfo')}
                        </h3>
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.customerName')}
                                </label>
                                <input
                                    type="text"
                                    name="customerName"
                                    value={formData.customerName}
                                    onChange={handleChange}
                                    required
                                    placeholder={t('serviceCall.customerNamePlaceholder')}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.phone')}
                                </label>
                                <input
                                    type="tel"
                                    name="phone"
                                    value={formData.phone}
                                    onChange={handleChange}
                                    required
                                    placeholder={t('serviceCall.phonePlaceholder')}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.email')}
                                </label>
                                <input
                                    type="email"
                                    name="email"
                                    value={formData.email}
                                    onChange={handleChange}
                                    placeholder={t('serviceCall.emailPlaceholder')}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.address')}
                                </label>
                                {googleMapsLoaded && googleMapsKey ? (
                                    <GooglePlacesAutocomplete
                                        onChange={handleAddressSelect}
                                        placeholder={t('serviceCall.addressPlaceholder')}
                                        apiKey={googleMapsKey}
                                    />
                                ) : (
                                    <input
                                        type="text"
                                        name="address"
                                        value={formData.address}
                                        onChange={handleChange}
                                        placeholder={t('serviceCall.addressPlaceholder')}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                )}
                            </div>
                        </div>
                    </div>

                    {/* ── Job Details ──────────────────────────────────────── */}
                    <div>
                        <h3 className="text-sm font-semibold text-gray-900 mb-3 uppercase tracking-wide">
                            {t('serviceCall.jobDetails')}
                        </h3>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.jobTitle')}
                                </label>
                                <input
                                    type="text"
                                    name="title"
                                    value={formData.title}
                                    onChange={handleChange}
                                    required
                                    placeholder={t('serviceCall.jobTitlePlaceholder')}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.description')}
                                </label>
                                <textarea
                                    name="description"
                                    value={formData.description}
                                    onChange={handleChange}
                                    rows={3}
                                    placeholder={t('serviceCall.descriptionPlaceholder')}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                />
                            </div>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">
                                        {t('serviceCall.scheduledDate')}
                                    </label>
                                    <input
                                        type="datetime-local"
                                        name="scheduledDate"
                                        value={formData.scheduledDate}
                                        onChange={handleChange}
                                        required
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700">
                                        {t('serviceCall.priority')}
                                    </label>
                                    <select
                                        name="priority"
                                        value={formData.priority}
                                        onChange={handleChange}
                                        className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                    >
                                        <option value="low">{t('serviceCall.priorityLow')}</option>
                                        <option value="medium">{t('serviceCall.priorityMedium')}</option>
                                        <option value="high">{t('serviceCall.priorityHigh')}</option>
                                    </select>
                                </div>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700">
                                    {t('serviceCall.technician')}
                                </label>
                                <select
                                    name="technicianId"
                                    value={formData.technicianId}
                                    onChange={handleChange}
                                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                                >
                                    <option value="">{t('serviceCall.selectTechnician')}</option>
                                    {activeWorkers.map((w) => (
                                        <option key={w.id} value={w.id}>
                                            {w.name}
                                        </option>
                                    ))}
                                </select>
                            </div>
                        </div>
                    </div>

                    {/* ── Buttons ──────────────────────────────────────────── */}
                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            {t('serviceCall.cancel')}
                        </button>
                        <button
                            type="submit"
                            disabled={submitting}
                            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                        >
                            {submitting ? t('serviceCall.submitting') : t('serviceCall.submit')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default ServiceCallModal;

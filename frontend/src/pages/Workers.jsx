import { useState, useEffect, useCallback } from 'react';
import { workersAPI } from '../api/client';
import Layout from '../components/Layout';
import { formatPhone } from "../utils/phone";
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import GooglePlacesAutocomplete from '../components/GooglePlacesAutocomplete';
import { loadGoogleMapsScript } from '../utils/googleMaps';
import axios from 'axios';

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

const splitPhone = (fullPhone) => {
    if (!fullPhone) return { countryCode: '+1', phoneNumber: '' };
    const match = COUNTRY_CODES.find((c) => fullPhone.startsWith(c.code));
    if (match) {
        return {
            countryCode: match.code,
            phoneNumber: fullPhone.slice(match.code.length),
        };
    }
    return { countryCode: '+1', phoneNumber: fullPhone };
};

const Workers = () => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const [workers, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingWorker, setEditingWorker] = useState(null);
    const [showActiveOnly, setShowActiveOnly] = useState(true);

    useEffect(() => {
        loadWorkers();
    }, [showActiveOnly]);

    const loadWorkers = async () => {
        try {
            const response = await workersAPI.getAll(showActiveOnly);
            setWorkers(response.data || []);
            setLoading(false);
        } catch (error) {
            console.error('Failed to load workers:', error);
            setLoading(false);
        }
    };

    const handleCreate = async (workerData) => {
        try {
            await workersAPI.create(workerData);
            await loadWorkers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create worker:', error);
            alert(t('workers.failedCreate'));
        }
    };

    const handleUpdate = async (workerData) => {
        try {
            await workersAPI.update(editingWorker.id, workerData);
            await loadWorkers();
            setEditingWorker(null);
        } catch (error) {
            console.error('Failed to update worker:', error);
            alert(t('workers.failedUpdate'));
        }
    };

    const handleDelete = async (workerId) => {
        if (!window.confirm(t('workers.deleteConfirm'))) return;

        try {
            await workersAPI.delete(workerId);
            await loadWorkers();
        } catch (error) {
            console.error('Failed to delete worker:', error);
            alert(t('workers.failedDelete'));
        }
    };

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">{t('workers.loading')}</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">{t(`industry.${industry}.workers`)}</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        style={{ backgroundColor: '#ff6b35' }}
                        className="hover:opacity-90 text-white px-4 py-2 rounded-md font-medium"
                    >
                        {t(`industry.${industry}.addWorker`)}
                    </button>
                </div>

                {/* Filter */}
                <div className="mb-6">
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={showActiveOnly}
                            onChange={(e) => setShowActiveOnly(e.target.checked)}
                            className="rounded border-gray-300"
                            style={{ accentColor: '#1e3a5f' }}
                        />
                        <span className="ml-2 text-sm text-gray-700">{t('workers.showActiveOnly')}</span>
                    </label>
                </div>

                {/* Workers List */}
                {workers.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">{t('workers.noWorkers')}</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {workers.map((worker) => (
                                <li key={worker.id} className="px-6 py-4 hover:bg-gray-50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center">
                                                <h3 className="text-lg font-medium text-gray-900">{worker.name}</h3>
                                                {worker.is_active ? (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                                                        {t('workers.active')}
                                                    </span>
                                                ) : (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded-full">
                                                        {t('workers.inactive')}
                                                    </span>
                                                )}
                                                {worker.role && (
                                                    <span
                                                        className="ml-3 px-2 py-1 text-xs font-medium rounded-full text-white"
                                                        style={{ backgroundColor: '#1e3a5f' }}
                                                    >
                                                        {worker.role}
                                                    </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500 mt-1">
                                                📞 {worker.phone}
                                                {worker.email && ` • ✉️ ${worker.email}`}
                                            </p>
                                            {worker.home_address && (
                                                <p className="text-sm text-gray-600 mt-1">
                                                    🏠 {worker.home_address}
                                                </p>
                                            )}
                                        </div>
                                        <div className="flex gap-3">
                                            <button
                                                onClick={() => setEditingWorker(worker)}
                                                style={{ color: '#1e3a5f' }}
                                                className="hover:opacity-70 font-medium"
                                            >
                                                {t('jobs.edit')}
                                            </button>
                                            <button
                                                onClick={() => handleDelete(worker.id)}
                                                className="text-red-600 hover:text-red-800 font-medium"
                                            >
                                                {t('jobs.delete')}
                                            </button>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {/* Create Modal */}
                {showModal && <WorkerModal onSave={handleCreate} onClose={() => setShowModal(false)} />}

                {/* Edit Modal */}
                {editingWorker && (
                    <WorkerModal
                        worker={editingWorker}
                        onSave={handleUpdate}
                        onClose={() => setEditingWorker(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

const WorkerModal = ({ worker, onSave, onClose }) => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const { countryCode: existingCode, phoneNumber: existingNumber } = splitPhone(worker?.phone);

    const [formData, setFormData] = useState({
        name: worker?.name || '',
        email: worker?.email || '',
        countryCode: existingCode,
        phoneNumber: existingNumber,
        role: worker?.role || '',
        is_active: worker?.is_active !== undefined ? worker.is_active : true,
        home_address: worker?.home_address || '',
        home_latitude: worker?.home_latitude || null,
        home_longitude: worker?.home_longitude || null,
        home_google_place_id: worker?.home_google_place_id || '',
        home_formatted_address: worker?.home_formatted_address || '',
        home_address_components: worker?.home_address_components || null,
    });
    const [googleMapsKey, setGoogleMapsKey] = useState('');
    const [googleMapsLoaded, setGoogleMapsLoaded] = useState(false);

    // Load Google Maps API key and script
    useEffect(() => {
        const loadGoogleMaps = async () => {
            try {
                const token = localStorage.getItem('token');
                const response = await axios.get(`${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/config/google-maps`, {
                    headers: { Authorization: `Bearer ${token}` }
                });

                if (response.data.enabled && response.data.api_key) {
                    setGoogleMapsKey(response.data.api_key);
                    await loadGoogleMapsScript(response.data.api_key);
                    setGoogleMapsLoaded(true);
                }
            } catch (error) {
                console.error('Failed to load Google Maps:', error);
                setGoogleMapsLoaded(false);
            }
        };

        loadGoogleMaps();
    }, []);

    const handleChange = (e) => {
        const value = e.target.type === 'checkbox' ? e.target.checked : e.target.value;
        setFormData({
            ...formData,
            [e.target.name]: value,
        });
    };

    const handleHomeAddressSelect = useCallback((addressData) => {
        setFormData(prev => ({
            ...prev,
            home_address: addressData.address || addressData.formattedAddress || '',
            home_latitude: addressData.latitude || null,
            home_longitude: addressData.longitude || null,
            home_google_place_id: addressData.placeId || '',
            home_formatted_address: addressData.formattedAddress || '',
            home_address_components: addressData.components || null,
        }));
    }, []);

    const handleSubmit = (e) => {
        e.preventDefault();
        const payload = {
            name: formData.name,
            email: formData.email,
            phone: formatPhone(formData.countryCode, formData.phoneNumber),
            role: formData.role,
            is_active: formData.is_active,
            home_address: formData.home_address,
            home_latitude: formData.home_latitude,
            home_longitude: formData.home_longitude,
            home_google_place_id: formData.home_google_place_id,
            home_formatted_address: formData.home_formatted_address,
            home_address_components: formData.home_address_components,
        };
        onSave(payload);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full">
                {/* Modal Header */}
                <div
                    className="px-6 py-4 border-b border-gray-200"
                    style={{ backgroundColor: '#1e3a5f' }}
                >
                    <h2 className="text-xl font-semibold text-white">
                        {worker ? t('workers.editWorker') : t('workers.addNewWorker')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    {/* Name */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('workers.name')}</label>
                        <input
                            type="text"
                            name="name"
                            value={formData.name}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                        />
                    </div>

                    {/* Phone */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('workers.phone')}</label>
                        <div className="flex gap-2 mt-1">
                            <select
                                name="countryCode"
                                value={formData.countryCode}
                                onChange={handleChange}
                                className="border border-gray-300 rounded-md px-2 py-2 bg-gray-50 focus:outline-none focus:ring-2"
                                style={{ minWidth: '100px' }}
                            >
                                {COUNTRY_CODES.map((item) => (
                                    <option key={item.code} value={item.code}>
                                        {item.flag} {item.code}
                                    </option>
                                ))}
                            </select>
                            <input
                                type="tel"
                                name="phoneNumber"
                                value={formData.phoneNumber}
                                onChange={handleChange}
                                placeholder="1234567890"
                                required
                                className="flex-1 border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                            />
                        </div>
                        <p className="text-xs text-gray-500 mt-1">
                            {t('workers.fullNumber')} {formData.countryCode}{formData.phoneNumber}
                        </p>
                    </div>

                    {/* Email */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('workers.email')}</label>
                        <input
                            type="email"
                            name="email"
                            value={formData.email}
                            onChange={handleChange}
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                        />
                    </div>

                    {/* Role */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('workers.role')}</label>
                        <select
                            name="role"
                            value={formData.role}
                            onChange={handleChange}
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 bg-white focus:outline-none focus:ring-2"
                        >
                            <option value="">{t('workers.selectRole')}</option>
                            <option value="Worker">{t('workers.roleWorker')}</option>
                            <option value="Foreman">{t('workers.roleForeman')}</option>
                            <option value="Supervisor">{t('workers.roleSupervisor')}</option>
                            <option value="Technician">{t('workers.roleTechnician')}</option>
                        </select>
                    </div>

                    {/* Home Address */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('workers.homeAddress')}
                        </label>
                        <p className="text-xs text-gray-500 mb-1">{t('workers.homeAddressHelp')}</p>
                        {googleMapsLoaded && googleMapsKey ? (
                            <GooglePlacesAutocomplete
                                onChange={handleHomeAddressSelect}
                                placeholder={t('workers.homeAddressPlaceholder')}
                                apiKey={googleMapsKey}
                            />
                        ) : (
                            <input
                                type="text"
                                name="home_address"
                                value={formData.home_address}
                                onChange={handleChange}
                                placeholder={t('workers.homeAddressPlaceholder')}
                                className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                            />
                        )}
                    </div>

                    {/* Active Toggle */}
                    <div>
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                name="is_active"
                                checked={formData.is_active}
                                onChange={handleChange}
                                className="rounded border-gray-300"
                                style={{ accentColor: '#1e3a5f' }}
                            />
                            <span className="ml-2 text-sm text-gray-700">{t('workers.activeToggle')}</span>
                        </label>
                    </div>

                    {/* Buttons */}
                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            {t('workers.cancel')}
                        </button>
                        <button
                            type="submit"
                            style={{ backgroundColor: '#ff6b35' }}
                            className="px-4 py-2 text-white rounded-md hover:opacity-90"
                        >
                            {worker ? t('workers.update') : t('workers.add')} {t(`industry.${industry}.workerSingle`)}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Workers;

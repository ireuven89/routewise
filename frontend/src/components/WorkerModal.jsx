import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import GooglePlacesAutocomplete from './GooglePlacesAutocomplete';
import { loadGoogleMapsScript } from '../utils/googleMaps';
import { formatPhone } from '../utils/phone';
import axios from 'axios';

export const COUNTRY_CODES = [
    { code: '+1', country: 'US/Canada', flag: '🇺🇸' },
    { code: '+972', country: 'Israel', flag: '🇮🇱' },
    { code: '+44', country: 'UK', flag: '🇬🇧' },
    { code: '+61', country: 'Australia', flag: '🇦🇺' },
    { code: '+91', country: 'India', flag: '🇮🇳' },
    { code: '+49', country: 'Germany', flag: '🇩🇪' },
    { code: '+33', country: 'France', flag: '🇫🇷' },
    { code: '+52', country: 'Mexico', flag: '🇲🇽' },
];

export const splitPhone = (fullPhone) => {
    if (!fullPhone) return { countryCode: '+1', phoneNumber: '' };
    const match = COUNTRY_CODES.find((c) => fullPhone.startsWith(c.code));
    if (match) {
        return { countryCode: match.code, phoneNumber: fullPhone.slice(match.code.length) };
    }
    return { countryCode: '+1', phoneNumber: fullPhone };
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
        setFormData({ ...formData, [e.target.name]: value });
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
                <div className="px-6 py-4 border-b border-gray-200" style={{ backgroundColor: '#1e3a5f' }}>
                    <h2 className="text-xl font-semibold text-white">
                        {worker ? t('workers.editWorker') : t('workers.addNewWorker')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
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

export default WorkerModal;

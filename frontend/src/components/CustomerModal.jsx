import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '../context/LanguageContext';
import GooglePlacesAutocomplete from './GooglePlacesAutocomplete';
import { loadGoogleMapsScript } from '../utils/googleMaps';
import axios from 'axios';

const CustomerModal = ({ customer, onSave, onClose }) => {
    const { t } = useLanguage();
    const [formData, setFormData] = useState({
        name: customer?.name || '',
        email: customer?.email || '',
        phone: customer?.phone || '',
        address: customer?.address || '',
        notes: customer?.notes || '',
        latitude: customer?.latitude || null,
        longitude: customer?.longitude || null,
        google_place_id: customer?.google_place_id || '',
        formatted_address: customer?.formatted_address || '',
        address_components: customer?.address_components || null,
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
        setFormData({ ...formData, [e.target.name]: e.target.value });
    };

    const handleAddressSelect = useCallback((addressData) => {
        setFormData(prev => ({
            ...prev,
            address: addressData.address || addressData.formattedAddress || '',
            latitude: addressData.latitude || null,
            longitude: addressData.longitude || null,
            google_place_id: addressData.placeId || '',
            formatted_address: addressData.formattedAddress || '',
            address_components: addressData.components || null,
        }));
    }, []);

    const handleSubmit = (e) => {
        e.preventDefault();
        onSave(formData);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900">
                        {customer ? t('customers.editCustomer') : t('customers.addNewCustomer')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('customers.name')}</label>
                        <input
                            type="text"
                            name="name"
                            value={formData.name}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2 focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('customers.phone')}</label>
                        <input
                            type="tel"
                            name="phone"
                            value={formData.phone}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2 focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('customers.email')}</label>
                        <input
                            type="email"
                            name="email"
                            value={formData.email}
                            onChange={handleChange}
                            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2 focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('customers.address')}</label>
                        {googleMapsLoaded && googleMapsKey ? (
                            <GooglePlacesAutocomplete
                                onChange={handleAddressSelect}
                                placeholder={t('customers.addressPlaceholder')}
                                required
                                apiKey={googleMapsKey}
                            />
                        ) : (
                            <input
                                type="text"
                                name="address"
                                value={formData.address}
                                onChange={handleChange}
                                required
                                placeholder={t('customers.addressPlaceholder')}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2 focus:ring-green-500 focus:border-green-500"
                            />
                        )}
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">{t('customers.notes')}</label>
                        <textarea
                            name="notes"
                            value={formData.notes}
                            onChange={handleChange}
                            rows={3}
                            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2 focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            {t('customers.cancel')}
                        </button>
                        <button
                            type="submit"
                            className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700"
                        >
                            {customer ? t('customers.update') : t('customers.add')} {t('customers.customer')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CustomerModal;

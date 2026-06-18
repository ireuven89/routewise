import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import GooglePlacesAutocomplete from '../components/GooglePlacesAutocomplete';
import apiClient, { organizationAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { loadGoogleMapsScript } from '../utils/googleMaps';

const OrganizationSettings = () => {
    const { organization } = useAuth();
    const { t } = useLanguage();

    const [googleMapsApiKey, setGoogleMapsApiKey] = useState(null);

    // Service area state
    const [serviceAreaLocation, setServiceAreaLocation] = useState(null);
    const [radius, setRadius] = useState(organization?.service_radius_km || 20);
    const [savingArea, setSavingArea] = useState(false);
    const [areaMsg, setAreaMsg] = useState('');

    // Pricing state
    const [visitFee, setVisitFee] = useState(organization?.visit_fee ?? '');
    const [repairMin, setRepairMin] = useState(organization?.repair_estimate_min ?? '');
    const [repairMax, setRepairMax] = useState(organization?.repair_estimate_max ?? '');
    const [savingPricing, setSavingPricing] = useState(false);
    const [pricingMsg, setPricingMsg] = useState('');

    useEffect(() => {
        apiClient.get('/api/v1/config/google-maps')
            .then(res => {
                if (res.data.enabled && res.data.api_key) {
                    setGoogleMapsApiKey(res.data.api_key);
                    loadGoogleMapsScript(res.data.api_key);
                }
            })
            .catch(() => {});
    }, []);

    const handleSaveArea = async () => {
        if (!serviceAreaLocation) return;
        setSavingArea(true);
        setAreaMsg('');
        try {
            await organizationAPI.updateServiceArea({
                latitude: serviceAreaLocation.latitude,
                longitude: serviceAreaLocation.longitude,
                address: serviceAreaLocation.address,
                google_place_id: serviceAreaLocation.placeId,
                formatted_address: serviceAreaLocation.formattedAddress,
                address_components: serviceAreaLocation.components,
                service_radius_km: parseFloat(radius) || 20,
            });
            setAreaMsg(t('settings.saved'));
        } catch {
            setAreaMsg(t('settings.saveError'));
        } finally {
            setSavingArea(false);
        }
    };

    const handleSavePricing = async () => {
        setSavingPricing(true);
        setPricingMsg('');
        try {
            await organizationAPI.updateServiceOffer({
                visit_fee: visitFee !== '' ? parseFloat(visitFee) : null,
                repair_estimate_min: repairMin !== '' ? parseFloat(repairMin) : null,
                repair_estimate_max: repairMax !== '' ? parseFloat(repairMax) : null,
            });
            setPricingMsg(t('settings.saved'));
        } catch {
            setPricingMsg(t('settings.saveError'));
        } finally {
            setSavingPricing(false);
        }
    };

    return (
        <Layout>
            <div className="max-w-2xl mx-auto px-4 py-8">
                <h1 className="text-2xl font-bold text-gray-900 mb-6">{t('settings.title')}</h1>

                {/* Service Area Section */}
                <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
                    <h2 className="text-lg font-semibold text-gray-900 mb-4">
                        {t('settings.serviceArea.title')}
                    </h2>

                    {organization?.formatted_address && (
                        <p className="text-sm text-gray-500 mb-3">
                            {t('settings.serviceArea.current')}: {organization.formatted_address}
                        </p>
                    )}

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                {t('settings.serviceArea.address')}
                            </label>
                            {googleMapsApiKey ? (
                                <GooglePlacesAutocomplete
                                    apiKey={googleMapsApiKey}
                                    onChange={setServiceAreaLocation}
                                    placeholder={t('settings.serviceArea.addressPlaceholder')}
                                />
                            ) : (
                                <div className="mt-1 block w-full rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-400 bg-gray-50">
                                    {t('settings.serviceArea.addressPlaceholder')}
                                </div>
                            )}
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                {t('settings.serviceArea.radius')} (km)
                            </label>
                            <input
                                type="number"
                                value={radius}
                                onChange={e => setRadius(e.target.value)}
                                min="1"
                                max="500"
                                className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                            />
                        </div>

                        <button
                            onClick={handleSaveArea}
                            disabled={!serviceAreaLocation || savingArea}
                            className="w-full bg-blue-600 text-white py-2.5 px-4 rounded-lg text-sm font-semibold hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {savingArea ? t('settings.saving') : t('settings.serviceArea.save')}
                        </button>

                        {areaMsg && (
                            <p className={`text-sm ${areaMsg === t('settings.saved') ? 'text-green-600' : 'text-red-500'}`}>
                                {areaMsg}
                            </p>
                        )}
                    </div>
                </div>

                {/* Pricing Section */}
                <div className="bg-white rounded-xl shadow-sm p-6">
                    <h2 className="text-lg font-semibold text-gray-900 mb-4">
                        {t('settings.pricing.title')}
                    </h2>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                {t('settings.pricing.visitFee')} (₪)
                            </label>
                            <input
                                type="number"
                                value={visitFee}
                                onChange={e => setVisitFee(e.target.value)}
                                min="0"
                                placeholder="150"
                                className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    {t('settings.pricing.repairMin')} (₪)
                                </label>
                                <input
                                    type="number"
                                    value={repairMin}
                                    onChange={e => setRepairMin(e.target.value)}
                                    min="0"
                                    placeholder="300"
                                    className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    {t('settings.pricing.repairMax')} (₪)
                                </label>
                                <input
                                    type="number"
                                    value={repairMax}
                                    onChange={e => setRepairMax(e.target.value)}
                                    min="0"
                                    placeholder="800"
                                    className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                                />
                            </div>
                        </div>

                        <button
                            onClick={handleSavePricing}
                            disabled={savingPricing}
                            className="w-full bg-blue-600 text-white py-2.5 px-4 rounded-lg text-sm font-semibold hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {savingPricing ? t('settings.saving') : t('settings.pricing.save')}
                        </button>

                        {pricingMsg && (
                            <p className={`text-sm ${pricingMsg === t('settings.saved') ? 'text-green-600' : 'text-red-500'}`}>
                                {pricingMsg}
                            </p>
                        )}
                    </div>
                </div>
            </div>
        </Layout>
    );
};

export default OrganizationSettings;

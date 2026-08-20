import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import GooglePlacesAutocomplete from '../components/GooglePlacesAutocomplete';
import apiClient, { organizationAPI, paymentAPI } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { loadGoogleMapsScript } from '../utils/googleMaps';
import { FaMapMarkerAlt, FaShekelSign, FaCheck, FaExclamationCircle, FaMoneyBillWave } from 'react-icons/fa';

const OrganizationSettings = () => {
    const { organization, updateOrganization } = useAuth();
    const { t } = useLanguage();

    const [googleMapsApiKey, setGoogleMapsApiKey] = useState(null);

    const [serviceAreaLocation, setServiceAreaLocation] = useState(null);
    const [radius, setRadius] = useState(organization?.service_radius_km || 20);
    const [savingArea, setSavingArea] = useState(false);
    const [areaMsg, setAreaMsg] = useState('');
    const [areaSuccess, setAreaSuccess] = useState(false);

    const [visitFee, setVisitFee] = useState(organization?.visit_fee ?? '');
    const [repairMin, setRepairMin] = useState(organization?.repair_estimate_min ?? '');
    const [repairMax, setRepairMax] = useState(organization?.repair_estimate_max ?? '');
    const [savingPricing, setSavingPricing] = useState(false);
    const [pricingMsg, setPricingMsg] = useState('');
    const [pricingSuccess, setPricingSuccess] = useState(false);

    const [bitEnabled, setBitEnabled] = useState(organization?.bit_payment_enabled || false);
    const [bitPhone, setBitPhone] = useState(organization?.bit_phone_number || '');
    const [bitBusinessName, setBitBusinessName] = useState(organization?.bit_business_name || '');
    const [autoSend, setAutoSend] = useState(organization?.auto_send_payment_sms || false);
    const [savingPayment, setSavingPayment] = useState(false);
    const [paymentMsg, setPaymentMsg] = useState('');
    const [paymentSuccess, setPaymentSuccess] = useState(false);

    useEffect(() => {
        apiClient.get('/api/v1/config/google-maps')
            .then(async (res) => {
                if (res.data.enabled && res.data.api_key) {
                    await loadGoogleMapsScript(res.data.api_key);
                    setGoogleMapsApiKey(res.data.api_key);
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
            setAreaSuccess(true);
        } catch {
            setAreaMsg(t('settings.saveError'));
            setAreaSuccess(false);
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
            setPricingSuccess(true);
        } catch {
            setPricingMsg(t('settings.saveError'));
            setPricingSuccess(false);
        } finally {
            setSavingPricing(false);
        }
    };

    const handleSavePayment = async () => {
        setSavingPayment(true);
        setPaymentMsg('');
        try {
            const settings = {
                bit_payment_enabled: bitEnabled,
                bit_phone_number: bitPhone,
                bit_business_name: bitBusinessName,
                auto_send_payment_sms: autoSend,
            };
            await paymentAPI.updateSettings(settings);
            updateOrganization(settings);
            setPaymentMsg(t('settings.saved'));
            setPaymentSuccess(true);
        } catch {
            setPaymentMsg(t('settings.saveError'));
            setPaymentSuccess(false);
        } finally {
            setSavingPayment(false);
        }
    };

    const inputClass = "block w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-[#1e3a5f] focus:ring-opacity-20 focus:border-transparent";

    return (
        <Layout>
            <div className="max-w-2xl mx-auto px-4 py-8">
                <h1 className="text-2xl font-bold text-gray-900 mb-6">{t('settings.title')}</h1>

                {/* Service Area Section */}
                <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 mb-6">
                    <div className="flex items-center gap-3 pb-4 mb-4 border-b border-gray-100">
                        <div className="w-8 h-8 rounded-full bg-blue-50 flex items-center justify-center flex-shrink-0">
                            <FaMapMarkerAlt className="w-3.5 h-3.5 text-blue-600" />
                        </div>
                        <h2 className="text-base font-semibold text-gray-900">
                            {t('settings.serviceArea.title')}
                        </h2>
                    </div>

                    {organization?.formatted_address && (
                        <p className="text-xs text-gray-400 mb-4">
                            {t('settings.serviceArea.current')}: {organization.formatted_address}
                        </p>
                    )}

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                {t('settings.serviceArea.address')}
                            </label>
                            {googleMapsApiKey ? (
                                <GooglePlacesAutocomplete
                                    apiKey={googleMapsApiKey}
                                    onChange={setServiceAreaLocation}
                                    placeholder={t('settings.serviceArea.addressPlaceholder')}
                                />
                            ) : (
                                <div className="block w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm text-gray-400 bg-gray-50">
                                    {t('settings.serviceArea.addressPlaceholder')}
                                </div>
                            )}
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                {t('settings.serviceArea.radius')} (km)
                            </label>
                            <input
                                type="number"
                                value={radius}
                                onChange={e => setRadius(e.target.value)}
                                min="1"
                                max="500"
                                className={inputClass}
                            />
                        </div>

                        <div className="flex items-center justify-between pt-1">
                            {areaMsg ? (
                                <span className={`inline-flex items-center gap-1.5 text-sm font-medium ${areaSuccess ? 'text-emerald-600' : 'text-red-500'}`}>
                                    {areaSuccess
                                        ? <FaCheck className="w-3.5 h-3.5" />
                                        : <FaExclamationCircle className="w-3.5 h-3.5" />
                                    }
                                    {areaMsg}
                                </span>
                            ) : <span />}
                            <button
                                onClick={handleSaveArea}
                                disabled={!serviceAreaLocation || savingArea}
                                className="bg-[#1e3a5f] hover:opacity-90 text-white px-6 py-2 rounded-xl text-sm font-semibold disabled:opacity-40 disabled:cursor-not-allowed transition-opacity"
                            >
                                {savingArea ? t('settings.saving') : t('settings.serviceArea.save')}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Pricing Section */}
                <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 mb-6">
                    <div className="flex items-center gap-3 pb-4 mb-4 border-b border-gray-100">
                        <div className="w-8 h-8 rounded-full bg-emerald-50 flex items-center justify-center flex-shrink-0">
                            <FaShekelSign className="w-3.5 h-3.5 text-emerald-600" />
                        </div>
                        <h2 className="text-base font-semibold text-gray-900">
                            {t('settings.pricing.title')}
                        </h2>
                    </div>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                {t('settings.pricing.visitFee')} (₪)
                            </label>
                            <input
                                type="number"
                                value={visitFee}
                                onChange={e => setVisitFee(e.target.value)}
                                min="0"
                                placeholder="150"
                                className={inputClass}
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                    {t('settings.pricing.repairMin')} (₪)
                                </label>
                                <input
                                    type="number"
                                    value={repairMin}
                                    onChange={e => setRepairMin(e.target.value)}
                                    min="0"
                                    placeholder="300"
                                    className={inputClass}
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                    {t('settings.pricing.repairMax')} (₪)
                                </label>
                                <input
                                    type="number"
                                    value={repairMax}
                                    onChange={e => setRepairMax(e.target.value)}
                                    min="0"
                                    placeholder="800"
                                    className={inputClass}
                                />
                            </div>
                        </div>

                        <div className="flex items-center justify-between pt-1">
                            {pricingMsg ? (
                                <span className={`inline-flex items-center gap-1.5 text-sm font-medium ${pricingSuccess ? 'text-emerald-600' : 'text-red-500'}`}>
                                    {pricingSuccess
                                        ? <FaCheck className="w-3.5 h-3.5" />
                                        : <FaExclamationCircle className="w-3.5 h-3.5" />
                                    }
                                    {pricingMsg}
                                </span>
                            ) : <span />}
                            <button
                                onClick={handleSavePricing}
                                disabled={savingPricing}
                                className="bg-[#1e3a5f] hover:opacity-90 text-white px-6 py-2 rounded-xl text-sm font-semibold disabled:opacity-40 disabled:cursor-not-allowed transition-opacity"
                            >
                                {savingPricing ? t('settings.saving') : t('settings.pricing.save')}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Bit Payment Section */}
                <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-6">
                    <div className="flex items-center gap-3 pb-4 mb-4 border-b border-gray-100">
                        <div className="w-8 h-8 rounded-full bg-orange-50 flex items-center justify-center flex-shrink-0">
                            <FaMoneyBillWave className="w-3.5 h-3.5 text-orange-600" />
                        </div>
                        <h2 className="text-base font-semibold text-gray-900">
                            {t('settings.payment.title')}
                        </h2>
                    </div>

                    <div className="space-y-4">
                        <label className="flex items-center gap-2.5 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={bitEnabled}
                                onChange={e => setBitEnabled(e.target.checked)}
                                className="w-4 h-4 rounded border-gray-300 text-[#1e3a5f] focus:ring-[#1e3a5f]"
                            />
                            <span className="text-sm font-medium text-gray-700">{t('settings.payment.enable')}</span>
                        </label>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                {t('settings.payment.phoneNumber')}
                            </label>
                            <input
                                type="text"
                                value={bitPhone}
                                onChange={e => setBitPhone(e.target.value)}
                                placeholder="050-123-4567"
                                className={inputClass}
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">
                                {t('settings.payment.businessName')}
                            </label>
                            <input
                                type="text"
                                value={bitBusinessName}
                                onChange={e => setBitBusinessName(e.target.value)}
                                className={inputClass}
                            />
                        </div>

                        <label className="flex items-center gap-2.5 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={autoSend}
                                onChange={e => setAutoSend(e.target.checked)}
                                disabled={!bitEnabled}
                                className="w-4 h-4 rounded border-gray-300 text-[#1e3a5f] focus:ring-[#1e3a5f] disabled:opacity-40"
                            />
                            <span className={`text-sm font-medium ${bitEnabled ? 'text-gray-700' : 'text-gray-400'}`}>
                                {t('settings.payment.autoSend')}
                            </span>
                        </label>

                        <div className="flex items-center justify-between pt-1">
                            {paymentMsg ? (
                                <span className={`inline-flex items-center gap-1.5 text-sm font-medium ${paymentSuccess ? 'text-emerald-600' : 'text-red-500'}`}>
                                    {paymentSuccess
                                        ? <FaCheck className="w-3.5 h-3.5" />
                                        : <FaExclamationCircle className="w-3.5 h-3.5" />
                                    }
                                    {paymentMsg}
                                </span>
                            ) : <span />}
                            <button
                                onClick={handleSavePayment}
                                disabled={savingPayment}
                                className="bg-[#1e3a5f] hover:opacity-90 text-white px-6 py-2 rounded-xl text-sm font-semibold disabled:opacity-40 disabled:cursor-not-allowed transition-opacity"
                            >
                                {savingPayment ? t('settings.saving') : t('settings.payment.save')}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </Layout>
    );
};

export default OrganizationSettings;

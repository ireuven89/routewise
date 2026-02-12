import { useState, useEffect } from 'react';
import { paymentAPI } from '../api/client';
import Layout from '../components/Layout';
import { useLanguage } from '../context/LanguageContext';
import { formatPhone } from '../utils/phone';

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

const PaymentSettings = () => {
    const { t } = useLanguage();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [settings, setSettings] = useState({
        bit_payment_enabled: false,
        bit_phone_number: '',
        bit_business_name: '',
        auto_send_on_completion: false,
        sms_template_he: '',
        sms_template_en: '',
    });
    const [countryCode, setCountryCode] = useState('+972');
    const [phoneNumber, setPhoneNumber] = useState('');

    useEffect(() => {
        loadSettings();
    }, []);

    const loadSettings = async () => {
        try {
            const response = await paymentAPI.getPaymentSettings();
            const data = response.data;
            setSettings(data);

            // Parse existing phone number into country code and number
            if (data.bit_phone_number) {
                const match = data.bit_phone_number.match(/^(\+\d{1,3})(.+)$/);
                if (match) {
                    setCountryCode(match[1]);
                    setPhoneNumber(match[2]);
                } else {
                    setPhoneNumber(data.bit_phone_number);
                }
            }

            setLoading(false);
        } catch (error) {
            console.error('Failed to load payment settings:', error);
            setLoading(false);
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setSaving(true);

        try {
            // Format phone number with country code
            const formattedSettings = {
                ...settings,
                bit_phone_number: settings.bit_payment_enabled ? formatPhone(countryCode, phoneNumber) : '',
            };

            await paymentAPI.updatePaymentSettings(formattedSettings);
            alert(t('paymentSettings.saveSuccess'));
        } catch (error) {
            console.error('Failed to update settings:', error);
            alert(error.response?.data?.error || t('paymentSettings.saveFailed'));
        } finally {
            setSaving(false);
        }
    };

    const handleChange = (field, value) => {
        setSettings(prev => ({ ...prev, [field]: value }));
    };

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">{t('paymentSettings.loading')}</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="max-w-4xl mx-auto px-4 sm:px-0">
                <h1 className="text-3xl font-bold text-gray-900 mb-6">
                    {t('paymentSettings.title')}
                </h1>

                <form onSubmit={handleSubmit} className="bg-white shadow rounded-xl p-6 space-y-6">
                    {/* Enable Bit Payments */}
                    <div className="flex items-center justify-between pb-4 border-b">
                        <div>
                            <h3 className="text-lg font-medium text-gray-900">
                                {t('paymentSettings.enableBit')}
                            </h3>
                            <p className="text-sm text-gray-500">
                                {t('paymentSettings.enableBitDesc')}
                            </p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                            <input
                                type="checkbox"
                                checked={settings.bit_payment_enabled}
                                onChange={(e) => handleChange('bit_payment_enabled', e.target.checked)}
                                className="sr-only peer"
                            />
                            <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-500"></div>
                        </label>
                    </div>

                    {/* Bit Configuration */}
                    {settings.bit_payment_enabled && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    {t('paymentSettings.bitPhone')} *
                                </label>
                                <div className="flex gap-2" dir="ltr">
                                    <select
                                        value={countryCode}
                                        onChange={(e) => setCountryCode(e.target.value)}
                                        className="px-3 py-2 border border-gray-300 rounded-md bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500"
                                        style={{ minWidth: '120px' }}
                                    >
                                        {COUNTRY_CODES.map((item) => (
                                            <option key={item.code} value={item.code}>
                                                {item.flag} {item.code}
                                            </option>
                                        ))}
                                    </select>
                                    <input
                                        type="tel"
                                        value={phoneNumber}
                                        onChange={(e) => setPhoneNumber(e.target.value)}
                                        placeholder="501234567"
                                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
                                        required={settings.bit_payment_enabled}
                                    />
                                </div>
                                <p className="text-xs text-gray-500 mt-1">
                                    {t('paymentSettings.bitPhoneDesc')} {phoneNumber && formatPhone(countryCode, phoneNumber)}
                                </p>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-1">
                                    {t('paymentSettings.bitBusinessName')}
                                </label>
                                <input
                                    type="text"
                                    value={settings.bit_business_name}
                                    onChange={(e) => handleChange('bit_business_name', e.target.value)}
                                    placeholder={t('paymentSettings.bitBusinessNamePlaceholder')}
                                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500"
                                />
                            </div>
                        </div>
                    )}

                    {/* Auto-send on Completion */}
                    <div className="flex items-center justify-between pb-4 border-b">
                        <div>
                            <h3 className="text-lg font-medium text-gray-900">
                                {t('paymentSettings.autoSend')}
                            </h3>
                            <p className="text-sm text-gray-500">
                                {t('paymentSettings.autoSendDesc')}
                            </p>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                            <input
                                type="checkbox"
                                checked={settings.auto_send_on_completion}
                                onChange={(e) => handleChange('auto_send_on_completion', e.target.checked)}
                                className="sr-only peer"
                                disabled={!settings.bit_payment_enabled}
                            />
                            <div className={`w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-orange-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-500 ${!settings.bit_payment_enabled ? 'opacity-50 cursor-not-allowed' : ''}`}></div>
                        </label>
                    </div>

                    {/* SMS Templates */}
                    <div className="space-y-4">
                        <h3 className="text-lg font-medium text-gray-900">
                            {t('paymentSettings.smsTemplates')}
                        </h3>
                        <p className="text-sm text-gray-500">
                            {t('paymentSettings.smsTemplatesDesc')}
                        </p>

                        {/* Hebrew Template */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                {t('paymentSettings.hebrewTemplate')}
                            </label>
                            <textarea
                                value={settings.sms_template_he}
                                onChange={(e) => handleChange('sms_template_he', e.target.value)}
                                rows={4}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 font-mono text-sm"
                                dir="rtl"
                            />
                        </div>

                        {/* English Template */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                {t('paymentSettings.englishTemplate')}
                            </label>
                            <textarea
                                value={settings.sms_template_en}
                                onChange={(e) => handleChange('sms_template_en', e.target.value)}
                                rows={4}
                                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 font-mono text-sm"
                            />
                        </div>

                        {/* Template Variables Info */}
                        <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
                            <p className="text-sm font-medium text-blue-900 mb-2">
                                {t('paymentSettings.availableVariables')}
                            </p>
                            <ul className="text-sm text-blue-700 space-y-1">
                                <li><code className="bg-blue-100 px-1 rounded">{'{{customer_name}}'}</code> - {t('paymentSettings.varCustomerName')}</li>
                                <li><code className="bg-blue-100 px-1 rounded">{'{{job_title}}'}</code> - {t('paymentSettings.varJobTitle')}</li>
                                <li><code className="bg-blue-100 px-1 rounded">{'{{amount}}'}</code> - {t('paymentSettings.varAmount')}</li>
                                <li><code className="bg-blue-100 px-1 rounded">{'{{link}}'}</code> - {t('paymentSettings.varLink')}</li>
                            </ul>
                        </div>
                    </div>

                    {/* Save Button */}
                    <div className="flex justify-end pt-4 border-t">
                        <button
                            type="submit"
                            disabled={saving}
                            className="bg-orange-500 hover:bg-orange-600 text-white px-6 py-2 rounded-md font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {saving ? t('paymentSettings.saving') : t('paymentSettings.save')}
                        </button>
                    </div>
                </form>
            </div>
        </Layout>
    );
};

export default PaymentSettings;

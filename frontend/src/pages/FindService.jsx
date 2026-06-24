import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { FiMapPin, FiClock, FiSearch, FiPhone, FiCheckCircle, FiWind } from 'react-icons/fi';
import { providersAPI, publicConfigAPI } from '../api/client';
import { useLanguage } from '../context/LanguageContext';
import GooglePlacesAutocomplete from '../components/GooglePlacesAutocomplete';
import { loadGoogleMapsScript } from '../utils/googleMaps';

const BRAND = '#1e3a5f';
const BRAND_LIGHT = '#2d5282';

const ProviderInitials = ({ name }) => {
    const initials = name
        .split(' ')
        .slice(0, 2)
        .map(w => w[0])
        .join('')
        .toUpperCase();
    return (
        <div
            className="w-12 h-12 rounded-full flex items-center justify-center text-white text-base font-bold flex-shrink-0"
            style={{ background: `linear-gradient(135deg, ${BRAND}, ${BRAND_LIGHT})` }}
        >
            {initials}
        </div>
    );
};

const ProviderCard = ({ provider, t }) => (
    <div className="bg-white rounded-2xl shadow-md hover:shadow-lg transition-shadow p-6 border border-gray-100">
        <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-4 min-w-0">
                <ProviderInitials name={provider.name} />
                <div className="min-w-0">
                    <h3 className="text-base font-bold text-gray-900 truncate">{provider.name}</h3>
                    {provider.address && (
                        <p className="flex items-center gap-1 text-sm text-gray-500 mt-0.5 truncate">
                            <FiMapPin className="w-3.5 h-3.5 flex-shrink-0" />
                            {provider.address}
                        </p>
                    )}
                    <span className="inline-block mt-1.5 text-xs font-medium text-blue-700 bg-blue-50 px-2 py-0.5 rounded-full">
                        {provider.distance_km.toFixed(1)} {t('findService.away')}
                    </span>
                </div>
            </div>
            <a
                href={`tel:${provider.phone}`}
                className="flex-shrink-0 flex items-center gap-2 text-white text-sm font-semibold px-4 py-2 rounded-xl transition-colors"
                style={{ background: BRAND }}
                onMouseEnter={e => e.currentTarget.style.background = BRAND_LIGHT}
                onMouseLeave={e => e.currentTarget.style.background = BRAND}
            >
                <FiPhone className="w-4 h-4" />
                {t('findService.call')}
            </a>
        </div>

        <div className="mt-5 grid grid-cols-2 gap-3 border-t border-gray-100 pt-4">
            <div className="bg-gray-50 rounded-xl px-4 py-3">
                <p className="text-xs text-gray-500 mb-0.5">{t('findService.visitFee')}</p>
                <p className="text-base font-bold text-gray-900">
                    {provider.visit_fee != null ? `₪${provider.visit_fee}` : '—'}
                </p>
            </div>
            {(provider.repair_estimate_min != null || provider.repair_estimate_max != null) && (
                <div className="bg-gray-50 rounded-xl px-4 py-3">
                    <p className="text-xs text-gray-500 mb-0.5">{t('findService.repairEstimate')}</p>
                    <p className="text-base font-bold text-gray-900">
                        ₪{provider.repair_estimate_min ?? 0} – ₪{provider.repair_estimate_max ?? '?'}
                    </p>
                </div>
            )}
        </div>
    </div>
);

const TrustBadge = ({ icon: Icon, text }) => (
    <div className="flex items-center gap-2 text-white/90 text-sm">
        <Icon className="w-4 h-4 text-green-400" />
        <span>{text}</span>
    </div>
);

const FindService = () => {
    const { t, language } = useLanguage();
    const isRTL = language === 'he';

    const [googleMapsApiKey, setGoogleMapsApiKey] = useState(null);
    const [location, setLocation] = useState(null);
    const [requestedTime, setRequestedTime] = useState('');
    const [description, setDescription] = useState('');
    const [providers, setProviders] = useState([]);
    const [loading, setLoading] = useState(false);
    const [searched, setSearched] = useState(false);
    const [error, setError] = useState('');

    useEffect(() => {
        publicConfigAPI.getGoogleMaps()
            .then(async (res) => {
                if (res.data.enabled && res.data.api_key) {
                    await loadGoogleMapsScript(res.data.api_key);
                    setGoogleMapsApiKey(res.data.api_key);
                }
            })
            .catch(() => {});
    }, []);

    const handleSearch = async () => {
        if (!location) return;
        setLoading(true);
        setSearched(true);
        setError('');
        try {
            const res = await providersAPI.search(location.latitude, location.longitude, 'hvac');
            setProviders(res.data.providers || []);
        } catch {
            setError(t('settings.saveError'));
            setProviders([]);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen bg-gray-50" dir={isRTL ? 'rtl' : 'ltr'}>

            {/* ── Hero ── */}
            <div style={{ background: `linear-gradient(160deg, ${BRAND} 0%, #1a4a7a 100%)` }}>

                {/* Nav */}
                <nav className="max-w-5xl mx-auto px-6 py-5 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <FiWind className="w-6 h-6 text-white" />
                        <span className="text-xl font-bold text-white tracking-tight">RouteWise</span>
                    </div>
                    <Link
                        to="/login"
                        className="text-sm font-medium text-white/80 hover:text-white border border-white/30 hover:border-white/60 px-4 py-1.5 rounded-full transition-all"
                    >
                        {t('findService.forProviders')}
                    </Link>
                </nav>

                {/* Headline */}
                <div className="max-w-5xl mx-auto px-6 pt-10 pb-6 text-center">
                    <h1 className="text-4xl sm:text-5xl font-extrabold text-white leading-tight mb-4">
                        {t('findService.title')}
                    </h1>
                    <p className="text-lg text-white/70 max-w-xl mx-auto">
                        {t('findService.subtitle')}
                    </p>
                </div>

                {/* Search card */}
                <div className="max-w-3xl mx-auto px-6 pb-0">
                    <div className="bg-white rounded-2xl shadow-2xl p-6">
                        <div className="grid sm:grid-cols-2 gap-4 mb-4">
                            <div>
                                <label className="flex items-center gap-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
                                    <FiMapPin className="w-3.5 h-3.5" />
                                    {t('findService.yourAddress')}
                                </label>
                                {googleMapsApiKey ? (
                                    <GooglePlacesAutocomplete
                                        apiKey={googleMapsApiKey}
                                        onChange={setLocation}
                                        placeholder={t('findService.addressPlaceholder')}
                                    />
                                ) : (
                                    <div className="h-10 rounded-lg border border-gray-200 bg-gray-50 animate-pulse" />
                                )}
                            </div>
                            <div>
                                <label className="flex items-center gap-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
                                    <FiClock className="w-3.5 h-3.5" />
                                    {t('findService.preferredTime')}
                                </label>
                                <input
                                    type="datetime-local"
                                    value={requestedTime}
                                    onChange={e => setRequestedTime(e.target.value)}
                                    className="block w-full h-10 rounded-lg border border-gray-200 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                />
                            </div>
                        </div>
                        {/* Problem description */}
                        <div>
                            <label className="flex items-center gap-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
                                {t('findService.problemLabel')}
                            </label>
                            <textarea
                                value={description}
                                onChange={e => setDescription(e.target.value)}
                                rows={3}
                                placeholder={t('findService.problemPlaceholder')}
                                className="block w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                            />
                        </div>
                        <button
                            onClick={handleSearch}
                            disabled={!location || loading}
                            className="w-full flex items-center justify-center gap-2 text-white font-bold py-3 rounded-xl text-sm transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                            style={{ background: location && !loading ? BRAND : undefined, backgroundColor: !location || loading ? '#94a3b8' : undefined }}
                        >
                            <FiSearch className="w-4 h-4" />
                            {loading ? t('findService.searching') : t('findService.searchBtn')}
                        </button>
                    </div>
                </div>

                {/* Trust badges */}
                <div className="max-w-3xl mx-auto px-6 py-6 flex flex-wrap justify-center gap-x-8 gap-y-3">
                    <TrustBadge icon={FiCheckCircle} text={t('findService.badge1')} />
                    <TrustBadge icon={FiCheckCircle} text={t('findService.badge2')} />
                    <TrustBadge icon={FiCheckCircle} text={t('findService.badge3')} />
                </div>
            </div>

            {/* ── Results ── */}
            <main className="max-w-3xl mx-auto px-6 py-10">
                {error && (
                    <p className="text-sm text-red-500 mb-4">{error}</p>
                )}

                {loading && (
                    <div className="space-y-4">
                        {[1, 2, 3].map(i => (
                            <div key={i} className="bg-white rounded-2xl shadow-sm p-6 animate-pulse">
                                <div className="flex gap-4">
                                    <div className="w-12 h-12 rounded-full bg-gray-200" />
                                    <div className="flex-1 space-y-2">
                                        <div className="h-4 bg-gray-200 rounded w-1/3" />
                                        <div className="h-3 bg-gray-100 rounded w-1/2" />
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}

                {searched && !loading && (
                    providers.length === 0 ? (
                        <div className="text-center py-20">
                            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mx-auto mb-4">
                                <FiSearch className="w-7 h-7 text-gray-400" />
                            </div>
                            <p className="text-gray-500 text-base">{t('findService.noResults')}</p>
                        </div>
                    ) : (
                        <>
                            <p className="text-sm font-medium text-gray-500 mb-4">
                                {providers.length} {t('findService.resultsFound')}
                            </p>
                            <div className="space-y-4">
                                {providers.map(provider => (
                                    <ProviderCard key={provider.id} provider={provider} t={t} />
                                ))}
                            </div>
                        </>
                    )
                )}
            </main>

            {/* ── Footer ── */}
            <footer className="border-t border-gray-200 py-6 text-center text-xs text-gray-400">
                {t('footer.copyright')}
            </footer>
        </div>
    );
};

export default FindService;

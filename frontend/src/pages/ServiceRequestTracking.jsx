import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { FiCheckCircle, FiClock, FiRefreshCw } from 'react-icons/fi';
import { serviceRequestsAPI } from '../api/client';
import { useLanguage } from '../context/LanguageContext';

const BRAND = '#1e3a5f';

const BidCard = ({ bid, t, onAward, awarding }) => (
    <div className="bg-white rounded-2xl shadow-sm p-5 border border-gray-100 flex items-center justify-between gap-4">
        <div className="min-w-0">
            <h3 className="text-base font-bold text-gray-900 truncate">{bid.organization_name}</h3>
            <p className="text-lg font-bold mt-1" style={{ color: BRAND }}>₪{bid.price}</p>
            {bid.eta_minutes != null && (
                <p className="flex items-center gap-1 text-xs text-gray-500 mt-1">
                    <FiClock className="w-3.5 h-3.5" />
                    {t('requestTracking.eta')}: {bid.eta_minutes} {t('requestTracking.minutes')}
                </p>
            )}
            {bid.message && <p className="text-sm text-gray-500 mt-2">{bid.message}</p>}
            {bid.status === 'awarded' && (
                <span className="inline-block mt-2 text-xs font-semibold text-green-700 bg-green-50 px-2 py-0.5 rounded-full">
                    {t('leads.bidAwarded')}
                </span>
            )}
            {bid.status === 'rejected' && (
                <span className="inline-block mt-2 text-xs font-semibold text-gray-500 bg-gray-100 px-2 py-0.5 rounded-full">
                    {t('leads.bidRejected')}
                </span>
            )}
        </div>
        {bid.status === 'submitted' && (
            <button
                onClick={() => onAward(bid.id)}
                disabled={awarding}
                className="flex-shrink-0 text-white text-sm font-semibold px-4 py-2 rounded-xl transition-colors disabled:opacity-50"
                style={{ background: BRAND }}
            >
                {awarding ? t('requestTracking.awarding') : t('requestTracking.award')}
            </button>
        )}
    </div>
);

const ServiceRequestTracking = () => {
    const { token } = useParams();
    const { t, language } = useLanguage();
    const isRTL = language === 'he';

    const [request, setRequest] = useState(null);
    const [bids, setBids] = useState([]);
    const [loading, setLoading] = useState(true);
    const [notFound, setNotFound] = useState(false);
    const [awardingId, setAwardingId] = useState(null);

    const load = useCallback(async () => {
        try {
            const res = await serviceRequestsAPI.getByToken(token);
            setRequest(res.data.request);
            setBids(res.data.bids || []);
            setNotFound(false);
        } catch {
            setNotFound(true);
        } finally {
            setLoading(false);
        }
    }, [token]);

    useEffect(() => {
        load();
    }, [load]);

    const handleAward = async (bidId) => {
        setAwardingId(bidId);
        try {
            await serviceRequestsAPI.awardBid(token, bidId);
            await load();
        } finally {
            setAwardingId(null);
        }
    };

    const statusLabel = {
        open: t('requestTracking.statusOpen'),
        awarded: t('requestTracking.statusAwarded'),
        cancelled: t('requestTracking.statusCancelled'),
    };

    const winningBid = bids.find(b => b.status === 'awarded');

    return (
        <div className="min-h-screen bg-gray-50" dir={isRTL ? 'rtl' : 'ltr'}>
            <div style={{ background: `linear-gradient(160deg, ${BRAND} 0%, #1a4a7a 100%)` }} className="py-10 px-6 text-center">
                <h1 className="text-3xl font-extrabold text-white">{t('requestTracking.title')}</h1>
                {request && (
                    <span className="inline-block mt-3 text-sm font-semibold text-white/90 bg-white/10 px-3 py-1 rounded-full">
                        {statusLabel[request.status] || request.status}
                    </span>
                )}
            </div>

            <main className="max-w-2xl mx-auto px-6 py-10">
                {loading && <p className="text-center text-gray-500">{t('requestTracking.loading')}</p>}
                {notFound && !loading && <p className="text-center text-gray-500">{t('requestTracking.notFound')}</p>}

                {request && !loading && (
                    <>
                        {request.status === 'awarded' && winningBid && (
                            <div className="bg-green-50 border border-green-200 rounded-2xl p-4 mb-6 flex items-center gap-3">
                                <FiCheckCircle className="w-6 h-6 text-green-600 flex-shrink-0" />
                                <p className="text-sm text-green-800">
                                    {t('requestTracking.awardedTo', { name: winningBid.organization_name })}
                                </p>
                            </div>
                        )}

                        <div className="flex items-center justify-between mb-4">
                            <h2 className="text-sm font-bold text-gray-700 uppercase tracking-wide">
                                {t('requestTracking.bidsHeading')}
                            </h2>
                            <button
                                onClick={load}
                                className="flex items-center gap-1.5 text-xs font-medium text-gray-500 hover:text-gray-700"
                            >
                                <FiRefreshCw className="w-3.5 h-3.5" />
                                {t('requestTracking.refresh')}
                            </button>
                        </div>

                        {bids.length === 0 ? (
                            <p className="text-center text-gray-500 py-10">{t('requestTracking.noBidsYet')}</p>
                        ) : (
                            <div className="space-y-4">
                                {bids.map(bid => (
                                    <BidCard
                                        key={bid.id}
                                        bid={bid}
                                        t={t}
                                        awarding={awardingId === bid.id}
                                        onAward={request.status === 'open' ? handleAward : () => {}}
                                    />
                                ))}
                            </div>
                        )}
                    </>
                )}
            </main>
        </div>
    );
};

export default ServiceRequestTracking;

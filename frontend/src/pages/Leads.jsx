import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { leadsAPI } from '../api/client';
import { useLanguage } from '../context/LanguageContext';
import { FaBriefcase } from 'react-icons/fa';
import { FiClock, FiMapPin } from 'react-icons/fi';

const BRAND = '#1e3a5f';

const BidForm = ({ leadId, myBid, t, onSubmitted }) => {
    const [price, setPrice] = useState(myBid?.price ?? '');
    const [etaMinutes, setEtaMinutes] = useState(myBid?.eta_minutes ?? '');
    const [message, setMessage] = useState(myBid?.message ?? '');
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');

    const locked = myBid && myBid.status !== 'submitted';

    const handleSubmit = async () => {
        if (!price) return;
        setSubmitting(true);
        setError('');
        try {
            await leadsAPI.upsertBid(leadId, {
                price: parseFloat(price),
                eta_minutes: etaMinutes ? parseInt(etaMinutes, 10) : null,
                message,
            });
            onSubmitted();
        } catch {
            setError(t('leads.bidError'));
        } finally {
            setSubmitting(false);
        }
    };

    if (locked) {
        return (
            <div className="mt-3 pt-3 border-t border-gray-100 text-sm">
                <span className={`font-semibold ${myBid.status === 'awarded' ? 'text-green-700' : 'text-gray-500'}`}>
                    {myBid.status === 'awarded' ? t('leads.bidAwarded') : t('leads.bidRejected')}
                </span>
                <span className="text-gray-400 ms-2">₪{myBid.price}</span>
            </div>
        );
    }

    return (
        <div className="mt-3 pt-3 border-t border-gray-100">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">{t('leads.yourBid')}</p>
            <div className="grid grid-cols-2 gap-2 mb-2">
                <input
                    type="number"
                    value={price}
                    onChange={e => setPrice(e.target.value)}
                    placeholder={t('leads.priceLabel')}
                    className="h-9 rounded-lg border border-gray-200 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                    type="number"
                    value={etaMinutes}
                    onChange={e => setEtaMinutes(e.target.value)}
                    placeholder={t('leads.etaLabel')}
                    className="h-9 rounded-lg border border-gray-200 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <textarea
                value={message}
                onChange={e => setMessage(e.target.value)}
                placeholder={t('leads.messageLabel')}
                rows={2}
                className="block w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none mb-2"
            />
            {error && <p className="text-sm text-red-500 mb-2">{error}</p>}
            <button
                onClick={handleSubmit}
                disabled={!price || submitting}
                className="w-full py-2 rounded-lg text-sm font-semibold text-white transition-colors disabled:opacity-50"
                style={{ background: BRAND }}
            >
                {submitting ? t('leads.submitting') : myBid ? t('leads.updateBid') : t('leads.submitBid')}
            </button>
        </div>
    );
};

const LeadCard = ({ lead, t, onSubmitted }) => {
    const { request, my_bid: myBid } = lead;
    return (
        <div className="bg-white rounded-2xl shadow-sm p-5 border border-gray-100">
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <h3 className="text-base font-bold text-gray-900">{request.service_type}</h3>
                    {request.address && (
                        <p className="flex items-center gap-1 text-sm text-gray-500 mt-1">
                            <FiMapPin className="w-3.5 h-3.5 flex-shrink-0" />
                            {request.address}
                        </p>
                    )}
                    {request.preferred_time && (
                        <p className="flex items-center gap-1 text-xs text-gray-400 mt-1">
                            <FiClock className="w-3.5 h-3.5 flex-shrink-0" />
                            {new Date(request.preferred_time).toLocaleString()}
                        </p>
                    )}
                </div>
            </div>
            {request.description && <p className="text-sm text-gray-600 mt-3">{request.description}</p>}
            <BidForm leadId={request.id} myBid={myBid} t={t} onSubmitted={onSubmitted} />
        </div>
    );
};

const Leads = () => {
    const { t } = useLanguage();
    const [leads, setLeads] = useState([]);
    const [loading, setLoading] = useState(true);

    const load = async () => {
        try {
            const res = await leadsAPI.getAll();
            setLeads(res.data.leads || []);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        load();
    }, []);

    return (
        <Layout>
            <div className="flex items-center gap-3 mb-6">
                <FaBriefcase className="w-5 h-5 text-gray-400" />
                <div>
                    <h1 className="text-xl font-bold text-gray-900">{t('leads.title')}</h1>
                    <p className="text-sm text-gray-500">{t('leads.subtitle')}</p>
                </div>
            </div>

            {loading ? (
                <p className="text-gray-500">...</p>
            ) : leads.length === 0 ? (
                <p className="text-gray-500">{t('leads.empty')}</p>
            ) : (
                <div className="grid sm:grid-cols-2 gap-4">
                    {leads.map(lead => (
                        <LeadCard key={lead.request.id} lead={lead} t={t} onSubmitted={load} />
                    ))}
                </div>
            )}
        </Layout>
    );
};

export default Leads;

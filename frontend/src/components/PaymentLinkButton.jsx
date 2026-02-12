import { useState } from 'react';
import { FiSend } from 'react-icons/fi';
import { useLanguage } from '../context/LanguageContext';
import { paymentAPI } from '../api/client';

const PaymentLinkButton = ({ job, onSuccess }) => {
    const { t } = useLanguage();
    const [sending, setSending] = useState(false);

    // Only show for completed jobs with price
    if (job.status !== 'completed' || !job.price) {
        return null;
    }

    const handleSend = async () => {
        setSending(true);
        try {
            await paymentAPI.sendPaymentLink(job.id);
            alert(t('payments.linkSent'));
            if (onSuccess) onSuccess();
        } catch (error) {
            const errorMsg = error.response?.data?.error || t('payments.linkFailed');
            alert(errorMsg);
        } finally {
            setSending(false);
        }
    };

    return (
        <button
            onClick={handleSend}
            disabled={sending}
            className="flex items-center gap-2 px-3 py-1.5 bg-orange-500 text-white text-sm rounded-md hover:bg-orange-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
            {sending ? (
                <>
                    <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
                    {t('payments.sending')}
                </>
            ) : (
                <>
                    <FiSend size={14} />
                    {t('payments.sendLink')}
                </>
            )}
        </button>
    );
};

export default PaymentLinkButton;

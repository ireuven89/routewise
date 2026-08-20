/**
 * Unit tests for the PaymentBadge component defined inside Jobs.jsx.
 *
 * Because PaymentBadge is not exported, we reproduce it here from source so we can
 * render it directly — the same approach used in Dashboard.WeekSchedulePanel.test.jsx
 * for un-exported inline components. It only needs `../context/LanguageContext` (useLanguage)
 * and `../api/client` (paymentAPI), both mocked below.
 */

import React, { useState, useEffect } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

// ── Mock LanguageContext (identity translation) ───────────────────────────────
jest.mock('../context/LanguageContext', () => ({
    useLanguage: () => ({ t: (key) => key }),
}));

// ── Mock the payment API ──────────────────────────────────────────────────────
const mockGetNotifications = jest.fn();
const mockSendRequest = jest.fn();
const mockMarkPaid = jest.fn();
jest.mock('../api/client', () => ({
    paymentAPI: {
        getNotifications: (...args) => mockGetNotifications(...args),
        sendRequest: (...args) => mockSendRequest(...args),
        markPaid: (...args) => mockMarkPaid(...args),
    },
}));

const { useLanguage } = require('../context/LanguageContext');
const { paymentAPI } = require('../api/client');

// ── Inline PaymentBadge (mirrors Jobs.jsx implementation exactly) ─────────────
const PaymentBadge = ({ job }) => {
    const { t } = useLanguage();
    const [notification, setNotification] = useState(null);
    const [loaded, setLoaded] = useState(false);
    const [busy, setBusy] = useState(false);

    const eligible = job.status === 'completed' && job.price;

    useEffect(() => {
        if (!eligible) return;
        let cancelled = false;
        paymentAPI.getNotifications(job.id)
            .then(res => {
                if (cancelled) return;
                const list = res.data || [];
                setNotification(list[0] || null);
                setLoaded(true);
            })
            .catch(() => setLoaded(true));
        return () => { cancelled = true; };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [job.id, job.status, job.price]);

    if (!eligible || !loaded) return null;

    const handleSend = async () => {
        setBusy(true);
        try {
            const res = await paymentAPI.sendRequest(job.id);
            setNotification(res.data);
        } catch (error) {
            alert(error.response?.data?.error || t('jobs.paymentSendFailed'));
        } finally {
            setBusy(false);
        }
    };

    const handleMarkPaid = async () => {
        setBusy(true);
        try {
            await paymentAPI.markPaid(notification.id);
            setNotification({ ...notification, payment_status: 'paid' });
        } catch (error) {
            alert(error.response?.data?.error || t('jobs.markPaidFailed'));
        } finally {
            setBusy(false);
        }
    };

    if (!notification || notification.payment_status === 'send_failed') {
        return (
            <button onClick={handleSend} disabled={busy}>
                {t('jobs.sendPaymentRequest')}
            </button>
        );
    }

    if (notification.payment_status === 'paid') {
        return <span>{t('jobs.paymentPaid')}</span>;
    }

    return (
        <>
            <span>{t('jobs.paymentSent')}</span>
            <button onClick={handleMarkPaid} disabled={busy}>
                {t('jobs.markPaid')}
            </button>
        </>
    );
};

// ── Test helpers ──────────────────────────────────────────────────────────────
const makeJob = (overrides) => ({
    id: 'job-1',
    status: 'completed',
    price: 350,
    ...overrides,
});

beforeEach(() => {
    jest.clearAllMocks();
    window.alert = jest.fn();
});

describe('PaymentBadge', () => {
    describe('eligibility', () => {
        it('renders nothing when the job is not completed', () => {
            const { container } = render(<PaymentBadge job={makeJob({ status: 'in_progress' })} />);
            expect(container).toBeEmptyDOMElement();
            expect(mockGetNotifications).not.toHaveBeenCalled();
        });

        it('renders nothing when the job has no price', () => {
            const { container } = render(<PaymentBadge job={makeJob({ price: null })} />);
            expect(container).toBeEmptyDOMElement();
            expect(mockGetNotifications).not.toHaveBeenCalled();
        });

        it('fetches notifications for a completed, priced job', async () => {
            mockGetNotifications.mockResolvedValueOnce({ data: [] });
            render(<PaymentBadge job={makeJob()} />);
            await waitFor(() => expect(mockGetNotifications).toHaveBeenCalledWith('job-1'));
        });
    });

    describe('no notification yet', () => {
        it('renders a "Request Payment" button', async () => {
            mockGetNotifications.mockResolvedValueOnce({ data: [] });
            render(<PaymentBadge job={makeJob()} />);
            expect(await screen.findByText('jobs.sendPaymentRequest')).toBeInTheDocument();
        });

        it('renders a "Request Payment" button when the fetch fails', async () => {
            mockGetNotifications.mockRejectedValueOnce(new Error('network error'));
            render(<PaymentBadge job={makeJob()} />);
            expect(await screen.findByText('jobs.sendPaymentRequest')).toBeInTheDocument();
        });

        it('renders a "Request Payment" button when the latest notification failed to send', async () => {
            mockGetNotifications.mockResolvedValueOnce({
                data: [{ id: 1, payment_status: 'send_failed' }],
            });
            render(<PaymentBadge job={makeJob()} />);
            expect(await screen.findByText('jobs.sendPaymentRequest')).toBeInTheDocument();
        });

        it('sends the payment request and transitions to the sent state on click', async () => {
            mockGetNotifications.mockResolvedValueOnce({ data: [] });
            mockSendRequest.mockResolvedValueOnce({
                data: { id: 5, payment_status: 'sent' },
            });
            render(<PaymentBadge job={makeJob()} />);

            fireEvent.click(await screen.findByText('jobs.sendPaymentRequest'));

            await waitFor(() => expect(mockSendRequest).toHaveBeenCalledWith('job-1'));
            expect(await screen.findByText('jobs.paymentSent')).toBeInTheDocument();
            expect(screen.getByText('jobs.markPaid')).toBeInTheDocument();
        });

        it('alerts on a failed send', async () => {
            mockGetNotifications.mockResolvedValueOnce({ data: [] });
            mockSendRequest.mockRejectedValueOnce({
                response: { data: { error: 'Bit payment is not configured for this organization' } },
            });
            render(<PaymentBadge job={makeJob()} />);

            fireEvent.click(await screen.findByText('jobs.sendPaymentRequest'));

            await waitFor(() =>
                expect(window.alert).toHaveBeenCalledWith('Bit payment is not configured for this organization')
            );
            // Stays in the "not sent" state
            expect(screen.getByText('jobs.sendPaymentRequest')).toBeInTheDocument();
        });
    });

    describe('sent state', () => {
        it('renders a "sent" badge and a "Mark as Paid" button', async () => {
            mockGetNotifications.mockResolvedValueOnce({
                data: [{ id: 5, payment_status: 'sent' }],
            });
            render(<PaymentBadge job={makeJob()} />);

            expect(await screen.findByText('jobs.paymentSent')).toBeInTheDocument();
            expect(screen.getByText('jobs.markPaid')).toBeInTheDocument();
        });

        it('marks as paid and transitions to the paid state on click', async () => {
            mockGetNotifications.mockResolvedValueOnce({
                data: [{ id: 5, payment_status: 'sent' }],
            });
            mockMarkPaid.mockResolvedValueOnce({});
            render(<PaymentBadge job={makeJob()} />);

            fireEvent.click(await screen.findByText('jobs.markPaid'));

            await waitFor(() => expect(mockMarkPaid).toHaveBeenCalledWith(5));
            expect(await screen.findByText('jobs.paymentPaid')).toBeInTheDocument();
            expect(screen.queryByText('jobs.markPaid')).not.toBeInTheDocument();
        });

        it('alerts on a failed mark-as-paid and stays in the sent state', async () => {
            mockGetNotifications.mockResolvedValueOnce({
                data: [{ id: 5, payment_status: 'sent' }],
            });
            mockMarkPaid.mockRejectedValueOnce({
                response: { data: { error: 'payment notification not found' } },
            });
            render(<PaymentBadge job={makeJob()} />);

            fireEvent.click(await screen.findByText('jobs.markPaid'));

            await waitFor(() =>
                expect(window.alert).toHaveBeenCalledWith('payment notification not found')
            );
            expect(screen.getByText('jobs.paymentSent')).toBeInTheDocument();
        });
    });

    describe('paid state', () => {
        it('renders a "Paid" badge with no action button', async () => {
            mockGetNotifications.mockResolvedValueOnce({
                data: [{ id: 5, payment_status: 'paid' }],
            });
            render(<PaymentBadge job={makeJob()} />);

            expect(await screen.findByText('jobs.paymentPaid')).toBeInTheDocument();
            expect(screen.queryByText('jobs.markPaid')).not.toBeInTheDocument();
            expect(screen.queryByText('jobs.sendPaymentRequest')).not.toBeInTheDocument();
        });
    });
});

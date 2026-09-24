import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import '@testing-library/jest-dom';

import ServiceRequestTracking from './ServiceRequestTracking';
import { LanguageProvider } from '../context/LanguageContext';
import { serviceRequestsAPI } from '../api/client';

jest.mock('../api/client', () => ({
    serviceRequestsAPI: {
        getByToken: jest.fn(),
        awardBid: jest.fn(),
    },
}));

const renderTracking = (token = 'tok-123') =>
    render(
        <MemoryRouter initialEntries={[`/find-service/requests/${token}`]}>
            <LanguageProvider>
                <Routes>
                    <Route path="/find-service/requests/:token" element={<ServiceRequestTracking />} />
                </Routes>
            </LanguageProvider>
        </MemoryRouter>
    );

const makeRequest = (overrides = {}) => ({
    id: 'req-1',
    status: 'open',
    ...overrides,
});

const makeBid = (overrides = {}) => ({
    id: 'bid-1',
    organization_name: 'Cool Air Co',
    price: 250,
    eta_minutes: 45,
    message: 'We can be there today',
    status: 'submitted',
    ...overrides,
});

beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('language', 'en');
    jest.clearAllMocks();
});

describe('ServiceRequestTracking page', () => {
    it('shows a loading state while the request is being fetched', async () => {
        let resolvePromise;
        serviceRequestsAPI.getByToken.mockReturnValue(
            new Promise((resolve) => {
                resolvePromise = resolve;
            })
        );

        renderTracking();

        expect(screen.getByText('Loading your request...')).toBeInTheDocument();

        resolvePromise({ data: { request: makeRequest(), bids: [] } });
        await waitFor(() => expect(screen.queryByText('Loading your request...')).not.toBeInTheDocument());
    });

    it('fetches the request by the :token route param', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({ data: { request: makeRequest(), bids: [] } });

        renderTracking('my-special-token');

        await waitFor(() => expect(serviceRequestsAPI.getByToken).toHaveBeenCalledWith('my-special-token'));
    });

    it('shows a not-found message when the request lookup fails', async () => {
        serviceRequestsAPI.getByToken.mockRejectedValue(new Error('404'));

        renderTracking();

        expect(await screen.findByText('This request could not be found.')).toBeInTheDocument();
    });

    it('shows the "waiting for offers" status label for an open request with no bids', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({ data: { request: makeRequest({ status: 'open' }), bids: [] } });

        renderTracking();

        expect(await screen.findByText('Waiting for offers')).toBeInTheDocument();
        expect(screen.getByText('No offers yet — providers nearby have been notified.')).toBeInTheDocument();
    });

    it('renders bid cards with organization name, price, eta and message', async () => {
        const bid = makeBid();
        serviceRequestsAPI.getByToken.mockResolvedValue({ data: { request: makeRequest(), bids: [bid] } });

        renderTracking();

        expect(await screen.findByText('Cool Air Co')).toBeInTheDocument();
        expect(screen.getByText('₪250')).toBeInTheDocument();
        expect(screen.getByText(/ETA: 45 min/)).toBeInTheDocument();
        expect(screen.getByText('We can be there today')).toBeInTheDocument();
    });

    it('shows the "Choose this provider" button for a submitted bid on an open request', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({
            data: { request: makeRequest({ status: 'open' }), bids: [makeBid({ status: 'submitted' })] },
        });

        renderTracking();

        expect(await screen.findByRole('button', { name: 'Choose this provider' })).toBeInTheDocument();
    });

    it('awards a bid: calls awardBid with token + bidId, then reloads the request', async () => {
        serviceRequestsAPI.getByToken
            .mockResolvedValueOnce({
                data: { request: makeRequest({ status: 'open' }), bids: [makeBid({ id: 'bid-9', status: 'submitted' })] },
            })
            .mockResolvedValueOnce({
                data: {
                    request: makeRequest({ status: 'awarded' }),
                    bids: [makeBid({ id: 'bid-9', status: 'awarded' })],
                },
            });
        serviceRequestsAPI.awardBid.mockResolvedValue({ data: {} });

        renderTracking('tok-abc');

        const awardBtn = await screen.findByRole('button', { name: 'Choose this provider' });
        await userEvent.click(awardBtn);

        await waitFor(() => expect(serviceRequestsAPI.awardBid).toHaveBeenCalledWith('tok-abc', 'bid-9'));
        await waitFor(() => expect(serviceRequestsAPI.getByToken).toHaveBeenCalledTimes(2));
        expect(await screen.findByText('You selected Cool Air Co for this job.')).toBeInTheDocument();
    });

    it('shows the awarded banner and badge once a request has a winning bid', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({
            data: {
                request: makeRequest({ status: 'awarded' }),
                bids: [makeBid({ status: 'awarded', organization_name: 'Winner Co' })],
            },
        });

        renderTracking();

        expect(await screen.findByText('Provider selected')).toBeInTheDocument();
        expect(screen.getByText('You selected Winner Co for this job.')).toBeInTheDocument();
        expect(screen.getByText('You won this lead')).toBeInTheDocument();
    });

    it('shows a rejected badge for a losing bid', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({
            data: {
                request: makeRequest({ status: 'awarded' }),
                bids: [
                    makeBid({ id: 'b1', status: 'awarded', organization_name: 'Winner Co' }),
                    makeBid({ id: 'b2', status: 'rejected', organization_name: 'Runner Up Co' }),
                ],
            },
        });

        renderTracking();

        expect(await screen.findByText('Runner Up Co')).toBeInTheDocument();
        expect(screen.getByText('Awarded to another provider')).toBeInTheDocument();
    });

    it('does not call awardBid when clicking a submitted bid on a non-open (awarded) request', async () => {
        // Edge case: request already awarded but a stray bid is still "submitted"
        // (its award button renders, but onAward is a no-op per component logic).
        serviceRequestsAPI.getByToken.mockResolvedValue({
            data: {
                request: makeRequest({ status: 'awarded' }),
                bids: [
                    makeBid({ id: 'winner', status: 'awarded', organization_name: 'Winner Co' }),
                    makeBid({ id: 'straggler', status: 'submitted', organization_name: 'Straggler Co' }),
                ],
            },
        });

        renderTracking();

        const awardBtn = await screen.findByRole('button', { name: 'Choose this provider' });
        await userEvent.click(awardBtn);

        expect(serviceRequestsAPI.awardBid).not.toHaveBeenCalled();
    });

    it('refreshes the request when the Refresh button is clicked', async () => {
        serviceRequestsAPI.getByToken.mockResolvedValue({ data: { request: makeRequest(), bids: [] } });

        renderTracking();

        await screen.findByText('No offers yet — providers nearby have been notified.');
        await userEvent.click(screen.getByRole('button', { name: /Refresh/ }));

        await waitFor(() => expect(serviceRequestsAPI.getByToken).toHaveBeenCalledTimes(2));
    });
});

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import '@testing-library/jest-dom';

import Leads from './Leads';
import { AuthProvider } from '../context/AuthContext';
import { LanguageProvider } from '../context/LanguageContext';
import { leadsAPI } from '../api/client';

// ── Mock the API client module used by Leads.jsx (and transitively by
//    AuthContext, which Navbar/Layout depend on). ─────────────────────────────
jest.mock('../api/client', () => ({
    leadsAPI: {
        getAll: jest.fn(),
        upsertBid: jest.fn(),
    },
    authAPI: {
        login: jest.fn(),
        register: jest.fn(),
        getCurrentUser: jest.fn(),
    },
}));

const renderLeads = () =>
    render(
        <MemoryRouter>
            <AuthProvider>
                <LanguageProvider>
                    <Leads />
                </LanguageProvider>
            </AuthProvider>
        </MemoryRouter>
    );

const makeLead = (overrides = {}) => ({
    request: {
        id: 'req-1',
        service_type: 'hvac',
        address: '123 Main St',
        preferred_time: '2026-10-01T10:00:00',
        description: 'AC not cooling',
        ...overrides.request,
    },
    my_bid: overrides.my_bid !== undefined ? overrides.my_bid : null,
});

beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('language', 'en');
    jest.clearAllMocks();
});

describe('Leads page', () => {
    it('shows a loading indicator while leads are being fetched', async () => {
        let resolvePromise;
        leadsAPI.getAll.mockReturnValue(
            new Promise((resolve) => {
                resolvePromise = resolve;
            })
        );

        renderLeads();

        expect(screen.getByText('...')).toBeInTheDocument();

        resolvePromise({ data: { leads: [] } });
        await waitFor(() => expect(screen.queryByText('...')).not.toBeInTheDocument());
    });

    it('shows the empty state when there are no open leads', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [] } });

        renderLeads();

        expect(await screen.findByText('No open leads in your area right now.')).toBeInTheDocument();
    });

    it('renders lead details without exposing customer name/phone', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [makeLead()] } });

        renderLeads();

        expect(await screen.findByText('AC not cooling')).toBeInTheDocument();
        expect(screen.getByText('hvac')).toBeInTheDocument();
        expect(screen.getByText('123 Main St')).toBeInTheDocument();
        // The /leads response no longer includes customer_name/customer_phone —
        // confirm none of that text leaks into the rendered lead card (scoped to
        // <main> to exclude the Navbar's unrelated "Customers" nav link).
        const main = screen.getByRole('main');
        expect(within(main).queryByText(/customer/i)).not.toBeInTheDocument();
    });

    it('renders a card per lead when multiple leads are open', async () => {
        const lead1 = makeLead({ request: { id: 'req-1', service_type: 'hvac' } });
        const lead2 = makeLead({ request: { id: 'req-2', service_type: 'plumbing' } });
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [lead1, lead2] } });

        renderLeads();

        expect(await screen.findByText('hvac')).toBeInTheDocument();
        expect(screen.getByText('plumbing')).toBeInTheDocument();
    });

    it('disables the submit-bid button until a price is entered', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [makeLead()] } });
        renderLeads();

        const submitBtn = await screen.findByRole('button', { name: 'Submit Bid' });
        expect(submitBtn).toBeDisabled();
    });

    it('submits a new bid with parsed price/eta and reloads the leads list', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [makeLead()] } });
        leadsAPI.upsertBid.mockResolvedValue({ data: {} });

        renderLeads();

        await screen.findByText('AC not cooling');

        await userEvent.type(screen.getByPlaceholderText('Your Price (₪)'), '250');
        await userEvent.type(screen.getByPlaceholderText('ETA (minutes)'), '45');
        await userEvent.type(screen.getByPlaceholderText('Message to customer'), 'Can be there this afternoon');

        const submitBtn = screen.getByRole('button', { name: 'Submit Bid' });
        expect(submitBtn).not.toBeDisabled();
        await userEvent.click(submitBtn);

        await waitFor(() =>
            expect(leadsAPI.upsertBid).toHaveBeenCalledWith('req-1', {
                price: 250,
                eta_minutes: 45,
                message: 'Can be there this afternoon',
            })
        );
        // onSubmitted triggers a reload of the leads list.
        await waitFor(() => expect(leadsAPI.getAll).toHaveBeenCalledTimes(2));
    });

    it('sends null eta_minutes when the ETA field is left blank', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [makeLead()] } });
        leadsAPI.upsertBid.mockResolvedValue({ data: {} });

        renderLeads();
        await screen.findByText('AC not cooling');

        await userEvent.type(screen.getByPlaceholderText('Your Price (₪)'), '99');
        await userEvent.click(screen.getByRole('button', { name: 'Submit Bid' }));

        await waitFor(() =>
            expect(leadsAPI.upsertBid).toHaveBeenCalledWith('req-1', {
                price: 99,
                eta_minutes: null,
                message: '',
            })
        );
    });

    it('shows an error message when submitting a bid fails', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [makeLead()] } });
        leadsAPI.upsertBid.mockRejectedValue(new Error('network error'));

        renderLeads();
        await screen.findByText('AC not cooling');

        await userEvent.type(screen.getByPlaceholderText('Your Price (₪)'), '120');
        await userEvent.click(screen.getByRole('button', { name: 'Submit Bid' }));

        expect(await screen.findByText('Failed to submit bid. Please try again.')).toBeInTheDocument();
        // A failed bid submission should not trigger a reload.
        expect(leadsAPI.getAll).toHaveBeenCalledTimes(1);
    });

    it('pre-fills the bid form and shows "Update Bid" for an existing submitted bid', async () => {
        const lead = makeLead({
            my_bid: { id: 'bid-1', status: 'submitted', price: 180, eta_minutes: 30, message: 'On my way' },
        });
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [lead] } });

        renderLeads();

        const priceInput = await screen.findByPlaceholderText('Your Price (₪)');
        expect(priceInput).toHaveValue(180);
        expect(screen.getByPlaceholderText('ETA (minutes)')).toHaveValue(30);
        expect(screen.getByPlaceholderText('Message to customer')).toHaveValue('On my way');
        expect(screen.getByRole('button', { name: 'Update Bid' })).toBeInTheDocument();
    });

    it('locks the bid form and shows "You won this lead" when the bid was awarded', async () => {
        const lead = makeLead({
            my_bid: { id: 'bid-1', status: 'awarded', price: 200 },
        });
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [lead] } });

        renderLeads();

        expect(await screen.findByText('You won this lead')).toBeInTheDocument();
        expect(screen.getByText('₪200')).toBeInTheDocument();
        expect(screen.queryByPlaceholderText('Your Price (₪)')).not.toBeInTheDocument();
    });

    it('locks the bid form and shows "Awarded to another provider" when the bid was rejected', async () => {
        const lead = makeLead({
            my_bid: { id: 'bid-1', status: 'rejected', price: 150 },
        });
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [lead] } });

        renderLeads();

        expect(await screen.findByText('Awarded to another provider')).toBeInTheDocument();
        expect(screen.queryByPlaceholderText('Your Price (₪)')).not.toBeInTheDocument();
    });

    it('does not render lead cards while still loading, even with stale state', async () => {
        leadsAPI.getAll.mockResolvedValue({ data: { leads: [] } });
        renderLeads();

        // Immediately after render (before the promise resolves) the loading text is shown.
        expect(screen.getByText('...')).toBeInTheDocument();
        await waitFor(() => expect(screen.getByText('No open leads in your area right now.')).toBeInTheDocument());
    });
});

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route, useParams } from 'react-router-dom';
import '@testing-library/jest-dom';

import FindService from './FindService';
import { LanguageProvider } from '../context/LanguageContext';
import { providersAPI, publicConfigAPI, serviceRequestsAPI } from '../api/client';

// ── Mock the API client module used by FindService.jsx. ──────────────────────
jest.mock('../api/client', () => ({
    providersAPI: { search: jest.fn() },
    publicConfigAPI: { getGoogleMaps: jest.fn() },
    serviceRequestsAPI: { create: jest.fn() },
}));

// ── Mock the Google Maps script loader so it never touches the network. ──────
jest.mock('../utils/googleMaps', () => ({
    loadGoogleMapsScript: jest.fn(() => Promise.resolve()),
}));

// ── Mock GooglePlacesAutocomplete with a plain input that reports a fixed
//    location whenever it receives a change event with a non-empty value. ────
jest.mock('../components/GooglePlacesAutocomplete', () => {
    return function MockGooglePlacesAutocomplete({ onChange, placeholder }) {
        return (
            <input
                data-testid="mock-address-input"
                placeholder={placeholder}
                onChange={(e) => {
                    const value = e.target.value;
                    onChange(
                        value
                            ? { address: value, latitude: 32.08, longitude: 34.78 }
                            : null
                    );
                }}
            />
        );
    };
});

// Stub for the destination route so we can assert on client-side navigation
// without depending on the real ServiceRequestTracking page.
const TrackingStub = () => {
    const { token } = useParams();
    return <div data-testid="tracking-page">tracking:{token}</div>;
};

const renderFindService = () =>
    render(
        <MemoryRouter initialEntries={['/find-service']}>
            <LanguageProvider>
                <Routes>
                    <Route path="/find-service" element={<FindService />} />
                    <Route path="/find-service/requests/:token" element={<TrackingStub />} />
                </Routes>
            </LanguageProvider>
        </MemoryRouter>
    );

beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('language', 'en');
    jest.clearAllMocks();
    publicConfigAPI.getGoogleMaps.mockResolvedValue({ data: { enabled: true, api_key: 'test-key' } });
    providersAPI.search.mockResolvedValue({ data: { providers: [] } });
});

/** Selects the HVAC service type and fills in a location via the mocked autocomplete. */
const fillLocationAndServiceType = async () => {
    await userEvent.click(screen.getByRole('button', { name: 'HVAC / A/C' }));
    const addressInput = await screen.findByTestId('mock-address-input');
    await userEvent.type(addressInput, '123 Main St');
};

describe('FindService — Post Job flow', () => {
    it('disables the "Post Job" button until location and service type are set', async () => {
        renderFindService();

        const postJobBtn = screen.getByRole('button', { name: /Post Job/ });
        expect(postJobBtn).toBeDisabled();

        // Let the async Google Maps config effect settle before the test ends.
        await screen.findByTestId('mock-address-input');
    });

    it('keeps the Post Job button disabled with only a service type selected', async () => {
        renderFindService();

        await userEvent.click(screen.getByRole('button', { name: 'HVAC / A/C' }));

        expect(screen.getByRole('button', { name: /Post Job/ })).toBeDisabled();

        // Let the async Google Maps config effect settle before the test ends.
        await screen.findByTestId('mock-address-input');
    });

    it('enables the "Post Job" button once both service type and location are set', async () => {
        renderFindService();

        await fillLocationAndServiceType();

        expect(screen.getByRole('button', { name: /Post Job/ })).not.toBeDisabled();
    });

    it('opens the Post Job modal when the button is clicked', async () => {
        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));

        expect(screen.getByText('Post Your Job')).toBeInTheDocument();
        expect(screen.getByText("We'll send it to nearby providers so they can send you offers")).toBeInTheDocument();
    });

    it('disables the modal submit button until name and phone are filled', async () => {
        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));

        const sendBtn = screen.getByRole('button', { name: 'Send Request' });
        expect(sendBtn).toBeDisabled();

        await userEvent.type(screen.getByPlaceholderText('John Smith'), 'Jane Doe');
        expect(sendBtn).toBeDisabled(); // phone still missing

        await userEvent.type(screen.getByPlaceholderText('050-123-4567'), '0501234567');
        expect(sendBtn).not.toBeDisabled();
    });

    it('closes the modal without submitting when Cancel is clicked', async () => {
        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));
        await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

        expect(screen.queryByText('Post Your Job')).not.toBeInTheDocument();
        expect(serviceRequestsAPI.create).not.toHaveBeenCalled();
    });

    it('submits the job with the expected payload and navigates to the tracking page on success', async () => {
        serviceRequestsAPI.create.mockResolvedValue({ data: { access_token: 'abc-token-123' } });

        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.type(
            screen.getByPlaceholderText('e.g. AC not cooling, strange noise, water leaking...'),
            'AC is not cooling at all'
        );

        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));
        await userEvent.type(screen.getByPlaceholderText('John Smith'), 'Jane Doe');
        await userEvent.type(screen.getByPlaceholderText('050-123-4567'), '0501234567');
        await userEvent.click(screen.getByRole('button', { name: 'Send Request' }));

        await waitFor(() =>
            expect(serviceRequestsAPI.create).toHaveBeenCalledWith({
                service_type: 'hvac',
                description: 'AC is not cooling at all',
                customer_name: 'Jane Doe',
                customer_phone: '0501234567',
                latitude: 32.08,
                longitude: 34.78,
                address: '123 Main St',
                preferred_time: null,
            })
        );

        expect(await screen.findByTestId('tracking-page')).toHaveTextContent('tracking:abc-token-123');
    });

    it('shows an error message and does not navigate when the request submission fails', async () => {
        serviceRequestsAPI.create.mockRejectedValue(new Error('network down'));

        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));
        await userEvent.type(screen.getByPlaceholderText('John Smith'), 'Jane Doe');
        await userEvent.type(screen.getByPlaceholderText('050-123-4567'), '0501234567');
        await userEvent.click(screen.getByRole('button', { name: 'Send Request' }));

        expect(await screen.findByText('Failed to send request. Please try again.')).toBeInTheDocument();
        expect(screen.queryByTestId('tracking-page')).not.toBeInTheDocument();
        // Modal should remain open, still showing the form.
        expect(screen.getByText('Post Your Job')).toBeInTheDocument();
    });

    it('trims whitespace from name and phone before submitting', async () => {
        serviceRequestsAPI.create.mockResolvedValue({ data: { access_token: 'tok-2' } });

        renderFindService();

        await fillLocationAndServiceType();
        await userEvent.click(screen.getByRole('button', { name: /Post Job/ }));
        await userEvent.type(screen.getByPlaceholderText('John Smith'), '  Jane Doe  ');
        await userEvent.type(screen.getByPlaceholderText('050-123-4567'), '  0501234567  ');
        await userEvent.click(screen.getByRole('button', { name: 'Send Request' }));

        await waitFor(() =>
            expect(serviceRequestsAPI.create).toHaveBeenCalledWith(
                expect.objectContaining({ customer_name: 'Jane Doe', customer_phone: '0501234567' })
            )
        );
    });
});

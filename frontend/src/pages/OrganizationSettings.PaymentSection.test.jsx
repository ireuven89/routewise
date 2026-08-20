/**
 * Unit tests for the "Bit Payment Collection" settings card in OrganizationSettings.jsx.
 *
 * Unlike WeekSchedulePanel (Dashboard.WeekSchedulePanel.test.jsx), this card's state/handlers
 * live inline in the page component and are tightly coupled to page-level hooks (useAuth,
 * useLanguage) and sibling sections (Service Area, Pricing), so it isn't practical to
 * reproduce in isolation from source. Instead we render the real page with its external
 * dependencies (Layout/Navbar, contexts, API client, Google Maps loader) mocked, and assert
 * only on the payment card's behavior.
 *
 * Layout is mocked (rather than mocking react-router-dom directly, as
 * Dashboard.WeekSchedulePanel.test.jsx does) because it transitively pulls in Navbar →
 * react-router-dom, and this environment's installed react-router-dom build is missing
 * dist/main.js (a pre-existing package/Jest-resolver mismatch, unrelated to this feature —
 * the same failure reproduces on the pre-existing WeekSchedulePanel test). Mocking Layout
 * avoids that dependency entirely while still exercising the real page component.
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

// ── Mock Layout (avoids pulling in Navbar → react-router-dom) ────────────────
jest.mock('../components/Layout', () => ({ children }) => <div>{children}</div>);

// ── Mock react-icons/fa so SVG rendering doesn't require the full icon set ────
jest.mock('react-icons/fa', () => ({
    FaMapMarkerAlt: () => <span data-testid="icon-map" />,
    FaShekelSign: () => <span data-testid="icon-shekel" />,
    FaCheck: () => <span data-testid="icon-check" />,
    FaExclamationCircle: () => <span data-testid="icon-error" />,
    FaMoneyBillWave: () => <span data-testid="icon-money" />,
}));

// ── Mock GooglePlacesAutocomplete (never actually rendered in these tests since
//    the Google Maps config fetch is mocked to fail, but stub it defensively) ──
jest.mock('../components/GooglePlacesAutocomplete', () => () => <div data-testid="places-autocomplete" />);

// ── Mock the API client ───────────────────────────────────────────────────────
const mockUpdateSettings = jest.fn();
const mockApiGet = jest.fn();
jest.mock('../api/client', () => ({
    __esModule: true,
    default: {
        get: (...args) => mockApiGet(...args),
    },
    organizationAPI: {
        updateServiceArea: jest.fn(),
        updateServiceOffer: jest.fn(),
    },
    paymentAPI: {
        updateSettings: (...args) => mockUpdateSettings(...args),
    },
}));

// ── Mock AuthContext ──────────────────────────────────────────────────────────
const mockUpdateOrganization = jest.fn();
let mockOrganization = {
    service_radius_km: 20,
    visit_fee: null,
    repair_estimate_min: null,
    repair_estimate_max: null,
    bit_payment_enabled: false,
    bit_phone_number: '',
    bit_business_name: '',
    auto_send_payment_sms: false,
};
jest.mock('../context/AuthContext', () => ({
    useAuth: () => ({
        organization: mockOrganization,
        updateOrganization: mockUpdateOrganization,
    }),
}));

// ── Mock LanguageContext (identity translation) ───────────────────────────────
jest.mock('../context/LanguageContext', () => ({
    useLanguage: () => ({ t: (key) => key, language: 'en', toggleLanguage: jest.fn() }),
}));

import OrganizationSettings from './OrganizationSettings';

describe('OrganizationSettings — Bit Payment Collection card', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        mockApiGet.mockRejectedValue(new Error('google maps config not available in tests'));
        mockOrganization = {
            service_radius_km: 20,
            visit_fee: null,
            repair_estimate_min: null,
            repair_estimate_max: null,
            bit_payment_enabled: false,
            bit_phone_number: '',
            bit_business_name: '',
            auto_send_payment_sms: false,
        };
    });

    it('renders the payment card title', () => {
        render(<OrganizationSettings />);
        expect(screen.getByText('settings.payment.title')).toBeInTheDocument();
    });

    it('seeds fields from the organization prop', () => {
        mockOrganization = {
            ...mockOrganization,
            bit_payment_enabled: true,
            bit_phone_number: '050-123-4567',
            bit_business_name: 'RouteWise Plumbing',
            auto_send_payment_sms: true,
        };
        render(<OrganizationSettings />);

        expect(screen.getByDisplayValue('050-123-4567')).toBeInTheDocument();
        expect(screen.getByDisplayValue('RouteWise Plumbing')).toBeInTheDocument();

        const enableCheckbox = screen.getByLabelText('settings.payment.enable');
        expect(enableCheckbox).toBeChecked();
        const autoSendCheckbox = screen.getByLabelText('settings.payment.autoSend');
        expect(autoSendCheckbox).toBeChecked();
    });

    it('disables the auto-send checkbox when Bit payment is not enabled', () => {
        render(<OrganizationSettings />);
        const autoSendCheckbox = screen.getByLabelText('settings.payment.autoSend');
        expect(autoSendCheckbox).toBeDisabled();
    });

    it('enables the auto-send checkbox once Bit payment is enabled', () => {
        render(<OrganizationSettings />);
        const enableCheckbox = screen.getByLabelText('settings.payment.enable');
        fireEvent.click(enableCheckbox);

        const autoSendCheckbox = screen.getByLabelText('settings.payment.autoSend');
        expect(autoSendCheckbox).not.toBeDisabled();
    });

    it('saves settings and shows success feedback', async () => {
        mockUpdateSettings.mockResolvedValueOnce({});
        render(<OrganizationSettings />);

        fireEvent.click(screen.getByLabelText('settings.payment.enable'));
        fireEvent.change(screen.getByPlaceholderText('050-123-4567'), {
            target: { value: '052-999-8888' },
        });

        fireEvent.click(screen.getByText('settings.payment.save'));

        await waitFor(() => {
            expect(mockUpdateSettings).toHaveBeenCalledWith({
                bit_payment_enabled: true,
                bit_phone_number: '052-999-8888',
                bit_business_name: '',
                auto_send_payment_sms: false,
            });
        });

        expect(mockUpdateOrganization).toHaveBeenCalledWith({
            bit_payment_enabled: true,
            bit_phone_number: '052-999-8888',
            bit_business_name: '',
            auto_send_payment_sms: false,
        });

        expect(await screen.findByText('settings.saved')).toBeInTheDocument();
    });

    it('shows error feedback when saving fails', async () => {
        mockUpdateSettings.mockRejectedValueOnce(new Error('network error'));
        render(<OrganizationSettings />);

        fireEvent.click(screen.getByText('settings.payment.save'));

        expect(await screen.findByText('settings.saveError')).toBeInTheDocument();
        expect(mockUpdateOrganization).not.toHaveBeenCalled();
    });
});

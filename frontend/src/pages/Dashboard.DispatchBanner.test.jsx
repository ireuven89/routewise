/**
 * Tests for the dispatch banner on the Dashboard: how many jobs need a technician
 * (unassigned or declined, not finished) and how many are waiting for the technician to
 * accept, plus the "Assign now" link to the pre-filtered Jobs page.
 */
import { act, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import '@testing-library/jest-dom';

import Dashboard from './Dashboard';
import { LanguageProvider } from '../context/LanguageContext';
import { jobsAPI, customersAPI, workersAPI, dashboardAPI } from '../api/client';

jest.mock('../api/client', () => ({
    jobsAPI: { getAll: jest.fn(), create: jest.fn() },
    customersAPI: { getAll: jest.fn(), create: jest.fn() },
    workersAPI: { getAll: jest.fn(), create: jest.fn() },
    dashboardAPI: { getStats: jest.fn() },
}));

jest.mock('../components/Layout', () => ({ children }) => <div>{children}</div>);
jest.mock('../context/AuthContext', () => ({
    useAuth: () => ({ organization: { industry: 'hvac', name: 'Acme', company_code: 'ACME1' } }),
}));
// Recharts' ResponsiveContainer needs a real layout engine; the chart isn't under test.
jest.mock('recharts', () => {
    const Stub = ({ children }) => <div>{children}</div>;
    return {
        BarChart: Stub, Bar: Stub, XAxis: Stub, YAxis: Stub,
        CartesianGrid: Stub, Tooltip: Stub, ResponsiveContainer: Stub,
    };
});

const job = (overrides) => ({
    id: Math.random(),
    title: 'Job',
    status: 'scheduled',
    scheduled_at: '2026-10-05T09:00:00',
    customer: { name: 'Dana' },
    worker_id: null,
    assignment_status: null,
    ...overrides,
});

const renderDashboard = (jobs) => {
    jobsAPI.getAll.mockResolvedValue({ data: jobs });
    customersAPI.getAll.mockResolvedValue({ data: [] });
    workersAPI.getAll.mockResolvedValue({ data: [] });
    dashboardAPI.getStats.mockResolvedValue({ data: { revenue: null } });
    return render(
        <MemoryRouter>
            <LanguageProvider>
                <Dashboard />
            </LanguageProvider>
        </MemoryRouter>
    );
};

beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('language', 'en');
    jest.clearAllMocks();
});

describe('Dashboard dispatch banner', () => {
    it('counts jobs needing a technician and jobs waiting for acceptance', async () => {
        renderDashboard([
            job({}),                                                   // needs technician
            job({ status: 'in_progress' }),                            // needs technician
            job({ declined_by_name: 'Avi' }),                          // declined -> needs technician
            job({ status: 'completed' }),                              // finished, ignored
            job({ status: 'cancelled' }),                              // finished, ignored
            job({ worker_id: 7, assignment_status: 'pending' }),       // waiting
            job({ worker_id: 8, assignment_status: 'pending' }),       // waiting
            job({ worker_id: 7, assignment_status: 'pending', status: 'cancelled' }), // ignored
            job({ worker_id: 7, assignment_status: 'accepted' }),      // fine
        ]);

        const banner = await screen.findByTestId('dispatch-banner');
        expect(banner).toHaveTextContent('3 jobs need a technician');
        expect(banner).toHaveTextContent('2 waiting for the technician to accept');
        const cta = screen.getByRole('link', { name: 'Assign now' });
        expect(cta).toHaveAttribute('href', '/jobs?filter=unassigned');
    });

    it('shows only the waiting line (no CTA) when every job has a technician', async () => {
        renderDashboard([
            job({ worker_id: 7, assignment_status: 'pending' }),
            job({ worker_id: 7, assignment_status: 'accepted' }),
        ]);

        const banner = await screen.findByTestId('dispatch-banner');
        expect(banner).toHaveTextContent('1 waiting for the technician to accept');
        expect(banner).not.toHaveTextContent(/need a technician/);
        expect(screen.queryByRole('link', { name: 'Assign now' })).not.toBeInTheDocument();
    });

    it('shows only the needs-technician line when nothing is waiting', async () => {
        renderDashboard([job({}), job({ worker_id: 7, assignment_status: 'accepted' })]);

        const banner = await screen.findByTestId('dispatch-banner');
        expect(banner).toHaveTextContent('1 jobs need a technician');
        expect(banner).not.toHaveTextContent(/waiting for the technician/);
        expect(screen.getByRole('link', { name: 'Assign now' })).toBeInTheDocument();
    });

    it('is hidden when every open job is accepted', async () => {
        renderDashboard([
            job({ worker_id: 7, assignment_status: 'accepted' }),
            job({ status: 'completed' }),
        ]);

        await waitFor(() => expect(dashboardAPI.getStats).toHaveBeenCalled());
        await act(async () => {}); // let the load promise chain settle
        expect(screen.queryByTestId('dispatch-banner')).not.toBeInTheDocument();
    });

    it('is hidden when loading jobs fails', async () => {
        const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
        jobsAPI.getAll.mockRejectedValue(new Error('network'));
        customersAPI.getAll.mockResolvedValue({ data: [] });
        workersAPI.getAll.mockResolvedValue({ data: [] });
        dashboardAPI.getStats.mockResolvedValue({ data: {} });
        render(
            <MemoryRouter>
                <LanguageProvider>
                    <Dashboard />
                </LanguageProvider>
            </MemoryRouter>
        );

        await waitFor(() => expect(spy).toHaveBeenCalled());
        expect(screen.queryByTestId('dispatch-banner')).not.toBeInTheDocument();
        spy.mockRestore();
    });

    it('renders the banner in Hebrew', async () => {
        localStorage.setItem('language', 'he');
        renderDashboard([job({}), job({ worker_id: 7, assignment_status: 'pending' })]);

        const banner = await screen.findByTestId('dispatch-banner');
        expect(banner).toHaveTextContent('1 עבודות ממתינות לשיבוץ טכנאי');
        expect(banner).toHaveTextContent('1 ממתינות לאישור הטכנאי');
        expect(screen.getByRole('link', { name: 'שבצו עכשיו' })).toBeInTheDocument();
    });
});

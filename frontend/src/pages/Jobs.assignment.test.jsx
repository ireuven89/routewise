/**
 * Tests for the technician accept/decline flow as the owner sees it on the Jobs page:
 *   - needsTechnician() helper
 *   - AssignmentBadge (waiting / accepted / needs technician / declined by / from bid)
 *   - the "Unassigned" filter (incl. ?filter=unassigned deep link from the dashboard banner)
 *   - editing a job only calls the assign endpoint when the technician changed
 */
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import '@testing-library/jest-dom';

import Jobs, { AssignmentBadge, needsTechnician } from './Jobs';
import { LanguageProvider } from '../context/LanguageContext';
import { jobsAPI, customersAPI, workersAPI } from '../api/client';

jest.mock('../api/client', () => ({
    jobsAPI: {
        getAll: jest.fn(),
        create: jest.fn(),
        update: jest.fn(),
        assignTechnician: jest.fn(),
        updateStatus: jest.fn(),
        delete: jest.fn(),
    },
    customersAPI: { getAll: jest.fn() },
    workersAPI: { getAll: jest.fn() },
}));

// Layout pulls in Navbar/AuthContext; it isn't what these tests are about.
jest.mock('../components/Layout', () => ({ children }) => <div>{children}</div>);
jest.mock('../context/AuthContext', () => ({
    useAuth: () => ({ organization: { industry: 'hvac', name: 'Acme' } }),
}));

const makeJob = (overrides = {}) => ({
    id: 1,
    title: 'Fix AC',
    status: 'scheduled',
    scheduled_at: '2026-10-01T09:00:00',
    customer: { name: 'Dana', phone: '050' },
    worker_id: null,
    worker: null,
    assignment_status: null,
    declined_by_name: '',
    decline_reason: '',
    metadata: {},
    ...overrides,
});

const avi = { id: 7, name: 'Avi' };

const renderWithLanguage = (ui, route = '/jobs') =>
    render(
        <MemoryRouter initialEntries={[route]}>
            <LanguageProvider>{ui}</LanguageProvider>
        </MemoryRouter>
    );

beforeEach(() => {
    localStorage.clear();
    localStorage.setItem('language', 'en');
    jest.clearAllMocks();
});

// ─── needsTechnician ─────────────────────────────────────────────────────────

describe('needsTechnician', () => {
    it.each([
        ['unassigned scheduled job', { worker_id: null, status: 'scheduled' }, true],
        ['unassigned in-progress job', { worker_id: null, status: 'in_progress' }, true],
        ['worker_id missing entirely', { status: 'scheduled' }, true],
        ['assigned job', { worker_id: 7, status: 'scheduled' }, false],
        ['unassigned completed job', { worker_id: null, status: 'completed' }, false],
        ['unassigned cancelled job', { worker_id: null, status: 'cancelled' }, false],
    ])('%s -> %s', (_name, job, expected) => {
        expect(needsTechnician(job)).toBe(expected);
    });
});

// ─── AssignmentBadge ─────────────────────────────────────────────────────────

describe('AssignmentBadge', () => {
    it('shows "waiting for" when the offer is pending', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ worker_id: 7, worker: avi, assignment_status: 'pending' })} />);
        expect(screen.getByText(/Waiting for Avi to accept/)).toBeInTheDocument();
        expect(screen.queryByText(/Accepted by/)).not.toBeInTheDocument();
        expect(screen.queryByText(/Needs technician/)).not.toBeInTheDocument();
    });

    it('shows "accepted by" once the technician accepted', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ worker_id: 7, worker: avi, assignment_status: 'accepted' })} />);
        expect(screen.getByText(/Accepted by Avi/)).toBeInTheDocument();
        expect(screen.queryByText(/Waiting for/)).not.toBeInTheDocument();
    });

    it('treats an assigned job without assignment_status as accepted (pre-feature jobs)', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ worker_id: 7, worker: avi, assignment_status: null })} />);
        expect(screen.getByText(/Accepted by Avi/)).toBeInTheDocument();
    });

    it('shows "needs technician" for an unassigned open job', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob()} />);
        expect(screen.getByText(/Needs technician/)).toBeInTheDocument();
        expect(screen.queryByText(/Declined by/)).not.toBeInTheDocument();
    });

    it('shows who declined and why', () => {
        renderWithLanguage(
            <AssignmentBadge job={makeJob({ declined_by_name: 'Avi', decline_reason: 'Too far away' })} />
        );
        expect(screen.getByText(/Needs technician/)).toBeInTheDocument();
        const declined = screen.getByText('Declined by Avi');
        expect(declined).toHaveAttribute('title', 'Reason: Too far away');
        expect(screen.getByText('Reason: Too far away')).toBeInTheDocument();
    });

    it('shows "declined by" without a reason and no tooltip', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ declined_by_name: 'Avi' })} />);
        expect(screen.getByText('Declined by Avi')).not.toHaveAttribute('title');
        expect(screen.queryByText(/Reason:/)).not.toBeInTheDocument();
    });

    it('does not show decline info once someone else is assigned', () => {
        renderWithLanguage(
            <AssignmentBadge job={makeJob({
                worker_id: 8, worker: { id: 8, name: 'Moshe' }, assignment_status: 'pending',
                declined_by_name: 'Avi', decline_reason: 'Too far away',
            })} />
        );
        expect(screen.getByText(/Waiting for Moshe to accept/)).toBeInTheDocument();
        expect(screen.queryByText(/Declined by/)).not.toBeInTheDocument();
    });

    it.each(['completed', 'cancelled'])('renders nothing for an unassigned %s job', (status) => {
        const { container } = renderWithLanguage(<AssignmentBadge job={makeJob({ status })} />);
        expect(container).toBeEmptyDOMElement();
    });

    it('adds the "from bid" tag for jobs won through find-service', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ metadata: { source: 'service_request' } })} />);
        expect(screen.getByText('From bid')).toBeInTheDocument();
        expect(screen.getByText(/Needs technician/)).toBeInTheDocument();
    });

    it('keeps the "from bid" tag on a finished unassigned job', () => {
        renderWithLanguage(<AssignmentBadge job={makeJob({ status: 'completed', metadata: { source: 'service_request' } })} />);
        expect(screen.getByText('From bid')).toBeInTheDocument();
        expect(screen.queryByText(/Needs technician/)).not.toBeInTheDocument();
    });

    it('renders in Hebrew', () => {
        localStorage.setItem('language', 'he');
        renderWithLanguage(<AssignmentBadge job={makeJob({ worker_id: 7, worker: avi, assignment_status: 'pending' })} />);
        expect(screen.queryByText(/Waiting for/)).not.toBeInTheDocument();
        expect(screen.getByText(/ממתין לאישור של Avi/)).toBeInTheDocument();
    });
});

// ─── Jobs page: unassigned filter ────────────────────────────────────────────

const pageJobs = [
    makeJob({ id: 1, title: 'Unassigned job' }),
    makeJob({ id: 2, title: 'Declined job', declined_by_name: 'Avi', decline_reason: 'Sick' }),
    makeJob({ id: 3, title: 'Waiting job', worker_id: 7, worker: avi, assignment_status: 'pending' }),
    makeJob({ id: 4, title: 'Accepted job', worker_id: 7, worker: avi, assignment_status: 'accepted', status: 'in_progress' }),
    makeJob({ id: 5, title: 'Done unassigned job', status: 'completed' }),
];

const mockLoad = (jobs = pageJobs) => {
    jobsAPI.getAll.mockResolvedValue({ data: jobs });
    customersAPI.getAll.mockResolvedValue({ data: [{ id: 3, name: 'Dana', address: 'Herzl 1' }] });
    workersAPI.getAll.mockResolvedValue({ data: [avi, { id: 8, name: 'Moshe' }] });
};

const unassignedFilterButton = () => screen.getByRole('button', { name: /^Unassigned/ });

describe('Jobs page — unassigned filter', () => {
    it('shows every job by default and counts jobs needing a technician on the filter', async () => {
        mockLoad();
        renderWithLanguage(<Jobs />);

        expect(await screen.findByText('Accepted job')).toBeInTheDocument();
        ['Unassigned job', 'Declined job', 'Waiting job', 'Done unassigned job'].forEach((title) =>
            expect(screen.getByText(title)).toBeInTheDocument()
        );
        expect(within(unassignedFilterButton()).getByText('2')).toBeInTheDocument();
    });

    it('filters to open jobs without a technician (including declined ones)', async () => {
        mockLoad();
        renderWithLanguage(<Jobs />);
        await screen.findByText('Accepted job');

        userEvent.click(unassignedFilterButton());

        expect(screen.getByText('Unassigned job')).toBeInTheDocument();
        expect(screen.getByText('Declined job')).toBeInTheDocument();
        expect(screen.getByText('Declined by Avi')).toBeInTheDocument();
        expect(screen.queryByText('Waiting job')).not.toBeInTheDocument();
        expect(screen.queryByText('Accepted job')).not.toBeInTheDocument();
        expect(screen.queryByText('Done unassigned job')).not.toBeInTheDocument();
    });

    it('opens pre-filtered from the dashboard link (?filter=unassigned)', async () => {
        mockLoad();
        renderWithLanguage(<Jobs />, '/jobs?filter=unassigned');

        expect(await screen.findByText('Unassigned job')).toBeInTheDocument();
        expect(screen.queryByText('Waiting job')).not.toBeInTheDocument();
    });

    it('hides the count when every open job has a technician', async () => {
        mockLoad([pageJobs[2], pageJobs[3]]);
        renderWithLanguage(<Jobs />);
        await screen.findByText('Accepted job');

        expect(unassignedFilterButton()).toHaveTextContent(/^Unassigned$/);
    });

    it('shows the empty state when nothing needs a technician', async () => {
        mockLoad([pageJobs[2]]);
        renderWithLanguage(<Jobs />, '/jobs?filter=unassigned');

        expect(await screen.findByText('No jobs found. Create your first job!')).toBeInTheDocument();
    });

    it('shows assignment badges in the list', async () => {
        mockLoad();
        renderWithLanguage(<Jobs />);
        await screen.findByText('Accepted job');

        expect(screen.getByText(/Waiting for Avi to accept/)).toBeInTheDocument();
        expect(screen.getByText(/Accepted by Avi/)).toBeInTheDocument();
        expect(screen.getAllByText(/Needs technician/)).toHaveLength(2);
    });
});

// ─── Jobs page: editing and reassignment ────────────────────────────────────

describe('Jobs page — editing a job', () => {
    const openEditFor = async (title) => {
        const item = (await screen.findByText(title)).closest('li');
        userEvent.click(within(item).getByRole('button', { name: 'Edit' }));
        return screen.getByRole('heading', { name: 'Edit Job' }).closest('div').parentElement;
    };

    beforeEach(() => {
        jobsAPI.update.mockResolvedValue({});
        jobsAPI.assignTechnician.mockResolvedValue({});
    });

    it('does not re-assign when the technician is unchanged', async () => {
        mockLoad([makeJob({ id: 3, title: 'Waiting job', worker_id: 7, worker: avi, assignment_status: 'pending', customer_id: 3 })]);
        renderWithLanguage(<Jobs />);

        const modal = await openEditFor('Waiting job');
        expect(within(modal).getByDisplayValue('Avi')).toBeInTheDocument();
        userEvent.click(within(modal).getByRole('button', { name: 'Update Job' }));

        await waitFor(() => expect(jobsAPI.update).toHaveBeenCalledTimes(1));
        expect(jobsAPI.update.mock.calls[0][0]).toBe(3);
        expect(jobsAPI.assignTechnician).not.toHaveBeenCalled();
    });

    it('calls the assign endpoint when the technician changes', async () => {
        mockLoad([makeJob({ id: 3, title: 'Waiting job', worker_id: 7, worker: avi, assignment_status: 'pending', customer_id: 3 })]);
        renderWithLanguage(<Jobs />);

        const modal = await openEditFor('Waiting job');
        userEvent.selectOptions(within(modal).getByDisplayValue('Avi'), '8');
        userEvent.click(within(modal).getByRole('button', { name: 'Update Job' }));

        await waitFor(() => expect(jobsAPI.assignTechnician).toHaveBeenCalledWith(3, 8));
    });

    it('unassigns through the assign endpoint', async () => {
        mockLoad([makeJob({ id: 3, title: 'Waiting job', worker_id: 7, worker: avi, assignment_status: 'pending', customer_id: 3 })]);
        renderWithLanguage(<Jobs />);

        const modal = await openEditFor('Waiting job');
        userEvent.selectOptions(within(modal).getByDisplayValue('Avi'), '');
        userEvent.click(within(modal).getByRole('button', { name: 'Update Job' }));

        await waitFor(() => expect(jobsAPI.assignTechnician).toHaveBeenCalledWith(3, null));
    });

    it('assigns from the inline technician picker', async () => {
        mockLoad([makeJob({ id: 1, title: 'Unassigned job' })]);
        jobsAPI.assignTechnician.mockResolvedValue({});
        renderWithLanguage(<Jobs />);

        const item = (await screen.findByText('Unassigned job')).closest('li');
        userEvent.selectOptions(within(item).getByRole('combobox'), '7');

        await waitFor(() => expect(jobsAPI.assignTechnician).toHaveBeenCalledWith(1, 7));
    });
});

/**
 * Unit tests for the WeekSchedulePanel component defined inside Dashboard.jsx.
 *
 * Because WeekSchedulePanel is not exported, we reproduce it here from source
 * so we can render it directly.  The only external imports it needs are:
 *   - react-router-dom  (Link)
 *   - react-icons/fa    (FaClock, FaCalendarCheck, FaPlus)
 *
 * Both are available in the project and are mocked/used normally below.
 *
 * We use jest.useFakeTimers() + jest.setSystemTime() to pin "today" to a
 * deterministic date so the "today highlight" tests are reliable regardless of
 * when the suite runs.
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// ── Mock react-router-dom so Link renders as a plain <a> ─────────────────────
jest.mock('react-router-dom', () => ({
    Link: ({ to, children, className }) => (
        <a href={to} className={className}>
            {children}
        </a>
    ),
}));

// ── Mock react-icons/fa so SVG rendering doesn't require full icon set ────────
jest.mock('react-icons/fa', () => ({
    FaClock:         () => <span data-testid="icon-clock" />,
    FaCalendarCheck: () => <span data-testid="icon-calendar" />,
    FaPlus:          () => <span data-testid="icon-plus" />,
}));

// ── Inline StatusBadge (mirrors Dashboard.jsx implementation exactly) ─────────
const StatusBadge = ({ status, t }) => {
    const config = {
        scheduled:   { bg: 'bg-blue-100',    text: 'text-blue-700',    key: 'status.scheduled' },
        in_progress: { bg: 'bg-amber-100',   text: 'text-amber-700',   key: 'status.inProgress' },
        completed:   { bg: 'bg-emerald-100', text: 'text-emerald-700', key: 'status.completed' },
        cancelled:   { bg: 'bg-red-100',     text: 'text-red-700',     key: 'status.cancelled' },
    }[status] || { bg: 'bg-gray-100', text: 'text-gray-700', key: null };

    return (
        <span
            className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${config.bg} ${config.text}`}
        >
            {config.key ? t(config.key) : status}
        </span>
    );
};

// ── Inline WeekSchedulePanel (mirrors Dashboard.jsx implementation exactly) ───
//
// We cannot import it directly because Dashboard.jsx does not export it.
// This copy is intentional and must be kept in sync with Dashboard.jsx.
const { Link } = require('react-router-dom');
const { FaClock, FaCalendarCheck, FaPlus } = require('react-icons/fa');

const WeekSchedulePanel = ({ weekJobs, t }) => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const dayOfWeek = today.getDay();
    const diffToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
    const weekDays = Array.from({ length: 7 }, (_, i) => {
        const d = new Date(today);
        d.setDate(today.getDate() + diffToMonday + i);
        return d;
    });

    const jobsByDay = weekDays.reduce((acc, day) => {
        const key = day.toDateString();
        acc[key] = weekJobs.filter((j) => {
            const d = new Date(j.scheduled_at);
            d.setHours(0, 0, 0, 0);
            return d.toDateString() === key;
        });
        return acc;
    }, {});

    const daysWithJobs = weekDays.filter((d) => jobsByDay[d.toDateString()].length > 0);

    return (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
                <h2 className="text-sm font-semibold text-gray-900 flex items-center">
                    <FaClock className="mr-2 text-blue-500 w-4 h-4" />
                    {t('dashboard.weekSchedule')}
                </h2>
                <Link to="/jobs" className="text-xs text-blue-600 hover:text-blue-700 font-medium">
                    {t('dashboard.viewAll')}
                </Link>
            </div>

            {daysWithJobs.length === 0 ? (
                <div className="px-5 py-10 text-center">
                    <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                        <FaCalendarCheck className="w-5 h-5 text-gray-400" />
                    </div>
                    <p className="text-gray-500 text-sm">{t('dashboard.noJobsThisWeek')}</p>
                    <Link
                        to="/jobs"
                        className="mt-3 inline-flex items-center text-sm text-blue-600 hover:text-blue-700 font-medium"
                    >
                        <FaPlus className="mr-1 w-3 h-3" />
                        {t('dashboard.scheduleAJob')}
                    </Link>
                </div>
            ) : (
                <div>
                    {daysWithJobs.map((day) => {
                        const isToday = day.toDateString() === today.toDateString();
                        const dayLabel = day.toLocaleDateString('en-US', {
                            weekday: 'long',
                            month: 'short',
                            day: 'numeric',
                        });
                        const jobs = jobsByDay[day.toDateString()];
                        return (
                            <div key={day.toDateString()}>
                                <div
                                    className={`px-5 py-2 text-xs font-semibold uppercase tracking-wide ${
                                        isToday ? 'bg-blue-600 text-white' : 'bg-gray-50 text-gray-500'
                                    }`}
                                    data-testid={`day-header-${day.toDateString()}`}
                                >
                                    {dayLabel}
                                </div>
                                <div className="divide-y divide-gray-100">
                                    {jobs.map((job) => (
                                        <Link
                                            key={job.id}
                                            to={`/jobs/${job.id}`}
                                            className="flex items-center px-5 py-3.5 hover:bg-gray-50 transition-colors duration-100 group"
                                        >
                                            <div className="flex-shrink-0 w-16 text-center mr-4">
                                                <span className="text-xs font-semibold text-blue-600 bg-blue-50 px-2 py-1 rounded-md">
                                                    {new Date(job.scheduled_at).toLocaleTimeString('en-US', {
                                                        hour: '2-digit',
                                                        minute: '2-digit',
                                                    })}
                                                </span>
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                                                    {job.title}
                                                </p>
                                                {job.description && (
                                                    <p className="text-xs text-gray-400 truncate mt-0.5">
                                                        {job.description}
                                                    </p>
                                                )}
                                            </div>
                                            <div className="flex-shrink-0 ml-3">
                                                <StatusBadge status={job.status} t={t} />
                                            </div>
                                        </Link>
                                    ))}
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
};

// ── Test helpers ──────────────────────────────────────────────────────────────

/** Identity t() function: returns the translation key unchanged. */
const t = (key) => key;

/**
 * Pinned week: Mon 2024-01-22 … Sun 2024-01-28.
 * "Today" is pinned to Wednesday 2024-01-24.
 *
 * All date strings deliberately omit a timezone suffix so they are parsed as
 * local time, which keeps them on the correct calendar day regardless of the
 * test runner's timezone.
 */
const PINNED_TODAY   = new Date('2024-01-24T12:00:00'); // Wednesday (local)
const WEEK_MONDAY    = new Date('2024-01-22T10:00:00'); // Mon (local)
const WEEK_TUESDAY   = new Date('2024-01-23T09:30:00'); // Tue (local)
const WEEK_THURSDAY  = new Date('2024-01-25T14:00:00'); // Thu (local)
const NEXT_WEEK_DATE = new Date('2024-01-29T10:00:00'); // following Mon, outside current week (local)

/** Build a minimal job fixture. */
const makeJob = (overrides) => ({
    id:           overrides.id           ?? 'job-1',
    title:        overrides.title        ?? 'Fix AC unit',
    description:  overrides.description  ?? 'Check compressor',
    scheduled_at: overrides.scheduled_at ?? WEEK_MONDAY.toISOString(),
    status:       overrides.status       ?? 'scheduled',
    ...overrides,
});

// ── Suite ─────────────────────────────────────────────────────────────────────

describe('WeekSchedulePanel', () => {
    beforeEach(() => {
        // Pin "today" to Wednesday 2024-01-24 for every test.
        jest.useFakeTimers();
        jest.setSystemTime(PINNED_TODAY);
    });

    afterEach(() => {
        jest.useRealTimers();
    });

    // ── Header ────────────────────────────────────────────────────────────────

    describe('panel header', () => {
        it('renders the weekSchedule translation key as the heading', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            expect(screen.getByText('dashboard.weekSchedule')).toBeInTheDocument();
        });

        it('renders a "View all" link that points to /jobs', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            const viewAllLink = screen.getByText('dashboard.viewAll');
            expect(viewAllLink.closest('a')).toHaveAttribute('href', '/jobs');
        });
    });

    // ── Empty state ───────────────────────────────────────────────────────────

    describe('empty state (no jobs this week)', () => {
        it('shows the noJobsThisWeek translation key', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            expect(screen.getByText('dashboard.noJobsThisWeek')).toBeInTheDocument();
        });

        it('renders a "Schedule a job" link to /jobs', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            const scheduleLink = screen.getByText('dashboard.scheduleAJob');
            expect(scheduleLink.closest('a')).toHaveAttribute('href', '/jobs');
        });

        it('does not render any day headers', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            // Day headers contain weekday names like "Monday", "Tuesday" etc.
            // None should appear in the empty state.
            expect(screen.queryByText(/monday/i)).not.toBeInTheDocument();
            expect(screen.queryByText(/tuesday/i)).not.toBeInTheDocument();
            expect(screen.queryByText(/wednesday/i)).not.toBeInTheDocument();
        });

        it('does not render any job titles', () => {
            render(<WeekSchedulePanel weekJobs={[]} t={t} />);
            expect(screen.queryByRole('link', { name: /fix ac/i })).not.toBeInTheDocument();
        });
    });

    // ── Job rendering ─────────────────────────────────────────────────────────

    describe('job rendering', () => {
        it('renders the job title', () => {
            const job = makeJob({ title: 'Repair boiler' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('Repair boiler')).toBeInTheDocument();
        });

        it('renders the job description when present', () => {
            const job = makeJob({ description: 'Needs new thermocouple' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('Needs new thermocouple')).toBeInTheDocument();
        });

        it('omits the description element when description is empty string', () => {
            const job = makeJob({ description: '' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            // There must be exactly one <p> (the title paragraph); no description row.
            const jobLink = screen.getByText('Fix AC unit').closest('a');
            const allPs = jobLink.querySelectorAll('p');
            expect(allPs).toHaveLength(1);
        });

        it('omits the description element when description is undefined/null', () => {
            const job = makeJob({ description: undefined });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            const titleEl = screen.getByText('Fix AC unit');
            const allPs = titleEl.closest('a').querySelectorAll('p');
            expect(allPs).toHaveLength(1);
        });

        it('renders a status badge with the correct translation key for "scheduled"', () => {
            const job = makeJob({ status: 'scheduled' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('status.scheduled')).toBeInTheDocument();
        });

        it('renders a status badge with the correct translation key for "in_progress"', () => {
            const job = makeJob({ status: 'in_progress' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('status.inProgress')).toBeInTheDocument();
        });

        it('renders a status badge with the correct translation key for "completed"', () => {
            const job = makeJob({ status: 'completed' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('status.completed')).toBeInTheDocument();
        });

        it('renders a status badge with the correct translation key for "cancelled"', () => {
            const job = makeJob({ status: 'cancelled' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('status.cancelled')).toBeInTheDocument();
        });

        it('falls back to rendering the raw status string for an unknown status', () => {
            const job = makeJob({ status: 'pending_review' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText('pending_review')).toBeInTheDocument();
        });

        it('renders a formatted time (HH:MM) in the time badge', () => {
            // 2024-01-22T09:30:00 local → "09:30 AM" in en-US 12-hour format
            // We only check that SOME time-like string appears; exact format
            // depends on the test runner locale but must match HH:MM pattern.
            const job = makeJob({ scheduled_at: '2024-01-22T09:30:00' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText(/\d{1,2}:\d{2}/)).toBeInTheDocument();
        });

        it('wraps each job row in a link to /jobs/:id', () => {
            const job = makeJob({ id: 'abc-123' });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            const titleEl = screen.getByText('Fix AC unit');
            expect(titleEl.closest('a')).toHaveAttribute('href', '/jobs/abc-123');
        });
    });

    // ── Day grouping ──────────────────────────────────────────────────────────

    describe('day grouping', () => {
        it('renders a day header for a day that has jobs', () => {
            // WEEK_MONDAY is 2024-01-22 → "Monday, Jan 22"
            const job = makeJob({ scheduled_at: WEEK_MONDAY.toISOString() });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.getByText(/monday/i)).toBeInTheDocument();
        });

        it('does not render a day header for days with no jobs', () => {
            // Only Monday has a job; Tuesday and Thursday should not appear.
            const job = makeJob({ scheduled_at: WEEK_MONDAY.toISOString() });
            render(<WeekSchedulePanel weekJobs={[job]} t={t} />);
            expect(screen.queryByText(/tuesday/i)).not.toBeInTheDocument();
            expect(screen.queryByText(/thursday/i)).not.toBeInTheDocument();
        });

        it('renders multiple day headers when jobs fall on different days', () => {
            const monday  = makeJob({ id: 'j1', scheduled_at: WEEK_MONDAY.toISOString() });
            const tuesday = makeJob({ id: 'j2', scheduled_at: WEEK_TUESDAY.toISOString() });
            render(<WeekSchedulePanel weekJobs={[monday, tuesday]} t={t} />);
            expect(screen.getByText(/monday/i)).toBeInTheDocument();
            expect(screen.getByText(/tuesday/i)).toBeInTheDocument();
        });

        it('renders all jobs under the same day header when they share a date', () => {
            const job1 = makeJob({ id: 'j1', title: 'Morning AC fix',   scheduled_at: '2024-01-22T08:00:00' });
            const job2 = makeJob({ id: 'j2', title: 'Afternoon furnace', scheduled_at: '2024-01-22T14:30:00' });
            render(<WeekSchedulePanel weekJobs={[job1, job2]} t={t} />);

            // Only one Monday header
            const mondayHeaders = screen.getAllByText(/monday/i);
            expect(mondayHeaders).toHaveLength(1);

            // Both job titles present
            expect(screen.getByText('Morning AC fix')).toBeInTheDocument();
            expect(screen.getByText('Afternoon furnace')).toBeInTheDocument();
        });

        it('ignores jobs scheduled outside the current week', () => {
            const futureJob = makeJob({
                id:           'future-1',
                title:        'Next week job',
                scheduled_at: NEXT_WEEK_DATE.toISOString(),
            });
            render(<WeekSchedulePanel weekJobs={[futureJob]} t={t} />);
            // weekJobs is passed in already — if the component filters by week it
            // won't show the job; if it doesn't filter it will. Since the component
            // only builds headers for daysWithJobs (Mon–Sun of current week) and
            // NEXT_WEEK_DATE is outside that window, no header/job should appear.
            expect(screen.queryByText('Next week job')).not.toBeInTheDocument();
            expect(screen.getByText('dashboard.noJobsThisWeek')).toBeInTheDocument();
        });
    });

    // ── Today highlight ───────────────────────────────────────────────────────

    describe('today highlight', () => {
        it("applies bg-blue-600 to today's day header", () => {
            // Pinned today = Wednesday 2024-01-24
            const todayJob = makeJob({
                id:           'today-job',
                scheduled_at: '2024-01-24T11:00:00',
            });
            render(<WeekSchedulePanel weekJobs={[todayJob]} t={t} />);

            const header = screen.getByText(/wednesday/i);
            expect(header).toHaveClass('bg-blue-600');
            expect(header).not.toHaveClass('bg-gray-50');
        });

        it("does not apply bg-blue-600 to a non-today day header", () => {
            // Monday is NOT today (today = Wednesday)
            const mondayJob = makeJob({
                id:           'mon-job',
                scheduled_at: WEEK_MONDAY.toISOString(),
            });
            render(<WeekSchedulePanel weekJobs={[mondayJob]} t={t} />);

            const header = screen.getByText(/monday/i);
            expect(header).not.toHaveClass('bg-blue-600');
            expect(header).toHaveClass('bg-gray-50');
        });

        it('applies bg-blue-600 to today and bg-gray-50 to another day simultaneously', () => {
            const todayJob  = makeJob({ id: 'j-today', scheduled_at: '2024-01-24T09:00:00' });
            const mondayJob = makeJob({ id: 'j-mon',   scheduled_at: WEEK_MONDAY.toISOString() });
            render(<WeekSchedulePanel weekJobs={[todayJob, mondayJob]} t={t} />);

            expect(screen.getByText(/wednesday/i)).toHaveClass('bg-blue-600');
            expect(screen.getByText(/monday/i)).toHaveClass('bg-gray-50');
        });
    });

    // ── Multiple jobs on same day ─────────────────────────────────────────────

    describe('multiple jobs on the same day', () => {
        it('renders three jobs on the same day all under one header', () => {
            const jobs = [
                makeJob({ id: 'j1', title: 'Job Alpha',   scheduled_at: '2024-01-25T08:00:00' }),
                makeJob({ id: 'j2', title: 'Job Beta',    scheduled_at: '2024-01-25T11:00:00' }),
                makeJob({ id: 'j3', title: 'Job Gamma',   scheduled_at: '2024-01-25T15:00:00' }),
            ];
            render(<WeekSchedulePanel weekJobs={jobs} t={t} />);

            const thursdayHeaders = screen.getAllByText(/thursday/i);
            expect(thursdayHeaders).toHaveLength(1);

            expect(screen.getByText('Job Alpha')).toBeInTheDocument();
            expect(screen.getByText('Job Beta')).toBeInTheDocument();
            expect(screen.getByText('Job Gamma')).toBeInTheDocument();
        });

        it('renders each job with its own time badge when multiple jobs share a day', () => {
            const jobs = [
                makeJob({ id: 'j1', title: 'Early',  scheduled_at: '2024-01-22T07:00:00', description: '' }),
                makeJob({ id: 'j2', title: 'Late',   scheduled_at: '2024-01-22T16:45:00', description: '' }),
            ];
            render(<WeekSchedulePanel weekJobs={jobs} t={t} />);
            // Two different time strings should be present
            const timeBadges = screen.getAllByText(/\d{1,2}:\d{2}/);
            expect(timeBadges.length).toBeGreaterThanOrEqual(2);
        });
    });

    // ── Smoke: no crash with large dataset ───────────────────────────────────

    it('renders without error when all 7 days of the week have jobs', () => {
        const dates = [
            '2024-01-22T10:00:00', // Mon
            '2024-01-23T10:00:00', // Tue
            '2024-01-24T10:00:00', // Wed (today)
            '2024-01-25T10:00:00', // Thu
            '2024-01-26T10:00:00', // Fri
            '2024-01-27T10:00:00', // Sat
            '2024-01-28T10:00:00', // Sun
        ];
        const jobs = dates.map((dt, i) =>
            makeJob({ id: `j${i}`, title: `Job ${i}`, scheduled_at: dt })
        );
        render(<WeekSchedulePanel weekJobs={jobs} t={t} />);
        expect(screen.getByText(/monday/i)).toBeInTheDocument();
        expect(screen.getByText(/sunday/i)).toBeInTheDocument();
        // Wednesday (today) must be highlighted
        expect(screen.getByText(/wednesday/i)).toHaveClass('bg-blue-600');
    });
});

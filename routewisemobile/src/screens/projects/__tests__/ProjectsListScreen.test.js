import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { jobs } from '../../../services/api';
import ProjectsListScreen from '../ProjectsListScreen';

jest.mock('../../../services/api', () => ({
  jobs: { getMyJobs: jest.fn() },
  storage: { clearAll: jest.fn() },
}));

const mockNavigate = jest.fn();
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: mockNavigate }),
  useFocusEffect: (cb) => require('react').useEffect(cb, []),
}));

const renderScreen = (language = 'en') =>
  render(
    <LanguageProvider initialLanguage={language}>
      <ProjectsListScreen />
    </LanguageProvider>
  );

const job = (id, overrides = {}) => ({
  id,
  title: `Job ${id}`,
  status: 'scheduled',
  assignment_status: 'accepted',
  scheduled_at: '2026-10-01T09:00:00Z',
  customer: { name: 'Dana', address: 'Herzl 1' },
  ...overrides,
});

afterEach(() => {
  jest.clearAllMocks();
});

describe('ProjectsListScreen', () => {
  it('marks pending offers with the "new — respond" badge', async () => {
    jobs.getMyJobs.mockResolvedValue([job(1), job(2, { assignment_status: 'pending' })]);
    renderScreen();

    expect(await screen.findByTestId('new-2')).toHaveTextContent(/New — respond/);
    expect(screen.queryByTestId('new-1')).toBeNull();
  });

  it('lists pending offers first, keeping the rest in server order', async () => {
    jobs.getMyJobs.mockResolvedValue([
      job(1),
      job(2, { assignment_status: 'pending' }),
      job(3),
      job(4, { assignment_status: 'pending' }),
    ]);
    renderScreen();

    await screen.findByText('Job 1');
    const titles = screen.getAllByText(/^Job \d$/).map((n) => n.props.children);
    expect(titles).toEqual(['Job 2', 'Job 4', 'Job 1', 'Job 3']);
  });

  it('shows no badge when nothing is pending', async () => {
    jobs.getMyJobs.mockResolvedValue([job(1), job(2, { status: 'in_progress' })]);
    renderScreen();

    await screen.findByText('Job 2');
    expect(screen.queryByText(/New — respond/)).toBeNull();
  });

  it('opens the job detail when a card is pressed', async () => {
    jobs.getMyJobs.mockResolvedValue([job(5, { assignment_status: 'pending' })]);
    renderScreen();

    fireEvent.press(await screen.findByText('Job 5'));

    expect(mockNavigate).toHaveBeenCalledWith('ProjectDetail', { jobId: 5 });
  });

  it('handles an empty / null response', async () => {
    jobs.getMyJobs.mockResolvedValue(null);
    renderScreen();

    await screen.findByText('My Jobs');
    expect(screen.queryByText(/New — respond/)).toBeNull();
  });

  it('renders the badge in Hebrew', async () => {
    jobs.getMyJobs.mockResolvedValue([job(2, { assignment_status: 'pending' })]);
    renderScreen('he');

    const badge = await screen.findByTestId('new-2');
    expect(badge).not.toHaveTextContent(/New — respond/);
  });
});

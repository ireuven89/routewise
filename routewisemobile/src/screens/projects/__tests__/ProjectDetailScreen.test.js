import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { LanguageProvider } from '../../../i18n/LanguageContext';
import { jobs } from '../../../services/api';
import { notify } from '../../../utils/confirm';
import ProjectDetailScreen from '../ProjectDetailScreen';

jest.mock('../../../services/api', () => ({
  jobs: {
    getJobDetails: jest.fn(),
    accept: jest.fn(),
    decline: jest.fn(),
    updateStatus: jest.fn(),
  },
}));

jest.mock('../../../utils/confirm', () => ({ notify: jest.fn(), confirm: jest.fn() }));

const makeJob = (overrides = {}) => ({
  id: 42,
  title: 'Fix AC',
  status: 'scheduled',
  assignment_status: 'pending',
  scheduled_at: '2026-10-01T09:00:00Z',
  customer: { name: 'Dana', phone: '050', address: 'Herzl 1' },
  ...overrides,
});

const renderScreen = (language = 'en') => {
  const navigation = { navigate: jest.fn(), goBack: jest.fn() };
  render(
    <LanguageProvider initialLanguage={language}>
      <ProjectDetailScreen navigation={navigation} route={{ params: { jobId: 42 } }} />
    </LanguageProvider>
  );
  return { navigation };
};

let consoleError;
beforeEach(() => {
  consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});
});
afterEach(() => {
  consoleError.mockRestore();
  jest.clearAllMocks();
});

describe('ProjectDetailScreen — pending offer', () => {
  it('shows the offer card with accept/decline and no start button', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    renderScreen();

    expect(await screen.findByTestId('assignment-offer')).toHaveTextContent(/You were assigned this job/);
    expect(screen.getByTestId('accept-job')).toBeTruthy();
    expect(screen.getByTestId('decline-job')).toBeTruthy();
    expect(screen.queryByTestId('start-job')).toBeNull();
    expect(screen.queryByTestId('complete-job')).toBeNull();
    expect(jobs.getJobDetails).toHaveBeenCalledWith(42);
  });

  it('accepting calls the API and reloads into the start button', async () => {
    jobs.getJobDetails
      .mockResolvedValueOnce(makeJob())
      .mockResolvedValueOnce(makeJob({ assignment_status: 'accepted' }));
    jobs.accept.mockResolvedValue({ message: 'Job accepted' });
    renderScreen();

    fireEvent.press(await screen.findByTestId('accept-job'));

    expect(await screen.findByTestId('start-job')).toHaveTextContent(/Start job/);
    expect(jobs.accept).toHaveBeenCalledWith(42);
    expect(jobs.getJobDetails).toHaveBeenCalledTimes(2);
    expect(screen.queryByTestId('assignment-offer')).toBeNull();
  });

  it('shows the server error when accepting fails and keeps the offer', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    jobs.accept.mockRejectedValue({ response: { data: { error: 'job is not in a state that allows this' } } });
    renderScreen();

    fireEvent.press(await screen.findByTestId('accept-job'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'job is not in a state that allows this'));
    expect(screen.getByTestId('accept-job')).toBeTruthy();
    expect(jobs.getJobDetails).toHaveBeenCalledTimes(1);
  });

  it('falls back to a generic message when the error has no body', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    jobs.accept.mockRejectedValue(new Error('Network Error'));
    renderScreen();

    fireEvent.press(await screen.findByTestId('accept-job'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith(
      'Error', 'Could not update the job. Pull to refresh and try again.',
    ));
  });
});

describe('ProjectDetailScreen — declining', () => {
  it('opens the decline dialog only after pressing decline', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    renderScreen();

    await screen.findByTestId('decline-job');
    expect(screen.queryByTestId('decline-reason')).toBeNull();

    fireEvent.press(screen.getByTestId('decline-job'));

    expect(screen.getByTestId('decline-reason')).toBeTruthy();
    expect(screen.getByText('Decline this job?')).toBeTruthy();
    expect(jobs.decline).not.toHaveBeenCalled();
  });

  it('declines with a trimmed reason and navigates back', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    jobs.decline.mockResolvedValue({ message: 'Job declined' });
    const { navigation } = renderScreen();

    fireEvent.press(await screen.findByTestId('decline-job'));
    fireEvent.changeText(screen.getByTestId('decline-reason'), '  Too far away  ');
    fireEvent.press(screen.getByTestId('confirm-decline'));

    await waitFor(() => expect(navigation.goBack).toHaveBeenCalled());
    expect(jobs.decline).toHaveBeenCalledWith(42, 'Too far away');
    // the job left this technician's list; no reload
    expect(jobs.getJobDetails).toHaveBeenCalledTimes(1);
  });

  it('declines without a reason', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    jobs.decline.mockResolvedValue({});
    const { navigation } = renderScreen();

    fireEvent.press(await screen.findByTestId('decline-job'));
    fireEvent.press(screen.getByTestId('confirm-decline'));

    await waitFor(() => expect(navigation.goBack).toHaveBeenCalled());
    expect(jobs.decline).toHaveBeenCalledWith(42, '');
  });

  it('cancel closes the dialog without declining', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    const { navigation } = renderScreen();

    fireEvent.press(await screen.findByTestId('decline-job'));
    fireEvent.press(screen.getByText('Cancel'));

    await waitFor(() => expect(screen.queryByTestId('decline-reason')).toBeNull());
    expect(jobs.decline).not.toHaveBeenCalled();
    expect(navigation.goBack).not.toHaveBeenCalled();
  });

  it('stays on the screen and reports the error when declining fails', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob({ status: 'in_progress' }));
    jobs.decline.mockRejectedValue({ response: { data: { error: 'job is not in a state that allows this' } } });
    const { navigation } = renderScreen();

    fireEvent.press(await screen.findByTestId('decline-job'));
    fireEvent.press(screen.getByTestId('confirm-decline'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'job is not in a state that allows this'));
    expect(navigation.goBack).not.toHaveBeenCalled();
  });
});

describe('ProjectDetailScreen — working an accepted job', () => {
  it('starts a scheduled accepted job', async () => {
    jobs.getJobDetails
      .mockResolvedValueOnce(makeJob({ assignment_status: 'accepted' }))
      .mockResolvedValueOnce(makeJob({ assignment_status: 'accepted', status: 'in_progress' }));
    jobs.updateStatus.mockResolvedValue({});
    renderScreen();

    fireEvent.press(await screen.findByTestId('start-job'));

    expect(await screen.findByTestId('complete-job')).toHaveTextContent(/Complete job/);
    expect(jobs.updateStatus).toHaveBeenCalledWith(42, 'in_progress');
  });

  it('completes an in-progress job', async () => {
    jobs.getJobDetails
      .mockResolvedValueOnce(makeJob({ assignment_status: 'accepted', status: 'in_progress' }))
      .mockResolvedValueOnce(makeJob({ assignment_status: 'accepted', status: 'completed' }));
    jobs.updateStatus.mockResolvedValue({});
    renderScreen();

    fireEvent.press(await screen.findByTestId('complete-job'));

    await waitFor(() => expect(jobs.updateStatus).toHaveBeenCalledWith(42, 'completed'));
    await waitFor(() => expect(screen.queryByTestId('complete-job')).toBeNull());
    expect(screen.queryByTestId('start-job')).toBeNull();
  });

  it.each(['completed', 'cancelled'])('shows no action for a %s job', async (status) => {
    jobs.getJobDetails.mockResolvedValue(makeJob({ assignment_status: 'accepted', status }));
    renderScreen();

    await screen.findByText('Fix AC');
    ['accept-job', 'decline-job', 'start-job', 'complete-job'].forEach((id) =>
      expect(screen.queryByTestId(id)).toBeNull());
  });

  it('surfaces "accept first" from the server when starting', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob({ assignment_status: 'accepted' }));
    jobs.updateStatus.mockRejectedValue({ response: { data: { error: 'Accept the job first' } } });
    renderScreen();

    fireEvent.press(await screen.findByTestId('start-job'));

    await waitFor(() => expect(notify).toHaveBeenCalledWith('Error', 'Accept the job first'));
  });
});

describe('ProjectDetailScreen — Hebrew', () => {
  it('renders the offer in Hebrew', async () => {
    jobs.getJobDetails.mockResolvedValue(makeJob());
    renderScreen('he');

    const offer = await screen.findByTestId('assignment-offer');
    expect(offer).not.toHaveTextContent(/You were assigned this job/);
    expect(screen.getByTestId('accept-job')).not.toHaveTextContent(/Accept job/);
  });
});

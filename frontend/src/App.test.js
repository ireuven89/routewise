import { render, screen } from '@testing-library/react';
import App from './App';

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem('language', 'en');
  window.history.pushState({}, '', '/');
});

test('redirects a signed-out visitor from / to the login page', async () => {
  render(<App />);
  expect(await screen.findByText('Welcome back')).toBeInTheDocument();
  expect(window.location.pathname).toBe('/login');
});

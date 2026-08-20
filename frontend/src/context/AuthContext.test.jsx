/**
 * Unit tests for AuthContext.jsx.
 *
 * Prior to this feature, the provider never exposed `organization` at all, even though
 * Login.jsx stored it to localStorage directly — every page reading useAuth().organization
 * silently got undefined. These tests cover the fix: hydration from localStorage on mount,
 * persistence via login()/register(), the new updateOrganization() helper, and logout().
 */

import React from 'react';
import { render, screen, act, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

const mockLogin = jest.fn();
const mockRegister = jest.fn();
jest.mock('../api/client', () => ({
    authAPI: {
        login: (...args) => mockLogin(...args),
        register: (...args) => mockRegister(...args),
    },
}));

import { AuthProvider, useAuth } from './AuthContext';

const ORG = { id: 1, name: 'RouteWise Plumbing', bit_payment_enabled: false };
const USER = { id: 1, email: 'owner@routewise.test' };

/** Test consumer exposing context state/actions as clickable buttons. */
const Consumer = () => {
    const { user, organization, login, register, logout, updateOrganization, loading } = useAuth();
    return (
        <div>
            <div data-testid="loading">{String(loading)}</div>
            <div data-testid="user">{user ? user.email : 'none'}</div>
            <div data-testid="organization">{organization ? organization.name : 'none'}</div>
            <button onClick={() => login('owner@routewise.test', 'secret')}>login</button>
            <button onClick={() => register({ email: 'owner@routewise.test' })}>register</button>
            <button onClick={() => updateOrganization({ bit_payment_enabled: true })}>update-org</button>
            <button onClick={logout}>logout</button>
        </div>
    );
};

const renderWithProvider = () =>
    render(
        <AuthProvider>
            <Consumer />
        </AuthProvider>
    );

beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
});

describe('AuthContext', () => {
    describe('mount hydration', () => {
        it('starts with no user/organization when localStorage is empty', async () => {
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
            expect(screen.getByTestId('user')).toHaveTextContent('none');
            expect(screen.getByTestId('organization')).toHaveTextContent('none');
        });

        it('hydrates both user and organization from localStorage when a token exists', async () => {
            localStorage.setItem('token', 'test-token');
            localStorage.setItem('user', JSON.stringify(USER));
            localStorage.setItem('organization', JSON.stringify(ORG));

            renderWithProvider();

            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
            expect(screen.getByTestId('user')).toHaveTextContent(USER.email);
            expect(screen.getByTestId('organization')).toHaveTextContent(ORG.name);
        });

        it('does not hydrate organization if no token is present, even if organization is in localStorage', async () => {
            localStorage.setItem('organization', JSON.stringify(ORG));
            renderWithProvider();

            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
            expect(screen.getByTestId('organization')).toHaveTextContent('none');
        });
    });

    describe('login', () => {
        it('persists and exposes organization from the login response', async () => {
            mockLogin.mockResolvedValueOnce({ data: { token: 't1', user: USER, organization: ORG } });
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));

            await act(async () => {
                screen.getByText('login').click();
            });

            expect(screen.getByTestId('organization')).toHaveTextContent(ORG.name);
            expect(JSON.parse(localStorage.getItem('organization'))).toEqual(ORG);
        });

        it('does not set organization if the login response omits it', async () => {
            mockLogin.mockResolvedValueOnce({ data: { token: 't1', user: USER } });
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));

            await act(async () => {
                screen.getByText('login').click();
            });

            expect(screen.getByTestId('organization')).toHaveTextContent('none');
            expect(localStorage.getItem('organization')).toBeNull();
        });
    });

    describe('register', () => {
        it('persists and exposes organization from the register response', async () => {
            mockRegister.mockResolvedValueOnce({ data: { token: 't1', user: USER, organization: ORG } });
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));

            await act(async () => {
                screen.getByText('register').click();
            });

            expect(screen.getByTestId('organization')).toHaveTextContent(ORG.name);
            expect(JSON.parse(localStorage.getItem('organization'))).toEqual(ORG);
        });
    });

    describe('updateOrganization', () => {
        it('merges the patch into the existing organization and persists it', async () => {
            localStorage.setItem('token', 'test-token');
            localStorage.setItem('user', JSON.stringify(USER));
            localStorage.setItem('organization', JSON.stringify(ORG));
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));

            await act(async () => {
                screen.getByText('update-org').click();
            });

            const stored = JSON.parse(localStorage.getItem('organization'));
            expect(stored).toEqual({ ...ORG, bit_payment_enabled: true });
            // Name is preserved (merge, not replace)
            expect(screen.getByTestId('organization')).toHaveTextContent(ORG.name);
        });
    });

    describe('logout', () => {
        it('clears user and organization from state and localStorage', async () => {
            localStorage.setItem('token', 'test-token');
            localStorage.setItem('user', JSON.stringify(USER));
            localStorage.setItem('organization', JSON.stringify(ORG));
            renderWithProvider();
            await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));

            await act(async () => {
                screen.getByText('logout').click();
            });

            expect(screen.getByTestId('user')).toHaveTextContent('none');
            expect(screen.getByTestId('organization')).toHaveTextContent('none');
            expect(localStorage.getItem('token')).toBeNull();
            expect(localStorage.getItem('user')).toBeNull();
            expect(localStorage.getItem('organization')).toBeNull();
        });
    });
});

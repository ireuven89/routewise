import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import PaymentLinkButton from './PaymentLinkButton';
import { paymentAPI } from '../api/client';
import { useLanguage } from '../context/LanguageContext';

// Mock the API module
jest.mock('../api/client', () => ({
    paymentAPI: {
        sendPaymentLink: jest.fn(),
    },
}));

// Mock the useLanguage hook
jest.mock('../context/LanguageContext', () => ({
    useLanguage: jest.fn(),
}));

// Mock window.alert
const mockAlert = jest.fn();
global.alert = mockAlert;

describe('PaymentLinkButton', () => {
    const mockTranslations = {
        'payments.sendLink': 'Send Payment Link',
        'payments.sending': 'Sending...',
        'payments.linkSent': 'Payment link sent successfully',
        'payments.linkFailed': 'Failed to send payment link',
    };

    const mockT = (key) => mockTranslations[key] || key;

    beforeEach(() => {
        jest.clearAllMocks();
        useLanguage.mockReturnValue({ t: mockT });
    });

    describe('Rendering Conditions', () => {
        test('should not render for jobs that are not completed', () => {
            const job = { id: 1, status: 'pending', price: 100 };
            const { container } = render(<PaymentLinkButton job={job} />);
            expect(container.firstChild).toBeNull();
        });

        test('should not render for jobs with in_progress status', () => {
            const job = { id: 1, status: 'in_progress', price: 100 };
            const { container } = render(<PaymentLinkButton job={job} />);
            expect(container.firstChild).toBeNull();
        });

        test('should not render for completed jobs without price', () => {
            const job = { id: 1, status: 'completed', price: null };
            const { container } = render(<PaymentLinkButton job={job} />);
            expect(container.firstChild).toBeNull();
        });

        test('should not render for completed jobs with zero price', () => {
            const job = { id: 1, status: 'completed', price: 0 };
            const { container } = render(<PaymentLinkButton job={job} />);
            expect(container.firstChild).toBeNull();
        });

        test('should render for completed jobs with price', () => {
            const job = { id: 1, status: 'completed', price: 100 };
            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');
            expect(button).toBeInTheDocument();
        });
    });

    describe('Button Display States', () => {
        test('should display "Send Payment Link" text when idle', () => {
            const job = { id: 1, status: 'completed', price: 100 };
            render(<PaymentLinkButton job={job} />);
            expect(screen.getByText('Send Payment Link')).toBeInTheDocument();
        });

        test('should display send icon when idle', () => {
            const job = { id: 1, status: 'completed', price: 100 };
            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');
            // Check for FiSend icon by checking the svg element
            const svg = button.querySelector('svg');
            expect(svg).toBeInTheDocument();
        });

        test('should show loading state with spinner when sending', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockImplementation(
                () => new Promise((resolve) => setTimeout(resolve, 100))
            );

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            // Check for loading state
            await waitFor(() => {
                expect(screen.getByText('Sending...')).toBeInTheDocument();
            });

            // Check for spinner
            const spinner = button.querySelector('.animate-spin');
            expect(spinner).toBeInTheDocument();
        });

        test('should return to idle state after sending completes', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(screen.getByText('Send Payment Link')).toBeInTheDocument();
            });
        });
    });

    describe('API Interaction', () => {
        test('should call paymentAPI.sendPaymentLink with correct job ID', async () => {
            const job = { id: 42, status: 'completed', price: 200 };
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(paymentAPI.sendPaymentLink).toHaveBeenCalledWith(42);
                expect(paymentAPI.sendPaymentLink).toHaveBeenCalledTimes(1);
            });
        });

        test('should show success alert on successful send', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledWith('Payment link sent successfully');
            });
        });

        test('should call onSuccess callback on success', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const onSuccess = jest.fn();
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} onSuccess={onSuccess} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(onSuccess).toHaveBeenCalledTimes(1);
            });
        });

        test('should not call onSuccess callback if not provided', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            // Should not throw error
            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledWith('Payment link sent successfully');
            });
        });
    });

    describe('Error Handling', () => {
        test('should show error alert with API error message on failure', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const apiError = {
                response: {
                    data: {
                        error: 'Customer phone number is missing',
                    },
                },
            };
            paymentAPI.sendPaymentLink.mockRejectedValue(apiError);

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledWith('Customer phone number is missing');
            });
        });

        test('should show default error message when API error has no message', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const apiError = {
                response: {
                    data: {},
                },
            };
            paymentAPI.sendPaymentLink.mockRejectedValue(apiError);

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledWith('Failed to send payment link');
            });
        });

        test('should handle network errors gracefully', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const networkError = new Error('Network Error');
            paymentAPI.sendPaymentLink.mockRejectedValue(networkError);

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledWith('Failed to send payment link');
            });
        });

        test('should return to idle state after error', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockRejectedValue(new Error('Test error'));

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalled();
            });

            // Button should be back to idle state
            expect(screen.getByText('Send Payment Link')).toBeInTheDocument();
            expect(button).not.toBeDisabled();
        });

        test('should not call onSuccess callback on failure', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const onSuccess = jest.fn();
            paymentAPI.sendPaymentLink.mockRejectedValue(new Error('Test error'));

            render(<PaymentLinkButton job={job} onSuccess={onSuccess} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalled();
            });

            expect(onSuccess).not.toHaveBeenCalled();
        });
    });

    describe('Button Interaction', () => {
        test('should be disabled while sending (prevent double-clicks)', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockImplementation(
                () => new Promise((resolve) => setTimeout(resolve, 100))
            );

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            // Button should be disabled immediately
            await waitFor(() => {
                expect(button).toBeDisabled();
            });
        });

        test('should prevent multiple API calls on rapid clicks', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockImplementation(
                () => new Promise((resolve) => setTimeout(() => resolve({ data: {} }), 100))
            );

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            // Rapidly click the button
            fireEvent.click(button);
            fireEvent.click(button);
            fireEvent.click(button);

            await waitFor(() => {
                expect(mockAlert).toHaveBeenCalledTimes(1);
            });

            // API should only be called once
            expect(paymentAPI.sendPaymentLink).toHaveBeenCalledTimes(1);
        });

        test('should re-enable button after successful send', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(button).not.toBeDisabled();
            });
        });

        test('should have correct CSS classes for styling', () => {
            const job = { id: 1, status: 'completed', price: 100 };
            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            expect(button).toHaveClass('flex');
            expect(button).toHaveClass('items-center');
            expect(button).toHaveClass('gap-2');
            expect(button).toHaveClass('px-3');
            expect(button).toHaveClass('py-1.5');
            expect(button).toHaveClass('bg-orange-500');
            expect(button).toHaveClass('text-white');
            expect(button).toHaveClass('rounded-md');
        });
    });

    describe('Translation Integration', () => {
        test('should use translation keys for all text', () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const mockT = jest.fn((key) => mockTranslations[key] || key);
            useLanguage.mockReturnValue({ t: mockT });

            render(<PaymentLinkButton job={job} />);

            expect(mockT).toHaveBeenCalledWith('payments.sendLink');
        });

        test('should use translated text in loading state', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const mockT = jest.fn((key) => mockTranslations[key] || key);
            useLanguage.mockReturnValue({ t: mockT });

            paymentAPI.sendPaymentLink.mockImplementation(
                () => new Promise((resolve) => setTimeout(resolve, 100))
            );

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockT).toHaveBeenCalledWith('payments.sending');
            });
        });

        test('should use translated text in alerts', async () => {
            const job = { id: 1, status: 'completed', price: 100 };
            const mockT = jest.fn((key) => mockTranslations[key] || key);
            useLanguage.mockReturnValue({ t: mockT });

            paymentAPI.sendPaymentLink.mockResolvedValue({ data: {} });

            render(<PaymentLinkButton job={job} />);
            const button = screen.getByRole('button');

            fireEvent.click(button);

            await waitFor(() => {
                expect(mockT).toHaveBeenCalledWith('payments.linkSent');
            });
        });
    });
});

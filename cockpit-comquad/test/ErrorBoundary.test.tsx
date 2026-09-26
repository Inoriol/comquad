import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorBoundary } from '../src/ErrorBoundary';

describe('ErrorBoundary', () => {
    const ProblemChild = () => {
        throw new Error('Test error');
    };

    const GoodChild = () => <div>All good</div>;

    it('should render children when there is no error', () => {
        render(
            <ErrorBoundary>
                <GoodChild />
            </ErrorBoundary>
        );

        expect(screen.getByText('All good')).toBeInTheDocument();
    });

    it('should render error UI when child throws', () => {
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

        render(
            <ErrorBoundary>
                <ProblemChild />
            </ErrorBoundary>
        );

        expect(screen.getByText('Something went wrong')).toBeInTheDocument();
        expect(screen.getByText('Test error')).toBeInTheDocument();
        expect(screen.getByText('Try again')).toBeInTheDocument();

        consoleError.mockRestore();
    });

    it('should render custom fallback when provided', () => {
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
        const customFallback = <div>Custom error message</div>;

        render(
            <ErrorBoundary fallback={customFallback}>
                <ProblemChild />
            </ErrorBoundary>
        );

        expect(screen.getByText('Custom error message')).toBeInTheDocument();
        expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();

        consoleError.mockRestore();
    });

    it('should recover when Try again is clicked', () => {
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

        let shouldThrow = true;
        const ConditionalChild = () => {
            if (shouldThrow) {
                throw new Error('Test error');
            }
            return <div>Recovered</div>;
        };

        render(
            <ErrorBoundary>
                <ConditionalChild />
            </ErrorBoundary>
        );

        expect(screen.getByText('Something went wrong')).toBeInTheDocument();

        shouldThrow = false;
        fireEvent.click(screen.getByText('Try again'));

        expect(screen.getByText('Recovered')).toBeInTheDocument();

        consoleError.mockRestore();
    });
});

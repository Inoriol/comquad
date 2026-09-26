import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from '../src/app';
import * as client from '../src/client';

vi.mock('../src/client');

describe('App', () => {
    const mockProjects = [
        { name: 'test-project', status: 'running', services: 2, source_path: '/home/user/test', files: 5 }
    ];

    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(client.listProjects).mockResolvedValue(mockProjects);
    });

    it('should render Comquad Stacks title', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Comquad Stacks')).toBeInTheDocument();
        });
    });

    it('should show loading state initially', () => {
        vi.mocked(client.listProjects).mockImplementation(
            () => new Promise(() => {})
        );

        render(<App />);

        expect(screen.getByText('Loading projects...')).toBeInTheDocument();
    });

    it('should render projects list after loading', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('test-project')).toBeInTheDocument();
        });
    });

    it('should show error state on fetch failure', async () => {
        vi.mocked(client.listProjects).mockRejectedValue(new Error('Network error'));

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Error loading projects')).toBeInTheDocument();
            expect(screen.getByText('Network error')).toBeInTheDocument();
        });
    });

    it('should show empty state when no projects', async () => {
        vi.mocked(client.listProjects).mockResolvedValue([]);

        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('No stacks deployed')).toBeInTheDocument();
            expect(screen.getByText('Deploy your first compose.yaml to get started.')).toBeInTheDocument();
        });
    });

    it('should have Deploy Stack button', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Deploy Stack')).toBeInTheDocument();
        });
    });

    it('should have Refresh button', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByLabelText('Refresh')).toBeInTheDocument();
        });
    });

    it('should reload projects when Refresh is clicked', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('test-project')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByLabelText('Refresh'));

        await waitFor(() => {
            expect(client.listProjects).toHaveBeenCalledTimes(2);
        });
    });

    it('should navigate to deploy view when Deploy Stack is clicked', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('Deploy Stack')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Deploy Stack'));

        await waitFor(() => {
            expect(screen.getByText('Project Directory')).toBeInTheDocument();
        });
    });

    it('should navigate to detail view when project is clicked', async () => {
        render(<App />);

        await waitFor(() => {
            expect(screen.getByText('test-project')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('test-project'));

        await waitFor(() => {
            expect(screen.getByText('Loading project details...')).toBeInTheDocument();
        });
    });
});

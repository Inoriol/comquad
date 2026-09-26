import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { DirectoryPicker } from '../src/DirectoryPicker';
import * as client from '../src/client';

vi.mock('../src/client');

describe('DirectoryPicker', () => {
    const mockOnClose = vi.fn();
    const mockOnSelect = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(client.listDirectory).mockResolvedValue(['dir1', 'dir2', 'dir3']);
    });

    it('should render Select Project Directory title', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Select Project Directory')).toBeInTheDocument();
        });
    });

    it('should render Cancel and Select buttons', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Cancel')).toBeInTheDocument();
            expect(screen.getByText('Select This Directory')).toBeInTheDocument();
        });
    });

    it('should call onClose when Cancel is clicked', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Cancel')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Cancel'));

        expect(mockOnClose).toHaveBeenCalled();
    });

    it('should call onSelect with current path when Select is clicked', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
                initialPath="/home/user"
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Select This Directory')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Select This Directory'));

        expect(mockOnSelect).toHaveBeenCalledWith('/home/user');
    });

    it('should display directories from API', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('dir1')).toBeInTheDocument();
            expect(screen.getByText('dir2')).toBeInTheDocument();
            expect(screen.getByText('dir3')).toBeInTheDocument();
        });
    });

    it('should show No directories found when list is empty', async () => {
        vi.mocked(client.listDirectory).mockResolvedValue([]);

        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('No directories found')).toBeInTheDocument();
        });
    });

    it('should show error on API failure', async () => {
        vi.mocked(client.listDirectory).mockRejectedValue(new Error('Permission denied'));

        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Error')).toBeInTheDocument();
            expect(screen.getByText('Permission denied')).toBeInTheDocument();
        });
    });

    it('should display current path', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
                initialPath="/home/user"
            />
        );

        await waitFor(() => {
            expect(screen.getByText(/Current path:/)).toBeInTheDocument();
            expect(screen.getByText('/home/user')).toBeInTheDocument();
        });
    });

    it('should navigate into directory when clicked', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
                initialPath="/home"
            />
        );

        await waitFor(() => {
            expect(screen.getByText('dir1')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('dir1'));

        await waitFor(() => {
            expect(client.listDirectory).toHaveBeenCalledWith('/home/dir1');
        });
    });

    it('should show Go Up button when not at root', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
                initialPath="/home/user"
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Go Up')).toBeInTheDocument();
        });
    });

    it('should navigate up when Go Up is clicked', async () => {
        render(
            <DirectoryPicker
                isOpen={true}
                onClose={mockOnClose}
                onSelect={mockOnSelect}
                initialPath="/home/user"
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Go Up')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Go Up'));

        await waitFor(() => {
            expect(client.listDirectory).toHaveBeenCalledWith('/home');
        });
    });
});

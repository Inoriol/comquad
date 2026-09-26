import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { StackDeploy } from '../src/StackDeploy';
import * as client from '../src/client';

vi.mock('../src/client');

describe('StackDeploy', () => {
    const mockOnDeployComplete = vi.fn();
    const mockOnCancel = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should render Deploy Stack title', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByRole('heading', { name: 'Deploy Stack' })).toBeInTheDocument();
    });

    it('should render Project Directory form', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByText('Project Directory')).toBeInTheDocument();
    });

    it('should render Browse button', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByText('Browse')).toBeInTheDocument();
    });

    it('should render Preview Changes button', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByText('Preview Changes')).toBeInTheDocument();
    });

    it('should render Cancel button', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByText('Cancel')).toBeInTheDocument();
    });

    it('should call onCancel when Cancel is clicked', async () => {
        const user = userEvent.setup();
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        await user.click(screen.getByText('Cancel'));

        expect(mockOnCancel).toHaveBeenCalled();
    });

    it('should have path input with placeholder', () => {
        render(
            <StackDeploy
                onDeployComplete={mockOnDeployComplete}
                onCancel={mockOnCancel}
            />
        );

        expect(screen.getByPlaceholderText('/home/user/myproject')).toBeInTheDocument();
    });
});

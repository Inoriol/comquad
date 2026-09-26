import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ProjectDetail } from '../src/ProjectDetail';
import * as client from '../src/client';
import type { Project, ViewData, Container } from '../src/types';

vi.mock('../src/client');

describe('ProjectDetail', () => {
    const mockProject: Project = {
        name: 'test-project',
        status: 'running',
        services: 2,
        source_path: '/home/user/test',
        files: 5
    };

    const mockViewData: ViewData = {
        project: 'test-project',
        services: [
            { name: 'web', status: 'running', image: 'nginx', networks: ['default'], volumes: ['/data'] },
            { name: 'db', status: 'running', image: 'postgres', networks: ['default'], volumes: [] }
        ],
        resources: [
            { name: 'default', type: 'network', info: 'bridge' },
            { name: 'data', type: 'volume', info: 'local' }
        ]
    };

    const mockContainers: Container[] = [
        { name: 'web', service: 'web', state: 'running', image: 'nginx', ports: [{ host_ip: '0.0.0.0', host_port: 8080, container_port: 80 }] },
        { name: 'db', service: 'db', state: 'running', image: 'postgres', ports: [] }
    ];

    const mockOnBack = vi.fn();
    const mockOnRefresh = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(client.viewProject).mockResolvedValue(mockViewData);
        vi.mocked(client.projectPs).mockResolvedValue(mockContainers);
    });

    it('should render project name', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('test-project')).toBeInTheDocument();
        });
    });

    it('should render project status', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getAllByText('running').length).toBeGreaterThan(0);
        });
    });

    it('should render source path', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('/home/user/test')).toBeInTheDocument();
        });
    });

    it('should render action buttons', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Update')).toBeInTheDocument();
            expect(screen.getByText('Stop')).toBeInTheDocument();
            expect(screen.getByText('Remove')).toBeInTheDocument();
        });
    });

    it('should render Back button', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByLabelText('Back')).toBeInTheDocument();
        });
    });

    it('should call onBack when Back is clicked', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByLabelText('Back')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByLabelText('Back'));

        expect(mockOnBack).toHaveBeenCalled();
    });

    it('should render Services tab with service data', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getAllByText('web').length).toBeGreaterThan(0);
            expect(screen.getAllByText('nginx').length).toBeGreaterThan(0);
            expect(screen.getAllByText('postgres').length).toBeGreaterThan(0);
        });
    });

    it('should render Containers tab', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Containers')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Containers'));

        await waitFor(() => {
            expect(screen.getAllByText('web').length).toBeGreaterThan(0);
        });
    });

    it('should render Resources tab', async () => {
        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Resources')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Resources'));

        await waitFor(() => {
            expect(screen.getAllByText('default').length).toBeGreaterThan(0);
            expect(screen.getByText('network')).toBeInTheDocument();
        });
    });

    it('should show loading state initially', () => {
        vi.mocked(client.viewProject).mockImplementation(
            () => new Promise(() => {})
        );

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.getByText('Loading project details...')).toBeInTheDocument();
    });

    it('should show error state on fetch failure', async () => {
        vi.mocked(client.viewProject).mockRejectedValue(new Error('Fetch failed'));

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Error loading project details')).toBeInTheDocument();
            expect(screen.getByText('Fetch failed')).toBeInTheDocument();
        });
    });

    it('should show Back to Projects button on error', async () => {
        vi.mocked(client.viewProject).mockRejectedValue(new Error('Fetch failed'));

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Back to Projects')).toBeInTheDocument();
        });
    });

    it('should show No services found when services list is empty', async () => {
        vi.mocked(client.viewProject).mockResolvedValue({
            ...mockViewData,
            services: []
        });

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('No services found')).toBeInTheDocument();
        });
    });

    it('should show No containers found when containers list is empty', async () => {
        vi.mocked(client.projectPs).mockResolvedValue([]);

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Containers')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Containers'));

        await waitFor(() => {
            expect(screen.getByText('No containers found')).toBeInTheDocument();
        });
    });

    it('should show No resources found when resources list is empty', async () => {
        vi.mocked(client.viewProject).mockResolvedValue({
            ...mockViewData,
            resources: []
        });

        render(
            <ProjectDetail
                project={mockProject}
                onBack={mockOnBack}
                onRefresh={mockOnRefresh}
            />
        );

        await waitFor(() => {
            expect(screen.getByText('Resources')).toBeInTheDocument();
        });

        fireEvent.click(screen.getByText('Resources'));

        await waitFor(() => {
            expect(screen.getByText('No resources found')).toBeInTheDocument();
        });
    });
});

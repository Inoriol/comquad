import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ProjectsList } from '../src/ProjectsList';
import type { Project } from '../src/types';

describe('ProjectsList', () => {
    const mockProjects: Project[] = [
        {
            name: 'project-a',
            status: 'running',
            services: 3,
            source_path: '/home/user/project-a',
            files: 5
        },
        {
            name: 'project-b',
            status: 'stopped',
            services: 1,
            source_path: '/home/user/project-b',
            files: 2
        }
    ];

    const mockOnSelect = vi.fn();
    const mockOnRefresh = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('should render table with project data', () => {
        render(
            <ProjectsList
                projects={mockProjects}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.getByText('project-a')).toBeInTheDocument();
        expect(screen.getByText('project-b')).toBeInTheDocument();
        expect(screen.getByText('running')).toBeInTheDocument();
        expect(screen.getByText('stopped')).toBeInTheDocument();
        expect(screen.getByText('/home/user/project-a')).toBeInTheDocument();
        expect(screen.getByText('/home/user/project-b')).toBeInTheDocument();
    });

    it('should render table headers', () => {
        render(
            <ProjectsList
                projects={mockProjects}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.getByText('Name')).toBeInTheDocument();
        expect(screen.getByText('Status')).toBeInTheDocument();
        expect(screen.getByText('Services')).toBeInTheDocument();
        expect(screen.getByText('Source Path')).toBeInTheDocument();
        expect(screen.getByText('Files')).toBeInTheDocument();
    });

    it('should call onSelect when a row is clicked', () => {
        render(
            <ProjectsList
                projects={mockProjects}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        fireEvent.click(screen.getByText('project-a'));

        expect(mockOnSelect).toHaveBeenCalledWith(mockProjects[0]);
    });

    it('should render empty table when no projects', () => {
        render(
            <ProjectsList
                projects={[]}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.queryByText('project-a')).not.toBeInTheDocument();
        expect(screen.queryByText('project-b')).not.toBeInTheDocument();
    });

    it('should display service count as badge', () => {
        render(
            <ProjectsList
                projects={mockProjects}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.getByText('3')).toBeInTheDocument();
        expect(screen.getByText('1')).toBeInTheDocument();
    });

    it('should display file count', () => {
        render(
            <ProjectsList
                projects={mockProjects}
                onSelect={mockOnSelect}
                onRefresh={mockOnRefresh}
            />
        );

        expect(screen.getByText('5')).toBeInTheDocument();
        expect(screen.getByText('2')).toBeInTheDocument();
    });
});

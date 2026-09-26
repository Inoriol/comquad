import React, { useState, useEffect, useCallback } from 'react';
import { Page, PageSection, Title, Button, EmptyState, EmptyStateBody, Spinner } from '@patternfly/react-core';
import { CubesIcon, SyncIcon } from '@patternfly/react-icons';
import { ProjectsList } from './ProjectsList';
import { ProjectDetail } from './ProjectDetail';
import { StackDeploy } from './StackDeploy';
import { ErrorBoundary } from './ErrorBoundary';
import * as client from './client';
import { _ } from './i18n';
import type { Project } from './types';

type View = 'list' | 'detail' | 'deploy';

export const App: React.FC = () => {
    const [view, setView] = useState<View>('list');
    const [projects, setProjects] = useState<Project[]>([]);
    const [selectedProject, setSelectedProject] = useState<Project | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [deploying, setDeploying] = useState(false);

    const loadProjects = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await client.listProjects();
            setProjects(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : _('Failed to load projects'));
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        loadProjects();
    }, [loadProjects]);

    const handleProjectSelect = (project: Project) => {
        setSelectedProject(project);
        setView('detail');
    };

    const handleBack = () => {
        setView('list');
        setSelectedProject(null);
        loadProjects(); // Refresh list
    };

    const handleDeploy = () => {
        setView('deploy');
    };

    const handleDeployComplete = () => {
        setView('list');
        loadProjects();
    };

    const handleDeployCancel = () => {
        setView('list');
    };

    if (view === 'deploy') {
        return (
            <StackDeploy
                onDeployComplete={handleDeployComplete}
                onCancel={handleDeployCancel}
          />
        );
    }

    if (view === 'detail' && selectedProject) {
        return (
            <ErrorBoundary>
                <ProjectDetail
                    project={selectedProject}
                    onBack={handleBack}
                    onRefresh={loadProjects}
              />
          </ErrorBoundary>
        );
    }

    return (
        <Page>
            <PageSection variant='light'>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Title headingLevel='h1' size='2xl'>
                        {_('Comquad Stacks')}
                  </Title>
                    <div>
                        <Button
                            variant='plain'
                            onClick={loadProjects}
                            isDisabled={loading}
                            aria-label={_('Refresh')}
                      >
                            <SyncIcon />
                      </Button>
                        <Button variant='primary' onClick={handleDeploy}>
                            {_('Deploy Stack')}
                      </Button>
                  </div>
              </div>
          </PageSection>
            <PageSection>
                {error && (
                    <div style={{ textAlign: 'center', padding: '3rem' }}>
                        <CubesIcon style={{ fontSize: '3rem', color: '#6a6e73', marginBottom: '1rem' }} />
                        <Title headingLevel='h2' size='lg'>{_('Error loading projects')}</Title>
                        <p style={{ color: '#6a6e73', marginBottom: '1rem' }}>{error}</p>
                        <Button variant='primary' onClick={loadProjects}>
                            {_('Retry')}
                      </Button>
                  </div>
                )}
                {loading && !error && (
                    <div style={{ textAlign: 'center', padding: '3rem' }}>
                        <Spinner size='xl' />
                        <Title headingLevel='h2' size='lg' style={{ marginTop: '1rem' }}>{_('Loading projects...')}</Title>
                  </div>
                )}
                {!loading && !error && projects.length === 0 && (
                    <div style={{ textAlign: 'center', padding: '3rem' }}>
                        <CubesIcon style={{ fontSize: '3rem', color: '#6a6e73', marginBottom: '1rem' }} />
                        <Title headingLevel='h2' size='lg'>{_('No stacks deployed')}</Title>
                        <p style={{ color: '#6a6e73', marginBottom: '1rem' }}>
                            {_('Deploy your first compose.yaml to get started.')}
                      </p>
                        <Button variant='primary' onClick={handleDeploy}>
                            {_('Deploy Stack')}
                      </Button>
                  </div>
                )}
                {!loading && !error && projects.length > 0 && (
                    <ProjectsList
                        projects={projects}
                        onSelect={handleProjectSelect}
                        onRefresh={loadProjects}
                  />
                )}
          </PageSection>
      </Page>
    );
};

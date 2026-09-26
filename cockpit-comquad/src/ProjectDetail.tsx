import React, { useState, useEffect, useCallback } from 'react';
import {
    Page,
    PageSection,
    Title,
    Button,
    Label,
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    Tabs,
    Tab,
    TabTitleText,
    Card,
    CardBody,
    Modal,
    ModalVariant,
    ModalHeader,
    ModalBody,
    ModalFooter,
    Alert,
    Spinner,
    EmptyState,
    EmptyStateBody,
    ExpandableSection,
    Tooltip
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { ArrowLeftIcon, SyncIcon, ExternalLinkAltIcon, LockIcon, UnlockIcon } from '@patternfly/react-icons';
import * as client from './client';
import { _ } from './i18n';
import type { Project, ViewData, Container, DryRunData } from './types';

interface ProjectDetailProps {
    project: Project;
    onBack: () => void;
    onRefresh: () => void;
}

const getStatusColor = (status: string): 'green' | 'grey' | 'orange' | 'red' | 'blue' => {
    switch (status) {
        case 'healthy':
        case 'running':
            return 'green';
        case 'up':
            return 'blue';
        case 'stopped':
            return 'grey';
        case 'degraded':
            return 'orange';
        case 'failed':
            return 'red';
        default:
            return 'grey';
    }
};

const getFileStatusColor = (status: string): 'green' | 'blue' | 'red' | 'orange' => {
    switch (status) {
        case 'created': return 'green';
        case 'changed': return 'blue';
        case 'removed': return 'red';
        default: return 'orange';
    }
};

export const ProjectDetail: React.FC<ProjectDetailProps> = ({ project, onBack, onRefresh }) => {
    const [activeTab, setActiveTab] = useState(0);
    const [viewData, setViewData] = useState<ViewData | null>(null);
    const [containers, setContainers] = useState<Container[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [actionLoading, setActionLoading] = useState(false);
    const [actionError, setActionError] = useState<string | null>(null);
    const [deleteVolumes, setDeleteVolumes] = useState(false);
    const [confirmModal, setConfirmModal] = useState<{ open: boolean; action: string; title: string }>({
        open: false,
        action: '',
        title: ''
    });
    const [dryRun, setDryRun] = useState<DryRunData | null>(null);
    const [dryRunLoading, setDryRunLoading] = useState(false);
    const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());
    const [resourceModal, setResourceModal] = useState<{ open: boolean; name: string; content: string; loading: boolean }>({
        open: false,
        name: '',
        content: '',
        loading: false
    });
    const [httpsPorts, setHttpsPorts] = useState<Set<string>>(new Set());

    const loadData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const [view, ps] = await Promise.all([
                client.viewProject(project.source_path),
                client.projectPs(project.source_path)
            ]);
            setViewData(view);
            setContainers(ps);
        } catch (err) {
            setError(err instanceof Error ? err.message : _('Failed to load project details'));
        } finally {
            setLoading(false);
        }
    }, [project.source_path]);

    useEffect(() => {
        loadData();
    }, [loadData]);

    const fetchDryRun = useCallback(async () => {
        setDryRunLoading(true);
        setDryRun(null);
        try {
            const data = await client.dryRunProject(project.source_path);
            setDryRun(data);
            const allFiles = new Set(data.files?.map(f => f.name) || []);
            setExpandedFiles(allFiles);
        } catch {
            // silently fail - user can still proceed without preview
        } finally {
            setDryRunLoading(false);
        }
    }, [project.source_path]);

    const handleAction = async (action: 'update' | 'stop' | 'remove') => {
        setActionLoading(true);
        setActionError(null);
        try {
            switch (action) {
                case 'update':
                    await client.updateProject(project.source_path);
                    break;
                case 'stop':
                    await client.stopProject(project.source_path);
                    break;
                case 'remove':
                    await client.removeProject(project.source_path, deleteVolumes);
                    onBack();
                    return;
            }
            await loadData();
            onRefresh();
        } catch (err) {
            setActionError(err instanceof Error ? err.message : _('Failed to ${action} project'));
        } finally {
            setActionLoading(false);
            setConfirmModal({ open: false, action: '', title: '' });
            setDeleteVolumes(false);
            setDryRun(null);
            setExpandedFiles(new Set());
        }
    };

    const openConfirmModal = (action: string, title: string) => {
        setConfirmModal({ open: true, action, title });
        if (action === 'update') {
            fetchDryRun();
        }
    };

    const closeModal = () => {
        setConfirmModal({ open: false, action: '', title: '' });
        setDryRun(null);
        setExpandedFiles(new Set());
        setDeleteVolumes(false);
    };

    const toggleFile = (name: string) => {
        setExpandedFiles(prev => {
            const next = new Set(prev);
            if (next.has(name)) {
                next.delete(name);
            } else {
                next.add(name);
            }
            return next;
        });
    };

    const handleResourceClick = async (resourceName: string) => {
        setResourceModal({ open: true, name: resourceName, content: '', loading: true });
        try {
            const unit = await client.viewUnit(project.source_path, resourceName);
            setResourceModal({ open: true, name: unit.filename, content: unit.content, loading: false });
        } catch {
            setResourceModal({ open: true, name: resourceName, content: _('Failed to load unit file.'), loading: false });
        }
    };

    const getPortUrl = (hostIp: string, hostPort: number, portKey: string) => {
        const host = (!hostIp || hostIp === '0.0.0.0' || hostIp === '::')
            ? window.location.hostname
            : hostIp;
        const scheme = httpsPorts.has(portKey) ? 'https' : 'http';
        return `${scheme}://${host}:${hostPort}`;
    };

    const togglePortScheme = (portKey: string) => {
        setHttpsPorts(prev => {
            const next = new Set(prev);
            if (next.has(portKey)) {
                next.delete(portKey);
            } else {
                next.add(portKey);
            }
            return next;
        });
    };

    if (loading) {
        return (
            <Page>
                <PageSection>
                    <div style={{ textAlign: 'center', padding: '2rem' }}>
                        <Spinner size='xl' />
                        <Title headingLevel='h2' size='lg' style={{ marginTop: '1rem' }}>{_('Loading project details...')}</Title>
                  </div>
              </PageSection>
          </Page>
        );
    }

    if (error) {
        return (
            <Page>
                <PageSection>
                    <Alert variant='danger' title={_('Error loading project details')}>
                        {error}
                  </Alert>
                    <Button variant='primary' onClick={onBack} style={{ marginTop: '1rem' }}>
                        {_('Back to Projects')}
                  </Button>
              </PageSection>
          </Page>
        );
    }

    return (
        <Page>
            <PageSection variant='light'>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                        <Button variant='plain' onClick={onBack} aria-label={_('Back')}>
                            <ArrowLeftIcon />
                      </Button>
                        <Title headingLevel='h1' size='2xl'>
                            {project.name}
                      </Title>
                        <Label color={getStatusColor(project.status)}>{project.status}</Label>
                  </div>
                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                        <Button variant='plain' onClick={loadData} isDisabled={loading} aria-label={_('Refresh')}>
                            <SyncIcon />
                      </Button>
                        <Button
                            variant='primary'
                            onClick={() => openConfirmModal('update', _('Update Project'))}
                            isDisabled={actionLoading}
                      >
                            {_('Update')}
                      </Button>
                        <Button
                            variant='warning'
                            onClick={() => openConfirmModal('stop', _('Stop Project'))}
                            isDisabled={actionLoading}
                      >
                            {_('Stop')}
                      </Button>
                        <Button
                            variant='danger'
                            onClick={() => openConfirmModal('remove', _('Remove Project'))}
                            isDisabled={actionLoading}
                      >
                            {_('Remove')}
                      </Button>
                  </div>
              </div>
          </PageSection>

            <PageSection>
                {actionError && (
                    <Alert
                        variant='danger'
                        title={_('Action failed')}
                        actionClose={<Button variant='plain' onClick={() => setActionError(null)}>×</Button>}
                        style={{ marginBottom: '1rem' }}
                  >
                        {actionError}
                  </Alert>
                )}

                <Card>
                    <CardBody>
                        <DescriptionList isHorizontal>
                            <DescriptionListGroup>
                                <DescriptionListTerm>{_('Source Path')}</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <code>{project.source_path}</code>
                              </DescriptionListDescription>
                          </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>{_('Services')}</DescriptionListTerm>
                                <DescriptionListDescription>{project.services}</DescriptionListDescription>
                          </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>{_('Files')}</DescriptionListTerm>
                                <DescriptionListDescription>{project.files}</DescriptionListDescription>
                          </DescriptionListGroup>
                      </DescriptionList>
                  </CardBody>
              </Card>

                <Tabs activeKey={activeTab} onSelect={(_, key) => setActiveTab(key as number)} style={{ marginTop: '1rem' }}>
                    <Tab eventKey={0} title={<TabTitleText>{_('Services')}</TabTitleText>}>
                        <div style={{ marginTop: '1rem' }}>
                            {viewData && viewData.services && viewData.services.length > 0
                                ? (
                                    <Table aria-label={_('Services table')} variant='compact'>
                                        <Thead>
                                            <Tr>
                                                <Th>{_('Name')}</Th>
                                                <Th>{_('Status')}</Th>
                                                <Th>{_('Image')}</Th>
                                                <Th>{_('Networks')}</Th>
                                                <Th>{_('Volumes')}</Th>
                                          </Tr>
                                      </Thead>
                                        <Tbody>
                                            {viewData.services.map((service) => (
                                                <Tr key={service.name}>
                                                    <Td dataLabel={_('Name')}>{service.name}</Td>
                                                    <Td dataLabel={_('Status')}>
                                                        <Label color={getStatusColor(service.status)}>
                                                            {service.status}
                                                      </Label>
                                                  </Td>
                                                    <Td dataLabel={_('Image')}>
                                                        <code>{service.image}</code>
                                                  </Td>
                                                    <Td dataLabel={_('Networks')}>
                                                        {service.networks && service.networks.length > 0
                                                            ? service.networks.join(', ')
                                                            : '—'}
                                                  </Td>
                                                    <Td dataLabel={_('Volumes')}>
                                                        {service.volumes && service.volumes.length > 0
                                                            ? service.volumes.join(', ')
                                                            : '—'}
                                                  </Td>
                                              </Tr>
                                            ))}
                                      </Tbody>
                                  </Table>
                                )
                                : (
                                    <EmptyState>
                                        <EmptyStateBody>{_('No services found')}</EmptyStateBody>
                                  </EmptyState>
                                )}
                      </div>
                  </Tab>
                    <Tab eventKey={1} title={<TabTitleText>{_('Containers')}</TabTitleText>}>
                        <div style={{ marginTop: '1rem' }}>
                            {containers && containers.length > 0
                                ? (
                                    <Table aria-label={_('Containers table')} variant='compact'>
                                        <Thead>
                                            <Tr>
                                                <Th>{_('Name')}</Th>
                                                <Th>{_('Service')}</Th>
                                                <Th>{_('State')}</Th>
                                                <Th>{_('Image')}</Th>
                                                <Th>{_('Ports')}</Th>
                                          </Tr>
                                      </Thead>
                                        <Tbody>
                                            {containers.map((container) => (
                                                <Tr key={container.name}>
                                                    <Td dataLabel={_('Name')}>{container.name}</Td>
                                                    <Td dataLabel={_('Service')}>{container.service}</Td>
                                                    <Td dataLabel={_('State')}>
                                                        <Label color={getStatusColor(container.state)}>
                                                            {container.state}
                                                      </Label>
                                                  </Td>
                                                    <Td dataLabel={_('Image')}>
                                                        <code>{container.image}</code>
                                                  </Td>
                                                    <Td dataLabel={_('Ports')}>
                                                        {container.ports && container.ports.length > 0
                                                            ? container.ports.map((p, i) => {
                                                                const portKey = `${container.name}-${p.host_port}-${p.container_port}-${i}`;
                                                                const isHttps = httpsPorts.has(portKey);
                                                                return (
                                                                    <React.Fragment key={portKey}>
                                                                        {i > 0 && ', '}
                                                                        <a
                                                                            href={getPortUrl(p.host_ip, p.host_port, portKey)}
                                                                            target='_blank'
                                                                            rel='noopener noreferrer'
                                                                            style={{ textDecoration: 'none' }}
                                                                      >
                                                                            {p.host_port}:{p.container_port}
                                                                            <ExternalLinkAltIcon style={{ marginLeft: '0.25rem', fontSize: '0.75rem' }} />
                                                                      </a>
                                                                        <Tooltip content={isHttps ? _('Switch to http') : _('Switch to https')}>
                                                                            <Button
                                                                                variant='plain'
                                                                                isInline
                                                                                onClick={() => togglePortScheme(portKey)}
                                                                                aria-label={isHttps ? _('Switch to http') : _('Switch to https')}
                                                                                style={{ padding: '0 0.25rem', marginLeft: '0.125rem' }}
                                                                          >
                                                                                {isHttps
                                                                                    ? <LockIcon style={{ fontSize: '0.75rem' }} />
                                                                                    : <UnlockIcon style={{ fontSize: '0.75rem' }} />}
                                                                          </Button>
                                                                      </Tooltip>
                                                                  </React.Fragment>
                                                                );
                                                            })
                                                            : '—'}
                                                  </Td>
                                              </Tr>
                                            ))}
                                      </Tbody>
                                  </Table>
                                )
                                : (
                                    <EmptyState>
                                        <EmptyStateBody>{_('No containers found')}</EmptyStateBody>
                                  </EmptyState>
                                )}
                      </div>
                  </Tab>
                    <Tab eventKey={2} title={<TabTitleText>{_('Resources')}</TabTitleText>}>
                        <div style={{ marginTop: '1rem' }}>
                            {viewData && viewData.resources && viewData.resources.length > 0
                                ? (
                                    <Table aria-label={_('Resources table')} variant='compact'>
                                        <Thead>
                                            <Tr>
                                                <Th>{_('Name')}</Th>
                                                <Th>{_('Type')}</Th>
                                                <Th>{_('Info')}</Th>
                                          </Tr>
                                      </Thead>
                                        <Tbody>
                                            {viewData.resources.map((resource) => (
                                                <Tr key={resource.name}>
                                                    <Td dataLabel={_('Name')}>
                                                        <Button
                                                            variant='link'
                                                            isInline
                                                            onClick={() => handleResourceClick(resource.name)}
                                                            style={{ padding: 0 }}
                                                      >
                                                            {resource.name}
                                                      </Button>
                                                  </Td>
                                                    <Td dataLabel={_('Type')}>
                                                        <Label>{resource.type}</Label>
                                                  </Td>
                                                    <Td dataLabel={_('Info')}>{resource.info || '—'}</Td>
                                              </Tr>
                                            ))}
                                      </Tbody>
                                  </Table>
                                )
                                : (
                                    <EmptyState>
                                        <EmptyStateBody>{_('No resources found')}</EmptyStateBody>
                                  </EmptyState>
                                )}
                      </div>
                  </Tab>
              </Tabs>
          </PageSection>

            <Modal
                variant={ModalVariant.large}
                isOpen={confirmModal.open}
                onClose={closeModal}
          >
                <ModalHeader title={confirmModal.title} />
                <ModalBody>
                    <div style={{ fontSize: '1.1rem', padding: '0.5rem 0' }}>
                        {_('Are you sure you want to ${action} project', { action: confirmModal.action })} <strong>{project.name}</strong>?
                  </div>
                    {confirmModal.action === 'remove' && (
                        <div style={{ marginTop: '0.5rem' }}>
                            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
                                <input
                                    type='checkbox'
                                    checked={deleteVolumes}
                                    onChange={(e) => setDeleteVolumes(e.target.checked)}
                                    style={{ marginRight: '0.5rem' }}
                              />
                                {_('Delete volumes (-d flag)')}
                          </label>
                      </div>
                    )}
                    {confirmModal.action === 'update' && (
                        <div style={{ marginTop: '1rem' }}>
                            {dryRunLoading && (
                                <div style={{ textAlign: 'center', padding: '1rem' }}>
                                    <Spinner size='md' />
                                    <div style={{ marginTop: '0.5rem' }}>{_('Computing changes...')}</div>
                              </div>
                            )}
                            {dryRun && !dryRunLoading && (
                                <div>
                                    {dryRun.images && dryRun.images.length > 0 && (
                                        <div style={{ marginBottom: '0.75rem' }}>
                                            <strong>{_('Images')}:</strong>
                                            <ul style={{ margin: '0.25rem 0', paddingLeft: '1.5rem', fontSize: '0.875rem' }}>
                                                {dryRun.images.map(img => (
                                                    <li key={img.name}>
                                                        <code>{img.name}</code> — {img.ref} ({img.action})
                                                  </li>
                                                ))}
                                          </ul>
                                      </div>
                                    )}
                                    {dryRun.builds && dryRun.builds.length > 0 && (
                                        <div style={{ marginBottom: '0.75rem' }}>
                                            <strong>{_('Builds')}:</strong>
                                            <ul style={{ margin: '0.25rem 0', paddingLeft: '1.5rem', fontSize: '0.875rem' }}>
                                                {dryRun.builds.map(b => (
                                                    <li key={b.name}>
                                                        <code>{b.name}</code> — {b.image_tag}
                                                  </li>
                                                ))}
                                          </ul>
                                      </div>
                                    )}
                                    {!dryRun.has_changes && (
                                        <Alert variant='info' title={_('No changes')} isInline isPlain>
                                            {_('Quadlet files are up to date.')}
                                      </Alert>
                                    )}
                                    {dryRun.files && dryRun.files.length > 0 && (
                                        <div>
                                            <strong>{_('File changes (${count})', { count: dryRun.files.length })}:</strong>
                                            <div style={{ marginTop: '0.5rem', maxHeight: '300px', overflow: 'auto' }}>
                                                {dryRun.files.map(file => (
                                                    <ExpandableSection
                                                        key={file.name}
                                                        toggleText={
                                                            <span>
                                                                <Label color={getFileStatusColor(file.status)} isCompact>
                                                                    {file.status}
                                                              </Label>
                                                                {' '}<code>{file.name}</code>
                                                          </span>
                                                        }
                                                        isExpanded={expandedFiles.has(file.name)}
                                                        onToggle={() => toggleFile(file.name)}
                                                        style={{ marginBottom: '0.25rem' }}
                                                  >
                                                        <pre style={{
                                                            background: 'var(--pf-v5-global--BackgroundColor--200)',
                                                            padding: '0.5rem',
                                                            borderRadius: '4px',
                                                            overflow: 'auto',
                                                            maxHeight: '200px',
                                                            fontSize: '0.75rem',
                                                            lineHeight: '1.4',
                                                            whiteSpace: 'pre-wrap',
                                                            wordBreak: 'break-all'
                                                        }}
                                                      >
                                                            {file.status === 'created' && file.new_content
                                                                ? file.new_content
                                                                : file.diff}
                                                      </pre>
                                                  </ExpandableSection>
                                                ))}
                                          </div>
                                      </div>
                                    )}
                              </div>
                            )}
                      </div>
                    )}
              </ModalBody>
                <ModalFooter>
                    <Button
                        key='cancel'
                        variant='link'
                        onClick={closeModal}
                        isDisabled={actionLoading}
                  >
                        {_('Cancel')}
                  </Button>
                    <Button
                        key='confirm'
                        variant={confirmModal.action === 'remove' ? 'danger' : confirmModal.action === 'stop' ? 'warning' : 'primary'}
                        onClick={() => handleAction(confirmModal.action as any)}
                        isLoading={actionLoading}
                  >
                        {confirmModal.action === 'remove' ? _('Remove') : confirmModal.action === 'stop' ? _('Stop') : _('Update')}
                  </Button>
              </ModalFooter>
          </Modal>

            <Modal
                variant={ModalVariant.large}
                isOpen={resourceModal.open}
                onClose={() => setResourceModal({ open: false, name: '', content: '', loading: false })}
          >
                <ModalHeader title={resourceModal.name} />
                <ModalBody>
                    {resourceModal.loading
                        ? (
                            <div style={{ textAlign: 'center', padding: '2rem' }}>
                                <Spinner size='lg' />
                          </div>
                        )
                        : (
                            <pre style={{
                                background: 'var(--pf-v5-global--BackgroundColor--200)',
                                padding: '1rem',
                                borderRadius: '4px',
                                overflow: 'auto',
                                maxHeight: '500px',
                                fontSize: '0.8125rem',
                                lineHeight: '1.5',
                                whiteSpace: 'pre-wrap',
                                wordBreak: 'break-all'
                            }}
                          >
                                {resourceModal.content}
                          </pre>
                        )}
              </ModalBody>
                <ModalFooter>
                    <Button
                        variant='primary'
                        onClick={() => setResourceModal({ open: false, name: '', content: '', loading: false })}
                  >
                        {_('Close')}
                  </Button>
              </ModalFooter>
          </Modal>
      </Page>
    );
};

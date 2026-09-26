import React, { useState, useEffect } from 'react';
import {
    Modal,
    ModalVariant,
    ModalHeader,
    ModalBody,
    ModalFooter,
    Button,
    Breadcrumb,
    BreadcrumbItem,
    Spinner,
    Alert
} from '@patternfly/react-core';
import { FolderIcon, FolderOpenIcon } from '@patternfly/react-icons';
import * as client from './client';
import { _ } from './i18n';

interface DirectoryPickerProps {
    isOpen: boolean;
    onClose: () => void;
    onSelect: (path: string) => void;
    initialPath?: string;
}

export const DirectoryPicker: React.FC<DirectoryPickerProps> = ({
    isOpen,
    onClose,
    onSelect,
    initialPath = '/home'
}) => {
    const [currentPath, setCurrentPath] = useState(initialPath);
    const [directories, setDirectories] = useState<string[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen) {
            loadDirectory(currentPath);
        }
    }, [isOpen]);

    const loadDirectory = async (path: string) => {
        setLoading(true);
        setError(null);
        try {
            const dirs = await client.listDirectory(path);
            setDirectories(dirs);
            setCurrentPath(path);
        } catch (err) {
            setError(err instanceof Error ? err.message : _('Failed to load directory'));
            setDirectories([]);
        } finally {
            setLoading(false);
        }
    };

    const handleDirectoryClick = (dir: string) => {
        const newPath = currentPath === '/' ? `/${dir}` : `${currentPath}/${dir}`;
        loadDirectory(newPath);
    };

    const handleNavigateUp = () => {
        if (currentPath === '/') return;
        const parts = currentPath.split('/').filter(Boolean);
        parts.pop();
        const parentPath = parts.length > 0 ? '/' + parts.join('/') : '/';
        loadDirectory(parentPath);
    };

    const handleBreadcrumbClick = (index: number) => {
        if (index === 0) {
            loadDirectory('/');
        } else {
            const parts = currentPath.split('/').filter(Boolean);
            const newPath = '/' + parts.slice(0, index).join('/');
            loadDirectory(newPath);
        }
    };

    const handleSelect = () => {
        onSelect(currentPath);
        onClose();
    };

    const pathParts = currentPath.split('/').filter(Boolean);

    return (
        <Modal
            variant={ModalVariant.large}
            isOpen={isOpen}
            onClose={onClose}
      >
            <ModalHeader title={_('Select Project Directory')} />
            <ModalBody>
                <div style={{ marginBottom: '1rem', marginTop: '0.5rem' }}>
                    <Breadcrumb>
                        <BreadcrumbItem
                            to='#'
                            onClick={(e) => {
                                e.preventDefault();
                                handleBreadcrumbClick(0);
                            }}
                      >
                          /
                      </BreadcrumbItem>
                        {pathParts.map((part, index) => (
                            <BreadcrumbItem
                                key={index}
                                to='#'
                                onClick={(e) => {
                                    e.preventDefault();
                                    handleBreadcrumbClick(index + 1);
                                }}
                          >
                                {part}
                          </BreadcrumbItem>
                        ))}
                  </Breadcrumb>
              </div>

                {currentPath !== '/' && (
                    <div style={{ marginBottom: '1rem' }}>
                        <Button
                            variant='secondary'
                            onClick={handleNavigateUp}
                            style={{ marginTop: '0.5rem' }}
                      >
                            <FolderOpenIcon style={{ marginRight: '0.5rem' }} /> {_('Go Up')}
                      </Button>
                  </div>
                )}

                {error && (
                    <Alert variant='danger' title={_('Error')} style={{ marginBottom: '1rem' }}>
                        {error}
                  </Alert>
                )}

                {loading
                    ? (
                        <div style={{ textAlign: 'center', padding: '2rem' }}>
                            <Spinner size='lg' />
                      </div>
                    )
                    : (
                        <div style={{ maxHeight: '400px', overflow: 'auto', border: '1px solid #ddd', padding: '0.5rem', borderRadius: '4px' }}>
                            {directories.length === 0
                                ? (
                                    <div style={{ padding: '1rem', color: '#666' }}>{_('No directories found')}</div>
                                )
                                : (
                                    directories.map((dir) => (
                                        <div
                                            key={dir}
                                            style={{
                                                cursor: 'pointer',
                                                padding: '0.5rem 0.75rem',
                                                display: 'flex',
                                                alignItems: 'center',
                                                borderRadius: '4px'
                                            }}
                                            onClick={() => handleDirectoryClick(dir)}
                                            onMouseEnter={(e) => {
                                                e.currentTarget.style.backgroundColor = '#f0f0f0';
                                            }}
                                            onMouseLeave={(e) => {
                                                e.currentTarget.style.backgroundColor = 'transparent';
                                            }}
                                      >
                                            <FolderIcon style={{ marginRight: '0.5rem' }} />
                                            {dir}
                                      </div>
                                    ))
                                )}
                      </div>
                    )}

                <div style={{ marginTop: '1rem', fontSize: '0.9rem', color: '#666', padding: '0.5rem 0' }}>
                    {_('Current path:')}: <strong>{currentPath}</strong>
              </div>
          </ModalBody>
            <ModalFooter>
                <Button key='cancel' variant='link' onClick={onClose}>
                    {_('Cancel')}
              </Button>
                <Button key='select' variant='primary' onClick={handleSelect}>
                    {_('Select This Directory')}
              </Button>
          </ModalFooter>
      </Modal>
    );
};

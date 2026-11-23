import type React from 'react';

export type StatusColor = 'green' | 'grey' | 'orange' | 'red' | 'blue';

export const getStatusColor = (status: string): StatusColor => {
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

export const getFileStatusColor = (status: string): 'green' | 'blue' | 'red' | 'orange' => {
    switch (status) {
        case 'created': return 'green';
        case 'changed': return 'blue';
        case 'removed': return 'red';
        default: return 'orange';
    }
};

export const toggleSetItem = (
    set: Set<string>,
    setFn: React.Dispatch<React.SetStateAction<Set<string>>>,
    item: string
) => {
    setFn(prev => {
        const next = new Set(prev);
        if (next.has(item)) {
            next.delete(item);
        } else {
            next.add(item);
        }
        return next;
    });
};

export const COMQAD_PATHS = [
    '/usr/local/bin/comquad',
    '/usr/bin/comquad',
    '/opt/comquad/comquad'
] as const;

export const COMPOSE_FILES = [
    'compose.yaml',
    'compose.yml',
    'docker-compose.yaml',
    'docker-compose.yml',
    'podman-compose.yaml',
    'podman-compose.yml'
] as const;

export const COLORS = {
    muted: '#6a6e73',
    border: '#ddd',
    textSecondary: '#666',
    background: '#f0f0f0'
} as const;

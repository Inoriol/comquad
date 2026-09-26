import React from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { Badge, Button, Label } from '@patternfly/react-core';
import { _ } from './i18n';
import type { Project } from './types';

interface ProjectsListProps {
    projects: Project[];
    onSelect: (project: Project) => void;
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
        default:
            return 'grey';
    }
};

export const ProjectsList: React.FC<ProjectsListProps> = ({ projects, onSelect }) => {
    return (
        <Table aria-label={_('Projects table')} variant='compact'>
            <Thead>
                <Tr>
                    <Th>{_('Name')}</Th>
                    <Th>{_('Status')}</Th>
                    <Th>{_('Services')}</Th>
                    <Th>{_('Source Path')}</Th>
                    <Th>{_('Files')}</Th>
              </Tr>
          </Thead>
            <Tbody>
                {projects.map((project) => (
                    <Tr
                        key={project.name}
                        isClickable
                        onRowClick={() => onSelect(project)}
                        isHoverable
                  >
                        <Td dataLabel={_('Name')}>
                            <strong>{project.name}</strong>
                      </Td>
                        <Td dataLabel={_('Status')}>
                            <Label color={getStatusColor(project.status)}>
                                {project.status}
                          </Label>
                      </Td>
                        <Td dataLabel={_('Services')}>
                            <Badge>{project.services}</Badge>
                      </Td>
                        <Td dataLabel={_('Source Path')}>
                            <code style={{ fontSize: '0.9em' }}>{project.source_path}</code>
                      </Td>
                        <Td dataLabel={_('Files')}>
                            {project.files}
                      </Td>
                  </Tr>
                ))}
          </Tbody>
      </Table>
    );
};

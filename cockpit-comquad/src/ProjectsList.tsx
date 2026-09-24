import React from "react";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { Badge, Button, Label } from "@patternfly/react-core";
import type { Project } from "./types";

interface ProjectsListProps {
    projects: Project[];
    onSelect: (project: Project) => void;
    onRefresh: () => void;
}

const getStatusColor = (status: string): "green" | "grey" | "orange" | "red" | "blue" => {
    switch (status) {
        case "healthy":
        case "running":
            return "green";
        case "up":
            return "blue";
        case "stopped":
            return "grey";
        case "degraded":
            return "orange";
        default:
            return "grey";
    }
};

export const ProjectsList: React.FC<ProjectsListProps> = ({ projects, onSelect }) => {
    return (
        <Table aria-label="Projects table" variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Status</Th>
                    <Th>Services</Th>
                    <Th>Source Path</Th>
                    <Th>Files</Th>
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
                        <Td dataLabel="Name">
                            <strong>{project.name}</strong>
                        </Td>
                        <Td dataLabel="Status">
                            <Label color={getStatusColor(project.status)}>
                                {project.status}
                            </Label>
                        </Td>
                        <Td dataLabel="Services">
                            <Badge>{project.services}</Badge>
                        </Td>
                        <Td dataLabel="Source Path">
                            <code style={{ fontSize: "0.9em" }}>{project.source_path}</code>
                        </Td>
                        <Td dataLabel="Files">
                            {project.files}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
};

import React, { useState, useEffect, useCallback } from "react";
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
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { ArrowLeftIcon, SyncIcon } from "@patternfly/react-icons";
import * as client from "./client";
import type { Project, ViewData, Container } from "./types";

interface ProjectDetailProps {
    project: Project;
    onBack: () => void;
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
        case "failed":
            return "red";
        default:
            return "grey";
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
        action: "",
        title: "",
    });

    const loadData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            console.log("Loading project details for:", project.source_path);
            const [view, ps] = await Promise.all([
                client.viewProject(project.source_path),
                client.projectPs(project.source_path),
            ]);
            console.log("View data:", view);
            console.log("Containers:", ps);
            setViewData(view);
            setContainers(ps);
        } catch (err) {
            console.error("Error loading project details:", err);
            setError(err instanceof Error ? err.message : "Failed to load project details");
        } finally {
            setLoading(false);
        }
    }, [project.source_path]);

    useEffect(() => {
        loadData();
    }, [loadData]);

    const handleAction = async (action: "update" | "stop" | "remove") => {
        setActionLoading(true);
        setActionError(null);
        try {
            switch (action) {
                case "update":
                    await client.updateProject(project.source_path);
                    break;
                case "stop":
                    await client.stopProject(project.source_path);
                    break;
                case "remove":
                    await client.removeProject(project.source_path, deleteVolumes);
                    onBack();
                    return;
            }
            await loadData();
            onRefresh();
        } catch (err) {
            setActionError(err instanceof Error ? err.message : `Failed to ${action} project`);
        } finally {
            setActionLoading(false);
            setConfirmModal({ open: false, action: "", title: "" });
            setDeleteVolumes(false);
        }
    };

    const openConfirmModal = (action: string, title: string) => {
        setConfirmModal({ open: true, action, title });
    };

    if (loading) {
        return (
            <Page>
                <PageSection>
                    <div style={{ textAlign: "center", padding: "2rem" }}>
                        <Spinner size="xl" />
                        <Title headingLevel="h2" size="lg" style={{ marginTop: "1rem" }}>Loading project details...</Title>
                    </div>
                </PageSection>
            </Page>
        );
    }

    if (error) {
        return (
            <Page>
                <PageSection>
                    <Alert variant="danger" title="Error loading project details">
                        {error}
                    </Alert>
                    <Button variant="primary" onClick={onBack} style={{ marginTop: "1rem" }}>
                        Back to Projects
                    </Button>
                </PageSection>
            </Page>
        );
    }

    return (
        <Page>
            <PageSection variant="light">
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
                        <Button variant="plain" onClick={onBack} aria-label="Back">
                            <ArrowLeftIcon />
                        </Button>
                        <Title headingLevel="h1" size="2xl">
                            {project.name}
                        </Title>
                        <Label color={getStatusColor(project.status)}>{project.status}</Label>
                    </div>
                    <div style={{ display: "flex", gap: "0.5rem" }}>
                        <Button variant="plain" onClick={loadData} isDisabled={loading} aria-label="Refresh">
                            <SyncIcon />
                        </Button>
                        <Button
                            variant="primary"
                            onClick={() => openConfirmModal("update", "Update Project")}
                            isDisabled={actionLoading}
                        >
                            Update
                        </Button>
                        <Button
                            variant="warning"
                            onClick={() => openConfirmModal("stop", "Stop Project")}
                            isDisabled={actionLoading}
                        >
                            Stop
                        </Button>
                        <Button
                            variant="danger"
                            onClick={() => openConfirmModal("remove", "Remove Project")}
                            isDisabled={actionLoading}
                        >
                            Remove
                        </Button>
                    </div>
                </div>
            </PageSection>

            <PageSection>
                {actionError && (
                    <Alert
                        variant="danger"
                        title="Action failed"
                        actionClose={<Button variant="plain" onClick={() => setActionError(null)}>×</Button>}
                        style={{ marginBottom: "1rem" }}
                    >
                        {actionError}
                    </Alert>
                )}

                <Card>
                    <CardBody>
                        <DescriptionList isHorizontal>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Source Path</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <code>{project.source_path}</code>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Services</DescriptionListTerm>
                                <DescriptionListDescription>{project.services}</DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Files</DescriptionListTerm>
                                <DescriptionListDescription>{project.files}</DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </CardBody>
                </Card>

                <Tabs activeKey={activeTab} onSelect={(_, key) => setActiveTab(key as number)} style={{ marginTop: "1rem" }}>
                    <Tab eventKey={0} title={<TabTitleText>Services</TabTitleText>}>
                        <div style={{ marginTop: "1rem" }}>
                            {viewData && viewData.services && viewData.services.length > 0 ? (
                                <Table aria-label="Services table" variant="compact">
                                    <Thead>
                                        <Tr>
                                            <Th>Name</Th>
                                            <Th>Status</Th>
                                            <Th>Image</Th>
                                            <Th>Networks</Th>
                                            <Th>Volumes</Th>
                                        </Tr>
                                    </Thead>
                                    <Tbody>
                                        {viewData.services.map((service) => (
                                            <Tr key={service.name}>
                                                <Td dataLabel="Name">{service.name}</Td>
                                                <Td dataLabel="Status">
                                                    <Label color={getStatusColor(service.status)}>
                                                        {service.status}
                                                    </Label>
                                                </Td>
                                                <Td dataLabel="Image">
                                                    <code>{service.image}</code>
                                                </Td>
                                                <Td dataLabel="Networks">
                                                    {service.networks && service.networks.length > 0
                                                        ? service.networks.join(", ")
                                                        : "—"}
                                                </Td>
                                                <Td dataLabel="Volumes">
                                                    {service.volumes && service.volumes.length > 0
                                                        ? service.volumes.join(", ")
                                                        : "—"}
                                                </Td>
                                            </Tr>
                                        ))}
                                    </Tbody>
                                </Table>
                            ) : (
                                <EmptyState>
                                    <EmptyStateBody>No services found</EmptyStateBody>
                                </EmptyState>
                            )}
                        </div>
                    </Tab>
                    <Tab eventKey={1} title={<TabTitleText>Containers</TabTitleText>}>
                        <div style={{ marginTop: "1rem" }}>
                            {containers && containers.length > 0 ? (
                                <Table aria-label="Containers table" variant="compact">
                                    <Thead>
                                        <Tr>
                                            <Th>Name</Th>
                                            <Th>Service</Th>
                                            <Th>State</Th>
                                            <Th>Image</Th>
                                            <Th>Ports</Th>
                                        </Tr>
                                    </Thead>
                                    <Tbody>
                                        {containers.map((container) => (
                                            <Tr key={container.name}>
                                                <Td dataLabel="Name">{container.name}</Td>
                                                <Td dataLabel="Service">{container.service}</Td>
                                                <Td dataLabel="State">
                                                    <Label color={getStatusColor(container.state)}>
                                                        {container.state}
                                                    </Label>
                                                </Td>
                                                <Td dataLabel="Image">
                                                    <code>{container.image}</code>
                                                </Td>
                                                <Td dataLabel="Ports">
                                                    {container.ports && container.ports.length > 0
                                                        ? container.ports.map(p => `${p.host_port}:${p.container_port}`).join(", ")
                                                        : "—"}
                                                </Td>
                                            </Tr>
                                        ))}
                                    </Tbody>
                                </Table>
                            ) : (
                                <EmptyState>
                                    <EmptyStateBody>No containers found</EmptyStateBody>
                                </EmptyState>
                            )}
                        </div>
                    </Tab>
                    <Tab eventKey={2} title={<TabTitleText>Resources</TabTitleText>}>
                        <div style={{ marginTop: "1rem" }}>
                            {viewData && viewData.resources && viewData.resources.length > 0 ? (
                                <Table aria-label="Resources table" variant="compact">
                                    <Thead>
                                        <Tr>
                                            <Th>Name</Th>
                                            <Th>Type</Th>
                                            <Th>Info</Th>
                                        </Tr>
                                    </Thead>
                                    <Tbody>
                                        {viewData.resources.map((resource) => (
                                            <Tr key={resource.name}>
                                                <Td dataLabel="Name">{resource.name}</Td>
                                                <Td dataLabel="Type">
                                                    <Label>{resource.type}</Label>
                                                </Td>
                                                <Td dataLabel="Info">{resource.info || "—"}</Td>
                                            </Tr>
                                        ))}
                                    </Tbody>
                                </Table>
                            ) : (
                                <EmptyState>
                                    <EmptyStateBody>No resources found</EmptyStateBody>
                                </EmptyState>
                            )}
                        </div>
                    </Tab>
                </Tabs>
            </PageSection>

            <Modal
                variant={ModalVariant.medium}
                isOpen={confirmModal.open}
                onClose={() => setConfirmModal({ open: false, action: "", title: "" })}
            >
                <ModalHeader title={confirmModal.title} />
                <ModalBody>
                    <div style={{ fontSize: "1.1rem", padding: "1rem 0" }}>
                        Are you sure you want to {confirmModal.action} project <strong>{project.name}</strong>?
                    </div>
                    {confirmModal.action === "remove" && (
                        <div style={{ marginTop: "1rem" }}>
                            <label style={{ display: "flex", alignItems: "center", cursor: "pointer" }}>
                                <input
                                    type="checkbox"
                                    checked={deleteVolumes}
                                    onChange={(e) => setDeleteVolumes(e.target.checked)}
                                    style={{ marginRight: "0.5rem" }}
                                />
                                Delete volumes (-d flag)
                            </label>
                        </div>
                    )}
                </ModalBody>
                <ModalFooter>
                    <Button
                        key="cancel"
                        variant="link"
                        onClick={() => setConfirmModal({ open: false, action: "", title: "" })}
                        isDisabled={actionLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        key="confirm"
                        variant={confirmModal.action === "remove" ? "danger" : confirmModal.action === "stop" ? "warning" : "primary"}
                        onClick={() => handleAction(confirmModal.action as any)}
                        isLoading={actionLoading}
                    >
                        {confirmModal.action === "remove" ? "Remove" : confirmModal.action === "stop" ? "Stop" : "Update"}
                    </Button>
                </ModalFooter>
            </Modal>
        </Page>
    );
};

import React, { useState } from "react";
import {
    Page,
    PageSection,
    Title,
    Button,
    Card,
    CardBody,
    Form,
    FormGroup,
    TextInput,
    Alert,
    ExpandableSection,
    Label,
    Spinner,
} from "@patternfly/react-core";
import { ArrowLeftIcon, FolderOpenIcon, EyeIcon } from "@patternfly/react-icons";
import * as client from "./client";
import { DirectoryPicker } from "./DirectoryPicker";
import type { DryRunData } from "./types";

interface StackDeployProps {
    onDeployComplete: () => void;
    onCancel: () => void;
}

const getStatusColor = (status: string): "green" | "blue" | "red" | "orange" => {
    switch (status) {
        case "created": return "green";
        case "changed": return "blue";
        case "removed": return "red";
        default: return "orange";
    }
};

export const StackDeploy: React.FC<StackDeployProps> = ({ onDeployComplete, onCancel }) => {
    const [path, setPath] = useState("");
    const [deploying, setDeploying] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [showSuccess, setShowSuccess] = useState(false);
    const [showPicker, setShowPicker] = useState(false);
    const [previewing, setPreviewing] = useState(false);
    const [dryRun, setDryRun] = useState<DryRunData | null>(null);
    const [previewError, setPreviewError] = useState<string | null>(null);
    const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());

    const handleDeploy = async () => {
        if (!path.trim()) {
            setError("Please select a directory");
            return;
        }

        setDeploying(true);
        setError(null);
        try {
            await client.deployProject(path.trim());
            setShowSuccess(true);
            setTimeout(() => {
                onDeployComplete();
            }, 2000);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to deploy stack");
            setDeploying(false);
        }
    };

    const handlePreview = async () => {
        if (!path.trim()) return;
        setPreviewing(true);
        setPreviewError(null);
        setDryRun(null);
        try {
            const data = await client.dryRunProject(path.trim());
            setDryRun(data);
            const allFiles = new Set(data.files?.map(f => f.name) || []);
            setExpandedFiles(allFiles);
        } catch (err) {
            setPreviewError(err instanceof Error ? err.message : "Failed to preview changes");
        } finally {
            setPreviewing(false);
        }
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

    const handleDirectorySelect = (selectedPath: string) => {
        setPath(selectedPath);
        setShowPicker(false);
        setDryRun(null);
        setPreviewError(null);
    };

    if (showSuccess) {
        return (
            <Page>
                <PageSection>
                    <Alert variant="success" title="Stack deployed successfully">
                        The stack has been deployed and is now running.
                    </Alert>
                </PageSection>
            </Page>
        );
    }

    return (
        <Page>
            <PageSection variant="light">
                <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
                    <Button variant="plain" onClick={onCancel} aria-label="Back">
                        <ArrowLeftIcon />
                    </Button>
                    <Title headingLevel="h1" size="2xl">
                        Deploy Stack
                    </Title>
                </div>
            </PageSection>

            <PageSection>
                <Card style={{ maxWidth: "800px" }}>
                    <CardBody>
                        <Form>
                            <FormGroup
                                label="Project Directory"
                                fieldId="path-input"
                                helperText="Select the directory containing your compose.yaml file"
                            >
                                <div style={{ display: "flex", gap: "0.5rem" }}>
                                    <TextInput
                                        id="path-input"
                                        value={path}
                                        onChange={setPath}
                                        placeholder="/home/user/myproject"
                                        isDisabled={deploying}
                                        style={{ flex: 1 }}
                                    />
                                    <Button
                                        variant="secondary"
                                        onClick={() => setShowPicker(true)}
                                        isDisabled={deploying}
                                    >
                                        <FolderOpenIcon /> Browse
                                    </Button>
                                </div>
                            </FormGroup>

                            {error && (
                                <Alert
                                    variant="danger"
                                    title="Deployment failed"
                                    style={{ marginBottom: "1rem" }}
                                >
                                    {error}
                                </Alert>
                            )}

                            {previewError && (
                                <Alert
                                    variant="danger"
                                    title="Preview failed"
                                    actionClose={<Button variant="plain" onClick={() => setPreviewError(null)}>×</Button>}
                                    style={{ marginBottom: "1rem" }}
                                >
                                    {previewError}
                                </Alert>
                            )}

                            <div style={{ marginTop: "1.5rem", display: "flex", gap: "0.5rem", flexWrap: "wrap" }}>
                                <Button
                                    variant="secondary"
                                    onClick={handlePreview}
                                    isDisabled={deploying || previewing || !path.trim()}
                                    isLoading={previewing}
                                    icon={<EyeIcon />}
                                >
                                    {previewing ? "Previewing..." : "Preview Changes"}
                                </Button>
                                <Button
                                    variant="primary"
                                    onClick={handleDeploy}
                                    isDisabled={deploying || !path.trim()}
                                    isLoading={deploying}
                                >
                                    {deploying ? "Deploying..." : "Deploy Stack"}
                                </Button>
                                <Button
                                    variant="link"
                                    onClick={onCancel}
                                    isDisabled={deploying}
                                >
                                    Cancel
                                </Button>
                            </div>
                        </Form>
                    </CardBody>
                </Card>

                {previewing && (
                    <Card style={{ marginTop: "1rem", maxWidth: "800px" }}>
                        <CardBody>
                            <div style={{ textAlign: "center", padding: "2rem" }}>
                                <Spinner size="lg" />
                                <div style={{ marginTop: "0.5rem" }}>Computing changes...</div>
                            </div>
                        </CardBody>
                    </Card>
                )}

                {dryRun && !previewing && (
                    <Card style={{ marginTop: "1rem", maxWidth: "800px" }}>
                        <CardBody>
                            <Title headingLevel="h3" size="lg" style={{ marginBottom: "1rem" }}>
                                Preview: {dryRun.project}
                            </Title>
                            <div style={{ marginBottom: "0.5rem", color: "var(--pf-v5-global--Color--200)", fontSize: "0.875rem" }}>
                                Target: <code>{dryRun.target_dir}</code> | Pull: {dryRun.pull_strategy}
                            </div>

                            {dryRun.images && dryRun.images.length > 0 && (
                                <div style={{ marginBottom: "1rem" }}>
                                    <strong>Images:</strong>
                                    <ul style={{ margin: "0.25rem 0", paddingLeft: "1.5rem" }}>
                                        {dryRun.images.map(img => (
                                            <li key={img.name}>
                                                <code>{img.name}</code> — {img.ref} ({img.action})
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {dryRun.builds && dryRun.builds.length > 0 && (
                                <div style={{ marginBottom: "1rem" }}>
                                    <strong>Builds:</strong>
                                    <ul style={{ margin: "0.25rem 0", paddingLeft: "1.5rem" }}>
                                        {dryRun.builds.map(b => (
                                            <li key={b.name}>
                                                <code>{b.name}</code> — {b.image_tag}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {!dryRun.has_changes && (
                                <Alert variant="info" title="No changes" isInline isPlain>
                                    Quadlet files are up to date. Nothing to deploy.
                                </Alert>
                            )}

                            {dryRun.files && dryRun.files.length > 0 && (
                                <div>
                                    <strong>File changes ({dryRun.files.length}):</strong>
                                    <div style={{ marginTop: "0.5rem" }}>
                                        {dryRun.files.map(file => (
                                            <ExpandableSection
                                                key={file.name}
                                                toggleText={
                                                    <span>
                                                        <Label color={getStatusColor(file.status)} isCompact>
                                                            {file.status}
                                                        </Label>
                                                        {" "}<code>{file.name}</code>
                                                    </span>
                                                }
                                                isExpanded={expandedFiles.has(file.name)}
                                                onToggle={() => toggleFile(file.name)}
                                                style={{ marginBottom: "0.25rem" }}
                                            >
                                                <pre style={{
                                                    background: "var(--pf-v5-global--BackgroundColor--200)",
                                                    padding: "0.75rem",
                                                    borderRadius: "4px",
                                                    overflow: "auto",
                                                    maxHeight: "400px",
                                                    fontSize: "0.8125rem",
                                                    lineHeight: "1.4",
                                                    whiteSpace: "pre-wrap",
                                                    wordBreak: "break-all",
                                                }}>
                                                    {file.status === "created" && file.new_content
                                                        ? file.new_content
                                                        : file.diff}
                                                </pre>
                                            </ExpandableSection>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </CardBody>
                    </Card>
                )}
            </PageSection>

            <DirectoryPicker
                isOpen={showPicker}
                onClose={() => setShowPicker(false)}
                onSelect={handleDirectorySelect}
            />
        </Page>
    );
};

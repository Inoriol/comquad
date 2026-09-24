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
} from "@patternfly/react-core";
import { ArrowLeftIcon, FolderOpenIcon } from "@patternfly/react-icons";
import * as client from "./client";
import { DirectoryPicker } from "./DirectoryPicker";

interface StackDeployProps {
    onDeployComplete: () => void;
    onCancel: () => void;
}

export const StackDeploy: React.FC<StackDeployProps> = ({ onDeployComplete, onCancel }) => {
    const [path, setPath] = useState("");
    const [deploying, setDeploying] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [showSuccess, setShowSuccess] = useState(false);
    const [showPicker, setShowPicker] = useState(false);

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

    const handleDirectorySelect = (selectedPath: string) => {
        setPath(selectedPath);
        setShowPicker(false);
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
                <Card style={{ maxWidth: "600px" }}>
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

                            <div style={{ marginTop: "1.5rem", display: "flex", gap: "0.5rem" }}>
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
            </PageSection>

            <DirectoryPicker
                isOpen={showPicker}
                onClose={() => setShowPicker(false)}
                onSelect={handleDirectorySelect}
            />
        </Page>
    );
};

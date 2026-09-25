// TypeScript types matching comquad JSON API schemas

export interface Envelope<T> {
    version: string;
    data: T;
}

export interface ErrorEnvelope {
    version: string;
    error: {
        code: string;
        message: string;
    };
}

// Project list response
export interface ProjectListData {
    projects: Project[];
}

export interface Project {
    name: string;
    source_path: string;
    files: number;
    status: 'healthy' | 'up' | 'stopped' | 'degraded' | 'unknown';
    services: string; // "running/total" format
    resources?: Resources;
}

export interface Resources {
    containers: string[];
    networks: string[];
    volumes: string[];
    images: string[];
    builds: string[];
}

// Container list response
export interface PSData {
    containers: Container[];
}

export interface Container {
    name: string;
    image: string;
    command: string;
    service: string;
    state: string;
    status: string;
    ports: Port[];
    exposed_ports?: string[];
    networks?: string[];
    mounts?: string[];
    created_at: string;
    exited_at?: string;
    exit_code?: number;
    dbus_active?: string;
    dbus_sub?: string;
}

export interface Port {
    protocol: string;
    container_port: number;
    host_ip: string;
    host_port: number;
}

// View response
export interface ViewData {
    project: string;
    source_path: string;
    status: string;
    services: Service[];
    resources: Resource[];
}

export interface Service {
    name: string;
    status: string;
    image: string;
    networks: string[];
    volumes: string[];
}

export interface Resource {
    name: string;
    type: string; // "image", "network", "volume", "build"
    info?: string;
}

export interface UnitFile {
    filename: string;
    content: string;
}

// Lifecycle response (start/stop/restart)
export interface LifecycleData {
    action: string; // "start", "stop", "restart"
    project: string;
    units: string[];
    success: boolean;
}

// Down response
export interface DownData {
    project: string;
    removed_files: string[];
    removed_networks: string[];
    removed_volumes?: string[];
    success: boolean;
}

// Up response
export interface UpData {
    project: string;
    source_path: string;
    files_written: string[];
    files_removed?: string[];
    units_started: string[];
    success: boolean;
}

// Logs response
export interface LogsData {
    project: string;
    entries: LogEntry[];
}

export interface LogEntry {
    timestamp: string;
    unit: string;
    priority: number;
    message: string;
}

export interface DryRunData {
    project: string;
    target_dir: string;
    pull_strategy: string;
    images?: DryRunImage[];
    builds?: DryRunBuild[];
    files?: DryRunFile[];
    has_changes: boolean;
}

export interface DryRunImage {
    name: string;
    ref: string;
    action: string;
}

export interface DryRunBuild {
    name: string;
    image_tag: string;
}

export interface DryRunFile {
    name: string;
    path: string;
    status: string;
    diff: string;
    new_content?: string;
}

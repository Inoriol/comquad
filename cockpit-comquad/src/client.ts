import type {
    Project,
    Container,
    ViewData,
    UnitFile,
    LifecycleData,
    DownData,
    UpData,
    LogsData,
} from "./types";

// Use the global cockpit object loaded by ../base1/cockpit.js
declare const cockpit: {
    spawn: (args: string[], options?: {
        directory?: string;
        err?: string;
        environ?: string[];
        superuser?: string | null;
    }) => Promise<string>;
};

interface ComquadResponse<T> {
    version: string;
    data: T;
}

interface ComquadError {
    version: string;
    error: {
        code: string;
        message: string;
    };
}

function parseResponse<T>(output: string): T {
    const parsed = JSON.parse(output);
    if (parsed.error) {
        throw new Error(parsed.error.message);
    }
    return parsed.data;
}

let comquadPath: string | null = null;

async function findComquad(): Promise<string> {
    if (comquadPath) return comquadPath;
    
    // Try common locations
    const paths = [
        "/usr/local/bin/comquad",
        "/usr/bin/comquad",
        "/opt/comquad/comquad",
    ];
    
    // Also check user's home directory
    try {
        const home = await cockpit.spawn(["sh", "-c", "echo $HOME"], { err: "message" });
        paths.push(`${home.trim()}/go/bin/comquad`);
        paths.push(`${home.trim()}/.local/bin/comquad`);
    } catch (e) {
        // Ignore errors finding home
    }
    
    for (const path of paths) {
        try {
            await cockpit.spawn(["test", "-x", path], { err: "message" });
            comquadPath = path;
            return path;
        } catch (e) {
            // Not found at this path, continue
        }
    }
    
    // Fallback to just "comquad" and hope it's in PATH
    comquadPath = "comquad";
    return comquadPath;
}

async function runComquad(args: string[], options?: { directory?: string }): Promise<string> {
    try {
        const comquad = await findComquad();
        const fullArgs = [comquad, ...args.slice(1)];
        const output = await cockpit.spawn(fullArgs, {
            ...options,
            err: "message",
        });
        return output;
    } catch (error: any) {
        const message = error?.message || error?.toString() || "Unknown error";
        const problem = error?.problem || "";
        throw new Error(`${problem ? problem + ": " : ""}${message}`);
    }
}

export async function listProjects(): Promise<Project[]> {
    const output = await runComquad(["comquad", "list", "--json"]);
    const data = parseResponse<{ projects: Project[] }>(output);
    return data.projects;
}

export async function projectPs(projectPath: string): Promise<Container[]> {
    const output = await runComquad(["comquad", "ps", "--json"], { directory: projectPath });
    const data = parseResponse<{ containers: Container[] }>(output);
    return data.containers;
}

export async function viewProject(projectPath: string): Promise<ViewData> {
    const output = await runComquad(["comquad", "view", "--json"], { directory: projectPath });
    return parseResponse<ViewData>(output);
}

export async function viewUnit(projectPath: string, service: string): Promise<UnitFile> {
    const output = await runComquad(["comquad", "view", "--json", service], { directory: projectPath });
    return parseResponse<UnitFile>(output);
}

export async function startProject(projectPath: string): Promise<LifecycleData> {
    const output = await runComquad(["comquad", "start", "--json"], { directory: projectPath });
    return parseResponse<LifecycleData>(output);
}

export async function updateProject(projectPath: string): Promise<UpData> {
    const output = await runComquad(["comquad", "up", "--json", "--no-diff"], { directory: projectPath });
    return parseResponse<UpData>(output);
}

export async function stopProject(projectPath: string): Promise<LifecycleData> {
    const output = await runComquad(["comquad", "stop", "--json"], { directory: projectPath });
    return parseResponse<LifecycleData>(output);
}

export async function restartProject(projectPath: string): Promise<LifecycleData> {
    const output = await runComquad(["comquad", "restart", "--json"], { directory: projectPath });
    return parseResponse<LifecycleData>(output);
}

export async function removeProject(projectPath: string, deleteVolumes: boolean = false): Promise<DownData> {
    const args = ["comquad", "down", "--json", "-y"];
    if (deleteVolumes) {
        args.push("-d");
    }
    const output = await runComquad(args, { directory: projectPath });
    return parseResponse<DownData>(output);
}

export async function deployProject(projectPath: string): Promise<UpData> {
    const output = await runComquad(["comquad", "up", "--json", "--no-diff"], { directory: projectPath });
    return parseResponse<UpData>(output);
}

export async function projectLogs(projectPath: string, service?: string): Promise<LogsData> {
    const args = ["comquad", "logs", "--json"];
    if (service) {
        args.push(service);
    }
    const output = await runComquad(args, { directory: projectPath });
    return parseResponse<LogsData>(output);
}

export async function listDirectory(dirPath: string): Promise<string[]> {
    const output = await cockpit.spawn(["ls", "-1", dirPath], { err: "message" });
    return output.trim().split("\n").filter(Boolean);
}

export async function checkComposeFile(dirPath: string): Promise<boolean> {
    const composeFiles = ["compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml", "podman-compose.yaml", "podman-compose.yml"];
    try {
        const files = await listDirectory(dirPath);
        return composeFiles.some(f => files.includes(f));
    } catch {
        return false;
    }
}

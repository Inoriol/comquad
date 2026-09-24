// Type declarations for Cockpit API
declare const cockpit: {
    spawn: (args: string[], options?: {
        directory?: string;
        err?: string;
        environ?: string[];
        superuser?: string | null;
    }) => Promise<string>;
    file: (path: string, options?: { superuser?: string | null }) => {
        read: () => Promise<string>;
        watch: (callback: (content: string | null, tag: string) => void) => void;
    };
    user: () => Promise<{ id: number; name: string; gid: number; groups: string[]; home: string; shell: string }>;
    location: {
        go: (path: string[], options?: Record<string, string>) => void;
        path: string[];
        options: Record<string, string>;
    };
    dbus: (name: string, options?: { bus?: string; superuser?: string | null }) => {
        call: (path: string, iface: string, method: string, args: unknown[]) => Promise<unknown>;
        subscribe: (match: Record<string, string>, callback: (...args: unknown[]) => void) => { remove: () => void };
        close: () => void;
    };
    gettext: (str: string) => string;
};

export default cockpit;

import '@testing-library/jest-dom';

// Mock cockpit global
const mockCockpit = {
    spawn: vi.fn(),
    gettext: vi.fn((str: string) => str),
    ngettext: vi.fn((singular: string, plural: string, n: number) => n === 1 ? singular : plural),
    format: vi.fn((fmt: string, ...args: any[]) => {
        if (args.length === 0) return fmt;
        let result = fmt;
        args.forEach((arg) => {
            if (typeof arg === 'object' && arg !== null) {
                Object.entries(arg).forEach(([key, value]) => {
                    result = result.replace(new RegExp(`\\$\\{${key}\\}`, 'g'), String(value));
                });
            }
        });
        return result;
    }),
    locale: vi.fn(),
    file: vi.fn(),
    user: vi.fn(),
    location: {
        go: vi.fn(),
        path: [],
        options: {},
    },
    dbus: vi.fn(),
};

(globalThis as any).cockpit = mockCockpit;

// Reset mocks between tests
beforeEach(() => {
    vi.clearAllMocks();
});

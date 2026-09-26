import { describe, it, expect, vi, beforeEach } from 'vitest';
import { _, ngettext, format } from '../src/i18n';

describe('i18n', () => {
    const mockCockpit = (globalThis as any).cockpit;

    beforeEach(() => {
        vi.clearAllMocks();
        // Reset to default implementations
        mockCockpit.gettext.mockImplementation((str: string) => str);
        mockCockpit.ngettext.mockImplementation((singular: string, plural: string, n: number) => n === 1 ? singular : plural);
        mockCockpit.format.mockImplementation((fmt: string, ...args: any[]) => {
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
        });
    });

    describe('_', () => {
        it('should call cockpit.gettext with the string', () => {
            mockCockpit.gettext.mockReturnValueOnce('Translated string');

            const result = _('Test string');

            expect(mockCockpit.gettext).toHaveBeenCalledWith('Test string');
            expect(result).toBe('Translated string');
        });

        it('should return the original string if no translation exists', () => {
            const result = _('Test string');

            expect(mockCockpit.gettext).toHaveBeenCalledWith('Test string');
            expect(result).toBe('Test string');
        });

        it('should handle template strings with format', () => {
            mockCockpit.gettext.mockReturnValueOnce('File changes (${count})');

            const result = _('File changes (${count})', { count: 5 });

            expect(mockCockpit.gettext).toHaveBeenCalledWith('File changes (${count})');
            expect(result).toBe('File changes (5)');
        });

        it('should handle multiple template variables', () => {
            mockCockpit.gettext.mockReturnValueOnce('${name} has ${count} services');

            const result = _('${name} has ${count} services', { name: 'myapp', count: 3 });

            expect(result).toBe('myapp has 3 services');
        });
    });

    describe('ngettext', () => {
        it('should call cockpit.ngettext with singular, plural, and count', () => {
            const result = ngettext('1 service', '${count} services', 1);

            expect(mockCockpit.ngettext).toHaveBeenCalledWith('1 service', '${count} services', 1);
            expect(result).toBe('1 service');
        });

        it('should return plural form for count > 1', () => {
            const result = ngettext('1 service', '${count} services', 5);

            expect(mockCockpit.ngettext).toHaveBeenCalledWith('1 service', '${count} services', 5);
            expect(result).toBe('${count} services');
        });
    });

    describe('format', () => {
        it('should call cockpit.format with the format string and args', () => {
            const result = format('Hello, ${name}!', { name: 'World' });

            expect(mockCockpit.format).toHaveBeenCalledWith('Hello, ${name}!', { name: 'World' });
            expect(result).toBe('Hello, World!');
        });

        it('should handle multiple arguments', () => {
            const result = format('File ${file} has ${lines} lines', { file: 'test.txt', lines: 42 });

            expect(mockCockpit.format).toHaveBeenCalledWith('File ${file} has ${lines} lines', { file: 'test.txt', lines: 42 });
            expect(result).toBe('File test.txt has 42 lines');
        });
    });
});

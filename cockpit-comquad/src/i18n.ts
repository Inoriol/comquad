declare global {
    interface Window {
        cockpit: {
            gettext: (str: string) => string;
            ngettext: (singular: string, plural: string, n: number) => string;
            format: (fmt: string, ...args: any[]) => string;
        };
    }
    const cockpit: {
        gettext: (str: string) => string;
        ngettext: (singular: string, plural: string, n: number) => string;
        format: (fmt: string, ...args: any[]) => string;
    };
}

export const _ = (str: string, ...args: any[]): string => {
    const translated = cockpit.gettext(str);
    if (args.length > 0 && typeof args[0] === 'object' && args[0] !== null) {
        return cockpit.format(translated, args[0]);
    }
    return translated;
};
export const ngettext = (singular: string, plural: string, n: number): string => cockpit.ngettext(singular, plural, n);
export const format = (fmt: string, ...args: any[]): string => cockpit.format(fmt, ...args);

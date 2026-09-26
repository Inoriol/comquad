import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as client from '../src/client';

describe('client', () => {
    const mockCockpit = (globalThis as any).cockpit;

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('listDirectory', () => {
        it('should parse directory listing', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('dir1\ndir2\ndir3\n');

            const result = await client.listDirectory('/home/user');

            expect(result).toEqual(['dir1', 'dir2', 'dir3']);
        });

        it('should filter empty lines', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('dir1\n\ndir2\n\n');

            const result = await client.listDirectory('/home/user');

            expect(result).toEqual(['dir1', 'dir2']);
        });
    });

    describe('checkComposeFile', () => {
        it('should return true when compose.yaml exists', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('compose.yaml\nother-file.txt\n');

            const result = await client.checkComposeFile('/home/user/test');

            expect(result).toBe(true);
        });

        it('should return true when docker-compose.yml exists', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('docker-compose.yml\nother-file.txt\n');

            const result = await client.checkComposeFile('/home/user/test');

            expect(result).toBe(true);
        });

        it('should return true when podman-compose.yaml exists', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('podman-compose.yaml\nother-file.txt\n');

            const result = await client.checkComposeFile('/home/user/test');

            expect(result).toBe(true);
        });

        it('should return false when no compose file exists', async () => {
            mockCockpit.spawn.mockResolvedValueOnce('other-file.txt\nreadme.md\n');

            const result = await client.checkComposeFile('/home/user/test');

            expect(result).toBe(false);
        });

        it('should return false on error', async () => {
            mockCockpit.spawn.mockRejectedValueOnce(new Error('Permission denied'));

            const result = await client.checkComposeFile('/home/user/test');

            expect(result).toBe(false);
        });
    });
});

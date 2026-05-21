import { reactive } from 'vue';
import { backend } from '../wailsbridge';

const mediaCache = reactive<Record<string, string | null>>({});
const mediaErrors = reactive<Record<string, string>>({});

function mediaKey(type: string, keys: string[]) {
    return `${type}:${keys.filter(Boolean).join(',')}`;
}

export function useMediaCache() {
    function getMediaSrc(type: string, keys: string[]): string | null {
        const k = mediaKey(type, keys);
        if (k in mediaCache) return mediaCache[k];
        mediaCache[k] = null;
        backend
            .GetMediaData(type, keys.filter(Boolean).join(','))
            .then(r => {
                mediaCache[k] = r.data || '';
            })
            .catch(e => {
                mediaCache[k] = '';
                mediaErrors[k] = String(e);
            });
        return null;
    }

    function getMediaError(type: string, keys: string[]): string {
        return mediaErrors[mediaKey(type, keys)] || '';
    }

    return { getMediaSrc, getMediaError };
}
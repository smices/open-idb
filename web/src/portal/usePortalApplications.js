import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../lib/api.ts';

function isAbortError(error) {
  return error instanceof DOMException && error.name === 'AbortError';
}

export function usePortalApplications() {
  const [applications, setApplications] = useState([]);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reloadToken, setReloadToken] = useState(0);
  const requestIdRef = useRef(0);

  const reload = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    const requestId = ++requestIdRef.current;

    setLoading(true);
    setError(null);
    api.portalApplications({ signal: controller.signal })
      .then((response) => {
        if (requestId === requestIdRef.current) {
          setApplications(Array.isArray(response?.applications) ? response.applications : []);
        }
      })
      .catch((nextError) => {
        if (requestId === requestIdRef.current && !isAbortError(nextError)) {
          setError(nextError);
        }
      })
      .finally(() => {
        if (requestId === requestIdRef.current) {
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [reloadToken]);

  return { applications, error, loading, reload };
}

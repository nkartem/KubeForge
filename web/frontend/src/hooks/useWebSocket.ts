import { useEffect, useRef, useState, useCallback } from 'react';
import type { ProvisionEvent } from '../api/client';

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';

interface UseWebSocketOptions {
  onMessage?: (event: ProvisionEvent) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
}

export const useWebSocket = (clusterId: number | null, options: UseWebSocketOptions = {}) => {
  const [isConnected, setIsConnected] = useState(false);
  const [events, setEvents] = useState<ProvisionEvent[]>([]);
  const wsRef = useRef<WebSocket | null>(null);

  const connect = useCallback(() => {
    if (!clusterId || wsRef.current) return;

    const ws = new WebSocket(`${WS_BASE_URL}/ws/clusters/${clusterId}/events`);

    ws.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
      options.onOpen?.();
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as ProvisionEvent;
        setEvents((prev) => [...prev, data]);
        options.onMessage?.(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
      wsRef.current = null;
      options.onClose?.();
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      options.onError?.(error);
    };

    wsRef.current = ws;
  }, [clusterId, options]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
      setIsConnected(false);
    }
  }, []);

  const send = useCallback((data: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    isConnected,
    events,
    send,
    disconnect,
  };
};

"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { operationsClient } from "@/lib/operations-client";
import type { AgentStatus, DashboardData, DemandForecast, Incident, RunAction } from "@/lib/types";

export function useOperations() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [forecasts, setForecasts] = useState<DemandForecast[]>([]);
  const [agent, setAgent] = useState<AgentStatus | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [detail, setDetail] = useState<Incident | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState("");
  const mounted = useRef(true);
  const selection = useRef(selected);
  const inFlight = useRef(false);

  const refresh = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    try {
      const next = await operationsClient.dashboard();
      if (!mounted.current) return;
      setData(next);
      const id = selection.current || next.incidents[0]?.id;
      if (id) {
        const incident = await operationsClient.incident(id);
        if (mounted.current && (!selection.current || selection.current === id)) {
          setDetail(incident);
          if (!selection.current) {
            selection.current = id;
            setSelected(id);
          }
        }
      }
      const [predictions, status] = await Promise.allSettled([
        operationsClient.forecasts(),
        operationsClient.status(),
      ]);
      if (mounted.current) {
        if (predictions.status === "fulfilled") setForecasts(predictions.value);
        if (status.status === "fulfilled") setAgent(status.value);
      }
    } catch (error) {
      if (mounted.current) setError((error as Error).message);
    } finally {
      inFlight.current = false;
    }
  }, []);

  useEffect(() => {
    mounted.current = true;
    void refresh();
    const interval = setInterval(() => void refresh(), 4000);
    return () => {
      mounted.current = false;
      clearInterval(interval);
    };
  }, [refresh]);

  const selectIncident = (id: string) => {
    selection.current = id;
    setSelected(id);
    void operationsClient
      .incident(id)
      .then((incident) => {
        if (mounted.current && selection.current === id) setDetail(incident);
      })
      .catch((error: Error) => setError(error.message));
  };

  const runAction: RunAction = async (name, task, successMessage) => {
    setBusy(name);
    setError("");
    setNotice("");
    try {
      await task();
      setNotice(successMessage);
      await refresh();
    } catch (error) {
      setError((error as Error).message);
    } finally {
      setBusy("");
    }
  };

  return {
    data,
    forecasts,
    agent,
    selected,
    detail,
    error,
    notice,
    busy,
    refresh,
    selectIncident,
    runAction,
    dismissError: () => setError(""),
    dismissNotice: () => setNotice(""),
  };
}

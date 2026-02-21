/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useContext, useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { FileContent, Project, ProjectDetail, RunInfo, TaskDetail, TaskSummary } from '../types'
import { APIClient, createClient } from '../api/client'
import type { TaskStartRequest } from '../api/client'

const APIClientContext = createContext<APIClient | null>(null)

export function APIProvider({
  children,
  client,
}: {
  children: React.ReactNode
  client?: APIClient
}) {
  const value = useMemo(() => client ?? createClient(), [client])
  return <APIClientContext.Provider value={value}>{children}</APIClientContext.Provider>
}

export function useAPIClient(): APIClient {
  const client = useContext(APIClientContext)
  if (!client) {
    throw new Error('API client not available; wrap with APIProvider')
  }
  return client
}

export function useProjects() {
  const api = useAPIClient()
  return useQuery<Project[]>({
    queryKey: ['projects'],
    queryFn: () => api.getProjects(),
    staleTime: 2000,
  })
}

export function useProject(projectId?: string) {
  const api = useAPIClient()
  return useQuery<ProjectDetail>({
    queryKey: ['project', projectId],
    queryFn: () => api.getProject(projectId ?? ''),
    enabled: Boolean(projectId),
    staleTime: 2000,
  })
}

export function useTasks(projectId?: string) {
  const api = useAPIClient()
  return useQuery<TaskSummary[]>({
    queryKey: ['tasks', projectId],
    queryFn: () => api.getTasks(projectId ?? ''),
    enabled: Boolean(projectId),
    staleTime: 2000,
  })
}

export function useTask(projectId?: string, taskId?: string) {
  const api = useAPIClient()
  return useQuery<TaskDetail>({
    queryKey: ['task', projectId, taskId],
    queryFn: () => api.getTask(projectId ?? '', taskId ?? ''),
    enabled: Boolean(projectId && taskId),
    staleTime: 2000,
  })
}

export function useRunInfo(projectId?: string, taskId?: string, runId?: string) {
  const api = useAPIClient()
  return useQuery<RunInfo>({
    queryKey: ['run', projectId, taskId, runId],
    queryFn: () => api.getRunInfo(projectId ?? '', taskId ?? '', runId ?? ''),
    enabled: Boolean(projectId && taskId && runId),
    staleTime: 2000,
  })
}

export function useTaskFile(projectId?: string, taskId?: string, name?: string) {
  const api = useAPIClient()
  return useQuery<FileContent>({
    queryKey: ['task-file', projectId, taskId, name],
    queryFn: () => api.getTaskFile(projectId ?? '', taskId ?? '', name ?? ''),
    enabled: Boolean(projectId && taskId && name),
    staleTime: 1000,
  })
}

export function useRunFile(projectId?: string, taskId?: string, runId?: string, name?: string, tail?: number) {
  const api = useAPIClient()
  return useQuery<FileContent>({
    queryKey: ['run-file', projectId, taskId, runId, name, tail],
    queryFn: () => api.getRunFile(projectId ?? '', taskId ?? '', runId ?? '', name ?? '', tail),
    enabled: Boolean(projectId && taskId && runId && name),
    staleTime: 1000,
  })
}

export function useStartTask(projectId?: string) {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (payload: TaskStartRequest) => api.startTask(projectId ?? '', payload),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['tasks', projectId] })
    },
  })
}

export function useDeleteRun(projectId?: string, taskId?: string) {
  const api = useAPIClient()
  const client = useQueryClient()
  return useMutation({
    mutationFn: (runId: string) => api.deleteRun(projectId ?? '', taskId ?? '', runId),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['task', projectId, taskId] })
      client.invalidateQueries({ queryKey: ['tasks', projectId] })
    },
  })
}

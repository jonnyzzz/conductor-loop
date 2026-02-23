import type { BusMessage } from '../types'

export const DEFAULT_MAX_MESSAGES = 1500

export function parseMessageTimestamp(timestamp: string): number {
  const parsed = Date.parse(timestamp)
  return Number.isNaN(parsed) ? 0 : parsed
}

function normalizedParentsSignature(
  parents: Array<string | { msg_id: string; kind?: string; meta?: Record<string, unknown> }> | undefined
): string {
  if (!parents || parents.length === 0) {
    return ''
  }
  return JSON.stringify(parents)
}

function normalizedMetaSignature(meta: Record<string, string> | undefined): string {
  if (!meta) {
    return ''
  }
  const keys = Object.keys(meta).sort()
  if (keys.length === 0) {
    return ''
  }
  return keys.map((key) => `${key}:${meta[key]}`).join('\u0000')
}

function hasSameMessagePayload(a: BusMessage, b: BusMessage): boolean {
  return a.msg_id === b.msg_id &&
    a.timestamp === b.timestamp &&
    a.type === b.type &&
    a.body === b.body &&
    a.run_id === b.run_id &&
    a.project_id === b.project_id &&
    a.task_id === b.task_id &&
    a.project === b.project &&
    a.task === b.task &&
    a.attachment_path === b.attachment_path &&
    a.issue_id === b.issue_id &&
    normalizedParentsSignature(a.parents) === normalizedParentsSignature(b.parents) &&
    normalizedMetaSignature(a.meta) === normalizedMetaSignature(b.meta)
}

function hasSameMessageSequence(current: BusMessage[], next: BusMessage[]): boolean {
  if (current.length !== next.length) {
    return false
  }
  for (let i = 0; i < current.length; i += 1) {
    if (!hasSameMessagePayload(current[i], next[i])) {
      return false
    }
  }
  return true
}

function compareByRecency(a: BusMessage, b: BusMessage): number {
  const byTime = parseMessageTimestamp(b.timestamp) - parseMessageTimestamp(a.timestamp)
  if (byTime !== 0) {
    return byTime
  }
  return b.msg_id.localeCompare(a.msg_id)
}

function shouldReplaceMessage(current: BusMessage, next: BusMessage): boolean {
  const nextTimestamp = parseMessageTimestamp(next.timestamp)
  const currentTimestamp = parseMessageTimestamp(current.timestamp)
  if (nextTimestamp > currentTimestamp) {
    return true
  }
  if (nextTimestamp < currentTimestamp) {
    return false
  }
  return !hasSameMessagePayload(current, next)
}

function shouldComeBefore(candidate: BusMessage, current: BusMessage): boolean {
  const candidateTs = parseMessageTimestamp(candidate.timestamp)
  const currentTs = parseMessageTimestamp(current.timestamp)
  if (candidateTs !== currentTs) {
    return candidateTs > currentTs
  }
  return candidate.msg_id.localeCompare(current.msg_id) > 0
}

function trimToLimit(messages: BusMessage[], maxMessages: number): BusMessage[] {
  if (maxMessages <= 0 || messages.length <= maxMessages) {
    return messages
  }
  return messages.slice(0, maxMessages)
}

function isRecencyOrdered(messages: BusMessage[]): boolean {
  for (let i = 1; i < messages.length; i += 1) {
    if (shouldComeBefore(messages[i], messages[i - 1])) {
      return false
    }
  }
  return true
}

function mergeOrderedByRecency(existing: BusMessage[], incoming: BusMessage[]): BusMessage[] {
  const merged: BusMessage[] = []
  let existingIndex = 0
  let incomingIndex = 0
  while (existingIndex < existing.length && incomingIndex < incoming.length) {
    if (compareByRecency(existing[existingIndex], incoming[incomingIndex]) <= 0) {
      merged.push(existing[existingIndex])
      existingIndex += 1
      continue
    }
    merged.push(incoming[incomingIndex])
    incomingIndex += 1
  }
  if (existingIndex < existing.length) {
    merged.push(...existing.slice(existingIndex))
  }
  if (incomingIndex < incoming.length) {
    merged.push(...incoming.slice(incomingIndex))
  }
  return merged
}

function mergeMessagesByIDWithSort(
  existing: BusMessage[],
  incoming: BusMessage[],
  maxMessages: number
): BusMessage[] {
  const byID = new Map<string, BusMessage>()
  for (const message of [...existing, ...incoming]) {
    const current = byID.get(message.msg_id)
    if (!current) {
      byID.set(message.msg_id, message)
      continue
    }
    if (shouldReplaceMessage(current, message)) {
      byID.set(message.msg_id, message)
    }
  }
  const merged = trimToLimit([...byID.values()].sort(compareByRecency), maxMessages)
  if (hasSameMessageSequence(existing, merged)) {
    return existing
  }
  return merged
}

function tryMergeMessagesByIDLinear(
  existing: BusMessage[],
  incoming: BusMessage[],
  maxMessages: number
): BusMessage[] | null {
  if (!isRecencyOrdered(existing) || !isRecencyOrdered(incoming)) {
    return null
  }

  const existingByID = new Map<string, BusMessage>()
  for (const message of existing) {
    if (existingByID.has(message.msg_id)) {
      return null
    }
    existingByID.set(message.msg_id, message)
  }

  const incomingByID = new Map<string, BusMessage>()
  const incomingOrder: string[] = []
  for (const message of incoming) {
    const current = incomingByID.get(message.msg_id)
    if (!current) {
      incomingByID.set(message.msg_id, message)
      incomingOrder.push(message.msg_id)
      continue
    }
    if (shouldReplaceMessage(current, message)) {
      incomingByID.set(message.msg_id, message)
    }
  }

  const replacementIDs = new Set<string>()
  const acceptedIncoming: BusMessage[] = []
  let changed = false
  for (const msgID of incomingOrder) {
    const candidate = incomingByID.get(msgID)
    if (!candidate) {
      continue
    }
    const current = existingByID.get(msgID)
    if (!current) {
      acceptedIncoming.push(candidate)
      changed = true
      continue
    }
    if (shouldReplaceMessage(current, candidate)) {
      acceptedIncoming.push(candidate)
      replacementIDs.add(msgID)
      changed = true
    }
  }

  if (!changed) {
    const trimmedExisting = trimToLimit(existing, maxMessages)
    if (hasSameMessageSequence(existing, trimmedExisting)) {
      return existing
    }
    return trimmedExisting
  }

  const keptExisting = replacementIDs.size > 0
    ? existing.filter((message) => !replacementIDs.has(message.msg_id))
    : existing
  const merged = trimToLimit(mergeOrderedByRecency(keptExisting, acceptedIncoming), maxMessages)
  if (hasSameMessageSequence(existing, merged)) {
    return existing
  }
  return merged
}

export function mergeMessagesByID(
  existing: BusMessage[],
  incoming: BusMessage[],
  maxMessages = DEFAULT_MAX_MESSAGES
): BusMessage[] {
  const linearMerged = tryMergeMessagesByIDLinear(existing, incoming, maxMessages)
  if (linearMerged !== null) {
    return linearMerged
  }
  return mergeMessagesByIDWithSort(existing, incoming, maxMessages)
}

export function upsertMessageBatch(
  existing: BusMessage[],
  incoming: BusMessage[],
  maxMessages = DEFAULT_MAX_MESSAGES
): BusMessage[] {
  if (incoming.length === 0) {
    return existing
  }
  if (incoming.length === 1) {
    return upsertMessageByID(existing, incoming[0], maxMessages)
  }
  return mergeMessagesByID(existing, incoming, maxMessages)
}

export function upsertMessageByID(
  existing: BusMessage[],
  incoming: BusMessage,
  maxMessages = DEFAULT_MAX_MESSAGES
): BusMessage[] {
  const existingIndex = existing.findIndex((message) => message.msg_id === incoming.msg_id)
  if (existingIndex >= 0) {
    const current = existing[existingIndex]
    const incomingTimestamp = parseMessageTimestamp(incoming.timestamp)
    const currentTimestamp = parseMessageTimestamp(current.timestamp)
    if (incomingTimestamp < currentTimestamp) {
      return existing
    }
    if (incomingTimestamp === currentTimestamp && hasSameMessagePayload(current, incoming)) {
      return existing
    }
  }

  const next = existingIndex >= 0
    ? [...existing.slice(0, existingIndex), ...existing.slice(existingIndex + 1)]
    : [...existing]

  let insertIndex = next.length
  for (let i = 0; i < next.length; i += 1) {
    if (shouldComeBefore(incoming, next[i])) {
      insertIndex = i
      break
    }
  }
  next.splice(insertIndex, 0, incoming)
  return trimToLimit(next, maxMessages)
}

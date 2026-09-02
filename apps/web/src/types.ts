export type Provider = 'claude-code' | 'codex' | 'cursor'
export type State = 'working' | 'waiting' | 'blocked' | 'done' | 'failed' | 'idle'

export interface Session {
  id: string
  provider: Provider
  cwd: string
  repo: string
  branch: string
  worktree: string
  state: State
  last_message: string
  created_at: string
  updated_at: string
}

// PilotProvider is narrower than Provider: only these two can be piloted
// (launched and driven by LiveAgentsView) in this MVP — see
// PilotedSessionView for what each one can and can't do live.
export type PilotProvider = 'claude-code' | 'cursor'

export type PilotEventKind =
  | 'user'
  | 'assistant'
  | 'tool_call'
  | 'permission_request'
  | 'permission_resolved'
  | 'system'
  | 'error'

export interface PilotEvent {
  kind: PilotEventKind
  text?: string
  tool_name?: string
  tool_input?: unknown
  request_id?: string
  approved?: boolean
  at: string
}

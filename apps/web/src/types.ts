export type Race = 'claude-code' | 'cursor'
export type Activity = 'ready' | 'working' | 'waiting' | 'failed'
export type Presence = 'awake' | 'asleep'
export type TerritoryMode = 'own' | 'shared'

export interface Territory {
  mode: TerritoryMode
  path: string
  source: string
  branch: string
}

export interface Character {
  id: string
  session_id: string
  race: Race
  class: string
  activity: Activity
  presence: Presence
  unread: boolean
  territory: Territory
  repo: string
  archived: boolean
  last_message: string
  created_at: string
  updated_at: string
}

// The three Claude Code model aliases the recruit panel offers — confirmed
// live against the installed CLI's --model flag.
export type ClaudeClassId = 'opus' | 'sonnet' | 'haiku'

// One entry from the daemon's live Cursor class catalog (`agent --list-models`).
export interface CursorClassOption {
  id: string
  label: string
}

export type PilotEventKind = 'user' | 'assistant' | 'tool_call' | 'thinking' | 'system' | 'error'

export interface PilotEvent {
  kind: PilotEventKind
  text?: string
  tool_name?: string
  tool_input?: unknown
  request_id?: string
  at: string
}

export type Provider = 'claude-code' | 'codex' | 'cursor'
export type Fidelity = 'driver' | 'hooks' | 'tailing'
export type State = 'working' | 'waiting' | 'blocked' | 'done' | 'failed' | 'idle'

export interface Session {
  id: string
  provider: Provider
  fidelity: Fidelity
  cwd: string
  repo: string
  branch: string
  worktree: string
  state: State
  last_message: string
  created_at: string
  updated_at: string
}

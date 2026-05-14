export interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  created_at: string;
}

export interface MatchPlayer {
  match_id: string;
  user_id: string;
  user_name?: string;
  active: boolean;
  joined_at: string;
}

export interface Match {
  id: string;
  name: string;
  creator_id: string;
  status: 'waiting' | 'playing' | 'finished';
  current_round: number;
  created_at: string;
  players?: MatchPlayer[];
}

export interface Round {
  id: string;
  match_id: string;
  round_number: number;
  letter: string;
  status: 'playing' | 'finished';
  started_at: string;
  ends_at: string;
}

export interface Answer {
  id: string;
  round_id: string;
  user_id: string;
  color: string;
  fruit: string;
  object: string;
  movie: string;
  city: string;
  animal: string;
  name: string;
  score: number;
  submitted_at: string;
}

export interface RoundResult {
  round_id: string;
  letter: string;
  answers: Answer[];
}

export interface PlayerScore {
  user_id: string;
  score: number;
}

export interface WSMessage {
  type: 'round_started' | 'round_ended' | 'timer_tick' | 'scores_updated' | 'player_joined' | 'player_left' | 'match_ended' | 'join_request' | 'join_accepted' | 'join_rejected' | 'match_invite';
  letter?: string;
  round_id?: string;
  ends_at?: string;
  seconds_remaining?: number;
  scores?: PlayerScore[];
  user_id?: string;
  user_name?: string;
  ranking?: RankingEntry[];
  request_id?: string;
  match_id?: string;
  match_name?: string;
}

export interface RankingEntry {
  user_id: string;
  user_name: string;
  total_score: number;
  position: number;
}

export interface OnlineUser {
  id: string;
  name: string;
  avatar: string;
}

export interface MatchState {
  match: Match;
  phase: 'lobby' | 'playing' | 'round_ended' | 'finished';
  current_round?: Round;
  round_result?: RoundResult;
  ranking?: RankingEntry[];
}

export interface OpenMatch {
  id: string;
  name: string;
  creator_name: string;
  player_count: number;
  status: string;
}

export interface JoinRequest {
  id: string;
  match_id: string;
  user_id: string;
  user_name: string;
  status: string;
  created_at: string;
}

export interface PendingInvite {
  id: string;
  match_id: string;
  match_name: string;
  inviter_name: string;
  status: string;
}

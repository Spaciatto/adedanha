const API_BASE = '';

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${url}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
}

export const api = {
  // Users
  createUser: (name: string, email: string) =>
    request('/api/users', {
      method: 'POST',
      body: JSON.stringify({ name, email }),
    }),

  loginUser: (email: string) =>
    request('/api/users/login', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),

  updateUser: (id: string, name: string, email: string) =>
    request(`/api/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, email }),
    }),

  getUser: (id: string) =>
    request(`/api/users/${id}`),

  leaveAllMatches: (userId: string) =>
    request(`/api/users/${userId}/leave-all`, {
      method: 'POST',
    }),

  getActiveMatch: (userId: string) =>
    request(`/api/users/${userId}/active-match`),

  getOnlineUsers: () =>
    request('/api/online-users'),

  getAvailablePlayers: () =>
    request('/api/available-players'),

  getPendingInvites: (userId: string) =>
    request(`/api/invites/${userId}`),

  respondInvite: (inviteId: string, userId: string, accepted: boolean) =>
    request(`/api/invites/${inviteId}/respond`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, accepted }),
    }),

  // Matches
  createMatch: (creatorId: string, name: string) =>
    request('/api/matches', {
      method: 'POST',
      body: JSON.stringify({ creator_id: creatorId, name }),
    }),

  listOpenMatches: () =>
    request('/api/matches/open'),

  joinMatch: (matchId: string, userId: string) =>
    request(`/api/matches/${matchId}/join`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    }),

  leaveMatch: (matchId: string, userId: string) =>
    request(`/api/matches/${matchId}/leave`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    }),

  requestJoinMatch: (matchId: string, userId: string) =>
    request(`/api/matches/${matchId}/request-join`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    }),

  getJoinRequests: (matchId: string) =>
    request(`/api/matches/${matchId}/join-requests`),

  respondJoinRequest: (matchId: string, requestId: string, userId: string, accepted: boolean) =>
    request(`/api/matches/${matchId}/join-requests/${requestId}/respond`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, accepted }),
    }),

  invitePlayer: (matchId: string, creatorId: string, playerId: string) =>
    request(`/api/matches/${matchId}/invite`, {
      method: 'POST',
      body: JSON.stringify({ creator_id: creatorId, player_id: playerId }),
    }),

  getMatch: (matchId: string) =>
    request(`/api/matches/${matchId}`),

  getMatchState: (matchId: string) =>
    request(`/api/matches/${matchId}/state`),

  endMatch: (matchId: string, userId: string) =>
    request(`/api/matches/${matchId}/end`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    }),

  startRound: (matchId: string, userId: string) =>
    request(`/api/matches/${matchId}/rounds/start`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    }),

  submitAnswers: (matchId: string, roundId: string, userId: string, answers: {
    color: string;
    fruit: string;
    object: string;
    movie: string;
    city: string;
    animal: string;
    name: string;
  }) =>
    request(`/api/matches/${matchId}/rounds/${roundId}/answers`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, ...answers }),
    }),

  updateScores: (matchId: string, roundId: string, userId: string, scores: { user_id: string; score: number }[]) =>
    request(`/api/matches/${matchId}/rounds/${roundId}/scores`, {
      method: 'PUT',
      body: JSON.stringify({ user_id: userId, scores }),
    }),

  getRoundResults: (matchId: string, roundId: string) =>
    request(`/api/matches/${matchId}/rounds/${roundId}/results`),
};

export function connectWebSocket(matchId: string, userId: string): WebSocket {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws/${matchId}/${userId}`);
  return ws;
}

export function connectPresenceWebSocket(userId: string): WebSocket {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws/presence/${userId}`);
  return ws;
}

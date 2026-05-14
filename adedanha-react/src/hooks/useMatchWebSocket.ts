import { useEffect, useRef } from 'react';
import { WSMessage, Match, RoundResult, RankingEntry, Answer, JoinRequest, Round } from '../types';
import { api, connectWebSocket } from '../api';

type GamePhase = 'lobby' | 'playing' | 'round_ended' | 'scores' | 'finished';

interface AutoSubmitRefs {
  color: React.MutableRefObject<string>;
  fruit: React.MutableRefObject<string>;
  object: React.MutableRefObject<string>;
  movie: React.MutableRefObject<string>;
  city: React.MutableRefObject<string>;
  animal: React.MutableRefObject<string>;
  playerName: React.MutableRefObject<string>;
  answersSubmitted: React.MutableRefObject<boolean>;
  currentRound: React.MutableRefObject<Round | null>;
  phase: React.MutableRefObject<GamePhase>;
}

interface WebSocketCallbacks {
  onRoundStarted: (letter: string, roundId: string, endsAt: string) => void;
  onTimerTick: (seconds: number) => void;
  onRoundEnded: (roundId: string) => void;
  onScoresUpdated: (scores: { user_id: string; score: number }[]) => void;
  onMatchEnded: (ranking: RankingEntry[]) => void;
  onMatchUpdated: (match: Match) => void;
  onJoinRequestsUpdated: (requests: JoinRequest[]) => void;
  onAnswersSubmitted: () => void;
}

export function useMatchWebSocket(
  matchId: string | undefined,
  userId: string,
  refs: AutoSubmitRefs,
  callbacks: WebSocketCallbacks,
) {
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!matchId || !userId) return;

    const ws = connectWebSocket(matchId, userId);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);

        switch (msg.type) {
          case 'round_started':
            callbacks.onRoundStarted(msg.letter || '', msg.round_id || '', msg.ends_at || '');
            break;

          case 'timer_tick':
            callbacks.onTimerTick(msg.seconds_remaining || 0);
            break;

          case 'round_ended':
            // Auto-submit if not already submitted
            if (!refs.answersSubmitted.current && refs.currentRound.current && refs.phase.current === 'playing') {
              const hasAny = refs.color.current || refs.fruit.current || refs.object.current ||
                refs.movie.current || refs.city.current || refs.animal.current || refs.playerName.current;
              if (hasAny) {
                api.submitAnswers(matchId, refs.currentRound.current.id, userId, {
                  color: refs.color.current.toUpperCase(),
                  fruit: refs.fruit.current.toUpperCase(),
                  object: refs.object.current.toUpperCase(),
                  movie: refs.movie.current.toUpperCase(),
                  city: refs.city.current.toUpperCase(),
                  animal: refs.animal.current.toUpperCase(),
                  name: refs.playerName.current.toUpperCase(),
                }).catch(console.error);
              }
              callbacks.onAnswersSubmitted();
            }

            callbacks.onRoundEnded(msg.round_id || '');
            break;

          case 'scores_updated':
            if (msg.scores) {
              callbacks.onScoresUpdated(msg.scores);
            }
            break;

          case 'match_ended':
            if (msg.ranking) {
              callbacks.onMatchEnded(msg.ranking);
            }
            break;

          case 'player_joined':
          case 'player_left':
            api.getMatch(matchId).then((m: any) => callbacks.onMatchUpdated(m as Match)).catch(console.error);
            break;

          case 'join_request':
            api.getJoinRequests(matchId).then((reqs: any) => callbacks.onJoinRequestsUpdated(reqs as JoinRequest[])).catch(() => {});
            break;
        }
      } catch (e) {
        console.error('Error handling WS message:', e);
      }
    };

    ws.onclose = () => {
      console.log('Match WebSocket disconnected');
    };

    return () => {
      ws.close();
    };
  }, [matchId, userId]);

  return wsRef;
}

import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { User, Match, Round, Answer, WSMessage, RoundResult, RankingEntry, MatchState, JoinRequest } from '../types';
import { api, connectWebSocket } from '../api';
import MatchHeader from '../components/MatchHeader';
import MatchLobby from '../components/MatchLobby';
import MatchFinished from '../components/MatchFinished';
import RoundPlaying from '../components/RoundPlaying';
import RoundResults from '../components/RoundResults';

interface MatchPageProps {
  user: User;
}

type GamePhase = 'lobby' | 'playing' | 'round_ended' | 'scores' | 'finished';

function MatchPage({ user }: MatchPageProps) {
  const { matchId } = useParams<{ matchId: string }>();
  const navigate = useNavigate();
  const [match, setMatch] = useState<Match | null>(null);
  const [phase, setPhase] = useState<GamePhase>('lobby');
  const [currentRound, setCurrentRound] = useState<Round | null>(null);
  const [letter, setLetter] = useState('');
  const [secondsRemaining, setSecondsRemaining] = useState(60);
  const [roundResults, setRoundResults] = useState<RoundResult | null>(null);
  const [ranking, setRanking] = useState<RankingEntry[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [joinRequests, setJoinRequests] = useState<JoinRequest[]>([]);
  const [availablePlayers, setAvailablePlayers] = useState<{id: string; name: string; avatar: string}[]>([]);
  const [invitedPlayers, setInvitedPlayers] = useState<Record<string, boolean>>({});
  const [validations, setValidations] = useState<Record<string, Record<string, string>>>({});
  const [validating, setValidating] = useState(false);

  // Answer fields
  const [color, setColor] = useState('');
  const [fruit, setFruit] = useState('');
  const [object, setObject] = useState('');
  const [movie, setMovie] = useState('');
  const [city, setCity] = useState('');
  const [animal, setAnimal] = useState('');
  const [playerName, setPlayerName] = useState('');
  const [answersSubmitted, setAnswersSubmitted] = useState(false);

  // Scores
  const [scores, setScores] = useState<Record<string, number>>({});

  const wsRef = useRef<WebSocket | null>(null);
  const matchRef = useRef<Match | null>(null);

  // Refs for auto-submit on round end
  const colorRef = useRef('');
  const fruitRef = useRef('');
  const objectRef = useRef('');
  const movieRef = useRef('');
  const cityRef = useRef('');
  const animalRef = useRef('');
  const playerNameRef = useRef('');
  const answersSubmittedRef = useRef(false);
  const currentRoundRef = useRef<Round | null>(null);
  const phaseRef = useRef<GamePhase>('lobby');

  // Keep refs in sync
  useEffect(() => { matchRef.current = match; }, [match]);
  useEffect(() => { colorRef.current = color; }, [color]);
  useEffect(() => { fruitRef.current = fruit; }, [fruit]);
  useEffect(() => { objectRef.current = object; }, [object]);
  useEffect(() => { movieRef.current = movie; }, [movie]);
  useEffect(() => { cityRef.current = city; }, [city]);
  useEffect(() => { animalRef.current = animal; }, [animal]);
  useEffect(() => { playerNameRef.current = playerName; }, [playerName]);
  useEffect(() => { answersSubmittedRef.current = answersSubmitted; }, [answersSubmitted]);
  useEffect(() => { currentRoundRef.current = currentRound; }, [currentRound]);
  useEffect(() => { phaseRef.current = phase; }, [phase]);

  const isCreator = match?.creator_id === user.id;

  const loadMatch = useCallback(async () => {
    if (!matchId) return;
    try {
      const m = await api.getMatch(matchId) as Match;
      setMatch(m);
    } catch (err: any) {
      setError(err.message || 'Erro ao carregar partida');
    }
  }, [matchId]);

  const loadRoundResults = useCallback(async (roundId: string) => {
    if (!matchId) return;
    try {
      const results = await api.getRoundResults(matchId, roundId) as RoundResult;
      setRoundResults(results);
      const initialScores: Record<string, number> = {};
      results.answers.forEach((a: Answer) => {
        initialScores[a.user_id] = a.score || 0;
      });
      setScores(initialScores);
    } catch (err: any) {
      console.error('Error loading results:', err);
    }
  }, [matchId]);

  const loadJoinRequests = useCallback(async () => {
    if (!matchId) return;
    try {
      const requests = await api.getJoinRequests(matchId) as JoinRequest[];
      setJoinRequests(requests);
    } catch {
      // Silently fail
    }
  }, [matchId]);

  // Reconnection: load match state on mount
  useEffect(() => {
    if (!matchId) return;

    const loadState = async () => {
      try {
        const state = await api.getMatchState(matchId) as MatchState;
        setMatch(state.match);

        if (state.phase === 'finished') {
          setPhase('finished');
          if (state.ranking) {
            setRanking(state.ranking);
          }
        } else if (state.phase === 'playing' && state.current_round) {
          setPhase('lobby');
          setCurrentRound(state.current_round);
          setLetter(state.current_round.letter);
        } else if (state.phase === 'round_ended' && state.current_round) {
          setPhase('round_ended');
          setCurrentRound(state.current_round);
          setLetter(state.current_round.letter);
          if (state.round_result) {
            setRoundResults(state.round_result);
            const initialScores: Record<string, number> = {};
            state.round_result.answers.forEach((a: Answer) => {
              initialScores[a.user_id] = a.score || 0;
            });
            setScores(initialScores);
          }
        } else {
          setPhase('lobby');
        }
      } catch {
        loadMatch();
      }
    };

    loadState();
  }, [matchId, loadMatch]);

  // Load join requests for creator (separate effect, polls)
  useEffect(() => {
    if (!isCreator || phase !== 'lobby') return;
    loadJoinRequests();
    const loadAvailable = () => {
      api.getAvailablePlayers().then((players: any) => {
        const filtered = (players as {id: string; name: string; avatar: string}[]).filter(
          (p) => !match?.players?.some((mp) => mp.user_id === p.id)
        );
        setAvailablePlayers(filtered);
      }).catch(() => {});
    };
    loadAvailable();
    const interval = setInterval(() => {
      loadJoinRequests();
      loadAvailable();
    }, 5000);
    return () => clearInterval(interval);
  }, [isCreator, phase, loadJoinRequests, match?.players]);

  // WebSocket connection - only depends on matchId and user.id
  useEffect(() => {
    if (!matchId || !user.id) return;

    const ws = connectWebSocket(matchId, user.id);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);

        switch (msg.type) {
          case 'round_started':
            setLetter(msg.letter || '');
            setCurrentRound({
              id: msg.round_id || '',
              match_id: matchId,
              round_number: 0,
              letter: msg.letter || '',
              status: 'playing',
              started_at: new Date().toISOString(),
              ends_at: msg.ends_at || '',
            });
            setPhase('playing');
            setSecondsRemaining(60);
            setAnswersSubmitted(false);
            setColor('');
            setFruit('');
            setObject('');
            setMovie('');
            setCity('');
            setAnimal('');
            setPlayerName('');
            setRoundResults(null);
            break;

          case 'timer_tick':
            setSecondsRemaining(msg.seconds_remaining || 0);
            break;

          case 'round_ended':
            // Auto-submit answers if not already submitted
            if (!answersSubmittedRef.current && currentRoundRef.current && phaseRef.current === 'playing') {
              const hasAnyAnswer = colorRef.current || fruitRef.current || objectRef.current || movieRef.current || cityRef.current || animalRef.current || playerNameRef.current;
              if (hasAnyAnswer) {
                api.submitAnswers(matchId, currentRoundRef.current.id, user.id, {
                  color: colorRef.current.toUpperCase(),
                  fruit: fruitRef.current.toUpperCase(),
                  object: objectRef.current.toUpperCase(),
                  movie: movieRef.current.toUpperCase(),
                  city: cityRef.current.toUpperCase(),
                  animal: animalRef.current.toUpperCase(),
                  name: playerNameRef.current.toUpperCase(),
                }).catch(console.error);
              }
              setAnswersSubmitted(true);
            }

            setPhase('round_ended');
            if (msg.round_id) {
              // Small delay to allow auto-submit to complete before fetching results
              setTimeout(() => {
                api.getRoundResults(matchId, msg.round_id!).then((results: any) => {
                  setRoundResults(results as RoundResult);
                  const initialScores: Record<string, number> = {};
                  (results as RoundResult).answers.forEach((a: Answer) => {
                    initialScores[a.user_id] = a.score || 0;
                  });
                  setScores(initialScores);
                }).catch(console.error);
              }, 500);
            }
            break;

          case 'scores_updated':
            if (msg.scores) {
              const updatedScores: Record<string, number> = {};
              msg.scores.forEach((s) => {
                updatedScores[s.user_id] = s.score;
              });
              setScores(updatedScores);
            }
            setPhase('scores');
            break;

          case 'match_ended':
            setPhase('finished');
            if (msg.ranking) {
              setRanking(msg.ranking);
            }
            break;

          case 'player_joined':
          case 'player_left':
            // Reload match data
            api.getMatch(matchId).then((m: any) => setMatch(m as Match)).catch(console.error);
            break;

          case 'join_request':
            // Reload join requests
            api.getJoinRequests(matchId).then((reqs: any) => setJoinRequests(reqs as JoinRequest[])).catch(() => {});
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
  }, [matchId, user.id]);

  const handleStartRound = async () => {
    if (!matchId) return;
    setError('');
    setLoading(true);
    try {
      const result = await api.startRound(matchId, user.id) as any;
      // If all letters were used, match ends automatically
      if (result.finished) {
        setPhase('finished');
        if (result.ranking) {
          setRanking(result.ranking);
        }
      }
    } catch (err: any) {
      setError(err.message || 'Erro ao iniciar rodada');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitAnswers = async () => {
    if (!matchId || !currentRound) return;
    setError('');
    setLoading(true);
    try {
      await api.submitAnswers(matchId, currentRound.id, user.id, {
        color: color.toUpperCase(),
        fruit: fruit.toUpperCase(),
        object: object.toUpperCase(),
        movie: movie.toUpperCase(),
        city: city.toUpperCase(),
        animal: animal.toUpperCase(),
        name: playerName.toUpperCase(),
      });
      setAnswersSubmitted(true);
    } catch (err: any) {
      setError(err.message || 'Erro ao enviar respostas');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateScores = async () => {
    if (!matchId || !currentRound) return;
    setError('');
    setLoading(true);
    try {
      const scoresList = Object.entries(scores).map(([odUserId, score]) => ({
        user_id: odUserId,
        score,
      }));
      await api.updateScores(matchId, currentRound.id, user.id, scoresList);
    } catch (err: any) {
      setError(err.message || 'Erro ao atualizar scores');
    } finally {
      setLoading(false);
    }
  };

  const handleEndMatch = async () => {
    if (!matchId) return;
    setError('');
    setLoading(true);
    try {
      await api.endMatch(matchId, user.id);
    } catch (err: any) {
      setError(err.message || 'Erro ao encerrar partida');
    } finally {
      setLoading(false);
    }
  };

  const handleLeaveMatch = async () => {
    if (!matchId) return;
    setError('');
    setLoading(true);
    try {
      await api.leaveMatch(matchId, user.id);
      navigate('/');
    } catch (err: any) {
      setError(err.message || 'Erro ao abandonar partida');
    } finally {
      setLoading(false);
    }
  };

  const handleRespondJoinRequest = async (requestId: string, accepted: boolean) => {
    if (!matchId) return;
    try {
      await api.respondJoinRequest(matchId, requestId, user.id, accepted);
      setJoinRequests((prev) => prev.filter((r) => r.id !== requestId));
      if (accepted) {
        loadMatch();
      }
    } catch (err: any) {
      setError(err.message || 'Erro ao processar solicitação');
    }
  };

  const handleInvitePlayer = async (playerId: string) => {
    if (!matchId) return;
    try {
      await api.invitePlayer(matchId, user.id, playerId);
      setInvitedPlayers((prev) => ({ ...prev, [playerId]: true }));
    } catch (err: any) {
      setError(err.message || 'Erro ao convidar jogador');
    }
  };

  const handleValidateRound = async () => {
    if (!matchId || !currentRound) return;
    setValidating(true);
    try {
      const results = await api.validateRound(matchId, currentRound.id) as Array<{
        user_id: string;
        validations: Array<{ field: string; value: string; status: string }>;
        suggested_score: number;
      }>;

      const validationMap: Record<string, Record<string, string>> = {};
      const suggestedScores: Record<string, number> = {};

      results.forEach((pv) => {
        validationMap[pv.user_id] = {};
        pv.validations.forEach((v) => {
          validationMap[pv.user_id][v.field] = v.status;
        });
        suggestedScores[pv.user_id] = pv.suggested_score;
      });

      setValidations(validationMap);
      setScores((prev) => ({ ...prev, ...suggestedScores }));
    } catch (err: any) {
      setError(err.message || 'Erro ao validar respostas');
    } finally {
      setValidating(false);
    }
  };

  const getPlayerName = (odUserId: string): string => {
    const player = match?.players?.find((p) => p.user_id === odUserId);
    return player?.user_name || odUserId.slice(0, 8);
  };

  const getValidationIcon = (userId: string, field: string): string => {
    const v = validations[userId]?.[field];
    if (!v) return '';
    if (v === 'valid') return ' ✅';
    if (v === 'invalid') return ' ❌';
    return ' ⚠️';
  };

  if (!match) {
    return (
      <div className="card">
        <p>Carregando partida...</p>
        {error && <div className="error-message">{error}</div>}
      </div>
    );
  }

  return (
    <>
      <MatchHeader
        match={match}
        isCreator={isCreator}
        phase={phase}
        loading={loading}
        error={error}
        onLeave={handleLeaveMatch}
      />

      {phase === 'finished' && (
        <MatchFinished ranking={ranking} onGoHome={() => navigate('/')} />
      )}

      {phase === 'lobby' && (
        <MatchLobby
          isCreator={isCreator}
          loading={loading}
          joinRequests={joinRequests}
          availablePlayers={availablePlayers}
          invitedPlayers={invitedPlayers}
          onStartRound={handleStartRound}
          onRespondRequest={handleRespondJoinRequest}
          onInvitePlayer={handleInvitePlayer}
        />
      )}

      {phase === 'playing' && (
        <RoundPlaying
          letter={letter}
          secondsRemaining={secondsRemaining}
          answersSubmitted={answersSubmitted}
          color={color}
          fruit={fruit}
          object={object}
          movie={movie}
          city={city}
          animal={animal}
          playerName={playerName}
          loading={loading}
          onColorChange={setColor}
          onFruitChange={setFruit}
          onObjectChange={setObject}
          onMovieChange={setMovie}
          onCityChange={setCity}
          onAnimalChange={setAnimal}
          onPlayerNameChange={setPlayerName}
          onSubmit={handleSubmitAnswers}
        />
      )}

      {(phase === 'round_ended' || phase === 'scores') && roundResults && (
        <RoundResults
          roundResults={roundResults}
          match={match}
          isCreator={isCreator}
          phase={phase}
          scores={scores}
          validations={validations}
          validating={validating}
          loading={loading}
          onScoreChange={(userId, score) => setScores((prev) => ({ ...prev, [userId]: score }))}
          onValidate={handleValidateRound}
          onSaveScores={handleUpdateScores}
          onStartRound={handleStartRound}
          onEndMatch={handleEndMatch}
        />
      )}
    </>
  );
}

export default MatchPage;

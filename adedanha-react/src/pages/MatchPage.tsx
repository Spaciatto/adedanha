import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { User, Match, Round, Answer, WSMessage, RoundResult, RankingEntry, MatchState, JoinRequest } from '../types';
import { api, connectWebSocket } from '../api';

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
  const [availablePlayers, setAvailablePlayers] = useState<{id: string; name: string}[]>([]);
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
        const filtered = (players as {id: string; name: string}[]).filter(
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
      {/* Match Info Header */}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h2>{match.name || 'Partida'}</h2>
            <span className={`status-badge ${match.status}`}>{match.status}</span>
          </div>
          <div style={{ textAlign: 'right' }}>
            <span style={{ fontSize: '0.85rem', color: '#718096', display: 'block' }}>Rodada: {match.current_round}</span>
            {!isCreator && phase !== 'finished' && (
              <button className="danger" onClick={handleLeaveMatch} disabled={loading} style={{ marginTop: '8px', fontSize: '0.8rem', padding: '6px 12px' }}>
                Abandonar
              </button>
            )}
          </div>
        </div>

        <div className="match-id-display">
          <strong>ID para convidar:</strong> {match.id}
        </div>

        <div>
          <strong style={{ fontSize: '0.9rem', color: '#718096' }}>Jogadores:</strong>
          <div className="players-list">
            {match.players?.map((p) => (
              <span
                key={p.user_id}
                className={`player-badge ${p.user_id === match.creator_id ? 'creator' : ''} ${!p.active ? 'inactive' : ''}`}
              >
                {p.user_name || p.user_id.slice(0, 8)}
                {p.user_id === match.creator_id && ' 👑'}
                {!p.active && ' (saiu)'}
              </span>
            ))}
          </div>
        </div>

        {error && <div className="error-message">{error}</div>}
      </div>

      {/* Join Requests for Creator */}
      {isCreator && joinRequests.length > 0 && phase === 'lobby' && (
        <div className="card">
          <h3>Solicitações de Entrada</h3>
          <div className="join-requests-list">
            {joinRequests.map((req) => (
              <div key={req.id} className="join-request-item">
                <span className="join-request-name">{req.user_name}</span>
                <div>
                  <button className="success" onClick={() => handleRespondJoinRequest(req.id, true)} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
                    Aceitar
                  </button>
                  <button className="danger" onClick={() => handleRespondJoinRequest(req.id, false)} style={{ fontSize: '0.85rem', padding: '6px 12px' }}>
                    Recusar
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Invite Players for Creator */}
      {isCreator && phase === 'lobby' && (
        <div className="card">
          <h3>Convidar Jogadores</h3>
          <p style={{ marginBottom: '12px', color: '#718096', fontSize: '0.9rem' }}>
            Jogadores online disponíveis (sem partida ativa):
          </p>
          {availablePlayers.length === 0 ? (
            <p style={{ color: '#a0aec0' }}>Nenhum jogador disponível no momento.</p>
          ) : (
            <div className="online-users-list">
              {availablePlayers.map((p) => (
                <div key={p.id} className="online-user-item">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <span className="online-dot"></span>
                    <span className="online-user-name">{p.name}</span>
                  </div>
                  <button
                    className={invitedPlayers[p.id] ? 'secondary' : 'primary'}
                    onClick={() => handleInvitePlayer(p.id)}
                    disabled={invitedPlayers[p.id]}
                    style={{ fontSize: '0.8rem', padding: '6px 12px', marginBottom: 0 }}
                  >
                    {invitedPlayers[p.id] ? '✓ Convidado' : 'Convidar'}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Finished Phase - Ranking */}
      {phase === 'finished' && (
        <div className="card">
          <div className="round-ended-banner" style={{ background: 'linear-gradient(135deg, #667eea, #764ba2)' }}>
            🏆 Partida Encerrada!
          </div>

          <h3>Ranking Final</h3>
          <table className="results-table">
            <thead>
              <tr>
                <th>Posição</th>
                <th>Jogador</th>
                <th>Pontuação Total</th>
              </tr>
            </thead>
            <tbody>
              {ranking.map((entry) => (
                <tr key={entry.user_id}>
                  <td>
                    <strong>
                      {entry.position === 1 && '🥇 '}
                      {entry.position === 2 && '🥈 '}
                      {entry.position === 3 && '🥉 '}
                      {entry.position}º
                    </strong>
                  </td>
                  <td><strong>{entry.user_name}</strong></td>
                  <td><strong>{entry.total_score} pts</strong></td>
                </tr>
              ))}
            </tbody>
          </table>
          <button className="primary" onClick={() => navigate('/')} style={{ marginTop: '20px' }}>
            Voltar ao Início
          </button>
        </div>
      )}

      {/* Lobby Phase */}
      {phase === 'lobby' && (
        <div className="card">
          <h3>Aguardando início da rodada...</h3>
          <p style={{ color: '#718096', marginBottom: '15px' }}>
            {isCreator
              ? 'Você é o criador da partida. Inicie a rodada quando todos estiverem prontos.'
              : 'Aguarde o criador da partida iniciar a rodada.'}
          </p>
          {isCreator && (
            <button className="primary" onClick={handleStartRound} disabled={loading}>
              {loading ? 'Iniciando...' : 'Iniciar Rodada'}
            </button>
          )}
        </div>
      )}

      {/* Playing Phase */}
      {phase === 'playing' && (
        <div className="card">
          <div className="letter-display">{letter}</div>

          <div className={`timer ${secondsRemaining <= 10 ? 'warning' : 'normal'}`}>
            ⏱️ {secondsRemaining}s
          </div>

          {!answersSubmitted ? (
            <>
              <div className="field-group">
                <label>🎨 Cor</label>
                <input
                  className="uppercase"
                  type="text"
                  value={color}
                  onChange={(e) => setColor(e.target.value.toUpperCase())}
                  placeholder={`Cor com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>🍎 Fruta</label>
                <input
                  className="uppercase"
                  type="text"
                  value={fruit}
                  onChange={(e) => setFruit(e.target.value.toUpperCase())}
                  placeholder={`Fruta com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>📦 Objeto</label>
                <input
                  className="uppercase"
                  type="text"
                  value={object}
                  onChange={(e) => setObject(e.target.value.toUpperCase())}
                  placeholder={`Objeto com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>🎬 Filme</label>
                <input
                  className="uppercase"
                  type="text"
                  value={movie}
                  onChange={(e) => setMovie(e.target.value.toUpperCase())}
                  placeholder={`Filme com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>🏙️ Cidade</label>
                <input
                  className="uppercase"
                  type="text"
                  value={city}
                  onChange={(e) => setCity(e.target.value.toUpperCase())}
                  placeholder={`Cidade com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>🐾 Animal</label>
                <input
                  className="uppercase"
                  type="text"
                  value={animal}
                  onChange={(e) => setAnimal(e.target.value.toUpperCase())}
                  placeholder={`Animal com a letra ${letter}`}
                />
              </div>
              <div className="field-group">
                <label>👤 Nome</label>
                <input
                  className="uppercase"
                  type="text"
                  value={playerName}
                  onChange={(e) => setPlayerName(e.target.value.toUpperCase())}
                  placeholder={`Nome com a letra ${letter}`}
                />
              </div>
              <button className="success" onClick={handleSubmitAnswers} disabled={loading}>
                {loading ? 'Enviando...' : 'Enviar Respostas'}
              </button>
            </>
          ) : (
            <div className="success-message">
              ✅ Respostas enviadas! Aguardando fim da rodada...
            </div>
          )}
        </div>
      )}

      {/* Round Ended Phase */}
      {(phase === 'round_ended' || phase === 'scores') && roundResults && (
        <div className="card">
          <div className="round-ended-banner">
            ⏰ Rodada Encerrada! Letra: {roundResults.letter}
          </div>

          <h3>Respostas dos Jogadores</h3>
          <div style={{ overflowX: 'auto' }}>
            <table className="results-table">
              <thead>
                <tr>
                  <th>Jogador</th>
                  <th>Cor</th>
                  <th>Fruta</th>
                  <th>Objeto</th>
                  <th>Filme</th>
                  <th>Cidade</th>
                  <th>Animal</th>
                  <th>Nome</th>
                  {isCreator && <th>Score</th>}
                </tr>
              </thead>
              <tbody>
                {roundResults.answers.map((answer) => (
                  <tr key={answer.user_id}>
                    <td><strong>{getPlayerName(answer.user_id)}</strong></td>
                    <td>{answer.color || '-'}{getValidationIcon(answer.user_id, 'color')}</td>
                    <td>{answer.fruit || '-'}{getValidationIcon(answer.user_id, 'fruit')}</td>
                    <td>{answer.object || '-'}{getValidationIcon(answer.user_id, 'object')}</td>
                    <td>{answer.movie || '-'}{getValidationIcon(answer.user_id, 'movie')}</td>
                    <td>{answer.city || '-'}{getValidationIcon(answer.user_id, 'city')}</td>
                    <td>{answer.animal || '-'}{getValidationIcon(answer.user_id, 'animal')}</td>
                    <td>{answer.name || '-'}{getValidationIcon(answer.user_id, 'name')}</td>
                    {isCreator && (
                      <td>
                        <input
                          className="score-input"
                          type="number"
                          min="0"
                          value={scores[answer.user_id] || 0}
                          onChange={(e) =>
                            setScores((prev) => ({
                              ...prev,
                              [answer.user_id]: parseInt(e.target.value) || 0,
                            }))
                          }
                        />
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {isCreator && phase === 'round_ended' && (
            <div style={{ marginTop: '20px' }}>
              <button className="secondary" onClick={handleValidateRound} disabled={validating} style={{ marginBottom: '12px', display: 'block' }}>
                {validating ? '🔍 Validando...' : '🔍 Validar Respostas (Internet)'}
              </button>
              {Object.keys(validations).length > 0 && (
                <p style={{ fontSize: '0.8rem', color: '#718096', marginBottom: '12px' }}>
                  ✅ Válida (10pts) &nbsp; ⚠️ Incerta (5pts) &nbsp; ❌ Inválida (0pts) — Scores pré-preenchidos como sugestão
                </p>
              )}
              <button className="success" onClick={handleUpdateScores} disabled={loading}>
                {loading ? 'Salvando...' : 'Salvar Scores'}
              </button>
              <button className="primary" onClick={handleStartRound} disabled={loading} style={{ marginLeft: '10px' }}>
                {loading ? 'Iniciando...' : 'Iniciar Nova Rodada'}
              </button>
              <button className="danger" onClick={handleEndMatch} disabled={loading} style={{ marginLeft: '10px' }}>
                {loading ? 'Encerrando...' : 'Encerrar Partida'}
              </button>
            </div>
          )}

          {isCreator && phase === 'scores' && (
            <div style={{ marginTop: '20px' }}>
              <div className="success-message">✅ Scores salvos!</div>
              <button className="primary" onClick={handleStartRound} disabled={loading}>
                {loading ? 'Iniciando...' : 'Iniciar Nova Rodada'}
              </button>
              <button className="danger" onClick={handleEndMatch} disabled={loading} style={{ marginLeft: '10px' }}>
                {loading ? 'Encerrando...' : 'Encerrar Partida'}
              </button>
            </div>
          )}

          {!isCreator && (
            <div style={{ marginTop: '15px' }}>
              <p style={{ color: '#718096' }}>
                Aguardando o criador da partida avaliar as respostas e iniciar a próxima rodada...
              </p>
            </div>
          )}
        </div>
      )}
    </>
  );
}

export default MatchPage;

import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { User, Match, OnlineUser, OpenMatch } from '../types';
import { api } from '../api';

interface HomeProps {
  user: User;
}

function Home({ user }: HomeProps) {
  const navigate = useNavigate();
  const [matchName, setMatchName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [openMatches, setOpenMatches] = useState<OpenMatch[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [requestSent, setRequestSent] = useState<Record<string, boolean>>({});

  useEffect(() => {
    loadOnlineUsers();
    loadOpenMatches();
    const interval = setInterval(() => {
      loadOnlineUsers();
      loadOpenMatches();
    }, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadOnlineUsers = async () => {
    try {
      setLoadingUsers(true);
      const users = await api.getOnlineUsers() as OnlineUser[];
      setOnlineUsers(users.filter((u) => u.id !== user.id));
    } catch {
      // Silently fail
    } finally {
      setLoadingUsers(false);
    }
  };

  const loadOpenMatches = async () => {
    try {
      const matches = await api.listOpenMatches() as OpenMatch[];
      setOpenMatches(matches);
    } catch {
      // Silently fail
    }
  };

  const handleCreateMatch = async () => {
    setError('');

    if (!matchName.trim()) {
      setError('Insira um nome para a partida');
      return;
    }

    setLoading(true);
    try {
      const match = await api.createMatch(user.id, matchName.trim()) as Match;
      navigate(`/match/${match.id}`);
    } catch (err: any) {
      setError(err.message || 'Erro ao criar partida');
    } finally {
      setLoading(false);
    }
  };

  const handleRequestJoin = async (matchId: string) => {
    setError('');
    try {
      await api.requestJoinMatch(matchId, user.id);
      setRequestSent((prev) => ({ ...prev, [matchId]: true }));
    } catch (err: any) {
      setError(err.message || 'Erro ao solicitar entrada');
    }
  };

  return (
    <>
      <div className="card">
        <h2>Criar Nova Partida</h2>
        <p style={{ marginBottom: '15px', color: '#718096' }}>
          Crie uma partida e convide seus amigos compartilhando o ID.
        </p>
        {error && <div className="error-message">{error}</div>}
        <div className="field-group">
          <label>Nome da Partida</label>
          <input
            type="text"
            value={matchName}
            onChange={(e) => setMatchName(e.target.value)}
            placeholder="Ex: Adedanha da Galera"
          />
        </div>
        <button className="primary" onClick={handleCreateMatch} disabled={loading}>
          {loading ? 'Criando...' : 'Criar Partida'}
        </button>
      </div>

      <div className="card">
        <h2>Partidas Abertas</h2>
        <p style={{ marginBottom: '15px', color: '#718096' }}>
          Partidas aguardando jogadores. Solicite entrada ao criador.
        </p>
        {openMatches.length === 0 ? (
          <p style={{ color: '#a0aec0' }}>Nenhuma partida aberta no momento.</p>
        ) : (
          <div className="open-matches-list">
            {openMatches.map((m) => (
              <div key={m.id} className="open-match-item">
                <div className="open-match-info">
                  <strong>{m.name}</strong>
                  <span className="open-match-meta">
                    Criador: {m.creator_name} • {m.player_count} jogador(es)
                  </span>
                </div>
                <button
                  className={requestSent[m.id] ? 'secondary' : 'primary'}
                  onClick={() => handleRequestJoin(m.id)}
                  disabled={requestSent[m.id]}
                  style={{ marginBottom: 0 }}
                >
                  {requestSent[m.id] ? '✓ Solicitado' : 'Solicitar Entrada'}
                </button>
              </div>
            ))}
          </div>
        )}
        <button className="secondary" onClick={loadOpenMatches} style={{ marginTop: '10px' }}>
          🔄 Atualizar
        </button>
      </div>

      <div className="card">
        <h2>Usuários Online</h2>
        <p style={{ marginBottom: '15px', color: '#718096' }}>
          Usuários conectados que você pode convidar para uma partida.
        </p>
        {loadingUsers && onlineUsers.length === 0 ? (
          <p style={{ color: '#a0aec0' }}>Carregando...</p>
        ) : onlineUsers.length === 0 ? (
          <p style={{ color: '#a0aec0' }}>Nenhum outro usuário online no momento.</p>
        ) : (
          <div className="online-users-list">
            {onlineUsers.map((u) => (
              <div key={u.id} className="online-user-item">
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <span className="online-dot"></span>
                  {u.avatar ? (
                    <img src={u.avatar} alt="" className="avatar-small" />
                  ) : (
                    <span className="avatar-small-placeholder">👤</span>
                  )}
                  <span className="online-user-name">{u.name}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="card">
        <h2>Meu Perfil</h2>
        <p style={{ color: '#718096' }}>
          <strong>Nome:</strong> {user.name}<br />
          <strong>Email:</strong> {user.email}
        </p>
        <button className="secondary" onClick={() => navigate('/profile')} style={{ marginTop: '10px' }}>
          Editar Perfil
        </button>
      </div>
    </>
  );
}

export default Home;

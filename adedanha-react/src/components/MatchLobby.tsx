import { JoinRequest } from '../types';

interface MatchLobbyProps {
  isCreator: boolean;
  loading: boolean;
  joinRequests: JoinRequest[];
  availablePlayers: { id: string; name: string; avatar: string }[];
  invitedPlayers: Record<string, boolean>;
  onStartRound: () => void;
  onRespondRequest: (requestId: string, accepted: boolean) => void;
  onInvitePlayer: (playerId: string) => void;
}

function MatchLobby({
  isCreator, loading, joinRequests, availablePlayers, invitedPlayers,
  onStartRound, onRespondRequest, onInvitePlayer,
}: MatchLobbyProps) {
  return (
    <>
      {/* Join Requests */}
      {isCreator && joinRequests.length > 0 && (
        <div className="card">
          <h3>Solicitações de Entrada</h3>
          <div className="join-requests-list">
            {joinRequests.map((req) => (
              <div key={req.id} className="join-request-item">
                <span className="join-request-name">{req.user_name}</span>
                <div>
                  <button className="success" onClick={() => onRespondRequest(req.id, true)}
                    style={{ fontSize: '0.85rem', padding: '6px 12px' }}>Aceitar</button>
                  <button className="danger" onClick={() => onRespondRequest(req.id, false)}
                    style={{ fontSize: '0.85rem', padding: '6px 12px' }}>Recusar</button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Invite Players */}
      {isCreator && (
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
                    {p.avatar ? (
                      <img src={p.avatar} alt="" className="avatar-small" />
                    ) : (
                      <span className="avatar-small-placeholder">👤</span>
                    )}
                    <span className="online-user-name">{p.name}</span>
                  </div>
                  <button
                    className={invitedPlayers[p.id] ? 'secondary' : 'primary'}
                    onClick={() => onInvitePlayer(p.id)}
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

      {/* Waiting message */}
      <div className="card">
        <h3>Aguardando início da rodada...</h3>
        <p style={{ color: '#718096', marginBottom: '15px' }}>
          {isCreator
            ? 'Você é o criador da partida. Inicie a rodada quando todos estiverem prontos.'
            : 'Aguarde o criador da partida iniciar a rodada.'}
        </p>
        {isCreator && (
          <button className="primary" onClick={onStartRound} disabled={loading}>
            {loading ? 'Iniciando...' : 'Iniciar Rodada'}
          </button>
        )}
      </div>
    </>
  );
}

export default MatchLobby;

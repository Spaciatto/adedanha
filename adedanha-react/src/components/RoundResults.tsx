import { RoundResult, Match } from '../types';

interface RoundResultsProps {
  roundResults: RoundResult;
  match: Match;
  isCreator: boolean;
  phase: string;
  scores: Record<string, number>;
  validations: Record<string, Record<string, string>>;
  validating: boolean;
  loading: boolean;
  onScoreChange: (userId: string, score: number) => void;
  onValidate: () => void;
  onSaveScores: () => void;
  onStartRound: () => void;
  onEndMatch: () => void;
}

function RoundResults({
  roundResults, match, isCreator, phase, scores, validations, validating, loading,
  onScoreChange, onValidate, onSaveScores, onStartRound, onEndMatch,
}: RoundResultsProps) {
  const getPlayerName = (userId: string): string => {
    const player = match.players?.find((p) => p.user_id === userId);
    return player?.user_name || userId.slice(0, 8);
  };

  const getValidationIcon = (userId: string, field: string): string => {
    const v = validations[userId]?.[field];
    if (!v) return '';
    if (v === 'valid') return ' ✅';
    if (v === 'invalid') return ' ❌';
    return ' ⚠️';
  };

  return (
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
                      onChange={(e) => onScoreChange(answer.user_id, parseInt(e.target.value) || 0)}
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
          <button className="secondary" onClick={onValidate} disabled={validating}
            style={{ marginBottom: '12px', display: 'block' }}>
            {validating ? '🔍 Validando...' : '🔍 Validar Respostas (Internet)'}
          </button>
          {Object.keys(validations).length > 0 && (
            <p style={{ fontSize: '0.8rem', color: '#718096', marginBottom: '12px' }}>
              ✅ Válida (10pts) &nbsp; ⚠️ Incerta (5pts) &nbsp; ❌ Inválida (0pts)
            </p>
          )}
          <button className="success" onClick={onSaveScores} disabled={loading}>
            {loading ? 'Salvando...' : 'Salvar Scores'}
          </button>
          <button className="primary" onClick={onStartRound} disabled={loading} style={{ marginLeft: '10px' }}>
            {loading ? 'Iniciando...' : 'Iniciar Nova Rodada'}
          </button>
          <button className="danger" onClick={onEndMatch} disabled={loading} style={{ marginLeft: '10px' }}>
            {loading ? 'Encerrando...' : 'Encerrar Partida'}
          </button>
        </div>
      )}

      {isCreator && phase === 'scores' && (
        <div style={{ marginTop: '20px' }}>
          <div className="success-message">✅ Scores salvos!</div>
          <button className="primary" onClick={onStartRound} disabled={loading}>
            {loading ? 'Iniciando...' : 'Iniciar Nova Rodada'}
          </button>
          <button className="danger" onClick={onEndMatch} disabled={loading} style={{ marginLeft: '10px' }}>
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
  );
}

export default RoundResults;

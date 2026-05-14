interface RoundPlayingProps {
  letter: string;
  secondsRemaining: number;
  answersSubmitted: boolean;
  color: string;
  fruit: string;
  object: string;
  movie: string;
  city: string;
  animal: string;
  playerName: string;
  loading: boolean;
  onColorChange: (v: string) => void;
  onFruitChange: (v: string) => void;
  onObjectChange: (v: string) => void;
  onMovieChange: (v: string) => void;
  onCityChange: (v: string) => void;
  onAnimalChange: (v: string) => void;
  onPlayerNameChange: (v: string) => void;
  onSubmit: () => void;
}

function RoundPlaying({
  letter, secondsRemaining, answersSubmitted,
  color, fruit, object, movie, city, animal, playerName,
  loading,
  onColorChange, onFruitChange, onObjectChange, onMovieChange,
  onCityChange, onAnimalChange, onPlayerNameChange, onSubmit,
}: RoundPlayingProps) {
  return (
    <div className="card">
      <div className="letter-display">{letter}</div>

      <div className={`timer ${secondsRemaining <= 10 ? 'warning' : 'normal'}`}>
        ⏱️ {secondsRemaining}s
      </div>

      {!answersSubmitted ? (
        <>
          <div className="field-group">
            <label>🎨 Cor</label>
            <input className="uppercase" type="text" value={color}
              onChange={(e) => onColorChange(e.target.value.toUpperCase())}
              placeholder={`Cor com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>🍎 Fruta</label>
            <input className="uppercase" type="text" value={fruit}
              onChange={(e) => onFruitChange(e.target.value.toUpperCase())}
              placeholder={`Fruta com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>📦 Objeto</label>
            <input className="uppercase" type="text" value={object}
              onChange={(e) => onObjectChange(e.target.value.toUpperCase())}
              placeholder={`Objeto com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>🎬 Filme</label>
            <input className="uppercase" type="text" value={movie}
              onChange={(e) => onMovieChange(e.target.value.toUpperCase())}
              placeholder={`Filme com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>🏙️ Cidade</label>
            <input className="uppercase" type="text" value={city}
              onChange={(e) => onCityChange(e.target.value.toUpperCase())}
              placeholder={`Cidade com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>🐾 Animal</label>
            <input className="uppercase" type="text" value={animal}
              onChange={(e) => onAnimalChange(e.target.value.toUpperCase())}
              placeholder={`Animal com a letra ${letter}`} />
          </div>
          <div className="field-group">
            <label>👤 Nome</label>
            <input className="uppercase" type="text" value={playerName}
              onChange={(e) => onPlayerNameChange(e.target.value.toUpperCase())}
              placeholder={`Nome com a letra ${letter}`} />
          </div>
          <button className="success" onClick={onSubmit} disabled={loading}>
            {loading ? 'Enviando...' : 'Enviar Respostas'}
          </button>
        </>
      ) : (
        <div className="success-message">
          ✅ Respostas enviadas! Aguardando fim da rodada...
        </div>
      )}
    </div>
  );
}

export default RoundPlaying;

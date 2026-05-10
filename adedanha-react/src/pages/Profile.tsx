import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { User } from '../types';
import { api } from '../api';

interface ProfileProps {
  user: User;
  onUpdate: (user: User) => void;
}

function Profile({ user, onUpdate }: ProfileProps) {
  const navigate = useNavigate();
  const [name, setName] = useState(user.name);
  const [email, setEmail] = useState(user.email);
  const [avatar, setAvatar] = useState(user.avatar || '');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!name.trim() || !email.trim()) {
      setError('Nome e email são obrigatórios');
      return;
    }

    setLoading(true);
    try {
      const updated = await api.updateUser(user.id, name.trim(), email.trim().toLowerCase()) as User;
      updated.avatar = avatar;
      onUpdate(updated);
      setSuccess('Perfil atualizado com sucesso!');
    } catch (err: any) {
      setError(err.message || 'Erro ao atualizar perfil');
    } finally {
      setLoading(false);
    }
  };

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      setError('Selecione um arquivo de imagem');
      return;
    }

    if (file.size > 500000) {
      setError('Imagem muito grande. Máximo 500KB.');
      return;
    }

    const reader = new FileReader();
    reader.onload = async () => {
      const base64 = reader.result as string;
      setAvatar(base64);
      setError('');
      setLoading(true);
      try {
        await api.uploadAvatar(user.id, base64);
        const updatedUser = { ...user, avatar: base64 };
        onUpdate(updatedUser);
        setSuccess('Foto atualizada!');
      } catch (err: any) {
        setError(err.message || 'Erro ao enviar foto');
      } finally {
        setLoading(false);
      }
    };
    reader.readAsDataURL(file);
  };

  const handleRemoveAvatar = async () => {
    setLoading(true);
    setError('');
    try {
      await api.uploadAvatar(user.id, '');
      setAvatar('');
      const updatedUser = { ...user, avatar: '' };
      onUpdate(updatedUser);
      setSuccess('Foto removida!');
    } catch (err: any) {
      setError(err.message || 'Erro ao remover foto');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="card">
      <h2>Editar Perfil</h2>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      {/* Avatar section */}
      <div className="avatar-section">
        <div className="avatar-preview" onClick={() => fileInputRef.current?.click()}>
          {avatar ? (
            <img src={avatar} alt="Avatar" className="avatar-image" />
          ) : (
            <div className="avatar-placeholder">
              <span>📷</span>
              <small>Adicionar foto</small>
            </div>
          )}
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleAvatarChange}
          style={{ display: 'none' }}
        />
        <div className="avatar-actions">
          <button type="button" className="secondary" onClick={() => fileInputRef.current?.click()} style={{ fontSize: '0.85rem' }}>
            {avatar ? 'Trocar foto' : 'Escolher foto'}
          </button>
          {avatar && (
            <button type="button" className="danger" onClick={handleRemoveAvatar} style={{ fontSize: '0.85rem' }}>
              Remover
            </button>
          )}
        </div>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="field-group">
          <label>Nome</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Seu nome"
          />
        </div>
        <div className="field-group">
          <label>Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="seu@email.com"
          />
        </div>
        <button type="submit" className="primary" disabled={loading}>
          {loading ? 'Salvando...' : 'Salvar Alterações'}
        </button>
        <button type="button" className="secondary" onClick={() => navigate('/')}>
          Voltar
        </button>
      </form>
    </div>
  );
}

export default Profile;

const h = React.createElement;
const { useEffect, useMemo, useState } = React;

const STORAGE_KEY = 'auth-jwt-session';
const API_URL = `${window.location.origin}/api`;

function request(path, options) {
  return fetch(`${API_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options?.headers || {}),
    },
    ...options,
  }).then(async (response) => {
    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      throw new Error(data.message || 'Erro na requisicao.');
    }

    return data;
  });
}

function navigate(path) {
  window.location.hash = path;
}

function currentRoute() {
  return window.location.hash.replace(/^#/, '') || '/login';
}

function App() {
  const [route, setRoute] = useState(currentRoute());
  const [token, setToken] = useState(localStorage.getItem(STORAGE_KEY) || '');
  const [user, setUser] = useState(null);
  const [booting, setBooting] = useState(true);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({ name: '', email: '', password: '' });

  useEffect(() => {
    const onHashChange = () => setRoute(currentRoute());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    if (!token) {
      setBooting(false);
      return;
    }

    request('/profile', {
      method: 'GET',
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((data) => {
        setUser(data.user);
        setBooting(false);
        if (route === '/' || route === '/login' || route === '/cadastro') {
          navigate('/perfil');
        }
      })
      .catch(() => {
        localStorage.removeItem(STORAGE_KEY);
        setToken('');
        setUser(null);
        setBooting(false);
      });
  }, [route, token]);

  const auth = useMemo(
    () => ({
      isAuthenticated: Boolean(token && user),
      logout() {
        localStorage.removeItem(STORAGE_KEY);
        setToken('');
        setUser(null);
        setForm({ name: '', email: '', password: '' });
        navigate('/login');
      },
    }),
    [token, user]
  );

  function updateField(field, value) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  function submitLogin(event) {
    event.preventDefault();
    setError('');
    setLoading(true);

    request('/login', {
      method: 'POST',
      body: JSON.stringify({ email: form.email, password: form.password }),
    })
      .then((data) => {
        localStorage.setItem(STORAGE_KEY, data.token);
        setToken(data.token);
        setUser(data.user);
        setForm({ name: '', email: '', password: '' });
        navigate('/perfil');
      })
      .catch((submitError) => setError(submitError.message))
      .finally(() => setLoading(false));
  }

  function submitRegister(event) {
    event.preventDefault();
    setError('');
    setLoading(true);

    request('/register', {
      method: 'POST',
      body: JSON.stringify(form),
    })
      .then((data) => {
        localStorage.setItem(STORAGE_KEY, data.token);
        setToken(data.token);
        setUser(data.user);
        setForm({ name: '', email: '', password: '' });
        navigate('/perfil');
      })
      .catch((submitError) => setError(submitError.message))
      .finally(() => setLoading(false));
  }

  function header() {
    return h(
      'header',
      { className: 'topbar' },
      h(
        'div',
        null,
        h('p', { className: 'eyebrow' }, 'React + Go + MySQL'),
        h('h1', null, 'Autenticacao JWT segura')
      ),
      h(
        'nav',
        null,
        auth.isAuthenticated
          ? h(
              'div',
              { className: 'nav-actions' },
              h('span', null, user?.name || ''),
              h('button', { type: 'button', className: 'ghost-button', onClick: auth.logout }, 'Sair')
            )
          : h(
              'div',
              { className: 'nav-actions' },
              h('a', { href: '#/login' }, 'Login'),
              h('a', { href: '#/cadastro' }, 'Cadastro')
            )
      )
    );
  }

  function authForm(isRegister) {
    return h('section', { className: 'card auth-card' }, [
      h('p', { className: 'eyebrow', key: 'eyebrow' }, isRegister ? 'Cadastro rapido' : 'Acesso protegido'),
      h('h2', { key: 'title' }, isRegister ? 'Criar conta' : 'Entrar na conta'),
      h(
        'p',
        { className: 'muted', key: 'desc' },
        isRegister
          ? 'O backend em Go cria o usuario, gera o token JWT e libera o acesso.'
          : 'Faca login para acessar a area protegida e carregar o seu perfil.'
      ),
      h(
        'form',
        { className: 'form-grid', onSubmit: isRegister ? submitRegister : submitLogin, key: 'form' },
        [
          isRegister
            ? h(
                'label',
                { key: 'name' },
                'Nome',
                h('input', {
                  type: 'text',
                  value: form.name,
                  placeholder: 'Seu nome',
                  onChange: (event) => updateField('name', event.target.value),
                  required: true,
                })
              )
            : null,
          h(
            'label',
            { key: 'email' },
            'Email',
            h('input', {
              type: 'email',
              value: form.email,
              placeholder: 'voce@empresa.com',
              onChange: (event) => updateField('email', event.target.value),
              required: true,
            })
          ),
          h(
            'label',
            { key: 'password' },
            'Senha',
            h('input', {
              type: 'password',
              value: form.password,
              placeholder: 'Min. 6 caracteres',
              minLength: 6,
              onChange: (event) => updateField('password', event.target.value),
              required: true,
            })
          ),
          error ? h('p', { className: 'error-text', key: 'error' }, error) : null,
          h(
            'button',
            { type: 'submit', disabled: loading, key: 'button' },
            loading ? (isRegister ? 'Criando...' : 'Entrando...') : isRegister ? 'Criar conta' : 'Entrar'
          ),
        ].filter(Boolean)
      ),
      h(
        'p',
        { className: 'muted', key: 'link' },
        isRegister ? 'Ja possui conta? ' : 'Ainda nao tem conta? ',
        h('a', { href: isRegister ? '#/login' : '#/cadastro' }, isRegister ? 'Fazer login' : 'Criar cadastro')
      ),
    ]);
  }

  function dashboard() {
    return h('section', { className: 'dashboard-grid' }, [
      h('article', { className: 'card hero-card', key: 'hero' }, [
        h('p', { className: 'eyebrow', key: 'eye' }, 'Area autenticada'),
        h('h2', { key: 'title' }, `Bem-vindo, ${user?.name || ''}`),
        h(
          'p',
          { className: 'muted', key: 'desc' },
          'Esta pagina so abre quando o token JWT e enviado no header Authorization.'
        ),
        h('div', { className: 'badge-row', key: 'badges' }, [
          h('span', { className: 'badge', key: 'a' }, 'JWT ativo'),
          h('span', { className: 'badge badge-secondary', key: 'b' }, 'Sessao autenticada'),
        ]),
      ]),
      h('article', { className: 'card info-card', key: 'profile' }, [
        h('h3', { key: 'title' }, 'Dados do usuario'),
        h('dl', { className: 'profile-list', key: 'list' }, [
          h('div', { key: 'name' }, [h('dt', null, 'Nome'), h('dd', null, user?.name || '')]),
          h('div', { key: 'email' }, [h('dt', null, 'Email'), h('dd', null, user?.email || '')]),
          h('div', { key: 'created' }, [h('dt', null, 'Criado em'), h('dd', null, user?.created_at || '')]),
        ]),
      ]),
      h('article', { className: 'card info-card', key: 'security' }, [
        h('h3', { key: 'title' }, 'Resumo de seguranca'),
        h('ul', { className: 'feature-list', key: 'list' }, [
          h('li', { key: 'a' }, 'Senha armazenada em hash no backend Go.'),
          h('li', { key: 'b' }, 'Token JWT assinado com HMAC SHA-256.'),
          h('li', { key: 'c' }, 'Rota /api/profile protegida por validacao do token.'),
          h('li', { key: 'd' }, 'Sessao salva no navegador para manter login.'),
        ]),
      ]),
      h('article', { className: 'card token-card', key: 'token' }, [
        h('h3', { key: 'title' }, 'Token atual'),
        h('code', { key: 'code' }, token || 'Sem token'),
      ]),
    ]);
  }

  function view() {
    if (booting) {
      return h('div', { className: 'screen-center' }, 'Carregando sessao...');
    }

    if (!auth.isAuthenticated && route === '/perfil') {
      navigate('/login');
      return h('div', { className: 'screen-center' }, 'Redirecionando...');
    }

    if (auth.isAuthenticated && (route === '/' || route === '/login' || route === '/cadastro')) {
      navigate('/perfil');
      return h('div', { className: 'screen-center' }, 'Redirecionando...');
    }

    if (route === '/cadastro') {
      return authForm(true);
    }

    if (route === '/perfil') {
      return dashboard();
    }

    return authForm(false);
  }

  return h('div', { className: 'app-shell' }, [header(), h('main', { className: 'content' }, view())]);
}

ReactDOM.createRoot(document.getElementById('root')).render(h(App));

# Sistema de Login + Autenticacao com JWT

Projeto full-stack com:

- Frontend em React
- Backend em Go
- Banco MySQL
- Arquitetura cliente-servidor
- Login, cadastro, JWT, rotas protegidas e perfil do usuario

## Estrutura

```text
backend/   API Go + JWT + MySQL
frontend/  Interface React servida pelo Go
```

## Banco

Crie o banco e a tabela com:

```sql
SOURCE backend/schema.sql;
```

## Variaveis de ambiente

Use [backend/.env.example](backend/.env.example) como referencia:

```bash
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=auth_jwt
DB_USER=root
DB_PASS=
APP_PORT=8080
JWT_SECRET=troque-esta-chave-em-producao
JWT_TTL=3600
```

## Como rodar

1. Instale as dependencias Go:

```bash
go mod tidy
```

2. Inicie o servidor:

```bash
go run ./backend
```

3. Abra no navegador:

```text
http://localhost:8080
```

Tambem funciona em:

```text
http://localhost:8080/frontend/index.html
```

## Endpoints

- `GET /api/health`
- `POST /api/register`
- `POST /api/login`
- `GET /api/profile`

## Seguranca aplicada

- Senhas com `bcrypt`
- JWT assinado com HS256
- Expiracao do token
- Header `Authorization: Bearer <token>`
- Protecao de rota no backend e no frontend

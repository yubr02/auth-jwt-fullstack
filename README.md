# Autenticação Full Stack com JWT

Aplicação full stack com cadastro, login e proteção de rotas usando tokens JWT.

## Demonstração

[![Assistir à demonstração](https://img.youtube.com/vi/1GZxRmRjBKo/maxresdefault.jpg)](https://www.youtube.com/watch?v=1GZxRmRjBKo)

## Funcionalidades

- Cadastro de usuários
- Login com geração de token JWT
- Senhas protegidas com bcrypt
- Perfil autenticado
- Rotas privadas
- Integração entre front-end e API

## Tecnologias

| Camada | Tecnologias |
| --- | --- |
| Front-end | React |
| Back-end | Go |
| Banco de dados | MySQL |
| Segurança | JWT, bcrypt |

## Como executar

### Back-end

```bash
git clone https://github.com/yubr02/SLAUTJWT.git
cd SLAUTJWT
go mod tidy
go run ./backend
```

Configure as variáveis de ambiente e a conexão com o MySQL antes de iniciar.

### Front-end

Entre na pasta do front-end indicada no projeto e execute:

```bash
npm install
npm run dev
```

## Objetivo técnico

Este projeto demonstra autenticação stateless, separação entre cliente e servidor, hash de senhas e autorização de rotas — componentes reutilizáveis em sistemas maiores.

## Autor

Desenvolvido por [Matheus Santos Carvalho](https://github.com/yubr02).

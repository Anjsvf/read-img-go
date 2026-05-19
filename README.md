# Read Img Go

API REST em Go para leitura automatizada de medidores de água e gás utilizando 
Inteligência Artificial. O usuário envia uma foto do medidor — via arquivo direto 
ou string base64 — e o sistema converte, processa e extrai o valor automaticamente.

## Tecnologias Utilizadas

- **Linguagem:** Go 1.22+
- **Framework Web:** [Gin](https://gin-gonic.com/)
- **Banco de Dados:** MongoDB Atlas (via `mongo-driver`)
- **Autenticação:** JWT (JSON Web Tokens) + bcrypt
- **Integração de IA:** Google Gemini Flash API
- **Armazenamento de Imagens:** Cloudinary API

## Estrutura do Projeto

Arquitetura em camadas inspirada em Clean Architecture:

- `config/`: Carregamento e validação de variáveis de ambiente
- `domain/`: Entidades, DTOs, tipos e contratos do domínio
- `handler/`: Controllers HTTP — recebem requisições e delegam aos services
- `middleware/`: Rate limiting, CORS, JWT, timeout e limite de body
- `repository/`: Acesso ao MongoDB — queries, índices e persistência
- `router/`: Configuração de rotas e injeção de dependências
- `service/`: Regras de negócio e integração com Gemini e Cloudinary

## Principais Funcionalidades

- **Autenticação completa:** Registro e login com JWT assinado e senha 
  criptografada com bcrypt. Cada usuário recebe um `customer_code` único.

- **Upload duplo:** O endpoint `/upload` aceita tanto arquivo de imagem direto 
  via `multipart/form-data` quanto string base64 via JSON. Quando o arquivo é 
  enviado diretamente, o backend converte para base64 internamente antes de 
  processar — o cliente nunca precisa fazer essa conversão.

- **Pipeline de processamento da imagem:**
  1. Recebe o arquivo ou base64
  2. Converte para base64 em memória (se necessário)
  3. Envia ao Google Gemini Vision para extração do valor numérico
  4. Faz upload da imagem ao Cloudinary e obtém a URL pública
  5. Persiste no MongoDB apenas a URL — o base64 nunca é gravado em disco ou banco

- **Validação mensal:** Impede duplicidade de leitura do mesmo tipo (WATER/GAS) 
  no mesmo mês para o mesmo usuário.

- **Confirmação de leitura:** Rota dedicada para confirmar ou corrigir 
  manualmente o valor extraído pela IA, sem nova consulta ao Gemini.

- **Listagem por usuário:** O `customer_code` é extraído do token JWT — 
  cada usuário visualiza apenas suas próprias leituras.

## Variáveis de Ambiente

```env
PORT=8080
MONGO_URI=mongodb+srv://...
MONGO_DB=shopper
MONGO_COLLECTION=measures
USERS_COLLECTION=users
GEMINI_API_KEY=...
CLOUDINARY_URL=cloudinary://...
CLOUDINARY_CLOUD_NAME=...
CLOUDINARY_API_KEY=...
CLOUDINARY_API_SECRET=...
CLOUDINARY_FOLDER=
JWT_SECRET=...
```

## Rotas da API

### Públicas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | Status da aplicação |
| POST | `/auth/register` | Cadastro de usuário |
| POST | `/auth/login` | Login e geração do JWT |

### Protegidas — `Authorization: Bearer <token>`

| Método | Rota | Descrição |
|---|---|---|
| POST | `/upload` | Envia foto, extrai valor via IA e salva |
| PATCH | `/confirm` | Confirma ou corrige o valor lido |
| GET | `/measures/list` | Lista leituras do usuário autenticado |

#### POST /upload — aceita dois formatos

**Multipart (arquivo direto):**

image         → arquivo .jpg/.png
measure_type  → WATER ou GAS
measure_datetime → 2024-08-01T10:00:00Z

**JSON (base64):**
```json
{
  "image": "<base64>",
  "measure_type": "WATER",
  "measure_datetime": "2024-08-01T10:00:00Z"
}
```

## Segurança

| Proteção | Detalhe |
|---|---|
| Rate Limiting | 20 req/s por IP com limpeza automática de visitantes inativos |
| CORS | Restrito à origem do frontend em produção |
| Max Body Size | Limite de 20MB por requisição |
| Security Headers | X-Frame-Options, XSS-Protection, CSP, HSTS |
| Timeout | Requisições abortadas após 60 segundos |
| JWT | Tokens assinados com HS256, expiração de 24h |
| bcrypt | Senhas com custo padrão — nunca armazenadas em texto puro |
| Base64 | Nunca persistido — processado em memória e descartado |

## Como rodar localmente

```bash
git clone https://github.com/Anjsvf/read-img-go
cd read-img-go
cp .env.example .env  # preencha as variáveis
go mod tidy
go run cmd/main.go
```

A API estará disponível em `http://localhost:8080`.
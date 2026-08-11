# Google Calendar MCP

Servidor MCP local em Go para autenticar no Google Calendar, listar eventos e criar eventos únicos ou recorrentes.

## Configuração

1. Habilite a Google Calendar API no Google Cloud.
2. Crie um cliente OAuth do tipo **Desktop app**.
3. Baixe a credencial para `google_credentials.json` na raiz do projeto.

O arquivo precisa ter uma chave `installed`. Credenciais com chave `web` não
servem para o callback loopback dinâmico usado por este servidor.

## Executar os testes

```bash
go test ./...
```

## Compilar

```bash
go build -o bin/mcp-calendar ./cmd/mcp-calendar
```

## Adicionar ao Codex

Use caminhos absolutos para que a autenticação não dependa do diretório atual:

```bash
codex mcp add calendar \
  --env GOOGLE_CREDENTIALS_FILE=/mnt/ssd/projects/mcp-calendar/google_credentials.json \
  --env GOOGLE_TOKEN_FILE=/mnt/ssd/projects/mcp-calendar/data/token.json \
  -- /mnt/ssd/projects/mcp-calendar/bin/mcp-calendar
```

Reinicie o Codex depois de alterar ou recompilar o binário.

## Ferramentas

- `authenticate`: inicia o OAuth, abre o navegador quando possível e retorna a URL de autorização.
- `auth_status`: informa se existe uma autenticação reutilizável.
- `list_events`: lista eventos dentro de um período.
- `create_event`: cria eventos únicos ou recorrentes.

Para recorrência semanal, use uma regra como:

```text
RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR
```

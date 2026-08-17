# Google Calendar MCP

Servidor MCP local em Go para autenticar no Google Calendar, listar e criar eventos e responder a convites. Suporta eventos únicos ou recorrentes, convidados e Google Meet.

## Configuração

1. Habilite a Google Calendar API no Google Cloud.
2. Crie um cliente OAuth do tipo **Desktop app**.
3. Baixe a credencial para `google_credentials.json` na raiz do projeto.

O arquivo precisa ter uma chave `installed`. Credenciais com chave `web` não
servem para o callback loopback dinâmico usado por este servidor.

### Convites de remetentes desconhecidos

O Google Calendar decide quais convites recebidos por e-mail serão adicionados à
agenda. Essa preferência não pode ser alterada pela Calendar API e precisa ser
configurada uma vez pela interface do Google Calendar:

1. Abra **Configurações**.
2. Acesse **Geral > Configurações de eventos**.
3. Em **Adicionar convites à minha agenda**, selecione **De todos**.

A alteração vale apenas para novos convites. Para um convite antigo que aparece
somente no Gmail, clique em **Adicionar à agenda** antes de tentar respondê-lo
pelo MCP.

O projeto não acessa o Gmail e não precisa habilitar a Gmail API nem solicitar
escopos do Gmail.

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
- `list_events`: lista os eventos do Calendar dentro de um período, incluindo convites ocultos que já tenham sido adicionados à agenda.
- `create_event`: cria eventos únicos ou recorrentes, adiciona convidados e pode gerar um link do Google Meet. Quando há convidados, o Google Calendar envia o convite por e-mail.
- `respond_to_event`: confirma, recusa ou marca como talvez um convite recebido pelo usuário autenticado.

Para recorrência semanal, use uma regra como:

```text
RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR
```

Exemplo de argumentos para criar uma reunião:

```json
{
  "summary": "Conversa sobre o projeto",
  "start": "2026-08-20T10:00:00-03:00",
  "end": "2026-08-20T10:30:00-03:00",
  "attendees": ["convidado@example.com"],
  "createMeet": true
}
```

Para responder a um convite, use o `id` retornado por `list_events`:

```json
{
  "eventId": "identificador-do-evento",
  "responseStatus": "accepted"
}
```

Os valores aceitos para `responseStatus` são `accepted`, `declined` e `tentative`.

O `list_events` solicita também os convites ocultos já conhecidos pelo Calendar.
Isso não faz com que um convite existente apenas no Gmail seja importado para a
agenda. Convites ainda não respondidos são retornados com `responseStatus` igual
a `needsAction`.

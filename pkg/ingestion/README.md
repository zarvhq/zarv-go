# ingestion

Envia eventos ao gateway de ingestão da Zarv, que grava cada um na sua tabela
`bronze`. O pacote é só a ponte: recebe o corpo atual do gateway, envia, e
devolve a resposta do gateway como erro tipado. Não valida nem transforma o
conteúdo do evento.

```go
import "github.com/zarvhq/zarv-go/pkg/ingestion"

client, err := ingestion.New(ingestion.Config{}) // lê as variáveis de ambiente
if err != nil {
    log.Fatal(err) // nomeia a variável que falta
}

err = client.Send(ctx, ingestion.Event{
    TableName: "billing_plan",
    Operation: "UPDATE",
    Data:      map[string]any{"id": "p1", "name": "Pro"},
})
switch {
case errors.Is(err, ingestion.ErrRefused):
    // o gateway recusou: reenviar dá a mesma resposta
case errors.Is(err, ingestion.ErrUnavailable):
    // o gateway não aceitou dentro das tentativas: pode reenviar depois
}
```

## Configuração

| Variável | Campo do `Config` | |
|---|---|---|
| `ZARV_INGESTION_URL` | `URL` | URL base do gateway, `http` ou `https`. Sem caminho, usa `/v1/ingestion` |
| `ZARV_INGESTION_KEY` | `Key` | Chave Bearer aceita pelo stream do gateway |

O campo do `Config`, quando preenchido, vence a variável. `New` falha na
partida se faltar a URL ou a chave, ou se a URL não for absoluta; a mensagem
nomeia a variável e nunca mostra o valor.

Opcionais: `HTTPClient` (padrão com timeout de 30s), `Logger` (padrão
`slog.Default()`) e `MaxAttempts` (padrão 4, contando a primeira).

## O evento

`Event` é o corpo do gateway, campo a campo:

| Campo | JSON | |
|---|---|---|
| `TableName` | `table_name` | Tabela `bronze` de destino. Obrigatório |
| `Data` | `data` | O registro. Obrigatório |
| `Operation` | `operation` | `INSERT`, `UPDATE` ou `DELETE`. Vazio é `INSERT` |
| `Description` | `description` | Texto livre, gravado com a linha |
| `UniqueKey` | `unique_key` | Campo de `Data` que identifica o registro. Vazio usa o padrão do gateway |

Só `TableName` e `Data` são checados antes do envio (`ErrInvalidEvent`), porque
sem eles não há requisição. O nome da tabela e a operação são do gateway julgar:
um nome fora do padrão do stream volta como recusa, com o motivo. O `Data` do
chamador nunca é alterado.

## Erros

| Erro | Quando | Reenviar? |
|---|---|---|
| `ErrInvalidEvent` | Sem `TableName` ou sem `Data`; nada foi enviado | Não, corrija o evento |
| `*RefusedError` (`ErrRefused`) | 400, 401, 413, ou um 202 cujo corpo recusa o evento. Traz `Status` e `Reason`, as palavras do gateway | Não |
| `*UnavailableError` (`ErrUnavailable`) | 503, 5xx, 429 ou erro de rede em todas as tentativas. Traz `Attempts`, o último `Status` e o último erro de rede | Sim, mais tarde |
| erro do `ctx` | O contexto foi cancelado ou venceu | Decisão do chamador |

**Um 202 não é sempre sucesso**: o gateway aceita a requisição e pode recusar o
evento dentro dela. `Send` lê o corpo e devolve a recusa como erro. Um evento
arquivado (`archived`) conta como sucesso.

## Novas tentativas

Um 503, um 5xx, um 429, um timeout ou um erro de conexão é reenviado, até
`MaxAttempts` e dentro do `ctx`. A espera segue o `Retry-After` do gateway,
limitado a 30s; sem ele, dobra a partir de 200ms até 5s.

**Reenviar não duplica.** O JSON é montado uma vez e todas as tentativas enviam
os mesmos bytes, e o gateway identifica a linha pelo conteúdo do registro: a
mesma entrada é a mesma linha.

## Logs

Um envio que falha gera **uma** linha em `Config.Logger`, em nível `ERROR`
(`WARN` quando o contexto foi cancelado), com `table`, `status`, `attempts` e
`reason`. Nunca o registro, nunca a chave. As novas tentativas saem em `DEBUG`;
um envio bem-sucedido não gera log.

## Onde roda

O gateway é interno ao cluster. Um produtor fora dele precisa de acesso de rede
combinado antes.

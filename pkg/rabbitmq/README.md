# RabbitMQ Package

Cliente para operações com RabbitMQ.

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/rabbitmq
```

```go
import "github.com/zarvhq/zarv-go/pkg/rabbitmq"
```

## 🚀 Uso

### Producer (Publicar mensagens)

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

func main() {
    ctx := context.Background()
    
    // Conectar ao RabbitMQ
    client, err := rabbitmq.NewClient(ctx, "amqp://guest:guest@localhost:5672/")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar producer
    producer, err := client.NewProducer()
    if err != nil {
        panic(err)
    }
    
    // Publicar mensagem
    err = producer.Publish("my-queue", map[string]interface{}{
        "type": "order",
        "id":   12345,
        "data": "some data",
    })
    if err != nil {
        panic(err)
    }
}
```

### Consumer (Consumir mensagens)

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

// Implementar o handler personalizado
type OrderHandler struct{}

func (h *OrderHandler) HandleMessage(data []byte) error {
    var order map[string]interface{}
    if err := json.Unmarshal(data, &order); err != nil {
        return err
    }
    
    log.Printf("Processing order: %v", order)
    // Processar a mensagem aqui
    
    return nil
}

func main() {
    ctx := context.Background()
    
    // Conectar ao RabbitMQ
    client, err := rabbitmq.NewClient(ctx, "amqp://guest:guest@localhost:5672/")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar consumer com handler
    handler := &OrderHandler{}
    consumer, err := client.NewConsumer(
        "order-consumer",  // nome do consumer
        "my-queue",        // nome da fila
        handler,           // handler para processar mensagens
    )
    if err != nil {
        panic(err)
    }
    
    // Iniciar consumo com 5 workers concorrentes
    err = consumer.Consume(5)
    if err != nil {
        panic(err)
    }
}
```

## 🔧 Interface ConsumerHandler

Para consumir mensagens, implemente a interface `ConsumerHandler`:

```go
type ConsumerHandler interface {
    HandleMessage([]byte) error
}
```

Qualquer struct que implemente o método `HandleMessage` pode ser usado como handler.
## 🛑 Graceful Shutdown

O consumer suporta graceful shutdown via contexto:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Capturar sinais do OS
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    client, _ := rabbitmq.NewClient(ctx, "amqp://localhost:5672/")
    defer client.Close()
    
    handler := &OrderHandler{}
    consumer, _ := client.NewConsumer("worker", "orders", handler)
    
    // Executar consumer em goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- consumer.Consume(5)
    }()
    
    // Aguardar sinal de shutdown
    select {
    case <-sigChan:
        log.Println("Received shutdown signal, stopping gracefully...")
        cancel() // Cancela o context
        
        // Consumer irá:
        // 1. Parar de aceitar novas mensagens
        // 2. Aguardar mensagens em processamento terminarem
        // 3. Retornar nil
        
        // Aguardar consumer terminar
        if err := <-errChan; err != nil {
            log.Printf("Consumer error: %v", err)
        }
        log.Println("Shutdown complete")
        
    case err := <-errChan:
        log.Printf("Consumer stopped: %v", err)
    }
}
```

**Comportamento:**
- ✅ Para de aceitar novas mensagens imediatamente
- ✅ Aguarda mensagens já em processamento
- ✅ Faz Ack/Nack de todas as mensagens em processamento
- ✅ Fecha channel graciosamente
- ✅ Retorna nil após cleanup completo

## ⚡ Reconexão Automática

### Producer

O producer possui **reconexão automática de channel**:

```go
producer, _ := client.NewProducer()
defer producer.Close()

// Se o channel fechar (erro de rede, broker restart):
// - Publish detecta automaticamente
// - Recria o channel
// - Publica a mensagem
err := producer.Publish("queue", data)
if err != nil {
    // Erro apenas se a CONNECTION estiver fechada
    log.Fatal(err)
}
```

**Importante:** Se a **conexão** fechar, você precisa criar um novo client:

```go
if client.IsClosed() {
    // Criar novo client
    client, err = rabbitmq.NewClient(ctx, url)
    producer, err = client.NewProducer()
}
```

### Consumer

O consumer **não** tem reconexão automática. Quando encontra erro, você tem duas opções:

#### Opção 1: Loop de retry na aplicação

```go
for {
    client, err := rabbitmq.NewClient(ctx, url)
    if err != nil {
        log.Printf("Failed to connect: %v. Retrying...", err)
        time.Sleep(5 * time.Second)
        continue
    }
    
    consumer, err := client.NewConsumer("name", "queue", handler)
    if err != nil {
        log.Printf("Failed to create consumer: %v", err)
        client.Close()
        time.Sleep(5 * time.Second)
        continue
    }
    
    err = consumer.Consume(5)
    if err != nil {
        log.Printf("Consumer error: %v. Reconnecting...", err)
        client.Close()
        time.Sleep(5 * time.Second)
        continue
    }
    
    // err == nil significa context foi cancelado (shutdown gracioso)
    log.Println("Consumer stopped gracefully")
    client.Close()
    break
}
```

#### Opção 2: Graceful shutdown + supervisor externo (Recomendado)

Deixe o supervisor externo (systemd, Kubernetes, Docker, etc) reiniciar a aplicação:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Canal para capturar sinais
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    // Canal para erro fatal do consumer
    fatalErr := make(chan error, 1)
    
    // Iniciar consumer
    go func() {
        if err := runConsumer(ctx); err != nil {
            fatalErr <- err
        }
    }()
    
    // Aguardar sinal de shutdown ou erro fatal
    select {
    case sig := <-sigChan:
        log.Printf("Received signal %v, shutting down gracefully...", sig)
        cancel() // Graceful shutdown do consumer
        time.Sleep(time.Second) // Aguarda consumer parar
        log.Println("Shutdown complete")
        os.Exit(0)
        
    case err := <-fatalErr:
        log.Printf("Fatal error: %v. Initiating graceful shutdown...", err)
        cancel() // Para outros componentes graciosamente
        time.Sleep(time.Second)
        log.Println("Exiting with error")
        os.Exit(1) // Supervisor irá reiniciar
    }
}

func runConsumer(ctx context.Context) error {
    client, err := rabbitmq.NewClient(ctx, "amqp://localhost:5672/")
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    defer client.Close()
    
    handler := &MyHandler{}
    consumer, err := client.NewConsumer("worker", "tasks", handler)
    if err != nil {
        return fmt.Errorf("failed to create consumer: %w", err)
    }
    
    // Consume retorna:
    // - nil: se context cancelado (shutdown normal)
    // - error: se canal/conexão caiu (erro fatal)
    return consumer.Consume(5)
}
```

**Com systemd:**
```ini
[Unit]
Description=My Consumer Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/my-consumer
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

**Com Docker Compose:**
```yaml
services:
  consumer:
    image: my-consumer:latest
    restart: always
    environment:
      RABBITMQ_URL: amqp://rabbitmq:5672/
```

**Com Kubernetes:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: consumer
        image: my-consumer:latest
        env:
        - name: RABBITMQ_URL
          value: amqp://rabbitmq:5672/
      restartPolicy: Always
```

**Vantagens do approach com supervisor:**
- ✅ Aplicação mais simples (sem lógica de retry)
- ✅ Graceful shutdown garantido
- ✅ Supervisor controla backoff/rate limiting
- ✅ Logs e métricas centralizados
- ✅ Funciona para qualquer tipo de erro fatal
- ✅ Permite upgrade/rollback fácil


## 🔒 Thread Safety

- **Producer.Publish()**: Thread-safe, pode ser chamado por múltiplas goroutines
- **Consumer.Consume()**: Deve ser chamado apenas uma vez por consumer
- **Client**: Thread-safe para criar múltiplos producers/consumers
## 📋 Funcionalidades

- ✅ Publicação de mensagens em filas
- ✅ Consumo concorrente de mensagens
- ✅ Serialização automática para JSON
- ✅ Gerenciamento de conexão
- ✅ Handlers personalizáveis
- ✅ Acknowledgement manual de mensagens

## 🔌 Formato da URL de Conexão

```
amqp://username:password@host:port/vhost
```

Exemplos:
- Local: `amqp://guest:guest@localhost:5672/`
- Produção: `amqp://user:pass@rabbitmq.example.com:5672/production`

## ⚙️ Concorrência

O método `Consume(concurrency int)` permite especificar quantos workers processarão mensagens simultaneamente:

```go
// 1 worker (sequencial)
consumer.Consume(1)

// 5 workers (concorrente)
consumer.Consume(5)

// 10 workers (alta concorrência)
consumer.Consume(10)
```

## 🛡️ Tratamento de Erros

- Se `HandleMessage` retornar erro, a mensagem será rejeitada (Nack)
- Se `HandleMessage` retornar `nil`, a mensagem será confirmada (Ack)
- Conexões fechadas são detectadas automaticamente

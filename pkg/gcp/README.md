# GCP Package

Pacote com clientes para serviços do Google Cloud Platform.

## 📦 Submódulos

### GCS (Google Cloud Storage)

Cliente para interagir com Google Cloud Storage.

#### Instalação

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
```

#### Uso

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
)

func main() {
    ctx := context.Background()
    
    // Configuração
    cfg := &gcs.Cfg{
        BucketName:      "my-bucket",
        DatalakeBucket:  "datalake-bucket",
        CredentialsJSON: nil, // Usa Workload Identity se nil
        Local:           false,
    }
    
    // Criar cliente
    client, err := gcs.NewClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    
    // Upload de arquivo
    err = client.PutObject("path/file.txt", "text/plain", "", []byte("content"))
    if err != nil {
        panic(err)
    }
    
    // Download de arquivo
    data, err := client.GetObject("path/file.txt")
    if err != nil {
        panic(err)
    }
    
    // Download com content type
    data, contentType, err := client.GetObjectWithContentType("path/file.txt")
    if err != nil {
        panic(err)
    }
}
```

### Document AI

Cliente para processar documentos com Google Cloud Document AI.

#### Instalação

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
```

#### Uso

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
)

func main() {
    ctx := context.Background()
    
    cfg := &documentai.Cfg{
        ProjectID:       "my-project",
        Location:        "us",
        CredentialsJSON: nil, // Usa Workload Identity se nil
    }
    
    client, err := documentai.NewClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    
    // Processar documento
    doc, err := client.ProcessDocument(
        ctx,
        fileBytes,
        "application/pdf",
        "processor-id",
    )
    if err != nil {
        panic(err)
    }
}
```

## 🔧 Desenvolvimento Local

Para desenvolvimento local com fake-gcs-server:

```go
cfg := &gcs.Cfg{
    BucketName: "local-bucket",
    Endpoint:   "http://localhost:4443",
    Local:      true,
}
```

## 🔐 Autenticação

Por padrão, os clientes usam **Workload Identity** no GKE. Para usar credenciais JSON:

```go
cfg := &gcs.Cfg{
    BucketName:      "my-bucket",
    CredentialsJSON: []byte(`{"type": "service_account", ...}`),
}
```

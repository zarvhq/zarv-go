# Fiber Package

Pacote com utilitários e middlewares para aplicações Fiber.

## 📦 Submódulos

### v2/middleware

Middlewares de autenticação e autorização para Fiber v2.

#### Instalação

```go
import "github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
```

#### Exemplo de Uso

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
)

func main() {
    app := fiber.New()

    // Adicionar middleware de autenticação
    app.Use(middleware.Authenticate)

    app.Get("/api/resource", func(c *fiber.Ctx) error {
        // Obter perfil de autenticação
        profile := middleware.GetAuthProfile(c)
        
        // Verificar permissões
        if profile.IsViewer() {
            return c.Status(403).JSON(fiber.Map{
                "error": "Permission denied",
            })
        }

        return c.JSON(fiber.Map{
            "workspaceId": profile.WorkspaceID,
            "userId": profile.UserID,
        })
    })

    app.Listen(":3000")
}
```

#### Headers Esperados

O middleware `Authenticate` valida os seguintes headers:

- `X-Issuer`: Identificador do emissor (ex: "ultron-app", "vision-app")
- `X-Workspace-Id`: ID do workspace Zarv
- `X-User-Id`: ID do usuário Zarv
- `X-Zarv-Role`: Role do usuário
- `X-Access-Level`: Nível de acesso (viewer, user, supervisor, admin)
- `X-Internal`: (Opcional) Indica requisição interna

#### Níveis de Acesso

- `viewer`: Acesso somente leitura
- `user`: Acesso de usuário padrão
- `supervisor`: Acesso de supervisor
- `admin`: Acesso administrativo completo

#### Métodos do AuthProfile

```go
profile := middleware.GetAuthProfile(c)

// Verificar se é administrador Zarv (role zarver + admin/supervisor)
profile.IsZarvAdmin() // bool

// Verificar se é administrador do workspace
profile.IsUserAdmin() // bool

// Verificar se tem apenas acesso de visualização
profile.IsViewer() // bool
```

## 📝 Versionamento

O pacote está organizado por versões do Fiber (`v2/`) para facilitar futuras atualizações.

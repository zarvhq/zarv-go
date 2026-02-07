# Contributing to Zarv Go

Obrigado por considerar contribuir com o Zarv Go! 🎉

## Como Contribuir

### Reportando Bugs

Se você encontrou um bug, por favor abra uma [issue](https://github.com/zarvhq/zarv-go/issues) com:

- Descrição clara do problema
- Passos para reproduzir
- Comportamento esperado vs. comportamento atual
- Versão do Go e das dependências
- Código de exemplo, se possível

### Sugerindo Melhorias

Sugestões de melhorias são bem-vindas! Abra uma [issue](https://github.com/zarvhq/zarv-go/issues) descrevendo:

- O problema que a melhoria resolve
- Como a melhoria funcionaria
- Exemplos de uso

### Pull Requests

1. Fork o repositório
2. Crie uma branch a partir de `main`:
   ```bash
   git checkout -b feature/minha-feature
   ```

3. Faça suas alterações seguindo as convenções:
   - Código limpo e bem documentado
   - Comentários em inglês no código
   - Documentação em português no README
   - Siga as [Effective Go guidelines](https://golang.org/doc/effective_go.html)

4. Certifique-se de que o código está formatado:
   ```bash
   go fmt ./...
   ```

5. Execute os testes (quando disponíveis):
   ```bash
   go test ./...
   ```

6. Commit suas alterações com mensagens claras:
   ```bash
   git commit -m "feat: adiciona nova funcionalidade X"
   ```

7. Push para sua branch:
   ```bash
   git push origin feature/minha-feature
   ```

8. Abra um Pull Request

### Convenções de Commit

Usamos convenções semelhantes ao [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` Nova funcionalidade
- `fix:` Correção de bug
- `docs:` Mudanças na documentação
- `style:` Formatação, ponto e vírgula faltando, etc
- `refactor:` Refatoração de código
- `test:` Adição ou modificação de testes
- `chore:` Tarefas de manutenção

### Padrões de Código

- Use `gofmt` para formatar o código
- Siga as diretrizes do [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Mantenha funções pequenas e focadas
- Escreva testes para novas funcionalidades
- Documente exports públicos com comentários GoDoc

### Documentação

- Todos os exports públicos devem ter comentários GoDoc
- README deve ser atualizado para novas funcionalidades
- Exemplos de uso são encorajados

## Código de Conduta

Seja respeitoso e construtivo em todas as interações. Não toleramos:

- Linguagem ofensiva ou discriminatória
- Assédio de qualquer tipo
- Comportamento não profissional

## Dúvidas?

Se tiver dúvidas sobre como contribuir, sinta-se à vontade para abrir uma issue com a tag `question`.

Obrigado! 🙏

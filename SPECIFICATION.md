# TUI Proxy - Especificação Técnica Completa

> Proxy HTTP/HTTPS com interface TUI em Go, inspirado no Proxyman.
> Documentação consolidada de https://proxy.jmota.org/

---

## 1. Visão Geral

### 1.1 Conceito
Construir um proxy inspirado no Proxyman, mas com UI terminal. O backend Go intercepta HTTP/S e expõe eventos para o MVVM, enquanto o frontend em tview dá controlo total via teclado.

### 1.2 Stack Tecnológica
- **Backend**: Go + goproxy para interceptação HTTPS
- **Frontend**: Tview/Tcell para TUI com padrão MVVM
- **Certificados**: Go crypto/tls para CA dinâmico

### 1.3 Componentes Principais

| Componente | Descrição |
|------------|-----------|
| **Backend** | Go + goproxy para interceptação, com store de flows e geração de certificados dinâmicos |
| **Frontend** | Tview/Tcell para MVVM no terminal, ligando teclas a filtros e mapeamentos |
| **Alertas** | (Fase 2) Mecanismos que sinalizam requisições críticas (5xx, latência elevada) |

---

## 2. Requisitos

### 2.1 Captura Segura
- HTTPS proxy com capacidades MITM condicionais (SSL Proxy List)
- Certificados dinâmicos gerados por domínio via Go
- Modo passivo (tunnel) para hosts não listados no SSL Proxy List
- Armazenamento efémero em memória

### 2.2 Observabilidade & Comandos

#### Painel de Requests
- Painel fixo à esquerda com filtros rápidos e atalhos de teclado
- Lista de requests mostrando: tempo, método, status, URL
- Conexões tunneled (CONNECT) mostradas separadamente
- Atalhos principais:
  - `l` - Map to Local (quick)
  - `L` - Map to Local Manager
  - `r` - Map to Remote (quick)
  - `R` - Map to Remote Manager
  - `w` - Add to Whitelist
  - `W` - Whitelist Manager

#### Filtros
- Opções de visualização: Todos (`1`), Whitelist (`2`), Pesquisa por keyword (`/`)
- Filtro custom persiste mesmo quando muda para "Todos"
- Detalhes de header/body com syntax highlighting JSON
- Map Local tem prioridade sobre Map Remote

### 2.3 Alertas Internos (Fase 2 - não implementado)
- Alertas configuráveis para respostas 5xx, latência elevada, payloads específicos
- Compatibilidade Linux/macOS
- Suporte a workflow TUI

### 2.4 Mapeamentos de Tráfego

#### Map to Remote
Reescreve respostas redirecionando requests para URLs alternativos (staging, mock servers) mantendo os requests originais do cliente. O proxy contacta o novo URL e devolve a resposta ao cliente como se viesse do URL original, sem HTTP redirects visíveis ao cliente.

- Headers `Location` em redirects são reescritos para o host original
- Suporta mapeamento HTTPS → HTTP (ex: `https://api.com/*` → `http://localhost:3000`)

#### Map to Local
Captura respostas originais e guarda em ficheiro JSONC para edição. Ao criar um mapeamento local (`l`), o ficheiro é automaticamente aberto no editor definido (`$EDITOR`, `$VISUAL` ou `vi`).

**Formato do ficheiro mock (JSONC):**
```jsonc
{
  // Mapped from: https://api.example.com/users
  // Generated: 2024-01-15T10:30:00Z

  "status": 200,
  "statusText": "OK",

  "headers": {
    "Content-Type": "application/json",
    "Cache-Control": "no-cache"
  },

  // Response body - edit below
  "body": {
    "id": "abc",
    "value": 42
  }
}
```

**Prioridade de mapeamentos:** Map Local é sempre verificado antes de Map Remote. Se um URL corresponder a ambos, Map Local é usado.

### 2.5 Gestão da Whitelist
- Padrões glob suportados (ex: `*.example.com`, `api.*`)
- Controlos de teclado:
  - `w` - adicionar padrão do request selecionado
  - `W` - abrir manager
  - `Space/Enter` - toggle enabled/disabled
  - `d/x/Delete` - remover
  - `a` - adicionar novo padrão

### 2.6 SSL Proxy List
- Lista de hosts para MITM condicional
- Hosts não listados passam em modo tunnel (sem MITM)
- Permite interceptar apenas o tráfego desejado

---

## 3. Arquitetura MVVM

### 3.1 Model
Structs implementadas:
- `Flow` - descreve temporização, estados e payloads
- `Request` - método, URL, headers, body
- `Response` - status, headers, body
- `FilterState` - tipo de filtro e query
- `MapRule` - padrão, destino, tipo (local/remote), enabled
- `HostPattern` - padrão de whitelist com estado enabled

### 3.2 ViewModel
- Canal `chan FlowEvent` alimentado pelo proxy
- Aplicação incremental de filtros e seleção
- Exposição de comandos e binding a teclas
- Gestão de MapRuleStore e WhitelistStore

### 3.3 View

#### Painel Principal (Esquerda)
Lista de flows com: tempo, método, status code (colorido), URL truncado

#### Painel de Detalhe (Direita)
- Request: método, URL completo, headers, body
- Response: status, headers, body com pretty-print JSON
- Toggle entre raw e pretty com `T`

### 3.4 Layout Responsivo ao Foco
- Foco alterna entre lista de requests e painel de detalhe via `Tab`
- Tecla `H` expande o painel com foco para 80%, escondendo o outro
- Alternância garante visualização completa sem perder contexto

### 3.5 Regra de Atribuição de Teclas
- Usar sempre tecla simples (sem modificadores) para cada ação
- Prioridade: letra minúscula para ação rápida; maiúscula para manager/menu
- Overlay "whichkey" (`?`) mostra teclas disponíveis por contexto

### 3.6 Interface Focada em Keybindings
- TUI usa teclas simples associadas ao nome da ação
- Painel whichkey lista teclas disponíveis por contexto (lista, detalhe)
- Atualiza quando foco muda
- `?` abre guia completo com scroll (`j/k`, setas, PgUp/PgDn)

### 3.7 Popups Modais
- Popups bloqueiam clicks fora da área do popup
- Navegação apenas por teclado quando popup está aberto
- URL patterns usam input multi-linha para melhor visualização
- Inputs sem fundo azul (estilo minimalista)

---

## 4. Detalhes do Request/Response

Painel direito mostra resumo do request e corpo/response:

```
┌──────────────────────────────────────────────────────────┐
│ ► Request                                                │
│ GET https://api.example.com/users?page=1                 │
│                                                          │
│ Headers:                                                 │
│   Accept: application/json                               │
│   Authorization: Bearer xxx...                           │
├──────────────────────────────────────────────────────────┤
│ ► Response · 200 OK                                      │
│                                                          │
│ Headers:                                                 │
│   Content-Type: application/json                         │
│                                                          │
│ Body:                                                    │
│ {                                                        │
│   "users": [                                             │
│     { "id": 1, "name": "John" }                          │
│   ]                                                      │
│ }                                                        │
└──────────────────────────────────────────────────────────┘
```

**Atalhos do painel de detalhe:**
- `T` - troca entre raw e pretty
- `j/k` - scroll
- `w` - adicionar host à whitelist
- `l` - map to local
- `r` - map to remote
- (Fase 2) `E` - abre mock no editor

---

## 5. Implementação Técnica

### 5.1 Arquitetura Go

#### Proxy Core (`internal/proxy/`)
- Utiliza `net/http` e `goproxy` para interceptação MITM
- CA carregado/gerado ao iniciar em `~/.config/proxy-tui/ca/`
- Gera certificados dinâmicos por domínio
- SSL Proxy List para MITM condicional

#### FlowStore (`internal/proxy/flowstore.go`)
- Cada request cria um `Flow` com metadados
- Flows enviados para `chan FlowEvent` consumido pela ViewModel
- Limite configurável de flows em memória

#### MapRuleStore (`internal/model/mapping.go`)
- Estrutura ordenada armazena regras
- Map Local verificado antes de Map Remote
- Suporta padrões glob e regex

### 5.2 MITM e CA

O CA é gerado automaticamente na primeira execução:

```
~/.config/proxy-tui/ca/
├── ca.key          # Chave privada (4096 bits RSA)
└── ca.crt          # Certificado raiz (válido 10 anos)
```

**Instalação manual do CA (se necessário):**

```bash
# Linux
sudo cp ~/.config/proxy-tui/ca/ca.crt /usr/local/share/ca-certificates/proxy-tui.crt
sudo update-ca-certificates

# macOS
sudo security add-trusted-cert -d -r trustRoot \
  -k "/Library/Keychains/System.keychain" ~/.config/proxy-tui/ca/ca.crt
```

**Considerações:**
- Apps com certificate pinning precisam de builds de debug
- CA deve ser mantido seguro - compromete todo o tráfego MITM
- Hosts não listados no SSL Proxy List passam em modo tunnel

### 5.3 TUI MVVM

#### ViewModel (`internal/viewmodel/`)
- Medeia entre FlowStore, MapRuleStore e UI
- Aplica filtros (Todos/Whitelist/Search) no slice de flows
- Publica eventos de atualização via channel

#### View (`internal/ui/`)
- `app.go` - coordenação principal e keybindings
- `layout.go` - grid layout com painéis
- `requests.go` - lista de requests
- `detail.go` - painel de detalhe
- `whichkey.go` - overlay de ajuda
- `whitelist.go` - manager de whitelist
- `maplocal.go` - manager de map local
- `mapremote.go` - manager de map remote

### 5.4 Persistência (Fase 2 - parcialmente implementado)

**Diretório de configuração:** `~/.config/proxy-tui/`

```
~/.config/proxy-tui/
├── ca/
│   ├── ca.key          # Chave privada CA
│   └── ca.crt          # Certificado CA
└── mappings/           # Ficheiros mock JSONC
    └── *.jsonc
```

**TODO Fase 2:**
- Persistir regras de mapping em `maps.json`
- Persistir whitelist
- Ficheiro de configuração `config.yml`

---

## 6. Keybindings Completos

### 6.1 Globais
| Tecla | Ação |
|-------|------|
| `Tab` | Alternar foco entre painéis |
| `H` | Expandir painel atual (80%) |
| `?` | Abrir guia de ajuda (scrollable) |
| `q` | Sair |

### 6.2 Lista de Requests
| Tecla | Ação |
|-------|------|
| `j/↓` | Próximo request |
| `k/↑` | Request anterior |
| `g g` | Ir para o topo |
| `G` | Ir para o fim |
| `Enter` | Ver detalhes (foca painel direito) |
| `l` | Map to Local (quick - cria ficheiro e abre editor) |
| `L` | Map Local Manager |
| `r` | Map to Remote (quick - abre form) |
| `R` | Map Remote Manager |
| `w` | Adicionar à Whitelist |
| `W` | Whitelist Manager |
| `/` | Pesquisar/Filtrar |
| `1` | Filtro: Todos |
| `2` | Filtro: Whitelist |
| `c` | Limpar flows |
| `C` | Limpar whitelist |

### 6.3 Painel de Detalhe
| Tecla | Ação |
|-------|------|
| `T` | Alternar raw/pretty |
| `j/↓` | Scroll down |
| `k/↑` | Scroll up |
| `PgDn` | Page down |
| `PgUp` | Page up |
| `w` | Adicionar à Whitelist |
| `W` | Whitelist Manager |
| `l` | Map to Local |
| `L` | Map Local Manager |
| `r` | Map to Remote |
| `R` | Map Remote Manager |

### 6.4 Gestão de Mappings/Whitelist (dentro dos managers)
| Tecla | Ação |
|-------|------|
| `a` | Adicionar novo |
| `e` | Editar selecionado |
| `d/x/Delete` | Remover selecionado |
| `Space/Enter` | Toggle enabled/disabled |
| `Esc/q` | Fechar manager |

### 6.5 Forms de Input
| Tecla | Ação |
|-------|------|
| `Tab` | Próximo campo |
| `Enter` | Submeter (em campos simples) |
| `Ctrl+S` | Guardar (em TextArea multi-linha) |
| `Esc` | Cancelar |

---

## 7. Wireframes

### 7.1 Layout Principal
```
┌─────────────────────────────────────────────────────────────────────────┐
│ 192.168.1.100:9090 │ 1:All  2:Whitelist  /:filter                       │
├─────────────────────────────────────────────────────────────────────────┤
│ Requests                    │ ► Request                                 │
│                             │ GET https://api.example.com/users         │
│ 10:30:01 GET  200 /users    │                                           │
│ 10:30:02 POST 201 /users    │ Headers:                                  │
│ 10:30:03 GET  404 /missing  │   Accept: application/json                │
│ 10:30:04 ─── CONNECT ───    │                                           │
│                             ├───────────────────────────────────────────│
│                             │ ► Response · 200 OK                       │
│                             │                                           │
│                             │ Body:                                     │
│                             │ { "users": [...] }                        │
├─────────────────────────────────────────────────────────────────────────┤
│ Ready │ Flows: 4 │ Mapped: 0                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Com Painel de Detalhe Expandido (H)
```
┌────────────┬────────────────────────────────────────────────────────────┐
│ Requests   │ Request Detail (80%)                                       │
│ (20%)      │ GET https://api.example.com/users?page=1&limit=10          │
├────────────┤                                                            │
│ 10:30 200  │ Headers:                                                   │
│ 10:31 201  │   Accept: application/json                                 │
│ 10:32 404  │   Authorization: Bearer eyJhbGc...                         │
│            │   User-Agent: Mozilla/5.0                                  │
│            ├────────────────────────────────────────────────────────────│
│            │ ► Response · 200 OK · 123ms                                │
│            │                                                            │
│            │ Headers:                                                   │
│            │   Content-Type: application/json                           │
│            │                                                            │
│            │ Body:                                                      │
│            │ {                                                          │
│            │   "users": [                                               │
│            │     { "id": 1, "name": "John" },                           │
│            │     { "id": 2, "name": "Jane" }                            │
│            │   ],                                                       │
│            │   "total": 42                                              │
│            │ }                                                          │
└────────────┴────────────────────────────────────────────────────────────┘
```

### 7.3 Map Remote Form
```
┌──────────────────────────────────────────────────────────────┐
│                   Add Map Remote Rule                        │
│                                                              │
│  URL Pattern:                                                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ https://api.example.com/users*                         │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Remote URL: http://localhost:3000___________________        │
│                                                              │
│  [ Add ]  [ Cancel ]                                         │
└──────────────────────────────────────────────────────────────┘
```

### 7.4 Whichkey Help Overlay
```
┌────────────────────────────────────────────────────────────┐
│          Keybindings (j/k to scroll, ? to close)           │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  Global                                                    │
│    q        Quit                                           │
│    Tab      Toggle focus                                   │
│    H        Expand panel                                   │
│    ?        Toggle help                                    │
│                                                            │
│  List                                                      │
│    j/↓      Next item                                      │
│    k/↑      Previous item                                  │
│    Enter    View detail                                    │
│    l        Map selected to local                          │
│    L        Map local manager                              │
│    r        Add map remote rule                            │
│    R        Map remote manager                             │
│    w        Add to whitelist                               │
│    W        Show whitelist                                 │
│    /        Filter: Custom                                 │
│    1        Filter: All                                    │
│    2        Filter: Whitelist                              │
│    c        Clear flows                                    │
│                                                            │
│  Press ? to close                                          │
└────────────────────────────────────────────────────────────┘
```

---

## 8. Roadmap

### Fase 1 - MVP ✅
1. ✅ Proxy básico Go com MITM condicional
2. ✅ TUI com keybindings contextuais (whichkey)
3. ✅ Map to Local com ficheiros JSONC
4. ✅ Map to Remote com reescrita de Location headers
5. ✅ Whitelist com padrões glob
6. ✅ Filtros (All/Whitelist/Custom)

### Fase 2 - Funcionalidades Avançadas (em progresso)
1. ⬜ Persistência de mappings e whitelist
2. ⬜ Replay de requests
3. ⬜ Export de flows (HAR, cURL)
4. ⬜ Alertas para 5xx e latência elevada
5. ⬜ Hit counters nos mappings
6. ⬜ Ficheiro de configuração

### Fase 3 - Expansão
1. ⬜ Distribuição (binários Linux/macOS)
2. ⬜ Breakpoints (pausar e editar requests)
3. ⬜ Scripting/plugins
4. ⬜ WebSocket support

---

## 9. Plano de Testes

### 9.1 Testes Unitários
- FlowStore (add/get/clear)
- MapRuleStore (add/remove/toggle/match)
- WhitelistStore (add/remove/match patterns)
- Parser de ficheiros JSONC mock

### 9.2 Testes de Integração
- Servidor de teste (`httptest`) valida:
  - Map Local serve ficheiro JSONC
  - Map Remote redireciona corretamente
  - HTTPS MITM com CA gerado
  - Tunnel mode para hosts não listados

### 9.3 Testes Manuais
- Verificar keybindings em cada contexto
- Testar popups não perdem foco com clicks externos
- Validar scroll no help overlay
- Confirmar editor abre após map local

---

## 10. Deploy e Distribuição

### Build
```bash
# Linux/macOS
go build -o proxy-tui ./cmd/proxy-tui
```

### Execução
```bash
# Iniciar na porta default (9090)
./proxy-tui

# Porta customizada
./proxy-tui --port 8080
```

### Estrutura de Diretórios
```
~/.config/proxy-tui/
├── ca/
│   ├── ca.key          # Chave privada CA (gerado automaticamente)
│   └── ca.crt          # Certificado CA (gerado automaticamente)
└── mappings/           # Ficheiros mock JSONC
    ├── api.example.com_users.jsonc
    └── api.example.com_products.jsonc
```

---

## 11. Estado de Implementação

### Implementado ✅
- [x] Proxy HTTP/HTTPS com goproxy
- [x] MITM condicional (SSL Proxy List)
- [x] Geração automática de CA
- [x] TUI com tview/tcell
- [x] Layout com dois painéis (requests/detail)
- [x] Navegação vim-style (j/k/g/G)
- [x] Filtros (All/Whitelist/Custom)
- [x] Map to Local com JSONC
- [x] Map to Remote com reescrita de headers
- [x] Whitelist com padrões glob
- [x] Managers para mappings e whitelist
- [x] Help overlay scrollable
- [x] Popups modais (bloqueiam clicks externos)
- [x] Inputs multi-linha para URL patterns
- [x] Editor integration ($EDITOR)

### Não Implementado ⬜
- [ ] Persistência de configuração
- [ ] Replay requests
- [ ] Export (HAR, cURL)
- [ ] Alertas (5xx, latência)
- [ ] Hit counters
- [ ] Breakpoints
- [ ] WebSocket support

---

## 12. Checklist de Utilização

- [x] Executar `./proxy-tui` (CA é gerado automaticamente)
- [ ] Instalar CA no sistema/browser se necessário
- [ ] Configurar proxy no sistema/app (ex: `HTTP_PROXY=http://localhost:9090`)
- [ ] Adicionar hosts ao SSL Proxy List para MITM
- [ ] Usar `w` para adicionar hosts à whitelist
- [ ] Usar `l` para criar mocks locais
- [ ] Usar `r` para redirecionar para servidores alternativos

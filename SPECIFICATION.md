# TUI Proxy - Especificação Técnica Completa

> Proxy HTTP/HTTPS com interface TUI em Go, inspirado no Proxyman.
> Documentação consolidada de https://proxy.jmota.org/

---

## 1. Visão Geral

### 1.1 Conceito
Construir um proxy inspirado no Proxyman, mas com UI terminal. O backend Go intercepta HTTP/S e expõe eventos para o MVVM, enquanto o frontend em glow/tview dá controlo total via teclado.

### 1.2 Stack Tecnológica
- **Backend**: Go + goproxy para interceptação HTTPS
- **Frontend**: Glow/Tview/Tcell para TUI com padrão MVVM
- **Certificados**: OpenSSL para CA dinâmico

### 1.3 Componentes Principais

| Componente | Descrição |
|------------|-----------|
| **Backend** | Go + goproxy para interceptação, com store de flows e geração de certificados dinâmicos |
| **Frontend** | Glow/Tview/Tcell para MVVM no terminal, ligando teclas a filtros, replay e exportação |
| **Alertas** | Mecanismos que sinalizam requisições críticas (5xx, latência elevada) via interface ou notificações locais |

---

## 2. Requisitos

### 2.1 Captura Segura
- HTTPS proxy com capacidades MITM opcionais (certificados dinâmicos via Go)
- Modo passivo para logging sem modificação
- Armazenamento efémero e exportável (dados auditáveis)

### 2.2 Observabilidade & Comandos

#### Painel de Requests
- Painel fixo no topo com filtros rápidos e atalhos de teclado
- Lista de requests mostrando: URL, método, status, flags de whitelist
- Atalhos principais:
  - `L` - Map to Local
  - `R` - Map to Remote
  - `r` - Refresh list

#### Filtros
- Filtrar por: método, domínio, estado, tempo
- Opções de visualização: Todos, Whitelist, Pesquisa por keyword
- Detalhes de header/body com syntax highlighting JSON/XML
- Comandos rápidos: replay, export, save flow, copy headers/payload

### 2.3 Alertas Internos (nice to have - Fase 2)
- Alertas configuráveis para respostas 5xx, latência elevada, payloads específicos
- Compatibilidade Linux/macOS
- Suporte a workflow TUI

### 2.4 Mapeamentos de Tráfego

#### Map to Remote
Reescreve respostas redirecionando requests para URLs alternativos (staging, mock servers) mantendo os requests originais do cliente. O proxy contacta o novo URL e devolve a resposta ao cliente como se viesse do URL original, sem HTTP redirects.

#### Map to Local
Captura respostas originais e abre-as no editor definido (`$EDITOR` ou nvim) para edição de mock. Mocks guardados aparecem na lista de mappings com toggles de ativação, contadores de hits e opções de reabertura no editor.

**Formato do ficheiro mock:**
```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "abc",
  "value": 42
}
```

### 2.5 Gestão da Whitelist
- Regras avaliadas de cima para baixo; conflitos favorecem a última regra aplicada
- Controlos de teclado:
  - `A` - adicionar
  - `d` - remover
  - `T` - toggle
  - `↑/↓` + `P` - reordenar
- Lista única de mappings (local/remote) com: padrões, destinos, hits, estado
- Confirmação contextual de remoção com opção undo
- Remoção de padrão para interceptação imediatamente, mantendo histórico de auditoria leve

---

## 3. Arquitetura MVVM

### 3.1 Model
Structs para:
- `Flow` - descreve temporização, estados e payloads
- `Request`
- `Response`
- `FilterState`
- `Alert`

### 3.2 ViewModel
- Canal `chan FlowEvent` alimentado pelo proxy
- Aplicação incremental de filtros e seleção, com estados guardados
- Exposição de comandos (replay, export, alertas) e binding a teclas

### 3.3 View

#### Painel Principal
Lista de flows, status, tempo e tags; suporte a atalhos (`Ctrl+F`, `PgUp`, etc.)

#### Painel de Detalhe
Headers, body e timeline; renderização com destaque JSON/XML e preview de imagens

#### Painel de Controlo
Consola de comandos com history, filtros e bookmarks. Rotas MVVM reativas.

### 3.4 Layout Responsivo ao Foco
- Foco alterna entre lista de requests e painel de detalhe via `Tab`
- Tecla `H` expande o painel com foco para 80%, escondendo o outro
- Alternância garante visualização completa sem perder contexto

### 3.5 Regra de Atribuição de Teclas
- Usar sempre tecla simples (sem modificadores) para cada ação
- Prioridade: letra minúscula; maiúscula quando houver segunda ação com mesma letra
- Se terceira ação precisar da mesma tecla, avisar antes de atribuir alternativa
- Overlay "whichkey" destaca se tecla ativa é minúscula/maiúscula e ação correspondente

### 3.6 Interface Focada em Keybindings
- TUI usa teclas simples associadas ao nome da ação
- Painel whichkey lista teclas disponíveis por contexto (lista, request, filtros)
- Atualiza quando foco muda
- `?` abre guia completo; cada painel tem linha de ajuda clara

### 3.7 Inspiração LazyGit
- Múltiplos painéis fixos, foco no teclado e overlay de teclas
- Layouts de painéis (lista/detalhe) espelham status/stage/log do LazyGit
- Navegação com letras simples (`h/j/k/l`, `1-5`, `/`) e whichkey inline

---

## 4. Detalhes do Request/Response

Painel direito mostra resumo do request e corpo/response com tabs:

```
┌──────────────────────────────────────────────────────────┐
│ Request Detail (GET https://api/.../info)                │
│ status: 200 · 123ms · Map=Local                          │
├──────────────────────────────────────────────────────────┤
│ Headers:                                                 │
│   content-type: application/json                         │
│   cache-control: no-cache                                │
├──────────────────────────────────────────────────────────┤
│ Body (pretty):                                           │
│ {                                                        │
│   "id": "abc",                                           │
│   "value": 42                                            │
│ }                                                        │
└──────────────────────────────────────────────────────────┘
```

**Atalhos do painel de detalhe:**
- `H` - expande painel
- `E` - abre mock no editor
- `T` - troca tabs (raw ↔ pretty)
- `Ctrl+R` - reprocessa response

---

## 5. Implementação Técnica

### 5.1 Arquitetura Go

#### Proxy Core
- Utiliza `net/http` e `goproxy` para interceptação MITM
- CA (`ca.key`/`ca.crt`) carregado ao iniciar
- Gera certificados dinâmicos por domínio

#### FlowStore
- Cada request cria um `Flow` com metadados:
  - timestamp, client, método, url, status, corpo
- Flows enviados para `chan FlowEvent` consumido pela ViewModel

#### MapList
- Estrutura ordenada armazena regras:
  - padrão, destino, tipo, enabled, hits
- UI permite: reordenar, adicionar (`A`), remover (`d`), alternar (`T`)

### 5.2 MITM e Script CA

```bash
#!/bin/bash
set -e
CA_DIR=~/proxy-ca
mkdir -p "$CA_DIR"
openssl req -x509 -newkey rsa:4096 -days 3650 -nodes \
  -keyout "$CA_DIR/ca.key" -out "$CA_DIR/ca.crt" -subj "/CN=ProxyTUI CA"

# Linux
sudo cp "$CA_DIR/ca.crt" /usr/local/share/ca-certificates/proxy-tui.crt
sudo update-ca-certificates

# macOS
sudo security add-trusted-cert -d -r trustRoot \
  -k "/Library/Keychains/System.keychain" "$CA_DIR/ca.crt"
```

**Workflow MITM:**
1. Gerar CA self-signed
2. Instalar CA em Linux/macOS
3. Proxy Go usa `tls.X509KeyPair` para gerar certificados baseados no CA
4. Cliente aceita CA → todo tráfego TLS visível para logging/replay

**Considerações:**
- Apps com pinning precisam de builds de debug ou certificados especiais
- Manter CA em local seguro - compromete todo o tráfego
- Modo passivo (only-log) evita MITM completo enquanto valida

### 5.3 TUI MVVM

#### ViewModel (Go)
- Medeia entre FlowStore, MapList e painel TUI
- Aplica filtros (Todos/Whitelist/Search) no slice de flows
- Publica request ativo

#### View
- Usa glow/tview/tcell para renderizar:
  - Topo contextual
  - Lista de requests
  - Painel de detalhe
- Overlay whichkey mostra atalhos (`h/j/k/l`, `L/R/r`, etc.)

#### Layout
- Responde a `Tab` (rotaciona foco)
- `H` expande painel com 80% da largura

### 5.4 Persistência

**Ficheiro de configuração:** `~/.proxy-tui/maps.json`

Campos:
- `pattern` - padrão URL
- `target` - destino
- `type` - local/remote
- `enabled` - boolean
- `mockPath` - caminho do ficheiro mock
- `hits` - contador

Mocks seguem formato HTTP/1.1 header + body e reabrem via `$EDITOR` com atalho `E`.

---

## 6. Keybindings Completos

### 6.1 Globais
| Tecla | Ação |
|-------|------|
| `Tab` | Alternar foco entre painéis |
| `H` | Expandir painel atual (80%) |
| `?` | Abrir guia de ajuda completo |
| `q` | Sair |

### 6.2 Lista de Requests
| Tecla | Ação |
|-------|------|
| `j/↓` | Próximo request |
| `k/↑` | Request anterior |
| `Enter` | Ver detalhes |
| `L` | Map to Local |
| `R` | Map to Remote |
| `r` | Refresh lista |
| `/` | Pesquisar |
| `1` | Filtro: Todos |
| `2` | Filtro: Whitelist |

### 6.3 Painel de Detalhe
| Tecla | Ação |
|-------|------|
| `T` | Alternar tabs (raw/pretty) |
| `E` | Abrir mock no editor |
| `Ctrl+R` | Reprocessar response |

### 6.4 Gestão de Mappings
| Tecla | Ação |
|-------|------|
| `A` | Adicionar mapping |
| `d` | Remover mapping |
| `T` | Toggle enabled/disabled |
| `P` | Reordenar (com setas) |
| `U` | Undo última ação |

---

## 7. Wireframes

### 7.1 Layout Principal
```
┌─────────────────────────────────────────────────────────────────────┐
│ Requests • Filter=Todos • Map=— • Status=Ready                      │
├─────────────────────────────────────────────────────────────────────┤
│ hh:mm:ss.ms  GET   clientA  200  https://api/.../foo                │
│ hh:mm:ss.ms  POST  clientB  201  https://api/.../bar                │
│ hh:mm:ss.ms  GET   clientA  404  https://api/.../baz                │
├─────────────────────────────────────────────────────────────────────┤
│ Whichkey: L=Map Local | R=Map Remote | r=Refresh | /=Search         │
└─────────────────────────────────────────────────────────────────────┘
```

### 7.2 Com Painel de Detalhe Expandido
```
┌────────────────────────┬────────────────────────────────────────────┐
│ Requests (20%)         │ Request Detail (80%)                       │
├────────────────────────┤ GET https://api/.../info                   │
│ hh:mm:ss GET 200       │ status: 200 · 123ms · Map=Local            │
│ hh:mm:ss POST 201      ├────────────────────────────────────────────┤
│ hh:mm:ss GET 404       │ Headers:                                   │
│                        │   content-type: application/json           │
│                        ├────────────────────────────────────────────┤
│                        │ Body: { "id": "abc", "value": 42 }         │
└────────────────────────┴────────────────────────────────────────────┘
```

### 7.3 Map to Local + Editor
```
┌─────────────────────────────────────────────────────────────────────┐
│ Requests • Map=Local → http://localhost:3000/ • Editor=active       │
├─────────────────────────────────────────────────────────────────────┤
│ Mock content:                                                        │
│ HTTP/1.1 200 OK                                                      │
│ Content-Type: application/json                                       │
│                                                                      │
│ { "mocked": true }                                                   │
├─────────────────────────────────────────────────────────────────────┤
│ E=Reopen editor | H=Expand | Ctrl+R=Reload                          │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Roadmap

### Fase 1 - MVP
1. Proxy básico Go + CLI minimal
2. ViewModel com mock flows e TUI inicial com keybindings contextuais (whichkey)
3. Documentação e site

### Fase 2 - Funcionalidades Avançadas
1. Filtros, alertas/automação, replay e exportação
2. Alertas e automações internas focadas na interface e logs locais
3. Distribuição inicial (binários Linux/macOS)

### Fase 3 - Expansão
1. Monetização: plano Pro, licenciamento para equipas
2. Expansão para Windows (opcional) e suporte enterprise

---

## 9. Plano de Testes

### 9.1 Testes Unitários
- FlowStore
- MapList (add/remove/toggle/reorder)
- Parser de mock files

### 9.2 Testes de Integração
- Servidor de teste (`httptest`) valida:
  - Map Local/Remote
  - Leituras de mock
  - CA trust
  - Keybindings e overlay

### 9.3 Testes Manuais
- Seguir wireframes para Map Local/Remote
- Testar lista/pesquisa
- Validar reordenação
- Confirmar atalhos `H` e `?`

---

## 10. Deploy e Distribuição

### Build
```bash
# Linux/macOS
go build -o proxy-tui ./cmd/proxy-tui
```

### Execução
```bash
./proxy-tui --config ~/.proxy-tui/config.yml
```

### Estrutura de Diretórios
```
~/.proxy-tui/
├── config.yml      # Configuração principal
├── maps.json       # Regras de mapping
├── ca.key          # Chave privada CA
├── ca.crt          # Certificado CA
└── mocks/          # Ficheiros mock
    └── *.mock
```

---

## 11. Tickets Iniciais

### Ticket 1 - Script CA
Automatizar geração/instalação certificado raiz e trust store Linux/macOS com validações e fallback passivo.

### Ticket 2 - Painel Requests
Lista central com hh:mm:ss.ms, método, client e HTTP code, filtros e atalhos. Inclui toggle filtros e whichkey.

---

## 12. Checklist de Preparação

- [ ] Gerar certificado raiz e instalar nos dispositivos alvo
- [ ] Confirmar confiança em navegadores (https://127.0.0.1)
- [ ] Testar interceptação com curl/wget antes de usar apps mobile
- [ ] Validar hipóteses (fluxo de valor, mercado)
- [ ] Obter feedback dos stakeholders
- [ ] Aprovar milestones e roadmap antes de kick-off

# Zyr Git

Um jeito mais simples de trabalhar com Git e GitHub. Os comandos continuam no terminal; o dashboard abre como aplicativo pelo menu Iniciar do Windows. Nele você pode consultar a versão instalada, navegar pelo guia e copiar os comandos.

```powershell
zyr git commit
```

Projeto de João Vital/Jovenzinho.

## Instalação

No Windows, execute:

```text
dist/ZyrGit-Setup.exe
```

O instalador identifica se precisa instalar, atualizar ou reparar o aplicativo e pede confirmação antes de fazer qualquer alteração. Ele cria o atalho **Zyr Git** no menu Iniciar. Depois da instalação, abra um novo terminal para usar os comandos.

Se o Windows mostrar o SmartScreen, isso acontece porque o executável ainda não possui assinatura digital.

## Uso

Abra **Zyr Git** pelo menu Iniciar para ver a versão instalada, a última atualização e a documentação dentro do aplicativo.

Entre na pasta do projeto que deseja enviar e execute:

```powershell
cd C:\caminho\do\projeto
zyr git commit
```

Na primeira execução, o Zyr verifica o Git, a identidade do usuário, o repositório local e o remote `origin`. Se o Git não estiver instalado, ele pode instalar após sua confirmação.

Quando não existe um `.gitignore`, o Zyr cria um modelo genérico automaticamente, que ignora a pasta `.vscode/` por completo. Um arquivo já existente nunca é alterado; se ele foi criado por uma versão anterior, adicione `.vscode/` manualmente para ignorar as configurações do editor.

Depois da configuração inicial, basta informar a mensagem do commit. O programa executa o equivalente a:

```text
git add .
git commit -m "mensagem"
git push
```

Se não houver alterações, ele encerra sem criar um commit vazio.

## Consultar o status

Na pasta do projeto, execute:

```powershell
zyr git status
```

O comando mostra a branch, os arquivos preparados, modificados, novos, excluídos ou em conflito, e a quantidade de commits para enviar ou receber. Ele não altera os arquivos nem executa `fetch`; a comparação com o upstream usa a última sincronização local do Git. Fora de um repositório, o Zyr informa o problema sem criar um projeto.

## Resetar o histórico

Para substituir o histórico da branch atual por um único commit com o estado atual dos arquivos:

```powershell
zyr git reset-history
```

Antes de alterar o repositório, o Zyr mostra o nome, a branch e o remote e pede uma confirmação explícita. Ao confirmar, os commits anteriores da branch são substituídos e a nova história é enviada com push forçado.

## Criar um repositório no GitHub

```powershell
zyr git add-repo
```

O comando pergunta o nome, a descrição, a visibilidade e se o novo repositório deve receber um README e um `.gitignore`. Se você escolher o `.gitignore`, também poderá selecionar um dos templates oficiais disponíveis no GitHub.

Antes de criar, o Zyr mostra um resumo e pede confirmação. O repositório é criado na conta autenticada, sem licença e sem alterar, clonar ou enviar arquivos da pasta atual.

Instalação do GitHub CLI e login são conduzidos pelo próprio comando quando necessário. Você não precisa executar comandos do `gh` manualmente.

## Excluir um repositório do GitHub

```powershell
zyr git delete-repo
```

O Zyr usa o GitHub CLI (`gh`) para mostrar os repositórios disponíveis. Escolha um número, confira os detalhes e digite o nome completo, como `JoaoVitalPortugal/projeto`, para confirmar.

Essa ação exclui permanentemente o repositório remoto, mas não altera nenhum arquivo ou repositório local. O comando só prossegue quando a conta autenticada possui permissão administrativa.

Se o GitHub CLI não estiver instalado no Windows, o Zyr pode instalá-lo após pedir autorização. Se você ainda não estiver autenticado, o próprio comando oferece abrir o login oficial do GitHub no navegador.

Antes de mostrar os repositórios, o Zyr também verifica a permissão `delete_repo`. Quando ela não existe, explica o que ela permite, pede sua confirmação e abre o fluxo oficial do GitHub para autorizá-la. O comando só continua depois de confirmar o login e a permissão.

Você não precisa executar nenhum comando do `gh` manualmente. Basta iniciar `zyr git delete-repo` e responder às confirmações exibidas pelo Zyr.

## Atualizações do Zyr Git e do GitHub CLI

Ao executar um comando `zyr git`, o Zyr verifica se há uma versão mais recente publicada nas Releases oficiais. Quando há, abre uma pequena janela com o progresso, instala a atualização e continua o comando original. Se a rede falhar, o comando continua com a versão instalada. Ao abrir o dashboard, a mesma verificação acontece; após uma atualização, a janela fecha e o dashboard não reabre automaticamente.

No Windows, execute `zyr git update-gh` para atualizar o `gh`. O Zyr mostra a versão instalada, pede confirmação e usa o gerenciador de pacotes associado à instalação (WinGet, Chocolatey ou Scoop). Se o `gh` ainda não estiver instalado, o comando pode instalá-lo. Essa atualização é independente de `zyr git reset-history`.

## GitHub

Para usar `zyr git commit` com um projeto existente, informe a URL do repositório remoto quando o programa solicitar. Para criar um repositório novo, use `zyr git add-repo`.

Senhas e tokens não são pedidos nem armazenados. A autenticação dos commits continua sendo feita pelo Git, enquanto a criação e a exclusão de repositórios usam a sessão do GitHub CLI.

## Sobre o projeto

O Zyr Git é um projeto independente. **Zyr** é o nome usado por João Vital/Jovenzinho em seus próprios projetos e não indica integração com outra ferramenta.

## Desinstalação

Abra **Configurações > Aplicativos > Aplicativos instalados**, procure por **Zyr Git** e selecione **Desinstalar**.

## Compilar

Requisitos: Go 1.24 ou superior e PowerShell.

```powershell
.\scripts\build.ps1
```

Os testes são executados durante o build. O resultado final é um único instalador:

```text
dist/ZyrGit-Setup.exe
```

## Publicar uma versão

Atualize `VERSION` e envie uma tag correspondente, por exemplo `v0.7.0`. O workflow de release executa os testes no Windows, compila o instalador e publica `ZyrGit-Setup.exe` e `SHA256SUMS.txt` nas Releases do repositório. O atualizador automático usa apenas a Release estável mais recente com o instalador e digest SHA-256 disponíveis. A primeira versão com atualização automática precisa ser instalada manualmente uma vez.

# Zyr Git Commit

Um jeito mais simples de trabalhar com Git e GitHub pelo terminal.

```powershell
zyr git commit
```

Projeto de João Vital/Jovenzinho.

## Instalação

No Windows, execute:

```text
dist/ZyrGitCommit-Setup.exe
```

O instalador identifica se precisa instalar, atualizar ou reparar o aplicativo e pede confirmação antes de fazer qualquer alteração. Depois da instalação, abra um novo terminal.

Se o Windows mostrar o SmartScreen, isso acontece porque o executável ainda não possui assinatura digital.

## Uso

Entre na pasta do projeto que deseja enviar e execute:

```powershell
cd C:\caminho\do\projeto
zyr git commit
```

Na primeira execução, o Zyr verifica o Git, a identidade do usuário, o repositório local e o remote `origin`. Se o Git não estiver instalado, ele pode instalar após sua confirmação.

Quando não existe um `.gitignore`, o Zyr cria um modelo genérico automaticamente. Um arquivo já existente nunca é alterado.

Depois da configuração inicial, basta informar a mensagem do commit. O programa executa o equivalente a:

```text
git add .
git commit -m "mensagem"
git push
```

Se não houver alterações, ele encerra sem criar um commit vazio.

## Resetar o histórico

Para substituir o histórico da branch atual por um único commit com o estado atual dos arquivos:

```powershell
zyr git reset-history
```

Antes de alterar o repositório, o Zyr mostra o nome, a branch e o remote e pede uma confirmação explícita. Ao confirmar, os commits anteriores da branch são substituídos e a nova história é enviada com push forçado.

## Excluir um repositório do GitHub

```powershell
zyr git delete-repo
```

O Zyr usa o GitHub CLI (`gh`) para mostrar os repositórios disponíveis. Escolha um número, confira os detalhes e digite o nome completo, como `JoaoVitalPortugal/projeto`, para confirmar.

Essa ação exclui permanentemente o repositório remoto, mas não altera nenhum arquivo ou repositório local. O comando só prossegue quando a conta autenticada possui permissão administrativa.

Se o GitHub CLI não estiver instalado no Windows, o Zyr pode instalá-lo após pedir autorização. Se você ainda não estiver autenticado, o próprio comando oferece abrir o login oficial do GitHub no navegador.

Antes de mostrar os repositórios, o Zyr também verifica a permissão `delete_repo`. Quando ela não existe, explica o que ela permite, pede sua confirmação e abre o fluxo oficial do GitHub para autorizá-la. O comando só continua depois de confirmar o login e a permissão.

Você não precisa executar nenhum comando do `gh` manualmente. Basta iniciar `zyr git delete-repo` e responder às confirmações exibidas pelo Zyr.

## GitHub

O Zyr não cria o repositório no GitHub. Crie o repositório remoto primeiro e informe a URL quando o programa solicitar.

Senhas e tokens não são pedidos nem armazenados. A autenticação dos commits continua sendo feita pelo Git, e a exclusão de repositórios usa a sessão do GitHub CLI.

## Sobre o projeto

O Zyr Git Commit é um projeto independente. **Zyr** é o nome usado por João Vital/Jovenzinho em seus próprios projetos e não indica integração com outra ferramenta.

## Desinstalação

Abra **Configurações > Aplicativos > Aplicativos instalados**, procure por **Zyr Git Commit** e selecione **Desinstalar**.

## Compilar

Requisitos: Go 1.24 ou superior e PowerShell.

```powershell
.\scripts\build.ps1 -Version 0.4.1
```

Os testes são executados durante o build. O resultado final é um único instalador:

```text
dist/ZyrGitCommit-Setup.exe
```

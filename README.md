# Zyr Git Commit

Um jeito mais simples de adicionar alterações, criar um commit e enviar tudo para o repositório remoto.

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

## GitHub

O Zyr não cria o repositório no GitHub. Crie o repositório remoto primeiro e informe a URL quando o programa solicitar.

Senhas e tokens não são pedidos nem armazenados. A autenticação continua sendo feita pelo próprio Git, pelo Git Credential Manager ou pelas suas chaves SSH.

## Sobre o projeto

O Zyr Git Commit é um projeto independente. **Zyr** é o nome usado por João Vital/Jovenzinho em seus próprios projetos e não indica integração com outra ferramenta.

## Desinstalação

Abra **Configurações > Aplicativos > Aplicativos instalados**, procure por **Zyr Git Commit** e selecione **Desinstalar**.

## Compilar

Requisitos: Go 1.24 ou superior e PowerShell.

```powershell
.\scripts\build.ps1 -Version 0.2.1
```

Os testes são executados durante o build. O resultado final é um único instalador:

```text
dist/ZyrGitCommit-Setup.exe
```

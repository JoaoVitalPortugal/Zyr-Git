$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationFramework
$data = Get-Content -LiteralPath $env:ZYR_GIT_DASHBOARD_FILE -Raw | ConvertFrom-Json
$xaml = @'
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation" xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml" Title="Zyr Git" Width="1040" Height="720" MinWidth="800" MinHeight="540" WindowStartupLocation="CenterScreen" Background="#0B1020" Foreground="#E6EDF7" FontFamily="Segoe UI">
  <Window.Resources>
    <Style TargetType="TextBlock"><Setter Property="Foreground" Value="#E6EDF7"/></Style>
    <Style x:Key="NavButton" TargetType="Button">
      <Setter Property="Foreground" Value="#9EABC8"/><Setter Property="Background" Value="Transparent"/><Setter Property="BorderThickness" Value="0"/><Setter Property="Padding" Value="15,11"/><Setter Property="HorizontalContentAlignment" Value="Left"/><Setter Property="Cursor" Value="Hand"/><Setter Property="FontSize" Value="14"/>
      <Style.Triggers><Trigger Property="IsMouseOver" Value="True"><Setter Property="Background" Value="#19233B"/><Setter Property="Foreground" Value="#E6EDF7"/></Trigger></Style.Triggers>
    </Style>
    <Style x:Key="CopyButton" TargetType="Button">
      <Setter Property="Foreground" Value="#B9D9FF"/><Setter Property="Background" Value="#1C3152"/><Setter Property="BorderBrush" Value="#30517E"/><Setter Property="BorderThickness" Value="1"/><Setter Property="Padding" Value="12,5"/><Setter Property="Cursor" Value="Hand"/><Setter Property="FontSize" Value="12"/>
      <Style.Triggers><Trigger Property="IsMouseOver" Value="True"><Setter Property="Background" Value="#28456E"/></Trigger></Style.Triggers>
    </Style>
    <Style x:Key="Command" TargetType="TextBlock"><Setter Property="Foreground" Value="#8BC7FF"/><Setter Property="FontFamily" Value="Consolas"/><Setter Property="FontSize" Value="15"/></Style>
    <Style x:Key="Body" TargetType="TextBlock"><Setter Property="Foreground" Value="#AFBED8"/><Setter Property="FontSize" Value="14"/><Setter Property="TextWrapping" Value="Wrap"/><Setter Property="LineHeight" Value="22"/></Style>
    <Style x:Key="Eyebrow" TargetType="TextBlock"><Setter Property="Foreground" Value="#6CB6FF"/><Setter Property="FontSize" Value="11"/><Setter Property="FontWeight" Value="SemiBold"/></Style>
    <Style x:Key="Rule" TargetType="Border"><Setter Property="Height" Value="1"/><Setter Property="Background" Value="#26314B"/></Style>
    <Style TargetType="ScrollBar">
      <Setter Property="Width" Value="9"/>
      <Setter Property="Template"><Setter.Value>
        <ControlTemplate TargetType="ScrollBar">
          <Grid Background="#0B1020">
            <Track Name="PART_Track" IsDirectionReversed="True">
              <Track.Thumb>
                <Thumb>
                  <Thumb.Template>
                    <ControlTemplate TargetType="Thumb">
                      <Border Background="#365EA8" CornerRadius="4" Margin="2,0"/>
                    </ControlTemplate>
                  </Thumb.Template>
                </Thumb>
              </Track.Thumb>
            </Track>
          </Grid>
        </ControlTemplate>
      </Setter.Value></Setter>
    </Style>
  </Window.Resources>
  <Grid>
    <Grid.RowDefinitions><RowDefinition Height="76"/><RowDefinition Height="*"/></Grid.RowDefinitions>
    <Border Background="#0A0F1E" BorderBrush="#202C45" BorderThickness="0,0,0,1" Padding="30,0">
      <Grid>
        <Grid.ColumnDefinitions><ColumnDefinition Width="Auto"/><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
        <Border Width="36" Height="36" CornerRadius="9" Background="#1B3456" BorderBrush="#396A9F" BorderThickness="1" VerticalAlignment="Center">
          <TextBlock Text="Z" FontSize="19" FontWeight="Bold" Foreground="#84C5FF" HorizontalAlignment="Center" VerticalAlignment="Center"/>
        </Border>
        <StackPanel Grid.Column="1" Orientation="Horizontal" VerticalAlignment="Center" Margin="14,0,0,0">
          <TextBlock Text="Zyr Git" FontSize="21" FontWeight="SemiBold"/>
          <Border Width="1" Height="18" Background="#32405C" Margin="17,0"/>
          <TextBlock Text="Guia de comandos" FontSize="13" Foreground="#9EABC8" VerticalAlignment="Center"/>
        </StackPanel>
        <Border Grid.Column="2" Background="#172640" CornerRadius="7" Padding="11,5" VerticalAlignment="Center">
          <TextBlock Name="HeaderVersion" Foreground="#A8D7FF" FontFamily="Consolas" FontSize="12"/>
        </Border>
      </Grid>
    </Border>
    <Grid Grid.Row="1">
      <Grid.ColumnDefinitions><ColumnDefinition Width="205"/><ColumnDefinition Width="*"/></Grid.ColumnDefinitions>
      <Border Background="#0C1425" BorderBrush="#202C45" BorderThickness="0,0,1,0">
        <Grid Margin="14,28,14,20">
          <Grid.RowDefinitions><RowDefinition Height="Auto"/><RowDefinition Height="*"/><RowDefinition Height="Auto"/></Grid.RowDefinitions>
          <StackPanel>
            <TextBlock Text="NAVEGAR" Foreground="#657795" FontSize="11" FontWeight="SemiBold" Margin="15,0,0,15"/>
            <Button Name="NavOverview" Content="Visão geral" Style="{StaticResource NavButton}" Foreground="#E6EDF7" Background="#192C49"/>
            <Button Name="NavCommit" Content="Dia a dia" Style="{StaticResource NavButton}" Margin="0,3,0,0"/>
            <Button Name="NavGitHub" Content="Repositórios" Style="{StaticResource NavButton}" Margin="0,3,0,0"/>
            <Button Name="NavTools" Content="Manutenção" Style="{StaticResource NavButton}" Margin="0,3,0,0"/>
          </StackPanel>
          <StackPanel Grid.Row="2" Margin="14,0,0,0">
            <Border Style="{StaticResource Rule}" Margin="0,0,0,13"/>
            <TextBlock Text="João Vital / Jovenzinho" Foreground="#657795" FontSize="11"/>
          </StackPanel>
        </Grid>
      </Border>
      <ScrollViewer Name="DocumentScroll" Grid.Column="1" VerticalScrollBarVisibility="Auto" HorizontalScrollBarVisibility="Disabled" Background="#0B1020">
        <StackPanel MaxWidth="770" Margin="42,36,42,48">
          <StackPanel Name="SectionOverview">
            <TextBlock Text="VISÃO GERAL" Style="{StaticResource Eyebrow}"/>
            <TextBlock Text="Versão e instalação" FontSize="27" FontWeight="SemiBold" Margin="0,8,0,20"/>
            <Border Background="#101B30" BorderBrush="#263A5A" BorderThickness="1" CornerRadius="9" Padding="22,19">
              <Grid>
                <Grid.ColumnDefinitions><ColumnDefinition Width="1.1*"/><ColumnDefinition Width="1.2*"/><ColumnDefinition Width="1.2*"/></Grid.ColumnDefinitions>
                <StackPanel><TextBlock Text="INSTALADA" Foreground="#8296B4" FontSize="10" FontWeight="SemiBold"/><TextBlock Name="InstalledVersion" FontSize="22" FontWeight="SemiBold" Margin="0,7,0,0"/></StackPanel>
                <StackPanel Grid.Column="1"><TextBlock Text="INSTALADA EM" Foreground="#8296B4" FontSize="10" FontWeight="SemiBold"/><TextBlock Name="InstalledDate" FontSize="15" Margin="0,11,0,0"/></StackPanel>
                <StackPanel Grid.Column="2"><TextBlock Text="NO GITHUB" Foreground="#8296B4" FontSize="10" FontWeight="SemiBold"/><TextBlock Name="LatestVersion" FontSize="15" Margin="0,9,0,0"/><TextBlock Name="PublishedDate" Foreground="#8296B4" FontSize="11" Margin="0,3,0,0"/></StackPanel>
              </Grid>
            </Border>
            <TextBlock Text="O Zyr procura uma nova versão antes de executar os comandos. Se encontrar, instala e continua de onde você parou." Style="{StaticResource Body}" Margin="1,16,0,0"/>
          </StackPanel>

          <StackPanel Name="SectionCommit" Margin="0,47,0,0">
            <TextBlock Text="DIA A DIA" Style="{StaticResource Eyebrow}"/>
            <TextBlock Text="Enviar alterações" FontSize="25" FontWeight="SemiBold" Margin="0,8,0,8"/>
            <TextBlock Text="Abra o terminal na pasta do projeto e execute:" Style="{StaticResource Body}" Margin="0,0,0,16"/>
            <Border Background="#111D33" BorderBrush="#2A3D5E" BorderThickness="1" CornerRadius="8" Padding="16,12">
              <Grid>
                <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
                <TextBlock Text="zyr git commit" Style="{StaticResource Command}" VerticalAlignment="Center"/>
                <Button Name="CopyCommit" Grid.Column="1" Tag="zyr git commit" Content="Copiar" Style="{StaticResource CopyButton}"/>
              </Grid>
            </Border>
            <Grid Margin="0,20,0,0">
              <Grid.ColumnDefinitions><ColumnDefinition Width="1*"/><ColumnDefinition Width="1*"/></Grid.ColumnDefinitions>
              <StackPanel Margin="0,0,17,0">
                <TextBlock Text="Na primeira vez" FontSize="15" FontWeight="SemiBold" Margin="0,0,0,6"/>
                <TextBlock Text="O Zyr confere Git, sua identidade e o remote origin. Se faltar algo, ele orienta a configuração." Style="{StaticResource Body}"/>
              </StackPanel>
              <StackPanel Grid.Column="1" Margin="17,0,0,0">
                <TextBlock Text="Depois" FontSize="15" FontWeight="SemiBold" Margin="0,0,0,6"/>
                <TextBlock Text="Adiciona os arquivos, pede a mensagem, cria o commit e faz push. Sem alterações, não cria commit vazio." Style="{StaticResource Body}"/>
              </StackPanel>
            </Grid>
            <Border Background="#182338" BorderBrush="#364567" BorderThickness="0,0,0,0" CornerRadius="6" Padding="12,10" Margin="0,20,0,0">
              <TextBlock Text="Antes de confirmar todos os arquivos, confira o .gitignore do projeto." Foreground="#C7D3E8" FontSize="13" TextWrapping="Wrap"/>
            </Border>
            <Border Style="{StaticResource Rule}" Margin="0,29,0,22"/>
            <TextBlock Text="Conferir o projeto" FontSize="19" FontWeight="SemiBold" Margin="0,0,0,13"/>
            <Border Background="#111D33" BorderBrush="#2A3D5E" BorderThickness="1" CornerRadius="8" Padding="16,12">
              <Grid>
                <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
                <TextBlock Text="zyr git status" Style="{StaticResource Command}" VerticalAlignment="Center"/>
                <Button Name="CopyStatus" Grid.Column="1" Tag="zyr git status" Content="Copiar" Style="{StaticResource CopyButton}"/>
              </Grid>
            </Border>
            <TextBlock Text="Mostra a branch, os arquivos alterados e os commits para enviar ou receber. A comparação usa a última sincronização local do Git; o comando não faz fetch." Style="{StaticResource Body}" Margin="0,12,0,0"/>
          </StackPanel>

          <StackPanel Name="SectionGitHub" Margin="0,54,0,0">
            <Border Style="{StaticResource Rule}" Margin="0,0,0,28"/>
            <TextBlock Text="GITHUB" Style="{StaticResource Eyebrow}"/>
            <TextBlock Text="Repositórios" FontSize="25" FontWeight="SemiBold" Margin="0,8,0,20"/>
            <Grid>
              <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
              <TextBlock Text="zyr git add-repo" Style="{StaticResource Command}" VerticalAlignment="Center"/>
              <Button Name="CopyAddRepo" Grid.Column="1" Tag="zyr git add-repo" Content="Copiar" Style="{StaticResource CopyButton}"/>
            </Grid>
            <TextBlock Text="Cria um repositório na sua conta e configura o remote do projeto local. Para enviar os arquivos, faça um commit." Style="{StaticResource Body}" Margin="0,10,0,19"/>
            <Border Style="{StaticResource Rule}" Margin="0,0,0,19"/>
            <Grid>
              <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
              <TextBlock Text="zyr git delete-repo" Style="{StaticResource Command}" VerticalAlignment="Center"/>
              <Button Name="CopyDeleteRepo" Grid.Column="1" Tag="zyr git delete-repo" Content="Copiar" Style="{StaticResource CopyButton}"/>
            </Grid>
            <TextBlock Text="Escolhe e exclui um repositório remoto da sua conta. O comando mostra o alvo e pede confirmação explícita." Style="{StaticResource Body}" Margin="0,10,0,0"/>
          </StackPanel>

          <StackPanel Name="SectionTools" Margin="0,54,0,0">
            <Border Style="{StaticResource Rule}" Margin="0,0,0,28"/>
            <TextBlock Text="FERRAMENTAS" Style="{StaticResource Eyebrow}"/>
            <TextBlock Text="Manutenção" FontSize="25" FontWeight="SemiBold" Margin="0,8,0,20"/>
            <Grid>
              <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
              <TextBlock Text="zyr git reset-history" Style="{StaticResource Command}" VerticalAlignment="Center"/>
              <Button Name="CopyReset" Grid.Column="1" Tag="zyr git reset-history" Content="Copiar" Style="{StaticResource CopyButton}"/>
            </Grid>
            <TextBlock Text="Mantém os arquivos atuais e troca os commits anteriores da branch por um único commit. Após sua confirmação, envia a nova história com push forçado." Style="{StaticResource Body}" Margin="0,10,0,19"/>
            <Border Style="{StaticResource Rule}" Margin="0,0,0,19"/>
            <Grid>
              <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="Auto"/></Grid.ColumnDefinitions>
              <TextBlock Text="zyr git update-gh" Style="{StaticResource Command}" VerticalAlignment="Center"/>
              <Button Name="CopyUpdateGh" Grid.Column="1" Tag="zyr git update-gh" Content="Copiar" Style="{StaticResource CopyButton}"/>
            </Grid>
            <TextBlock Text="Atualiza o GitHub CLI (gh), usado para criar e excluir repositórios. Essa ferramenta tem atualização separada do Zyr Git." Style="{StaticResource Body}" Margin="0,10,0,0"/>
          </StackPanel>
        </StackPanel>
      </ScrollViewer>
    </Grid>
  </Grid>
</Window>
'@
$reader = New-Object System.Xml.XmlNodeReader ([xml]$xaml)
$window = [Windows.Markup.XamlReader]::Load($reader)
if (Test-Path -LiteralPath $env:ZYR_GIT_ICON) { $window.Icon = [Windows.Media.Imaging.BitmapFrame]::Create([uri]$env:ZYR_GIT_ICON) }
$window.FindName('HeaderVersion').Text = 'v' + $data.version
$window.FindName('InstalledVersion').Text = $data.version
$window.FindName('LatestVersion').Text = if ($data.latest) { $data.latest } else { 'Ainda não publicada' }
$window.FindName('InstalledDate').Text = if ($data.installedAt) { ([datetime]$data.installedAt).ToString('dd/MM/yyyy') } else { 'Não disponível' }
$window.FindName('PublishedDate').Text = if ($data.publishedAt) { 'Publicada em ' + ([datetime]$data.publishedAt).ToString('dd/MM/yyyy') } else { '' }
foreach ($item in @(
  @('NavOverview', 'SectionOverview'),
  @('NavCommit', 'SectionCommit'),
  @('NavGitHub', 'SectionGitHub'),
  @('NavTools', 'SectionTools')
)) {
  $button = $window.FindName($item[0])
  $section = $window.FindName($item[1])
  $button.Add_Click({ $section.BringIntoView() }.GetNewClosure())
}
foreach ($name in @('CopyCommit', 'CopyStatus', 'CopyAddRepo', 'CopyDeleteRepo', 'CopyReset', 'CopyUpdateGh')) {
  $window.FindName($name).Add_Click({
    param($sender, $eventArgs)
    [Windows.Clipboard]::SetText([string]$sender.Tag)
    $sender.Content = 'Copiado'
  })
}
if ($env:ZYR_GIT_UI_SCREENSHOT) {
  $window.Add_ContentRendered({
    if ($env:ZYR_GIT_UI_SCREENSHOT_SECTION) {
      $window.FindName($env:ZYR_GIT_UI_SCREENSHOT_SECTION).BringIntoView()
      $window.UpdateLayout()
    }
    $width = [int]$window.Content.ActualWidth
    $height = [int]$window.Content.ActualHeight
    $bitmap = New-Object Windows.Media.Imaging.RenderTargetBitmap($width,$height,96,96,[Windows.Media.PixelFormats]::Pbgra32)
    $bitmap.Render($window.Content)
    $encoder = New-Object Windows.Media.Imaging.PngBitmapEncoder
    $encoder.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($bitmap))
    $stream = [IO.File]::Create($env:ZYR_GIT_UI_SCREENSHOT)
    try { $encoder.Save($stream) } finally { $stream.Dispose() }
  })
}
if ($env:ZYR_GIT_UI_SMOKE -eq '1') {
  $timer = New-Object Windows.Threading.DispatcherTimer
  $timer.Interval = [TimeSpan]::FromMilliseconds(600)
  $timer.Add_Tick({ $timer.Stop(); $window.Close() })
  $timer.Start()
}
$null = $window.ShowDialog()

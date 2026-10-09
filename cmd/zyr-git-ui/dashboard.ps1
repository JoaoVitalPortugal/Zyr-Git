$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationFramework
$data = Get-Content -LiteralPath $env:ZYR_GIT_DASHBOARD_FILE -Raw | ConvertFrom-Json
$xaml = @'
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation" xmlns:x="http://schemas.microsoft.com/winfx/2006/xaml" Title="Zyr Git" Width="960" Height="680" MinWidth="720" MinHeight="500" WindowStartupLocation="CenterScreen" Background="#0B1020" Foreground="#E6EDF7" FontFamily="Segoe UI">
  <Window.Resources>
    <Style TargetType="TextBlock"><Setter Property="Foreground" Value="#E6EDF7"/></Style>
    <Style TargetType="Border" x:Key="Card"><Setter Property="Background" Value="#0F1629"/><Setter Property="BorderBrush" Value="#1B2440"/><Setter Property="BorderThickness" Value="1"/><Setter Property="CornerRadius" Value="10"/><Setter Property="Padding" Value="20"/><Setter Property="Margin" Value="0,0,0,14"/></Style>
    <Style TargetType="ScrollBar"><Setter Property="Width" Value="9"/><Setter Property="Template"><Setter.Value><ControlTemplate TargetType="ScrollBar"><Grid Background="#0B1020"><Track Name="PART_Track" IsDirectionReversed="True"><Track.Thumb><Thumb Background="#365EA8" BorderThickness="0"/></Track.Thumb></Track></Grid></ControlTemplate></Setter.Value></Setter></Style>
  </Window.Resources>
  <Grid>
    <Grid.RowDefinitions><RowDefinition Height="Auto"/><RowDefinition Height="*"/></Grid.RowDefinitions>
    <Border Background="#0A0E1B" BorderBrush="#1B2440" BorderThickness="0,0,0,1" Padding="32,22">
      <StackPanel>
        <TextBlock Text="ZYR GIT" Foreground="#6CB6FF" FontSize="13" FontWeight="Bold"/>
        <TextBlock Text="Seu Git, em um lugar só." FontSize="26" FontWeight="SemiBold" Margin="0,7,0,0"/>
      </StackPanel>
    </Border>
    <ScrollViewer Grid.Row="1" VerticalScrollBarVisibility="Auto" Padding="32,25,24,25" Background="#0B1020">
      <StackPanel MaxWidth="880">
        <Border Style="{StaticResource Card}">
          <Grid>
            <Grid.ColumnDefinitions><ColumnDefinition Width="*"/><ColumnDefinition Width="*"/><ColumnDefinition Width="*"/></Grid.ColumnDefinitions>
            <StackPanel><TextBlock Text="VERSÃO INSTALADA" Foreground="#8A94B5" FontSize="11"/><TextBlock Name="InstalledVersion" FontSize="21" FontWeight="SemiBold" Margin="0,7,0,0"/></StackPanel>
            <StackPanel Grid.Column="1"><TextBlock Text="ÚLTIMA ATUALIZAÇÃO" Foreground="#8A94B5" FontSize="11"/><TextBlock Name="InstalledDate" FontSize="16" Margin="0,10,0,0"/></StackPanel>
            <StackPanel Grid.Column="2"><TextBlock Text="VERSÃO PUBLICADA" Foreground="#8A94B5" FontSize="11"/><TextBlock Name="LatestVersion" FontSize="16" Margin="0,10,0,0"/><TextBlock Name="PublishedDate" Foreground="#8A94B5" FontSize="11" Margin="0,4,0,0"/></StackPanel>
          </Grid>
        </Border>
        <TextBlock Text="Como usar" FontSize="22" FontWeight="SemiBold" Margin="0,10,0,16"/>
        <Border Style="{StaticResource Card}"><StackPanel><TextBlock Text="01  Primeiros passos" Foreground="#E8B84A" FontSize="17" FontWeight="SemiBold"/><TextBlock Text="Abra um terminal na pasta do seu projeto. O Zyr Git verifica o Git, sua identidade, o repositório local e o remote origin quando você usa o comando de commit pela primeira vez." TextWrapping="Wrap" Margin="0,12,0,0" LineHeight="23"/></StackPanel></Border>
        <Border Style="{StaticResource Card}"><StackPanel><TextBlock Text="02  Enviar alterações" Foreground="#E8B84A" FontSize="17" FontWeight="SemiBold"/><TextBlock Text="zyr git commit" Foreground="#6CB6FF" FontFamily="Consolas" FontSize="16" Margin="0,12,0,8"/><TextBlock Text="O Zyr adiciona as alterações, pede uma mensagem, cria o commit e envia para o remote origin. Confira o .gitignore antes de confirmar a adição de todos os arquivos." TextWrapping="Wrap" LineHeight="23"/></StackPanel></Border>
        <Border Style="{StaticResource Card}"><StackPanel><TextBlock Text="03  Repositórios no GitHub" Foreground="#E8B84A" FontSize="17" FontWeight="SemiBold"/><TextBlock Text="zyr git add-repo    •    zyr git delete-repo" Foreground="#6CB6FF" FontFamily="Consolas" FontSize="15" Margin="0,12,0,8"/><TextBlock Text="Crie um repositório remoto ou escolha um para excluir. A exclusão exige confirmação explícita e permissão administrativa na sua conta." TextWrapping="Wrap" LineHeight="23"/></StackPanel></Border>
        <Border Style="{StaticResource Card}"><StackPanel><TextBlock Text="04  Histórico e ferramentas" Foreground="#E8B84A" FontSize="17" FontWeight="SemiBold"/><TextBlock Text="zyr git reset-history    •    zyr git update-gh" Foreground="#6CB6FF" FontFamily="Consolas" FontSize="15" Margin="0,12,0,8"/><TextBlock Text="reset-history substitui os commits anteriores da branch por um novo commit com os arquivos atuais. update-gh atualiza o GitHub CLI; é separado da atualização do Zyr Git." TextWrapping="Wrap" LineHeight="23"/></StackPanel></Border>
        <TextBlock Text="Zyr Git  •  João Vital / Jovenzinho" Foreground="#8A94B5" FontSize="12" Margin="0,8,0,20"/>
      </StackPanel>
    </ScrollViewer>
  </Grid>
</Window>
'@
$reader = New-Object System.Xml.XmlNodeReader ([xml]$xaml)
$window = [Windows.Markup.XamlReader]::Load($reader)
if (Test-Path -LiteralPath $env:ZYR_GIT_ICON) { $window.Icon = [Windows.Media.Imaging.BitmapFrame]::Create([uri]$env:ZYR_GIT_ICON) }
$window.FindName('InstalledVersion').Text = $data.version
$window.FindName('LatestVersion').Text = if ($data.latest) { $data.latest } else { 'Ainda não publicada' }
$window.FindName('InstalledDate').Text = if ($data.installedAt) { ([datetime]$data.installedAt).ToString('dd/MM/yyyy') } else { 'Não disponível' }
$window.FindName('PublishedDate').Text = if ($data.publishedAt) { 'Publicada em ' + ([datetime]$data.publishedAt).ToString('dd/MM/yyyy') } else { '' }
if ($env:ZYR_GIT_UI_SCREENSHOT) {
  $window.Add_ContentRendered({
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

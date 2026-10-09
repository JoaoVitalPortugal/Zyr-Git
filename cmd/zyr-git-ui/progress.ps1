$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationFramework
$xaml = @'
<Window xmlns="http://schemas.microsoft.com/winfx/2006/xaml/presentation" Title="Atualizando Zyr Git" Width="420" Height="220" ResizeMode="NoResize" WindowStartupLocation="CenterScreen" Background="#0B1020" Foreground="#E6EDF7" FontFamily="Segoe UI" ShowInTaskbar="True">
  <Grid Margin="24" Background="#0B1020">
    <Grid.RowDefinitions><RowDefinition Height="Auto"/><RowDefinition Height="Auto"/><RowDefinition Height="Auto"/><RowDefinition Height="*"/></Grid.RowDefinitions>
    <TextBlock Text="ZYR GIT" Foreground="#6CB6FF" FontSize="12" FontWeight="Bold"/>
    <TextBlock Name="VersionText" Grid.Row="1" Text="Preparando atualização" FontSize="19" FontWeight="SemiBold" Margin="0,12,0,0"/>
    <TextBlock Name="StageText" Grid.Row="2" Text="Aguarde..." Foreground="#8A94B5" TextWrapping="Wrap" Margin="0,10,0,13"/>
    <ProgressBar Name="Progress" Grid.Row="3" Height="8" VerticalAlignment="Top" Minimum="0" Maximum="100" Background="#1B2440" Foreground="#E8B84A" IsIndeterminate="True"/>
  </Grid>
</Window>
'@
$reader = New-Object System.Xml.XmlNodeReader ([xml]$xaml)
$window = [Windows.Markup.XamlReader]::Load($reader)
if (Test-Path -LiteralPath $env:ZYR_GIT_ICON) { $window.Icon = [Windows.Media.Imaging.BitmapFrame]::Create([uri]$env:ZYR_GIT_ICON) }
$versionText = $window.FindName('VersionText')
$stageText = $window.FindName('StageText')
$bar = $window.FindName('Progress')
$timer = New-Object Windows.Threading.DispatcherTimer
$timer.Interval = [TimeSpan]::FromMilliseconds(180)
$timer.Add_Tick({
  try {
    $state = Get-Content -LiteralPath $env:ZYR_GIT_PROGRESS_FILE -Raw | ConvertFrom-Json
    $versionText.Text = 'Atualizando para ' + $state.version
    $stageText.Text = $state.stage
    $bar.IsIndeterminate = ($state.percent -lt 0)
    if ($state.percent -ge 0) { $bar.Value = $state.percent }
    if ($state.done) {
      $timer.Stop()
      $delay = if ($state.error) { 3500 } else { 600 }
      $closeTimer = New-Object Windows.Threading.DispatcherTimer
      $closeTimer.Interval = [TimeSpan]::FromMilliseconds($delay)
      $closeTimer.Add_Tick({ $closeTimer.Stop(); $window.Close() })
      $closeTimer.Start()
    }
  } catch {}
})
$timer.Start()
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
  $smokeTimer = New-Object Windows.Threading.DispatcherTimer
  $smokeTimer.Interval = [TimeSpan]::FromMilliseconds(600)
  $smokeTimer.Add_Tick({ $smokeTimer.Stop(); $window.Close() })
  $smokeTimer.Start()
}
$null = $window.ShowDialog()

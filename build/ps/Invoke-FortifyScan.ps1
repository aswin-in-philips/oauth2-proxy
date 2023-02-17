param(
    $FortifyProjectId = "edifoundation-oauth2proxy",
    $FortifyVersionId = "Main",
    $FortifyBuildId = "fortify_fl",
    $FortifyFprPath = "$PSScriptRoot\$FortifyProjectId.fpr",
    $PublishURL = "https://fortify.philips.com/ssc",
    [string]$PublishAuthToken,
    $RepositoryRoot = "$PSScriptRoot\..\..\",
    $Sourceanalyzer = "C:\Program Files\Fortify\Fortify_SCA_and_Apps_22.1.1\bin\sourceanalyzer.exe",
    $FortifyClient = "C:\Program Files\Fortify\Fortify_SCA_and_Apps_22.1.1\bin\fortifyclient.bat"
)

function Invoke-VswhereDownload (
    [string]$Uri = "https://github.com/microsoft/vswhere/releases/download/2.8.4/vswhere.exe",
    [string]$Outfile = "$PSScriptRoot\vswhere.exe"
) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Write-Output "downloading from '$Uri' to '$Outfile'"
    Write-Output "please wait..."
    Invoke-WebRequest -Uri $Uri -OutFile $Outfile
}

try {
    $RepositoryRoot = [System.IO.Path]::GetFullPath($RepositoryRoot)


    & sourceanalyzer -b $FortifyBuildId -clean -logfile "$PSScriptRoot\fortify-clean.txt"

    & sourceanalyzer  -Xmx8G -b $FortifyBuildId "$RepositoryRoot\contrib"
    & sourceanalyzer  -b $FortifyBuildId -show-files 
    & sourceanalyzer  -Xmx8G -b $FortifyBuildId  -Dcom.fortify.sca.Phase0HigherOrder.Languages=go -scan -f $FortifyFprPath -logfile "$PSScriptRoot\fortify-scan.txt"

    #Upload fpr reports to fortify server
    Write-Output "Uploading '$FortifyFprPath' of '$FortifyProjectId' with version '$FortifyVersionId' to '$PublishURL'"
    & cmd /c fortifyclient.bat -url $PublishURL -authtoken $PublishAuthToken uploadFPR -file $FortifyFprPath -application $FortifyProjectId -applicationVersion "$FortifyVersionId"

    Write-Output "Completed Fortify scan"
}
catch {
    $ErrorMessage = $_.Exception.Message
    Write-Error $ErrorMessage
    Write-Output $ErrorMessage
    exit -1
}

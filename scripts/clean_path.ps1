# clean_path.ps1 - Clean Node.js and Git entries from user PATH
$path = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($path) {
    $dirs = $path -split ';' | Where-Object {
        $d = $_.Trim()
        $d -ne '' -and
        $d -notlike '*nodejs*' -and
        $d -notlike '*npm*' -and
        $d -ne 'C:\Program Files\Git\cmd' -and
        $d -ne 'C:\Program Files\Git\bin'
    }
    $newPath = $dirs -join ';'
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host '       PATH cleaned.'
} else {
    Write-Host '       User PATH is empty, skip.'
}

# Broadcast env change
Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class Env{[DllImport("user32.dll")]public static extern IntPtr SendMessageTimeout(IntPtr h,uint m,IntPtr w,string l,uint f,uint t,out IntPtr r);public static void Broadcast(){IntPtr r;SendMessageTimeout((IntPtr)0xffff,0x001A,IntPtr.Zero,"Environment",0x0002,0x2710,out r);}}'
[Env]::Broadcast()

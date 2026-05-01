# SSH 快速连接测试环境

## 目标机器

| 项目 | 值 |
|------|-----|
| IP | `192.168.101.17` |
| 用户名 | `administrator` |
| 密码 | `123456` |
| OS | Windows 11 |
| 桌面路径 | `C:\Users\administrator\Desktop` |

## Python 一行连接

```python
import paramiko
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)
stdin, stdout, stderr = ssh.exec_command('whoami')
print(stdout.read().decode())
ssh.close()
```

## 常用操作

### 执行远程命令

```python
import paramiko

def run_remote(cmd, timeout=30):
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)
    stdin, stdout, stderr = ssh.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode('utf-8', errors='replace')
    err = stderr.read().decode('utf-8', errors='replace')
    ssh.close()
    return out, err

# 示例
out, err = run_remote(r'cmd /c "dir /b C:\Users\administrator\Desktop\*.exe"')
print(out)
```

### 上传文件到远程桌面

```python
import paramiko

def upload_to_desktop(local_path):
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)
    sftp = ssh.open_sftp()
    filename = local_path.split('\\')[-1].split('/')[-1]
    remote_path = f'C:\\Users\\administrator\\Desktop\\{filename}'
    sftp.put(local_path, remote_path)
    sftp.close()
    ssh.close()
    print(f'Uploaded: {local_path} -> {remote_path}')

upload_to_desktop(r'C:\Users\Administrator\Downloads\ccswitch-gui\ccswitch-gui.exe')
```

### 下载远程文件

```python
def download_from_desktop(filename, local_dir='.'):
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)
    sftp = ssh.open_sftp()
    remote_path = f'C:\\Users\\administrator\\Desktop\\{filename}'
    local_path = f'{local_dir}\\{filename}'
    sftp.get(remote_path, local_path)
    sftp.close()
    ssh.close()
    print(f'Downloaded: {remote_path} -> {local_path}')

download_from_desktop('screenshot.png', r'C:\Users\Administrator\Downloads')
```

### 查看 Claude 配置

```python
def check_claude_settings():
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)

    stdin, stdout, stderr = ssh.exec_command(
        r'type C:\Users\administrator\.claude\settings.json 2>nul'
    )
    print('=== settings.json ===')
    print(stdout.read().decode('utf-8', errors='replace'))

    stdin, stdout, stderr = ssh.exec_command(
        r'type C:\Users\administrator\.ccswitch\ccs.json 2>nul'
    )
    print('=== ccs.json ===')
    print(stdout.read().decode('utf-8', errors='replace'))

    ssh.close()

check_claude_settings()
```

### 远程清理测试环境

```python
def clean_test_env():
    """执行自动清理测试环境（无 pause，适合 SSH）"""
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect('192.168.101.17', username='administrator', password='123456', timeout=10)

    cmds = [
        r'call "C:\Program Files\nodejs\npm.cmd" uninstall -g @anthropic-ai/claude-code 2>nul',
        r'powershell -Command "Remove-Item -Recurse -Force ''C:\Program Files\nodejs'' -ErrorAction SilentlyContinue"',
        r'powershell -Command "Remove-Item -Recurse -Force ''C:\Program Files\Git'' -ErrorAction SilentlyContinue"',
        r'powershell -Command "Remove-Item -Recurse -Force (Join-Path $env:USERPROFILE ''.claude'') -ErrorAction SilentlyContinue"',
        r'powershell -Command "Remove-Item -Recurse -Force (Join-Path $env:USERPROFILE ''.ccswitch'') -ErrorAction SilentlyContinue"',
    ]
    for cmd in cmds:
        stdin, stdout, stderr = ssh.exec_command(cmd, timeout=60)
        stdout.read()  # drain

    print('Clean complete')
    ssh.close()

clean_test_env()
```

## OpenSSH 命令行（备选）

如果安装了 OpenSSH 客户端：

```bash
# 连接（需要手动输入密码）
ssh administrator@192.168.101.17

# 单命令执行
ssh administrator@192.168.101.17 "whoami"

# SCP 上传
scp ccswitch-gui.exe administrator@192.168.101.17:C:/Users/administrator/Desktop/
```

## 注意事项

- SSH 执行 `cmd /c` 命令时，路径中的 `\` 需要用原始字符串 `r'...'`
- PowerShell 命令在 SSH 下可能路径解析异常，优先用 `cmd /c`
- 远程执行耗时命令时注意设置 `timeout`，默认 30 秒可能不够
- `sftp.put/get` 大文件时注意网络超时

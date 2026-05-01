import paramiko, os

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect('192.168.101.17', username='Administrator', password='123456', timeout=10)
sftp = ssh.open_sftp()

desktop = 'C:/Users/Administrator/Desktop'
scripts_dir = os.path.join(os.path.dirname(__file__))

files_to_upload = [
    ('自动清理测试环境.bat', '自动清理测试环境.bat'),
    ('一键卸载Claude.bat', '一键卸载Claude.bat'),
    ('clean_path.ps1', 'clean_path.ps1'),
]

for local_name, remote_name in files_to_upload:
    local_path = os.path.join(scripts_dir, local_name)
    remote_path = desktop + '/' + remote_name

    with open(local_path, 'rb') as f:
        data = f.read()

    # bat 文件转 GBK 编码
    if local_name.endswith('.bat'):
        text = data.decode('utf-8')
        data = text.encode('gbk')
        print(f'{local_name}: UTF-8 -> GBK ({len(data)} bytes)')
    else:
        print(f'{local_name}: {len(data)} bytes (no conversion)')

    with sftp.open(remote_path, 'wb') as f:
        f.write(data)
    print(f'  uploaded to {remote_path}')

# 删除旧的清理测试环境.bat
try:
    sftp.remove(desktop + '/清理测试环境.bat')
    print('deleted old 清理测试环境.bat')
except:
    print('清理测试环境.bat not found, skip')

sftp.close()
ssh.close()
print('DONE')

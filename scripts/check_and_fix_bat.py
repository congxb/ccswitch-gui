import paramiko

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect('192.168.101.17', username='Administrator', password='123456', timeout=10)
sftp = ssh.open_sftp()

desktop = 'C:/Users/Administrator/Desktop'

for f in sftp.listdir(desktop):
    if not f.endswith('.bat'):
        continue
    fp = desktop + '/' + f
    with sftp.open(fp, 'rb') as fh:
        data = fh.read()

    # utf-8 decode
    try:
        text = data.decode('utf-8')
        print(f'{f}: UTF-8, {len(data)} bytes')
        # re-encode as GBK and upload
        gbk_data = text.encode('gbk')
        with sftp.open(fp, 'wb') as fh:
            fh.write(gbk_data)
        print(f'  -> converted to GBK, {len(gbk_data)} bytes')
    except UnicodeDecodeError:
        print(f'{f}: not UTF-8, skip')

sftp.close()
ssh.close()
print('DONE')

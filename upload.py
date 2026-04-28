import paramiko, sys

host = "64.118.128.179"
user = "root"
password = "jow@Qp0i"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(host, 22, user, password, look_for_keys=False, allow_agent=False, timeout=15, banner_timeout=30)
    print("Uploading...")
    sftp = client.open_sftp()
    sftp.put(r"C:\Users\1\new-api-lite\backend\new-api-lite-linux", "/opt/api-lite/backend/new-api-lite")
    sftp.close()

    stdin, stdout, _ = client.exec_command("chmod +x /opt/api-lite/backend/new-api-lite && systemctl restart api-lite && sleep 2 && systemctl status api-lite --no-pager 2>&1 | head -5", timeout=15, get_pty=True)
    ec = stdout.channel.recv_exit_status()
    print(stdout.read().decode(errors="replace"))
    print("Done!")
except Exception as e:
    print(f"Error: {e}")
finally:
    client.close()

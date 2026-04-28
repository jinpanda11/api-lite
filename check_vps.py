import paramiko, os, sys

os.environ['PYTHONIOENCODING'] = 'utf-8'
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

host = "64.118.128.179"
user = "root"
password = "jow@Qp0i"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

def run(cmd, timeout=60):
    stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout, get_pty=True)
    exit_status = stdout.channel.recv_exit_status()
    out = stdout.read().decode(errors="replace")
    return exit_status, out

try:
    client.connect(host, 22, user, password, look_for_keys=False, allow_agent=False, timeout=15, banner_timeout=30)
    ec, out = run("cd /opt/api-lite && git log --oneline -5")
    print(out.strip())
    print("\n---")
    ec2, out2 = run("cd /opt/api-lite && git status --short")
    print(out2.strip() if out2.strip() else "clean")
except Exception as e:
    print(f"FAIL: {e}")
finally:
    client.close()

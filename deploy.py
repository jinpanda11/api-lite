import paramiko, sys

host = "64.118.128.179"
user = "root"
password = "jow@Qp0i"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

def run(cmd, timeout=120):
    stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout, get_pty=True)
    exit_status = stdout.channel.recv_exit_status()
    out = stdout.read().decode(errors="replace")
    err = stderr.read().decode(errors="replace")
    return exit_status, out, err

try:
    client.connect(host, 22, user, password, look_for_keys=False, allow_agent=False, timeout=15, banner_timeout=30)
    print("OK")

    steps = [
        ("Go", "wget -q https://go.dev/dl/go1.23.0.linux-amd64.tar.gz -O /tmp/go.tar.gz && rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && ln -sf /usr/local/go/bin/go /usr/local/bin/go && go version"),
        ("Node", "curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt-get install -y nodejs && node -v"),
        ("Clone", "cd /opt && rm -rf api-lite && git clone https://github.com/jinpanda11/api-lite.git && cd api-lite && git log --oneline -1"),
        ("Frontend", "cd /opt/api-lite/frontend && npm install && npm run build"),
        ("Backend", "cd /opt/api-lite/backend && go build -ldflags='-s -w' -o new-api-lite ."),
        ("Start", "cd /opt/api-lite && cp backend/config.yaml.example backend/config.yaml && cd backend && nohup ./new-api-lite > /tmp/api.log 2>&1 & sleep 3 && ss -tlnp | grep 3000"),
    ]

    for name, cmd in steps:
        print(f"[{name}]", end=" ", flush=True)
        ec, out, err = run(cmd, timeout=300)
        lines = [l for l in out.strip().split("\n") if l.strip()]
        print(lines[-1] if lines else "OK" if ec == 0 else f"FAIL({ec})")
        if ec != 0:
            print("  ERR:", err.strip()[:200])

    print("DONE")
except Exception as e:
    print(f"FAIL: {e}")
finally:
    client.close()

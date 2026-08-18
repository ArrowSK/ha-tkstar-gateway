from pathlib import Path
import re

import yaml

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "tkstar_gateway"

required = [
    ROOT / "repository.yaml",
    ROOT / "README.md",
    ROOT / "LICENSE",
    ROOT / "NOTICE",
    ROOT / "SECURITY.md",
    ROOT / "THIRD_PARTY_LICENSES.md",
    APP / "config.yaml",
    APP / "Dockerfile",
    APP / "run.sh",
    APP / "apparmor.txt",
    APP / "go.mod",
    APP / "cmd/tkstar-gateway/main.go",
]
missing = [str(path.relative_to(ROOT)) for path in required if not path.exists()]
if missing:
    raise SystemExit("missing: " + ", ".join(missing))

cfg = yaml.safe_load((APP / "config.yaml").read_text())
assert cfg["slug"] == "tkstar_gateway"
assert cfg["ingress"] is True and cfg["ingress_port"] == 8099
assert "mqtt:need" in cfg["services"]
for port in ("5093/tcp", "5013/tcp", "5023/tcp"):
    assert port in cfg["ports"]
assert cfg["arch"] == ["aarch64", "amd64"]
assert cfg["image"] == "ghcr.io/arrowsk/ha-tkstar-gateway"

version = cfg["version"]
main = (APP / "cmd/tkstar-gateway/main.go").read_text()
match = re.search(r'const\s+version\s*=\s*"([^"]+)"', main)
assert match and match.group(1) == version

docker = (APP / "Dockerfile").read_text()
assert re.search(rf"ARG\s+BUILD_VERSION={re.escape(version)}(?:\s|$)", docker)

for path in ROOT.rglob("*"):
    if path.is_file() and path.suffix.lower() in {".har", ".pcap", ".pcapng"}:
        raise SystemExit("packet captures must not be committed")

print("Repository validation passed")

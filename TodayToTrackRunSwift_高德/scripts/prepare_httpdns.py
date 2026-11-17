#!/usr/bin/env python3
"""Inject SDK signing parameters into the built app without printing their values."""
import json
import os
from pathlib import Path
import plistlib
import re
import sys

source = Path(os.environ.get("RUI_HTTPDNS_CONFIG", str(Path(os.environ["SRCROOT"]) / "HTTPDNS.private.json"))).expanduser()
config = json.loads(source.read_text()) if source.is_file() else {}
sdk = config.get("httpdns", config)
account = str(sdk.get("account_id", sdk.get("accountId", ""))).strip()
secret = sdk.get("secret_key", sdk.get("secretKey", ""))
secret = secret.strip() if isinstance(secret, str) else ""
valid = account.isascii() and account.isdigit() and 0 < int(account) < 2**63 and bool(re.fullmatch(r"[0-9a-fA-F]{32}", secret))
if not valid and os.environ.get("CONFIGURATION", "").lower() == "release":
    sys.exit("error: Release requires Rui HTTPDNS accountId/secretKey in RUI_HTTPDNS_CONFIG or HTTPDNS.private.json")
target = Path(os.environ["TARGET_BUILD_DIR"]) / os.environ["UNLOCALIZED_RESOURCES_FOLDER_PATH"] / "RuiHTTPDNS.private.plist"
target.parent.mkdir(parents=True, exist_ok=True)
target.write_bytes(plistlib.dumps({"accountId": account, "secretKey": secret} if valid else {}))
target.chmod(0o600)
print("Rui HTTPDNS: private parameters injected" if valid else "warning: Rui HTTPDNS parameters missing; Debug only, system DNS fallback")

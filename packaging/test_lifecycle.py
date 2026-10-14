#!/usr/bin/env python3
"""Static lifecycle invariants independent of privileged hardware."""
from pathlib import Path
import re
r=Path('packaging/rc.d/ccu-modbus').read_text()
i=Path('packaging/update_script').read_text()
init=re.search(r'  init\)\n(.*?)\n    ;;',r,re.S).group(1)
start=re.search(r'  start\|""\)\n(.*?)\n    ;;',r,re.S).group(1)
assert 'start_daemon' not in init
assert 'start_daemon' in start
assert i.index('ARCH="$(uname -m)"') < i.index('"$RCD/$ADDON" stop')
assert i.index('[ -f "$SRC" ]') < i.index('"$RCD/$ADDON" stop')
assert 'if [ $EXITCODE -eq 0 ] && [ ! -f /etc/init.d/S00InstallAddon ]; then' in i
u=Path('packaging/www/update-check.cgi').read_text()
assert '/releases/latest' in u and 'releases/download/' in u
assert '/commits/main' not in u
print('PASS: early init, safe preflight, update restart, published releases')

#!/usr/bin/env python3
"""Regression guards for the CCU WebUI entrypoints (Tcl 8.2 compatible)."""
from pathlib import Path
import re
session=Path('packaging/lib/session.tcl').read_text()
index=Path('packaging/www/index.cgi').read_text()
api=Path('packaging/www/api.cgi').read_text()
js=Path('packaging/www/app.js').read_text()
assert 'regexp {^@([0-9a-zA-Z]{10})@$} $sid all sidnr' in session
assert 'check_session $sid' in index and 'check_session $sid' in api
assert 'window.CCU_MODBUS_SID' in index
assert 'if {$needPost && $method != "POST"}' in api
for action in ('save','test_connection','test_register'):
    assert re.search(r'\$action == "'+action+r'"[\s\S]{0,90}set needPost 1',api), action
assert 'encodeURIComponent(sid)' in js
assert 'regexp {^@' in session
assert '-w "\\n%{http_code}"' in api, 'CGI must retain backend HTTP status'
assert 'set httpStatus [expr {$statusCode + 0}]' in api
assert 'puts "Status: $httpStatus\\r"' in api
print('PASS: exact session format, protected pages, POST-only mutations, backend error status forwarding')

assert "update_start" in api and "check_session $sid" in api
assert "update_start" in api and "$method != \"POST\"" in api
assert "Auf Update prüfen" in index and "Update installieren" in index

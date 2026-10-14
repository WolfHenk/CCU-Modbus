#!/usr/bin/env python3
"""Isolated tests for version-list refresh after CCU-Modbus installation."""
import pathlib
import shlex
import subprocess
import tempfile
import time

ROOT = pathlib.Path(__file__).resolve().parent.parent
RCD = (ROOT / 'packaging/rc.d/ccu-modbus').read_text()
INSTALL = (ROOT / 'packaging/update_script').read_text()

start = RCD.index('\nrefresh_addon_versions() {\n') + 1
end = RCD.index('\n}\n', start) + 2
function = RCD[start:end]
assert function.startswith('refresh_addon_versions() {')


def wait_for(done, timeout=6):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if done():
            return
        time.sleep(0.06)
    raise AssertionError('background updater did not reach expected state')


def launch(tmp, *, check_exit=0, invokes=1, delay=0):
    pending = tmp / 'refresh.pending'
    checker = tmp / 'fake-checker.sh'
    output = tmp / 'check-invocations.log'
    lock = tmp / 'refresh.lock'
    checker.write_text(
        '#!/bin/sh\n'
        f'echo check >> {shlex.quote(str(output))}\n'
        'sleep 0.25\n'
        f'exit {check_exit}\n'
    )
    checker.chmod(0o755)
    content = (
        f'REFRESH_PENDING={shlex.quote(str(pending))}\n'
        f'REFRESH_CHECKER={shlex.quote(str(checker))}\n'
        f'REFRESH_LOCK={shlex.quote(str(lock))}\n'
        f'REFRESH_DELAY={delay}\n'
        + function + '\n'
        + 'refresh_addon_versions\n' * invokes
    )
    subprocess.run(['/bin/sh', '-c', content], check=True, timeout=4)
    return pending, output


def main():
    # Check critical install ordering: successful registration -> pending marker,
    # first-install reboot decision -> live-update trigger.
    assert INSTALL.index('"$BASE/bin/update_addon" ccu-modbus') < INSTALL.index('touch "$CONF/addon_updates_refresh.pending"')
    assert INSTALL.index('touch "$CONF/addon_updates_refresh.pending"') < INSTALL.index('if [ $NEW_INTERFACE -eq 1 ]; then')
    assert INSTALL.index('if [ $NEW_INTERFACE -eq 1 ]; then') < INSTALL.index('"$RCD/$ADDON" refresh-addon-versions')
    assert '"$BASE/bin/update_addon" ccu-modbus' in INSTALL and '|| exit 14' in INSTALL
    assert 'refresh_addon_versions' in RCD[RCD.index('  init)'):RCD.index('  stop)')]
    for script in ('packaging/update_script', 'packaging/rc.d/ccu-modbus'):
        subprocess.run(['/bin/sh', '-n', str(ROOT / script)], check=True)

    with tempfile.TemporaryDirectory(prefix='ccu-modbus-version-refresh-') as td:
        tmp = pathlib.Path(td)
        pending, output = launch(tmp)
        time.sleep(0.35)
        assert not output.exists(), 'ran without a pending update'
        pending.touch()
        pending, output = launch(tmp, invokes=2)
        wait_for(lambda: output.exists() and not pending.exists())
        assert output.read_text().splitlines() == ['check'], 'duplicate requests ran twice'

    with tempfile.TemporaryDirectory(prefix='ccu-modbus-version-refresh-error-') as td:
        tmp = pathlib.Path(td)
        pending = tmp / 'refresh.pending'
        pending.touch()
        _, output = launch(tmp, check_exit=9)
        wait_for(lambda: output.exists())
        time.sleep(0.4)
        assert pending.exists(), 'failed update check lost retry marker'
        _, output = launch(tmp, check_exit=0)
        wait_for(lambda: not pending.exists())
        assert output.read_text().splitlines() == ['check', 'check']

    print('PASS: no pending marker, success, concurrent trigger, error/retry, reboot/install ordering')


if __name__ == '__main__':
    main()

#!/bin/tclsh
#
# CCU Zusatzsoftware update check.
# The WebUI calls:
#   ?cmd=check_version&version=<installed>
#   ?cmd=download&version=<installed>
#
# While the repository is private, the unauthenticated CCU cannot read
# VERSION from GitHub and this returns n/a. The download redirect still
# leads the logged-in browser to the repository download folder.

set checkURL    "https://raw.githubusercontent.com/WolfHenk/CCU-Modbus/main/VERSION"
set downloadURL "https://github.com/WolfHenk/CCU-Modbus/tree/main/releases"

set cmd ""
if {[info exists env(QUERY_STRING)]} {
    foreach pair [split $env(QUERY_STRING) "&"] {
        if {[regexp {^([^=]*)=(.*)$} $pair dummy key value]} {
            if {$key == "cmd"} { set cmd $value }
        }
    }
}

if {$cmd == "download"} {
    puts -nonewline "Content-Type: text/html; charset=utf-8\r\n\r\n"
    puts -nonewline "<html><head><meta http-equiv='refresh' content='0; url=$downloadURL'></head><body></body></html>"
    exit 0
}

puts -nonewline "Content-Type: text/plain; charset=utf-8\r\n\r\n"

set newversion ""
catch {
    set newversion [string trim [exec /usr/bin/env curl -fsSL --max-time 10 $checkURL]]
}
if {$newversion != ""} {
    puts -nonewline $newversion
} else {
    puts -nonewline "n/a"
}

#!/bin/tclsh
#
# CCU Zusatzsoftware update check.
# The WebUI calls:
#   ?cmd=check_version&version=<installed>
#   ?cmd=download&version=<installed>
#
# Downloading a mutable raw.githubusercontent.com/main/.../latest URL is not
# safe immediately after a release because the CDN may serve the previous
# object for several minutes. Therefore resolve main to a commit SHA first and
# use that immutable SHA for VERSION and package download.

set repoApi "https://api.github.com/repos/WolfHenk/CCU-Modbus/commits/main"
set rawBase "https://raw.githubusercontent.com/WolfHenk/CCU-Modbus"

proc current_commit {repoApi} {
    set body ""
    if {[catch {set body [exec /usr/bin/env curl -fsSL --max-time 10 -H {Cache-Control: no-cache} -H {Accept: application/vnd.github+json} $repoApi]}]} {
        return ""
    }
    if {[regexp {"sha"[[:space:]]*:[[:space:]]*"([0-9a-fA-F]{40})"} $body dummy sha]} {
        return [string tolower $sha]
    }
    return ""
}

set cmd ""
if {[info exists env(QUERY_STRING)]} {
    foreach pair [split $env(QUERY_STRING) "&"] {
        if {[regexp {^([^=]*)=(.*)$} $pair dummy key value]} {
            if {$key == "cmd"} { set cmd $value }
        }
    }
}

set sha [current_commit $repoApi]

if {$cmd == "download"} {
    if {$sha == ""} {
        puts -nonewline "Status: 503 Service Unavailable\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n"
        puts -nonewline "Aktueller CCU-Modbus-Stand konnte nicht ermittelt werden."
        exit 0
    }
    set downloadURL "$rawBase/$sha/releases/ccu-modbus-latest.tar.gz"
    puts -nonewline "Status: 302 Found\r\n"
    puts -nonewline "Location: $downloadURL\r\n"
    puts -nonewline "Cache-Control: no-store\r\n"
    puts -nonewline "Content-Type: text/plain; charset=utf-8\r\n\r\n"
    exit 0
}

puts -nonewline "Content-Type: text/plain; charset=utf-8\r\nCache-Control: no-store\r\n\r\n"

if {$sha != ""} {
    set versionURL "$rawBase/$sha/VERSION"
    set newversion ""
    catch {
        set newversion [string trim [exec /usr/bin/env curl -fsSL --max-time 10 $versionURL]]
    }
    if {$newversion != ""} {
        puts -nonewline $newversion
        exit 0
    }
}
puts -nonewline "n/a"

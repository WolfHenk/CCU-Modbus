#!/bin/tclsh
# Only published releases are advertised; no mutable main branch checks.
set repoApi "https://api.github.com/repos/WolfHenk/CCU-Modbus/releases/latest"
set cmd ""
if {[info exists env(QUERY_STRING)]} {
    foreach pair [split $env(QUERY_STRING) "&"] {
        if {[regexp {^([^=]*)=(.*)$} $pair dummy key value]} {
            if {$key == "cmd"} { set cmd $value }
        }
    }
}
set body ""
catch {set body [exec /usr/bin/env curl -fsSL --max-time 10 -H {Accept: application/vnd.github+json} $repoApi]}
set version ""
set asset ""
regexp {"tag_name"[[:space:]]*:[[:space:]]*"v?([0-9]+\.[0-9]+\.[0-9]+)"} $body dummy version
# Never offer a release without a real installable asset.
regexp {"browser_download_url"[[:space:]]*:[[:space:]]*"(https://github.com/WolfHenk/CCU-Modbus/releases/download/[^" ]+\.tar\.gz)"} $body dummy asset
if {$cmd == "download"} {
    if {$version != "" && $asset != ""} {
        puts -nonewline "Status: 302 Found\r\nLocation: $asset\r\nCache-Control: no-store\r\n\r\n"
    } else {
        puts -nonewline "Status: 503 Service Unavailable\r\nContent-Type: text/plain\r\n\r\nRelease nicht verfuegbar"
    }
    exit 0
}
puts -nonewline "Content-Type: text/plain; charset=utf-8\r\nCache-Control: no-store\r\n\r\n"
if {$version != "" && $asset != ""} {puts -nonewline $version} else {puts -nonewline "n/a"}

#!/usr/bin/env tclsh
set script [file join [file dirname [info script]] bin update_addon]
set path [file join /tmp "ccu_modbus_addon_test_[pid].cfg"]
set ::env(CCU_MODBUS_HM_ADDONS_CFG) $path
proc readFile {path} {
    if {![file exists $path]} {return ""}
    set h [open $path r]; set data [read $h]; close $h; return $data
}
proc assert {test message} {if {![uplevel 1 [list expr $test]]} {error $message}}
set other [list CONFIG_URL /addons/ccu-jack/config.cgi CONFIG_DESCRIPTION [list en {<li>Jack</li>} de {<li>Jack</li>}] ID ccu-jack CONFIG_NAME CCU-Jack]
set initial [list ccu-jack $other cuxd [list ID cuxd CONFIG_NAME CUx-Daemon CONFIG_URL /addons/cuxd/index.ccc CONFIG_DESCRIPTION [list en {<li>CUxD</li>} de {<li>CUxD</li>}]]]
set f [open $path w];puts -nonewline $f $initial;close $f
set cmd [list tclsh $script ccu-modbus /addons/ccu-modbus/index.cgi CCU-Modbus {Modbus TCP interface} {Modbus-TCP-Schnittstelle}]
if {[catch {exec {*}$cmd} e]} {error "registration failed: $e"}
set changed [readFile $path]
assert {[string first $initial $changed] == 0} {old entries not byte-preserved on add}
array set data $changed
assert {[info exists data(ccu-jack)] && $data(ccu-jack) eq $other} {CCU-Jack was changed}
assert {[info exists data(ccu-modbus)]} {CCU-Modbus missing}
array set newentry $data(ccu-modbus)
assert {$newentry(CONFIG_URL) eq "/addons/ccu-modbus/index.cgi"} {URL missing}
assert {$newentry(CONFIG_NAME) eq "CCU-Modbus"} {name missing}
if {[catch {exec {*}$cmd} e]} {error "re-registration failed: $e"}
assert {[readFile $path] eq $changed} {idempotent add changed file}
if {[catch {exec tclsh $script delete ccu-modbus} e]} {error "delete failed: $e"}
array unset data
array set data [readFile $path]
assert {![info exists data(ccu-modbus)]} {uninstall left CCU-Modbus}
assert {$data(ccu-jack) eq $other} {uninstall damaged CCU-Jack}
if {[catch {exec tclsh $script delete ccu-modbus} e]} {error "second delete failed: $e"}
file delete -force -- $path
puts {PASS: registration, other addons preserved, idempotence, uninstall}

load tclrega.so

proc check_session {sid} {
    if {[regexp {@([0-9a-zA-Z]{10})@} $sid all sidnr]} {
        set res [lindex [rega_script "Write(system.GetSessionVarStr('$sidnr'));"] 1]
        if {$res != ""} { return 1 }
    }
    return 0
}

proc query_value {name} {
    if {![info exists ::env(QUERY_STRING)]} { return "" }
    foreach part [split $::env(QUERY_STRING) "&"] {
        set p [string first "=" $part]
        if {$p < 0} {
            set key $part
            set val ""
        } else {
            set key [string range $part 0 [expr {$p-1}]]
            set val [string range $part [expr {$p+1}] end]
        }
        if {$key == $name} { return $val }
    }
    return ""
}

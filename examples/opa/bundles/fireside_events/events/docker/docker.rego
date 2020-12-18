package falco_test

import input.output_fields as outf

default cname_match = false

cname_match {
    cname := outf["container.name"]
    cname == "tag-me-bad"
}

default priority_match = false

priority_match {
    pri := input.priority
    pri == "Notice"
}


default multi_match = false

multi_match {
    cname := outf["container.name"]
    cname == "tag-me-bad"
    uname := outf["user.name"]
    uname == "root"
}

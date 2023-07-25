#!/bin/bash
#
# Synopsis: 
#    run_parallel_experiment.sh <aida-db> <carmen-impl> <carmen-variant> <tosca-impl> <output-directory>

#  HardwareDescription() queries the hardware configuration of the server.
#  Requires Linux as an operating system, and hwinfo must be installed.
HardwareDescription()  {
    processor=`cat /proc/cpuinfo | grep "^model name" | head -n 1 | awk -F': ' '{print $2}'`
    memory=`free | grep "^Mem:" | awk '{printf("%dGB RAM\n", $2/1024/1024)}'` 
    disks=`hwinfo  --disk | grep Model | awk -F ': \"' '{if (NR > 1) printf(", "); printf("%s", substr($2,1,length($2)-1));}  END {printf("\n")}'`
    echo "$processor, $memory, disks: $disks"
}

# OperatingSystemDescription() queries the operating system of the server.
# Requires linux as an operating system.
OperatingSystemDescription() {
    cat /etc/*-release | awk -F'=\"' '{if ($1 == "DISTRIB_DESCRIPTION") print substr($2,1,length($2)-1)}'
}

## GitHash() queries the git hash of the Aida repository
# Requires a git command installed and the AIDA repository.
# TODO: Make Aida directory parameterisable
GitHash() {
   git rev-parse HEAD
}

## GoVersion() queries the GO version installed on the server
# Requires a local GO installation on the system.
GoVersion() {
    go version
}

# Run full parallel experiment
./build/aida-profile parallelisation --aida-db $1 --db-impl $2 --db-variant $3 --vm-impl=$4 --db-tmp /var/data/tmp-andrei 4564026 4664026 "$5/profile.db"

# Query the configuration
hw=`HardwareDescription`
os=`OperatingSystemDescription`
gh=`GitHash`
go=`GoVersion`
statedb="$2 $3"
vm="$4"

# Render R Markdown file
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', GoInfo='$go', VM='$vm', StateDB='$statedb'" -i "$5/profile.db" -f html_pdf -d "$5" -o parallel_experiment reports/parallel_experiment.rmd 

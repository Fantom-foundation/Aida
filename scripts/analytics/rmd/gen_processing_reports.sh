#!/bin/bash
# Name:
#    gen_processing_reports.sh -  script for generating the block processing reports
#
# Synopsis: 
#    gen_processing_report.sh <db-impl> <db-variant> <carmen-schema> <vm-impl> <output-dir>
#
# Description: 
#    Produces block processing reports in the HTML format.
#
#    The script requires a linux environment with installed commands hwinfo, free, git, go, sqlite3, and curl.
#    The script must be invoked in the main directory of the Aida repository.
# 

# check the number of command line arguments
if [ "$#" -ne 5 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
dbimpl=$1
dbvariant=$2
carmenschema=$3
vmimpl=$4
outputdir=$5

# logging 
log() {
    echo "$(date) $1" | tee -a "$outputdir/gen_processing_reports.log"
}

#  HardwareDescription() queries the hardware configuration of the server.
HardwareDescription()  {
    processor=`cat /proc/cpuinfo | grep "^model name" | head -n 1 | awk -F': ' '{print $2}'`
    memory=`free | grep "^Mem:" | awk '{printf("%dGB RAM\n", $2/1024/1024)}'` 
    disks=`hwinfo  --disk | grep Model | awk -F ': \"' '{if (NR > 1) printf(", "); printf("%s", substr($2,1,length($2)-1));}  END {printf("\n")}'`
    echo "$processor, $memory, disks: $disks"
}

# OperatingSystemDescription() queries the operating system of the server.
OperatingSystemDescription() {
    bash -lic 'source /etc/*-release; echo $DISTRIB_DESCRIPTION'
}

## GitHash() queries the git hash of the Aida repository
GitHash() {
   git rev-parse HEAD
}

## GoVersion() queries the Go version installed on the server
GoVersion() {
    go version
}

## Machine() queries machine name and IP address of server
Machine() {
    echo "`hostname`(`curl -s api.ipify.org`)"
}

# query the configuration
log "query configuration ..."
hw=`HardwareDescription`
os=`OperatingSystemDescription`
machine=`Machine`
gh=`GitHash`
go=`GoVersion`
statedb="$dbimpl($dbvariant $carmenschema)"

log "render s3 report ..."
/home/rapolt/dev/aida/scripts/analytics/rmd/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d /var/opera/Aida/tmp-rapolt/register/s5-f1.db -f html -o test.html -O $outputdir /home/rapolt/dev/aida/scripts/analytics/rmd/f1.rmd

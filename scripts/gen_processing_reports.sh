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
    echo "$(date) $1" | tee -a "$outputdir/block_processing.log"
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

# render R Markdown file
log "render block processing report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o block_processing.html -O $outputdir reports/block_processing.rmd

# produce mainnet report
log "render mainnet report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o mainnet_report.html -O $outputdir reports/mainnet_report.rmd

# produce wallet transfer report
log "render wallet transfer report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o wallet_transfer.html -O $outputdir reports/wallet_transfer.rmd

# produce contract creation report
log "render contract creation report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o contract_creation.html -O $outputdir reports/contract_creation.rmd

# produce contract execution report
log "render contract execution report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o contract_execution.html -O $outputdir reports/contract_execution.rmd

# produce parallel experiment report
log "render parallel experiment report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o parallel_experiment.html -O $outputdir reports/parallel_experiment.rmd

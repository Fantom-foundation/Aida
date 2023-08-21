#!/bin/bash
# Name:
#    gen_processing_reports.sh -  script for generating the block processing reports
#
# Synopsis: 
#    gen_processing_report.sh <db-impl> <db-variant> <vm-impl> <output-dir>
#
# Description: 
#    Produces block processing reports in the HTML format.
#
#    The script requires a linux environment with installed commands hwinfo, free, git, go, sqlite3, and curl.
#    The script must be invoked in the main directory of the Aida repository.
# 

# check the number of command line arguments
if [ "$#" -ne 4 ]; then
    echo "Invalid number of command line arguments supplied"
    exit 1
fi

# assign variables for command line arguments
dbimpl=$1
dbvariant=$2
vmimpl=$3
outputdir=$4

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

## Reduce data set
ReduceData() {
sqlite3 $1 << EOF
-- create view groupedBlockProfile to group block data for every 100,000 blocks
DROP VIEW IF EXISTS groupedBlockProfile;
CREATE VIEW groupedBlockProfile(block, tBlock, tCommit, numTx, speedup, gasBlock) AS
 SELECT block/100000, tBlock, tCommit, numTx, log(speedup), gasBlock FROM blockProfile;
-- aggregate block data
DROP TABLE IF EXISTS AggregatedBlockProfile;
CREATE TABLE AggregatedBlockProfile(block INTEGER, tBlock REAL, tCommit REAL, numTx REAL,  speedup REAL, gasBlock REAL, gps REAL, tps REAL);
INSERT INTO AggregatedBlockProfile SELECT min(block)*100000, avg(tBlock)/1e6, avg(tCommit)/1e3, avg(numTx), exp(avg(speedup)), avg(gasBlock), sum(gasBlock)/(sum(tBlock)/1e9), sum(numTx)/(sum(tBlock)/1e9) FROM groupedBlockProfile GROUP BY block;
DROP VIEW groupedBlockProfile;
-- create view groupedeTxProfile to group transaction data for every 1,000,000 transactions
DROP VIEW IF EXISTS  groupedTxProfile;
CREATE VIEW groupedTxProfile(tx, duration, gas) AS
 SELECT rowid/1000000, duration, gas FROM txProfile ORDER BY block ASC, tx ASC;
-- aggregate transaction data
DROP TABLE IF EXISTS aggregatedTxProfile;
CREATE TABLE aggregatedTxProfile(tx INTEGER, duration REAL, gas REAL);
INSERT INTO aggregatedTxProfile SELECT min(tx)*1000000, avg(duration)/1e3, avg(gas) FROM groupedTxProfile GROUP BY tx;
DROP VIEW groupedTxProfile;
EOF
}

# reduce dataset in sqlite3 (NB: R consumes too much memory/is too slow for the reduction)
log "reduce data set ..."
ReduceData $outputdir/profile.db

# query the configuration
log "query configuration ..."
hw=`HardwareDescription`
os=`OperatingSystemDescription`
machine=`Machine`
gh=`GitHash`
go=`GoVersion`
statedb="$dbimpl($dbvariant)"

# render R Markdown file
log "render block processing report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o block_processing.html -O $outputdir reports/block_processing.rmd

# produce parallel experiment report
log "render parallel experiment report ..."
./scripts/knit.R -p "GitHash='$gh', HwInfo='$hw', OsInfo='$os', Machine='$machine', GoInfo='$go', VM='$vmimpl', StateDB='$statedb'" \
                 -d "$outputdir/profile.db" -f html -o parallel_experiment.html -O $outputdir reports/parallel_experiment.rmd
